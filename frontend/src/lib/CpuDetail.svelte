<script>
  import { onDestroy, createEventDispatcher } from 'svelte';
  import { GetCPUDetail } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  const dispatch = createEventDispatcher();

  let data = null;
  let loading = true;
  let error = null;
  let interval = null;

  // Sparkline history — last 30 readings of total CPU usage
  let history = [];
  const HISTORY_MAX = 30;

  async function fetchData() {
    try {
      const result = await GetCPUDetail();
      data = result;
      error = null;

      // Append to history
      history = [...history, result.usage].slice(-HISTORY_MAX);
    } catch (e) {
      console.error('Failed to get CPU detail:', e);
      error = 'Failed to load CPU data.';
    }
    loading = false;
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 2000);
  }

  function stopPolling() {
    if (interval) {
      clearInterval(interval);
      interval = null;
    }
  }

  $: if (visible) {
    startPolling();
  } else {
    stopPolling();
  }

  onDestroy(() => stopPolling());

  // --- Donut chart helpers ---
  // Radius and stroke for the ring
  const CX = 100;
  const CY = 100;
  const R = 78;
  const STROKE = 18;
  const CIRCUMFERENCE = 2 * Math.PI * R;

  function ringSegments(user, sys, idle) {
    // Returns array of { offset, dasharray, color, label, value }
    const segments = [
      { value: user, color: 'var(--blue)',   label: 'User'   },
      { value: sys,  color: 'var(--yellow)', label: 'System' },
      { value: idle, color: 'var(--bg-tertiary)', label: 'Idle' },
    ];
    let offset = 0; // start from top (rotated via transform)
    return segments.map(s => {
      const dash = (s.value / 100) * CIRCUMFERENCE;
      const gap  = CIRCUMFERENCE - dash;
      const seg = {
        ...s,
        dasharray: `${dash} ${gap}`,
        // strokeDashoffset shifts the segment: we go counter-clockwise by
        // subtracting the accumulated offset from full circumference
        dashoffset: CIRCUMFERENCE - offset,
      };
      offset += dash;
      return seg;
    });
  }

  // --- Sparkline helpers ---
  const SPARK_W = 260;
  const SPARK_H = 48;

  function buildSparkPath(pts) {
    if (!pts || pts.length < 2) return '';
    const minV = 0;
    const maxV = 100;
    const xStep = SPARK_W / (HISTORY_MAX - 1);

    const coords = pts.map((v, i) => {
      const x = i * xStep;
      const y = SPARK_H - ((v - minV) / (maxV - minV)) * SPARK_H;
      return [x, y];
    });

    // Pad missing slots to the left so the line always ends at the right edge
    const pad = HISTORY_MAX - pts.length;
    const shifted = coords.map(([x, y]) => [x + pad * xStep, y]);

    return shifted
      .map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`)
      .join(' ');
  }

  function buildSparkArea(pts) {
    if (!pts || pts.length < 2) return '';
    const path = buildSparkPath(pts);
    const pad = HISTORY_MAX - pts.length;
    const xStep = SPARK_W / (HISTORY_MAX - 1);
    const lastX = ((pts.length - 1) + pad) * xStep;
    const firstX = pad * xStep;
    return `${path} L${lastX.toFixed(1)},${SPARK_H} L${firstX.toFixed(1)},${SPARK_H} Z`;
  }

  function fmt(n, decimals = 1) {
    if (n == null) return '—';
    return Number(n).toFixed(decimals);
  }
</script>

<div class="cpu-detail">
  <!-- Header row with back button -->
  <div class="detail-header">
    <button class="back-btn" on:click={() => dispatch('back')} title="Go back">
      &#8592; Back
    </button>
    <div class="header-text">
      <h1 class="page-title">CPU</h1>
      {#if data}
        <p class="page-subtitle">{data.model}</p>
      {/if}
    </div>
  </div>

  {#if loading && !data}
    <div class="loading-state">Loading CPU data&hellip;</div>
  {:else if error && !data}
    <div class="error-state">{error}</div>
  {:else if data}
    <div class="content">

      <!-- Top section: donut + breakdown side by side -->
      <div class="top-section">

        <!-- Donut ring chart -->
        <div class="card donut-card">
          <div class="card-label">Usage Breakdown</div>
          <div class="donut-wrap">
            <svg
              width="200"
              height="200"
              viewBox="0 0 200 200"
              aria-label="CPU usage ring chart"
            >
              <!-- Rotate so first segment starts at 12 o'clock -->
              <g transform="rotate(-90 100 100)">
                {#each ringSegments(data.user, data.sys, data.idle) as seg}
                  <circle
                    cx={CX}
                    cy={CY}
                    r={R}
                    fill="none"
                    stroke={seg.color}
                    stroke-width={STROKE}
                    stroke-dasharray={seg.dasharray}
                    stroke-dashoffset={seg.dashoffset}
                    stroke-linecap="butt"
                  />
                {/each}
              </g>
              <!-- Centre label -->
              <text
                x="100"
                y="94"
                text-anchor="middle"
                dominant-baseline="middle"
                class="donut-pct"
              >{fmt(data.usage)}%</text>
              <text
                x="100"
                y="114"
                text-anchor="middle"
                dominant-baseline="middle"
                class="donut-sub"
              >total</text>
            </svg>

            <!-- Legend -->
            <div class="donut-legend">
              <div class="legend-item">
                <span class="legend-dot" style="background: var(--blue);"></span>
                <span class="legend-name">User</span>
                <span class="legend-val">{fmt(data.user)}%</span>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background: var(--yellow);"></span>
                <span class="legend-name">System</span>
                <span class="legend-val">{fmt(data.sys)}%</span>
              </div>
              <div class="legend-item">
                <span class="legend-dot" style="background: var(--bg-tertiary); border: 1px solid var(--border);"></span>
                <span class="legend-name">Idle</span>
                <span class="legend-val">{fmt(data.idle)}%</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Usage bars -->
        <div class="card bars-card">
          <div class="card-label">Usage Bars</div>
          <div class="bars">
            <div class="bar-row">
              <span class="bar-name">User</span>
              <div class="bar-track">
                <div
                  class="bar-fill blue"
                  style="width: {data.user}%"
                ></div>
              </div>
              <span class="bar-pct">{fmt(data.user)}%</span>
            </div>
            <div class="bar-row">
              <span class="bar-name">System</span>
              <div class="bar-track">
                <div
                  class="bar-fill yellow"
                  style="width: {data.sys}%"
                ></div>
              </div>
              <span class="bar-pct">{fmt(data.sys)}%</span>
            </div>
            <div class="bar-row">
              <span class="bar-name">Idle</span>
              <div class="bar-track">
                <div
                  class="bar-fill muted"
                  style="width: {data.idle}%"
                ></div>
              </div>
              <span class="bar-pct">{fmt(data.idle)}%</span>
            </div>
          </div>

          <!-- Sparkline -->
          <div class="sparkline-section">
            <div class="sparkline-label">
              Usage history
              <span class="sparkline-hint">(last {history.length} readings)</span>
            </div>
            <div class="sparkline-wrap">
              {#if history.length >= 2}
                <svg
                  width={SPARK_W}
                  height={SPARK_H}
                  viewBox="0 0 {SPARK_W} {SPARK_H}"
                  preserveAspectRatio="none"
                  aria-label="CPU usage sparkline"
                >
                  <defs>
                    <linearGradient id="spark-grad" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stop-color="var(--accent)" stop-opacity="0.3" />
                      <stop offset="100%" stop-color="var(--accent)" stop-opacity="0" />
                    </linearGradient>
                  </defs>
                  <path
                    d={buildSparkArea(history)}
                    fill="url(#spark-grad)"
                  />
                  <path
                    d={buildSparkPath(history)}
                    fill="none"
                    stroke="var(--accent)"
                    stroke-width="1.5"
                    stroke-linejoin="round"
                    stroke-linecap="round"
                  />
                </svg>
              {:else}
                <div class="spark-placeholder">Collecting data&hellip;</div>
              {/if}
            </div>
          </div>
        </div>
      </div>

      <!-- Stats grid -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon-wrap blue">&#9670;</div>
          <div class="stat-body">
            <div class="stat-name">Cores</div>
            <div class="stat-val">{data.cores}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon-wrap accent">&#9689;</div>
          <div class="stat-body">
            <div class="stat-name">Architecture</div>
            <div class="stat-val mono">{data.architecture}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon-wrap green">&#9203;</div>
          <div class="stat-body">
            <div class="stat-name">Uptime</div>
            <div class="stat-val mono">{data.uptime}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon-wrap yellow">&#9632;</div>
          <div class="stat-body">
            <div class="stat-name">Processes</div>
            <div class="stat-val">{(data.processCount ?? 0).toLocaleString()}</div>
          </div>
        </div>

        <div class="stat-card">
          <div class="stat-icon-wrap yellow">&#9632;</div>
          <div class="stat-body">
            <div class="stat-name">Threads</div>
            <div class="stat-val">{(data.threadCount ?? 0).toLocaleString()}</div>
          </div>
        </div>

        <div class="stat-card load-card">
          <div class="stat-icon-wrap accent">&#8776;</div>
          <div class="stat-body">
            <div class="stat-name">Load Average</div>
            <div class="load-row">
              <div class="load-item">
                <span class="load-lbl">1m</span>
                <span class="load-val">{fmt(data.loadAvg1)}</span>
              </div>
              <div class="load-sep"></div>
              <div class="load-item">
                <span class="load-lbl">5m</span>
                <span class="load-val">{fmt(data.loadAvg5)}</span>
              </div>
              <div class="load-sep"></div>
              <div class="load-item">
                <span class="load-lbl">15m</span>
                <span class="load-val">{fmt(data.loadAvg15)}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

    </div>
  {/if}
</div>

<style>
  /* ── Layout ── */
  .cpu-detail {
    padding: 0 32px 32px;
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  /* ── Header ── */
  .detail-header {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 24px;
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

  .page-title {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.5px;
    line-height: 1;
  }

  .page-subtitle {
    color: var(--text-secondary);
    font-size: 13px;
    margin-top: 4px;
  }

  /* ── States ── */
  .loading-state,
  .error-state {
    color: var(--text-secondary);
    padding: 60px 0;
    text-align: center;
  }

  .error-state {
    color: var(--red);
  }

  /* ── Content ── */
  .content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  /* ── Top section ── */
  .top-section {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 20px;
    align-items: start;
  }

  /* ── Cards ── */
  .card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
  }

  .card-label {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.6px;
    text-transform: uppercase;
    color: var(--text-muted);
    margin-bottom: 20px;
  }

  /* ── Donut card ── */
  .donut-card {
    display: flex;
    flex-direction: column;
  }

  .donut-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
  }

  .donut-pct {
    fill: var(--text-primary);
    font-size: 26px;
    font-weight: 700;
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif;
  }

  .donut-sub {
    fill: var(--text-muted);
    font-size: 11px;
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .donut-legend {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
  }

  .legend-dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .legend-name {
    flex: 1;
    color: var(--text-secondary);
  }

  .legend-val {
    font-weight: 600;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }

  /* ── Bars card ── */
  .bars-card {
    display: flex;
    flex-direction: column;
  }

  .bars {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .bar-row {
    display: grid;
    grid-template-columns: 56px 1fr 44px;
    align-items: center;
    gap: 12px;
  }

  .bar-name {
    font-size: 13px;
    color: var(--text-secondary);
    white-space: nowrap;
  }

  .bar-track {
    height: 6px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.5s ease;
  }

  .bar-fill.blue   { background: var(--blue); }
  .bar-fill.yellow { background: var(--yellow); }
  .bar-fill.muted  { background: var(--text-muted); }

  .bar-pct {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-primary);
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  /* ── Sparkline ── */
  .sparkline-section {
    margin-top: 24px;
    border-top: 1px solid var(--border);
    padding-top: 20px;
  }

  .sparkline-label {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 10px;
  }

  .sparkline-hint {
    color: var(--text-muted);
    font-weight: 400;
  }

  .sparkline-wrap {
    width: 100%;
    overflow: hidden;
  }

  .sparkline-wrap svg {
    display: block;
    width: 100%;
    height: 48px;
  }

  .spark-placeholder {
    height: 48px;
    display: flex;
    align-items: center;
    color: var(--text-muted);
    font-size: 12px;
  }

  /* ── Stats grid ── */
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
    display: flex;
    align-items: flex-start;
    gap: 14px;
  }

  .stat-icon-wrap {
    width: 34px;
    height: 34px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    flex-shrink: 0;
  }

  .stat-icon-wrap.blue   { background: var(--blue-dim);   color: var(--blue);   }
  .stat-icon-wrap.green  { background: var(--green-dim);  color: var(--green);  }
  .stat-icon-wrap.yellow { background: var(--yellow-dim); color: var(--yellow); }
  .stat-icon-wrap.accent { background: var(--accent-dim); color: var(--accent); }

  .stat-body {
    flex: 1;
    min-width: 0;
  }

  .stat-name {
    font-size: 11px;
    font-weight: 600;
    letter-spacing: 0.5px;
    text-transform: uppercase;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .stat-val {
    font-size: 20px;
    font-weight: 700;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stat-val.mono {
    font-size: 15px;
    font-family: 'SF Mono', 'Menlo', 'Consolas', monospace;
    letter-spacing: 0;
  }

  /* Load average card spans full width in 3-col grid when it's the last item
     (it lands in position 6 which fills the third row's first column unless
     we make it span). We keep it naturally sized but widen it slightly. */
  .load-card {
    grid-column: span 1;
  }

  .load-row {
    display: flex;
    align-items: center;
    gap: 0;
    margin-top: 2px;
  }

  .load-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    flex: 1;
  }

  .load-lbl {
    font-size: 10px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.4px;
    color: var(--text-muted);
  }

  .load-val {
    font-size: 15px;
    font-weight: 700;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }

  .load-sep {
    width: 1px;
    height: 28px;
    background: var(--border);
    flex-shrink: 0;
  }

  /* ── Responsive ── */
  @media (max-width: 900px) {
    .top-section {
      grid-template-columns: 1fr;
    }

    .donut-wrap {
      flex-direction: row;
      justify-content: center;
      align-items: center;
    }

    .donut-legend {
      width: auto;
    }

    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 600px) {
    .cpu-detail {
      padding: 0 16px 24px;
    }

    .stats-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
