<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { GetNetworkDetail } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  const dispatch = createEventDispatcher();

  let data = null;
  let loading = true;
  let error = null;
  let interval = null;

  // Sparkline: track per-poll throughput deltas (bytesSent + bytesRecv difference)
  const HISTORY_MAX = 30;
  let throughputHistory = []; // bytes/s values
  let prevBytes = null;       // { sent, recv, ts } from last poll

  // ---- Helpers ----

  function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function formatBytesPerSec(bps) {
    if (!bps || bps === 0) return '0 B/s';
    const k = 1024;
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    const i = Math.floor(Math.log(bps) / Math.log(k));
    return parseFloat((bps / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function isActive(status) {
    return (status || '').toLowerCase() === 'active';
  }

  // ---- Sparkline ----
  const SPARK_W = 600;
  const SPARK_H = 64;

  function sparklinePath(pts, w, h) {
    if (!pts || pts.length < 2) return '';
    const maxV = Math.max(...pts, 1);
    const step = w / (HISTORY_MAX - 1);
    const pad = HISTORY_MAX - pts.length;
    return pts
      .map((v, i) => {
        const x = (i + pad) * step;
        const y = h - (v / maxV) * (h * 0.9);
        return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  }

  function sparklineArea(pts, w, h) {
    if (!pts || pts.length < 2) return '';
    const path = sparklinePath(pts, w, h);
    const maxV = Math.max(...pts, 1);
    const step = w / (HISTORY_MAX - 1);
    const pad = HISTORY_MAX - pts.length;
    const firstX = pad * step;
    const lastX = (pts.length - 1 + pad) * step;
    return `${path} L${lastX.toFixed(1)},${h} L${firstX.toFixed(1)},${h} Z`;
  }

  // ---- Data fetching ----
  async function fetchData() {
    try {
      const result = await GetNetworkDetail();
      data = result;
      error = null;

      // Compute throughput delta
      const now = Date.now();
      if (prevBytes) {
        const dtSec = (now - prevBytes.ts) / 1000;
        if (dtSec > 0) {
          const deltaSent = Math.max(0, result.bytesSent - prevBytes.sent);
          const deltaRecv = Math.max(0, result.bytesRecv - prevBytes.recv);
          const bps = (deltaSent + deltaRecv) / dtSec;
          throughputHistory = [...throughputHistory, bps].slice(-HISTORY_MAX);
        }
      }
      prevBytes = { sent: result.bytesSent, recv: result.bytesRecv, ts: now };
    } catch (e) {
      console.error('NetworkDetail fetch error:', e);
      error = 'Failed to load network data.';
    }
    loading = false;
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 3000);
  }

  function stopPolling() {
    if (interval) { clearInterval(interval); interval = null; }
  }

  $: if (visible) { startPolling(); } else { stopPolling(); }

  onDestroy(() => stopPolling());

  // Current throughput for display
  $: currentThroughput = throughputHistory.length
    ? throughputHistory[throughputHistory.length - 1]
    : 0;
</script>

<div class="network-detail">

  <!-- Header -->
  <div class="detail-header">
    <button class="back-btn" on:click={() => dispatch('back')} aria-label="Back">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M10 3L5 8l5 5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      Back
    </button>

    <div class="header-title">
      <h1>Network</h1>
      {#if data}
        <span class="header-subtitle">
          <span class="status-dot" class:active={isActive(data.status)} class:inactive={!isActive(data.status)}></span>
          {data.status || 'Unknown'}
        </span>
      {/if}
    </div>

    {#if data}
      <span class="status-badge" class:badge-active={isActive(data.status)} class:badge-inactive={!isActive(data.status)}>
        {data.status || 'Unknown'}
      </span>
    {/if}
  </div>

  {#if loading && !data}
    <div class="loading-state">Loading network data&hellip;</div>
  {:else if error && !data}
    <div class="error-state">{error}</div>
  {:else if data}
    <div class="content">

      <!-- Connection card -->
      <div class="card connection-card">
        <div class="conn-primary">
          <div class="conn-wifi-name">
            {data.wifiNetwork || data.interface || 'Network'}
          </div>
          <div class="conn-meta">
            <span class="conn-ip">{data.ipAddress || '—'}</span>
            {#if data.configMethod}
              <span class="config-badge">{data.configMethod}</span>
            {/if}
          </div>
        </div>
        <div class="conn-secondary">
          {#if data.hostname}
            <span class="conn-hostname">{data.hostname}</span>
          {/if}
          {#if data.externalIP}
            <span class="conn-ext-ip">External: {data.externalIP}</span>
          {/if}
          {#if data.wifiSecurity}
            <span class="conn-security">{data.wifiSecurity}</span>
          {/if}
        </div>
      </div>

      <!-- Traffic stats row -->
      <div class="traffic-section">
        <div class="traffic-row">
          <!-- Data Sent -->
          <div class="card traffic-card sent-card">
            <div class="traffic-direction">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <path d="M7 11V3M3 7l4-4 4 4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              Data Sent
            </div>
            <div class="traffic-value sent-color">{formatBytes(data.bytesSent)}</div>
            <div class="traffic-packets">{(data.packetsSent || 0).toLocaleString()} packets</div>
          </div>

          <!-- Data Received -->
          <div class="card traffic-card recv-card">
            <div class="traffic-direction">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true">
                <path d="M7 3v8M3 7l4 4 4-4" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
              </svg>
              Data Received
            </div>
            <div class="traffic-value recv-color">{formatBytes(data.bytesRecv)}</div>
            <div class="traffic-packets">{(data.packetsRecv || 0).toLocaleString()} packets</div>
          </div>
        </div>
      </div>

      <!-- Throughput sparkline -->
      <div class="card sparkline-card">
        <div class="sparkline-header">
          <div>
            <div class="card-label">Throughput History</div>
            <div class="sparkline-sub">Combined sent + received &mdash; last {throughputHistory.length} readings</div>
          </div>
          <div class="sparkline-cur">
            {#if throughputHistory.length > 0}
              {formatBytesPerSec(currentThroughput)}
            {:else}
              <span class="text-muted">Collecting&hellip;</span>
            {/if}
          </div>
        </div>
        <div class="sparkline-wrap">
          {#if throughputHistory.length >= 2}
            <svg
              width="100%"
              height={SPARK_H}
              viewBox="0 0 {SPARK_W} {SPARK_H}"
              preserveAspectRatio="none"
              aria-label="Network throughput sparkline"
            >
              <defs>
                <linearGradient id="net-spark-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.3"/>
                  <stop offset="100%" stop-color="var(--accent)" stop-opacity="0"/>
                </linearGradient>
              </defs>
              <path
                d={sparklineArea(throughputHistory, SPARK_W, SPARK_H)}
                fill="url(#net-spark-grad)"
              />
              <path
                d={sparklinePath(throughputHistory, SPARK_W, SPARK_H)}
                fill="none"
                stroke="var(--accent)"
                stroke-width="2"
                stroke-linejoin="round"
                stroke-linecap="round"
              />
            </svg>
          {:else}
            <div class="spark-placeholder">Collecting throughput data&hellip;</div>
          {/if}
        </div>
        <div class="sparkline-footer">
          <span class="spark-tick">Older</span>
          <span class="spark-tick">3s interval</span>
          <span class="spark-tick">Now</span>
        </div>
      </div>

      <!-- Network details grid -->
      <div class="card details-card">
        <div class="card-label">Network Details</div>
        <div class="details-grid">

          <div class="detail-item">
            <div class="detail-label">Interface</div>
            <div class="detail-value mono">{data.interface || '—'}</div>
          </div>

          <div class="detail-item">
            <div class="detail-label">MAC Address</div>
            <div class="detail-value mono">{data.macAddress || '—'}</div>
          </div>

          <div class="detail-item">
            <div class="detail-label">Subnet Mask</div>
            <div class="detail-value mono">{data.subnetMask || '—'}</div>
          </div>

          <div class="detail-item">
            <div class="detail-label">Router / Gateway</div>
            <div class="detail-value mono">{data.router || '—'}</div>
          </div>

          <div class="detail-item">
            <div class="detail-label">IPv6 Address</div>
            <div class="detail-value mono ipv6">{data.ipv6Address || '—'}</div>
          </div>

          <div class="detail-item">
            <div class="detail-label">Hostname</div>
            <div class="detail-value mono">{data.hostname || '—'}</div>
          </div>

          {#if data.linkSpeed}
            <div class="detail-item">
              <div class="detail-label">Link Speed</div>
              <div class="detail-value">{data.linkSpeed}</div>
            </div>
          {/if}

          {#if data.externalIP}
            <div class="detail-item">
              <div class="detail-label">External IP</div>
              <div class="detail-value mono">{data.externalIP}</div>
            </div>
          {/if}

          <!-- DNS Servers spans full row -->
          <div class="detail-item detail-item-dns">
            <div class="detail-label">DNS Servers</div>
            <div class="dns-pills">
              {#if data.dns && data.dns.length > 0}
                {#each data.dns as server}
                  <span class="dns-pill">{server}</span>
                {/each}
              {:else}
                <span class="detail-value">—</span>
              {/if}
            </div>
          </div>

        </div>
      </div>

    </div>
  {/if}
</div>

<style>
  /* ---- Layout ---- */
  .network-detail {
    padding: 0 32px 32px;
    height: 100%;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }

  /* ---- Header ---- */
  .detail-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 28px;
  }

  .back-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 7px 14px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    color: var(--text-secondary);
    font-size: 13px;
    font-weight: 500;
    transition: all var(--transition);
    flex-shrink: 0;
  }

  .back-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
    border-color: var(--accent);
  }

  .header-title {
    flex: 1;
  }

  .header-title h1 {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.5px;
    line-height: 1.1;
  }

  .header-subtitle {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 13px;
    color: var(--text-secondary);
    margin-top: 4px;
  }

  .status-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .status-dot.active   { background: var(--green); box-shadow: 0 0 0 2px var(--green-dim); }
  .status-dot.inactive { background: var(--red);   box-shadow: 0 0 0 2px var(--red-dim);   }

  .status-badge {
    padding: 5px 14px;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.3px;
    flex-shrink: 0;
  }

  .badge-active   { background: var(--green-dim); color: var(--green); }
  .badge-inactive { background: var(--red-dim);   color: var(--red);   }

  /* ---- Loading / Error ---- */
  .loading-state,
  .error-state {
    color: var(--text-secondary);
    padding: 60px 0;
    text-align: center;
  }

  .error-state { color: var(--red); }

  /* ---- Content ---- */
  .content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  /* ---- Cards ---- */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px 24px;
  }

  .card-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
    margin-bottom: 16px;
  }

  /* ---- Connection card ---- */
  .connection-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    flex-wrap: wrap;
  }

  .conn-primary {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .conn-wifi-name {
    font-size: 28px;
    font-weight: 700;
    letter-spacing: -0.5px;
    color: var(--text-primary);
    line-height: 1;
  }

  .conn-meta {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .conn-ip {
    font-size: 15px;
    font-weight: 500;
    color: var(--text-secondary);
    font-family: 'SF Mono', 'Fira Code', 'Menlo', monospace;
  }

  .config-badge {
    padding: 3px 10px;
    border-radius: 20px;
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    background: var(--accent-dim);
    color: var(--accent);
  }

  .conn-secondary {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 6px;
    text-align: right;
  }

  .conn-hostname {
    font-size: 13px;
    color: var(--text-secondary);
    font-family: 'SF Mono', 'Fira Code', 'Menlo', monospace;
  }

  .conn-ext-ip {
    font-size: 12px;
    color: var(--text-muted);
    font-family: 'SF Mono', 'Fira Code', 'Menlo', monospace;
  }

  .conn-security {
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 10px;
    background: var(--blue-dim);
    color: var(--blue);
    font-weight: 600;
  }

  /* ---- Traffic section ---- */
  .traffic-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .traffic-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .traffic-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 20px 22px;
  }

  .traffic-direction {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
  }

  .sent-card .traffic-direction { color: var(--blue); }
  .recv-card .traffic-direction { color: var(--green); }

  .traffic-value {
    font-size: 30px;
    font-weight: 700;
    letter-spacing: -0.5px;
    line-height: 1;
  }

  .sent-color { color: var(--blue); }
  .recv-color { color: var(--green); }

  .traffic-packets {
    font-size: 12px;
    color: var(--text-muted);
    font-variant-numeric: tabular-nums;
  }

  /* ---- Sparkline card ---- */
  .sparkline-card {
    padding: 20px 24px;
  }

  .sparkline-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 12px;
  }

  .sparkline-header .card-label {
    margin-bottom: 4px;
  }

  .sparkline-sub {
    font-size: 12px;
    color: var(--text-muted);
  }

  .sparkline-cur {
    font-size: 18px;
    font-weight: 700;
    color: var(--accent);
    white-space: nowrap;
    flex-shrink: 0;
  }

  .text-muted {
    color: var(--text-muted);
    font-size: 14px;
    font-weight: 400;
  }

  .sparkline-wrap {
    height: 64px;
    border-radius: var(--radius-sm);
    overflow: hidden;
    background: var(--bg-secondary);
  }

  .sparkline-wrap svg {
    display: block;
  }

  .spark-placeholder {
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    color: var(--text-muted);
  }

  .sparkline-footer {
    display: flex;
    justify-content: space-between;
    margin-top: 8px;
  }

  .spark-tick {
    font-size: 11px;
    color: var(--text-muted);
  }

  /* ---- Details grid ---- */
  .details-card {
    /* inherits .card */
  }

  .details-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }

  .detail-item {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 5px;
    min-width: 0;
  }

  .detail-item-dns {
    grid-column: 1 / -1;
  }

  .detail-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .detail-value {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    word-break: break-all;
    line-height: 1.4;
  }

  .detail-value.mono {
    font-family: 'SF Mono', 'Fira Code', 'Menlo', monospace;
    font-size: 13px;
    font-weight: 500;
  }

  .detail-value.ipv6 {
    font-size: 12px;
    letter-spacing: 0.2px;
  }

  /* ---- DNS pills ---- */
  .dns-pills {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    padding-top: 2px;
  }

  .dns-pill {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 20px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border);
    color: var(--text-primary);
    font-size: 12px;
    font-weight: 500;
    font-family: 'SF Mono', 'Fira Code', 'Menlo', monospace;
    letter-spacing: 0.2px;
  }

  /* ---- Responsive ---- */
  @media (max-width: 860px) {
    .details-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .traffic-value {
      font-size: 24px;
    }
  }

  @media (max-width: 600px) {
    .network-detail {
      padding: 0 16px 24px;
    }

    .connection-card {
      flex-direction: column;
      align-items: flex-start;
    }

    .conn-secondary {
      align-items: flex-start;
      text-align: left;
    }

    .traffic-row {
      grid-template-columns: 1fr;
    }

    .details-grid {
      grid-template-columns: 1fr;
    }

    .conn-wifi-name {
      font-size: 22px;
    }
  }
</style>
