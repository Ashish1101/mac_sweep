<script>
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import { GetDiskDetail } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  const dispatch = createEventDispatcher();

  let data = null;
  let loading = true;
  let error = null;
  let interval = null;

  // SVG ring chart dimensions
  const SIZE = 200;
  const STROKE = 18;
  const R = (SIZE / 2) - (STROKE / 2);
  const CIRCUMFERENCE = 2 * Math.PI * R;

  function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function usageColor(pct) {
    if (pct > 90) return 'var(--red)';
    if (pct > 70) return 'var(--yellow)';
    return 'var(--accent)';
  }

  function usageColorDim(pct) {
    if (pct > 90) return 'var(--red-dim)';
    if (pct > 70) return 'var(--yellow-dim)';
    return 'var(--accent-dim)';
  }

  $: usedDash   = data ? (data.usage / 100) * CIRCUMFERENCE : 0;
  $: freeDash   = data ? CIRCUMFERENCE - usedDash : CIRCUMFERENCE;
  $: ringColor  = data ? usageColor(data.usage) : 'var(--accent)';

  async function fetchData() {
    try {
      data = await GetDiskDetail();
      error = null;
    } catch (e) {
      error = 'Failed to load disk information.';
      console.error('DiskDetail fetch error:', e);
    }
    loading = false;
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 10000);
  }

  function stopPolling() {
    if (interval) { clearInterval(interval); interval = null; }
  }

  $: if (visible) { startPolling(); } else { stopPolling(); }

  onDestroy(() => stopPolling());
</script>

<div class="disk-detail">
  <!-- Header -->
  <div class="page-header">
    <button class="back-btn" on:click={() => dispatch('back')}>
      &#8592; Back
    </button>
    <div class="header-text">
      <h1>Disk</h1>
      {#if data}
        <p class="subtitle">{data.volumeName}</p>
      {:else}
        <p class="subtitle">Storage overview</p>
      {/if}
    </div>
  </div>

  {#if loading}
    <div class="loading">Loading disk information...</div>
  {:else if error}
    <div class="error-msg">{error}</div>
  {:else if data}
    <div class="content">

      <!-- Top section: ring + usage bar side by side -->
      <div class="top-section">

        <!-- Usage Ring -->
        <div class="card ring-card">
          <div class="card-title">Usage</div>
          <div class="ring-wrap">
            <svg width={SIZE} height={SIZE} viewBox="0 0 {SIZE} {SIZE}">
              <!-- Track -->
              <circle
                cx={SIZE / 2}
                cy={SIZE / 2}
                r={R}
                fill="none"
                stroke="var(--bg-tertiary)"
                stroke-width={STROKE}
              />
              <!-- Used arc -->
              <circle
                cx={SIZE / 2}
                cy={SIZE / 2}
                r={R}
                fill="none"
                stroke={ringColor}
                stroke-width={STROKE}
                stroke-dasharray="{usedDash} {freeDash}"
                stroke-dashoffset={CIRCUMFERENCE / 4}
                stroke-linecap="round"
                style="transition: stroke-dasharray 0.6s ease;"
              />
              <!-- Center text -->
              <text
                x={SIZE / 2}
                y={SIZE / 2 - 8}
                text-anchor="middle"
                dominant-baseline="middle"
                fill={ringColor}
                font-size="28"
                font-weight="700"
                font-family="-apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif"
              >{data.usage.toFixed(1)}%</text>
              <text
                x={SIZE / 2}
                y={SIZE / 2 + 18}
                text-anchor="middle"
                dominant-baseline="middle"
                fill="var(--text-secondary)"
                font-size="11"
                font-family="-apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif"
              >used</text>
            </svg>
          </div>
          <div class="ring-legend">
            <div class="legend-item">
              <span class="legend-dot" style="background: {ringColor}"></span>
              <span class="legend-label">Used</span>
              <span class="legend-val">{formatBytes(data.used)}</span>
            </div>
            <div class="legend-item">
              <span class="legend-dot" style="background: var(--bg-tertiary)"></span>
              <span class="legend-label">Available</span>
              <span class="legend-val">{formatBytes(data.available)}</span>
            </div>
          </div>
        </div>

        <!-- Usage bar + I/O stats stacked -->
        <div class="right-col">

          <!-- Usage Bar -->
          <div class="card">
            <div class="card-title">Storage Breakdown</div>
            <div class="usage-bar-wrap">
              <div class="usage-bar-track">
                <div
                  class="usage-bar-fill"
                  style="width: {Math.min(data.usage, 100)}%; background: {ringColor};"
                ></div>
              </div>
              <div class="usage-bar-labels">
                <div class="bar-label-group">
                  <span class="bar-dot" style="background: {ringColor}"></span>
                  <span class="bar-label-text">Used — {formatBytes(data.used)}</span>
                </div>
                <div class="bar-label-group">
                  <span class="bar-dot" style="background: var(--bg-tertiary)"></span>
                  <span class="bar-label-text">Available — {formatBytes(data.available)}</span>
                </div>
              </div>
              <div class="total-line">
                Total capacity: <strong>{formatBytes(data.total)}</strong>
              </div>
            </div>
          </div>

          <!-- I/O Stats -->
          <div class="card io-card">
            <div class="card-title">I/O Operations</div>
            <div class="io-grid">
              <div class="io-stat">
                <div class="io-label">Read Ops</div>
                <div class="io-value blue">{(data.readOps ?? 0).toLocaleString()}</div>
              </div>
              <div class="io-stat">
                <div class="io-label">Write Ops</div>
                <div class="io-value accent">{(data.writeOps ?? 0).toLocaleString()}</div>
              </div>
            </div>
          </div>

        </div>
      </div>

      <!-- Info Cards Grid -->
      <div class="card info-section">
        <div class="card-title">Volume Details</div>
        <div class="info-grid">
          <div class="info-card">
            <div class="info-label">Volume Name</div>
            <div class="info-value">{data.volumeName}</div>
          </div>
          <div class="info-card">
            <div class="info-label">Mount Point</div>
            <div class="info-value mono">{data.mountPath}</div>
          </div>
          <div class="info-card">
            <div class="info-label">Filesystem</div>
            <div class="info-value">{(data.fsType ?? '—').toUpperCase()}</div>
          </div>
          <div class="info-card">
            <div class="info-label">Disk Type</div>
            <div class="info-value type-badge" class:ssd={data.diskType === 'SSD'} class:hdd={data.diskType === 'HDD'}>
              {data.diskType}
            </div>
          </div>
          <div class="info-card">
            <div class="info-label">Used</div>
            <div class="info-value" style="color: {ringColor}">{formatBytes(data.used)}</div>
          </div>
          <div class="info-card">
            <div class="info-label">Available</div>
            <div class="info-value" style="color: var(--green)">{formatBytes(data.available)}</div>
          </div>
          <div class="info-card">
            <div class="info-label">Total</div>
            <div class="info-value">{formatBytes(data.total)}</div>
          </div>
        </div>
      </div>

    </div>
  {/if}
</div>

<style>
  .disk-detail {
    padding: 0 32px 32px;
    height: 100%;
    overflow-y: auto;
  }

  /* Header */
  .page-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 28px;
  }

  .back-btn {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
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

  .header-text h1 {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }

  .subtitle {
    color: var(--text-secondary);
    margin-top: 3px;
    font-size: 13px;
  }

  /* Loading / error */
  .loading,
  .error-msg {
    color: var(--text-secondary);
    padding: 60px 0;
    text-align: center;
    font-size: 14px;
  }

  .error-msg {
    color: var(--red);
  }

  /* Layout */
  .content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .top-section {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 20px;
    align-items: stretch;
  }

  .right-col {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  /* Card base */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px 24px;
  }

  .card-title {
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-secondary);
    margin-bottom: 16px;
  }

  /* Ring card */
  .ring-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    min-width: 240px;
  }

  .ring-wrap {
    margin-bottom: 16px;
  }

  .ring-legend {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .legend-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .legend-label {
    font-size: 12px;
    color: var(--text-secondary);
    flex: 1;
  }

  .legend-val {
    font-size: 13px;
    font-weight: 600;
    color: var(--text-primary);
  }

  /* Usage bar */
  .usage-bar-wrap {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .usage-bar-track {
    height: 12px;
    background: var(--bg-tertiary);
    border-radius: 6px;
    overflow: hidden;
  }

  .usage-bar-fill {
    height: 100%;
    border-radius: 6px;
    transition: width 0.6s ease;
  }

  .usage-bar-labels {
    display: flex;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 8px;
  }

  .bar-label-group {
    display: flex;
    align-items: center;
    gap: 6px;
  }

  .bar-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .bar-label-text {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .total-line {
    font-size: 12px;
    color: var(--text-muted);
    border-top: 1px solid var(--border);
    padding-top: 10px;
  }

  .total-line strong {
    color: var(--text-primary);
    font-weight: 600;
  }

  /* I/O stats */
  .io-card {
    flex: 1;
  }

  .io-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .io-stat {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 16px 18px;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .io-label {
    font-size: 12px;
    color: var(--text-secondary);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.4px;
  }

  .io-value {
    font-size: 26px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }

  .io-value.blue   { color: var(--blue); }
  .io-value.accent { color: var(--accent); }

  /* Info grid */
  .info-section {
    /* inherits .card */
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
  }

  .info-card {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: 14px 16px;
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .info-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .info-value {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-primary);
    word-break: break-all;
  }

  .info-value.mono {
    font-family: 'SF Mono', 'Fira Code', 'Menlo', monospace;
    font-size: 13px;
  }

  .type-badge {
    display: inline-block;
    padding: 2px 10px;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.5px;
    width: fit-content;
  }

  .type-badge.ssd {
    background: var(--accent-dim);
    color: var(--accent);
  }

  .type-badge.hdd {
    background: var(--blue-dim);
    color: var(--blue);
  }

  /* Responsive */
  @media (max-width: 800px) {
    .top-section {
      grid-template-columns: 1fr;
    }

    .ring-card {
      min-width: unset;
    }

    .info-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 500px) {
    .disk-detail {
      padding: 0 16px 24px;
    }

    .info-grid {
      grid-template-columns: 1fr;
    }

    .io-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
