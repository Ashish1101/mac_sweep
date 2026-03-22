<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { GetMemoryDetail } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  const dispatch = createEventDispatcher();

  let data = null;
  let loading = true;
  let interval = null;
  let history = []; // last 30 usage % readings
  const MAX_HISTORY = 30;

  // ---- Helpers ----

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function formatBytesShort(bytes) {
    if (!bytes) return '0';
    const gb = bytes / (1024 * 1024 * 1024);
    if (gb >= 1) return gb % 1 === 0 ? gb + ' GB' : gb.toFixed(1) + ' GB';
    const mb = bytes / (1024 * 1024);
    if (mb >= 1) return mb % 1 === 0 ? mb + ' MB' : mb.toFixed(0) + ' MB';
    return formatBytes(bytes);
  }

  function pressureColor(pressure) {
    if (!pressure) return 'var(--text-muted)';
    const p = pressure.toLowerCase();
    if (p === 'normal') return 'var(--green)';
    if (p === 'warning' || p === 'warn') return 'var(--yellow)';
    if (p === 'critical') return 'var(--red)';
    return 'var(--text-muted)';
  }

  function pressureBg(pressure) {
    if (!pressure) return 'var(--bg-tertiary)';
    const p = pressure.toLowerCase();
    if (p === 'normal') return 'var(--green-dim)';
    if (p === 'warning' || p === 'warn') return 'var(--yellow-dim)';
    if (p === 'critical') return 'var(--red-dim)';
    return 'var(--bg-tertiary)';
  }

  // ---- Ring chart (SVG donut) ----
  // Returns SVG arc path for a donut segment
  function ringArc(cx, cy, r, startPct, endPct) {
    const startAngle = (startPct / 100) * 2 * Math.PI - Math.PI / 2;
    const endAngle   = (endPct   / 100) * 2 * Math.PI - Math.PI / 2;
    const x1 = cx + r * Math.cos(startAngle);
    const y1 = cy + r * Math.sin(startAngle);
    const x2 = cx + r * Math.cos(endAngle);
    const y2 = cy + r * Math.sin(endAngle);
    const large = (endPct - startPct) > 50 ? 1 : 0;
    return `M ${x1} ${y1} A ${r} ${r} 0 ${large} 1 ${x2} ${y2}`;
  }

  // ---- Sparkline ----
  function sparklinePath(data, w, h) {
    if (!data || data.length < 2) return '';
    const max = Math.max(...data, 100);
    const step = w / (data.length - 1);
    return data
      .map((v, i) => {
        const x = i * step;
        const y = h - (v / max) * h;
        return `${i === 0 ? 'M' : 'L'} ${x.toFixed(2)} ${y.toFixed(2)}`;
      })
      .join(' ');
  }

  function sparklineFill(data, w, h) {
    if (!data || data.length < 2) return '';
    const path = sparklinePath(data, w, h);
    return `${path} L ${w} ${h} L 0 ${h} Z`;
  }

  // ---- Data fetching ----
  async function fetchData() {
    try {
      data = await GetMemoryDetail();
      history = [...history, data.usage].slice(-MAX_HISTORY);
    } catch (e) {
      console.error('MemoryDetail fetch error:', e);
    }
    loading = false;
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 2000);
  }

  function stopPolling() {
    if (interval) { clearInterval(interval); interval = null; }
  }

  $: if (visible) { startPolling(); } else { stopPolling(); }

  onDestroy(() => stopPolling());

  // ---- Derived layout values ----
  // Ring chart
  const CX = 100, CY = 100, OUTER_R = 80, INNER_R = 52;
  const STROKE = OUTER_R - INNER_R; // 28

  $: usedPct   = data ? data.usage : 0;
  $: availPct  = 100 - usedPct;

  // Breakdown bar segments (Active, Wired, Compressed, Inactive, Free)
  $: total = data && data.total > 0 ? data.total : 1;
  $: segments = data ? [
    { label: 'Active',     value: data.active ?? 0,     color: 'var(--blue)',   pct: ((data.active ?? 0)     / total) * 100 },
    { label: 'Wired',      value: data.wired ?? 0,      color: 'var(--yellow)', pct: ((data.wired ?? 0)      / total) * 100 },
    { label: 'Compressed', value: data.compressed ?? 0, color: 'var(--accent)', pct: ((data.compressed ?? 0) / total) * 100 },
    { label: 'Inactive',   value: data.inactive ?? 0,   color: 'var(--text-muted)', pct: ((data.inactive ?? 0) / total) * 100 },
    { label: 'Free',       value: data.free ?? 0,       color: 'var(--bg-tertiary)', pct: ((data.free ?? 0)   / total) * 100 },
  ] : [];
</script>

<div class="memory-detail">
  <!-- Header row -->
  <div class="detail-header">
    <button class="back-btn" on:click={() => dispatch('back')} aria-label="Back">
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M10 3L5 8l5 5" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      Back
    </button>

    <div class="header-title">
      <h1>Memory</h1>
      {#if data}
        <span class="subtitle">{formatBytesShort(data.total)} RAM</span>
      {/if}
    </div>

    {#if data}
      <span
        class="pressure-badge"
        style="color:{pressureColor(data.pressure)};background:{pressureBg(data.pressure)}"
      >
        {data.pressure ?? '—'}
      </span>
    {/if}
  </div>

  {#if loading}
    <div class="loading">Loading memory data...</div>
  {:else if data}
    <div class="content">

      <!-- Top row: ring chart + breakdown -->
      <div class="top-row">

        <!-- Usage ring -->
        <div class="card ring-card">
          <div class="card-label">Memory Usage</div>
          <div class="ring-wrap">
            <svg width="200" height="200" viewBox="0 0 200 200">
              <!-- Track -->
              <circle
                cx={CX} cy={CY} r={(OUTER_R + INNER_R) / 2}
                fill="none"
                stroke="var(--bg-tertiary)"
                stroke-width={STROKE}
              />
              <!-- Used arc -->
              {#if usedPct > 0}
                <circle
                  cx={CX} cy={CY} r={(OUTER_R + INNER_R) / 2}
                  fill="none"
                  stroke="var(--accent)"
                  stroke-width={STROKE}
                  stroke-dasharray="{(usedPct / 100) * 2 * Math.PI * ((OUTER_R + INNER_R) / 2)} {2 * Math.PI * ((OUTER_R + INNER_R) / 2)}"
                  stroke-dashoffset="{2 * Math.PI * ((OUTER_R + INNER_R) / 2) * 0.25}"
                  stroke-linecap="round"
                />
              {/if}
              <!-- Center text -->
              <text x={CX} y={CY - 8} text-anchor="middle" class="ring-pct">{usedPct.toFixed(1)}%</text>
              <text x={CX} y={CY + 14} text-anchor="middle" class="ring-label">Used</text>
            </svg>
          </div>
          <div class="ring-legend">
            <div class="legend-item">
              <span class="legend-dot" style="background:var(--accent)"></span>
              <span class="legend-text">Used — {formatBytes(data.used)}</span>
            </div>
            <div class="legend-item">
              <span class="legend-dot" style="background:var(--bg-tertiary)"></span>
              <span class="legend-text">Available — {formatBytes(data.available)}</span>
            </div>
          </div>
        </div>

        <!-- Breakdown stacked bar -->
        <div class="card breakdown-card">
          <div class="card-label">Memory Breakdown</div>
          <div class="stacked-bar">
            {#each segments as seg}
              {#if seg.pct > 0.5}
                <div
                  class="bar-seg"
                  style="width:{seg.pct.toFixed(2)}%;background:{seg.color}"
                  title="{seg.label}: {formatBytes(seg.value)} ({seg.pct.toFixed(1)}%)"
                ></div>
              {/if}
            {/each}
          </div>
          <div class="breakdown-legend">
            {#each segments as seg}
              <div class="breakdown-row">
                <span class="bd-dot" style="background:{seg.color}"></span>
                <span class="bd-label">{seg.label}</span>
                <span class="bd-value">{formatBytes(seg.value)}</span>
                <span class="bd-pct">{seg.pct.toFixed(1)}%</span>
              </div>
            {/each}
          </div>
        </div>
      </div>

      <!-- Stats grid -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">App Memory</div>
          <div class="stat-value">{formatBytes(data.appMemory)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Wired Memory</div>
          <div class="stat-value" style="color:var(--yellow)">{formatBytes(data.wired)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Compressed</div>
          <div class="stat-value" style="color:var(--accent)">{formatBytes(data.compressed)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Cached Files</div>
          <div class="stat-value" style="color:var(--blue)">{formatBytes(data.cachedFiles)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Swap Used</div>
          <div class="stat-value">{formatBytes(data.swapUsed)}</div>
          <div class="stat-sub">of {formatBytes(data.swapTotal)}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Free</div>
          <div class="stat-value" style="color:var(--green)">{formatBytes(data.free)}</div>
        </div>
      </div>

      <!-- Sparkline history -->
      <div class="card sparkline-card">
        <div class="sparkline-header">
          <span class="card-label">Usage History</span>
          <span class="sparkline-cur">{usedPct.toFixed(1)}%</span>
        </div>
        <div class="sparkline-wrap">
          {#if history.length >= 2}
            <svg
              width="100%"
              height="72"
              viewBox="0 0 600 72"
              preserveAspectRatio="none"
            >
              <defs>
                <linearGradient id="mem-grad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.35"/>
                  <stop offset="100%" stop-color="var(--accent)" stop-opacity="0"/>
                </linearGradient>
              </defs>
              <!-- Fill area -->
              <path
                d={sparklineFill(history, 600, 72)}
                fill="url(#mem-grad)"
              />
              <!-- Line -->
              <path
                d={sparklinePath(history, 600, 72)}
                fill="none"
                stroke="var(--accent)"
                stroke-width="2"
                stroke-linejoin="round"
                stroke-linecap="round"
              />
            </svg>
          {:else}
            <div class="sparkline-empty">Collecting data...</div>
          {/if}
        </div>
        <div class="sparkline-footer">
          <span class="sparkline-tick">0%</span>
          <span class="sparkline-tick">Last {history.length} readings</span>
          <span class="sparkline-tick">100%</span>
        </div>
      </div>

    </div>
  {/if}
</div>

<style>
  .memory-detail {
    padding: 0 32px 32px;
    height: 100%;
    overflow-y: auto;
  }

  /* Header */
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

  .subtitle {
    display: block;
    color: var(--text-secondary);
    font-size: 13px;
    margin-top: 3px;
  }

  .pressure-badge {
    padding: 5px 14px;
    border-radius: 20px;
    font-size: 12px;
    font-weight: 600;
    letter-spacing: 0.3px;
    flex-shrink: 0;
  }

  /* Loading */
  .loading {
    color: var(--text-secondary);
    text-align: center;
    padding: 60px 0;
  }

  /* Content layout */
  .content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .top-row {
    display: grid;
    grid-template-columns: 240px 1fr;
    gap: 20px;
    align-items: start;
  }

  /* Cards */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px;
  }

  .card-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
    margin-bottom: 16px;
  }

  /* Ring chart card */
  .ring-card {
    display: flex;
    flex-direction: column;
    align-items: center;
  }

  .ring-card .card-label {
    align-self: flex-start;
  }

  .ring-wrap {
    display: flex;
    justify-content: center;
    margin-bottom: 16px;
  }

  .ring-pct {
    font-size: 26px;
    font-weight: 700;
    fill: var(--text-primary);
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif;
  }

  .ring-label {
    font-size: 12px;
    fill: var(--text-secondary);
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif;
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

  .legend-text {
    font-size: 12px;
    color: var(--text-secondary);
  }

  /* Breakdown card */
  .breakdown-card {
    display: flex;
    flex-direction: column;
  }

  .stacked-bar {
    display: flex;
    height: 20px;
    border-radius: 6px;
    overflow: hidden;
    background: var(--bg-tertiary);
    margin-bottom: 20px;
    gap: 1px;
  }

  .bar-seg {
    height: 100%;
    transition: width 0.5s ease;
    min-width: 2px;
  }

  .breakdown-legend {
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .breakdown-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .bd-dot {
    width: 10px;
    height: 10px;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .bd-label {
    flex: 1;
    font-size: 13px;
    color: var(--text-secondary);
  }

  .bd-value {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-primary);
    min-width: 72px;
    text-align: right;
  }

  .bd-pct {
    font-size: 12px;
    color: var(--text-muted);
    min-width: 44px;
    text-align: right;
  }

  /* Stats grid */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
  }

  .stat-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 18px 20px;
  }

  .stat-label {
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-muted);
    margin-bottom: 8px;
  }

  .stat-value {
    font-size: 22px;
    font-weight: 700;
    line-height: 1;
  }

  .stat-sub {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  /* Sparkline card */
  .sparkline-card {
    padding: 20px;
  }

  .sparkline-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .sparkline-header .card-label {
    margin-bottom: 0;
  }

  .sparkline-cur {
    font-size: 18px;
    font-weight: 700;
    color: var(--accent);
  }

  .sparkline-wrap {
    height: 72px;
    border-radius: var(--radius-sm);
    overflow: hidden;
    background: var(--bg-secondary);
  }

  .sparkline-wrap svg {
    display: block;
  }

  .sparkline-empty {
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

  .sparkline-tick {
    font-size: 11px;
    color: var(--text-muted);
  }

  @media (max-width: 860px) {
    .top-row {
      grid-template-columns: 1fr;
    }

    .ring-card {
      align-items: flex-start;
    }

    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
