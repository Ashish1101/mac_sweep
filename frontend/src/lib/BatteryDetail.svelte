<script>
  import { createEventDispatcher, onDestroy } from 'svelte';
  import { GetBatteryDetail } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  const dispatch = createEventDispatcher();

  let data = null;
  let loading = true;
  let error = null;
  let interval = null;

  // Sparkline history — last 30 percentage readings
  const MAX_HISTORY = 30;
  let history = []; // [{ pct: number, ts: number }]

  // ---- Helpers ----

  function batteryColor(pct) {
    if (pct > 50) return 'var(--green)';
    if (pct > 20) return 'var(--yellow)';
    return 'var(--red)';
  }

  function batteryColorClass(pct) {
    if (pct > 50) return 'green';
    if (pct > 20) return 'yellow';
    return 'red';
  }

  function healthColorClass(pct) {
    if (pct > 80) return 'green';
    if (pct > 50) return 'yellow';
    return 'red';
  }

  // SVG ring for the big percentage dial
  // Returns the stroke-dasharray offset for an arc of `pct` out of 100
  const RING_RADIUS = 84;
  const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

  function ringOffset(pct) {
    const filled = (pct / 100) * RING_CIRCUMFERENCE;
    return `${filled} ${RING_CIRCUMFERENCE - filled}`;
  }

  // Sparkline polyline points from history array
  function sparklinePoints(hist, w, h) {
    if (hist.length < 2) return '';
    const maxPct = 100;
    const step = w / (hist.length - 1);
    return hist
      .map((d, i) => {
        const x = i * step;
        const y = h - (d.pct / maxPct) * h;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  }

  // Sparkline area fill path
  function sparklineArea(hist, w, h) {
    if (hist.length < 2) return '';
    const maxPct = 100;
    const step = w / (hist.length - 1);
    const pts = hist
      .map((d, i) => {
        const x = i * step;
        const y = h - (d.pct / maxPct) * h;
        return `${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' L ');
    const lastX = ((hist.length - 1) * step).toFixed(1);
    return `M 0,${h} L ${pts} L ${lastX},${h} Z`;
  }

  // ---- Fetch ----

  async function fetchData() {
    try {
      const result = await GetBatteryDetail();
      data = result;
      history = [...history, { pct: result.percentage, ts: Date.now() }].slice(-MAX_HISTORY);
      error = null;
    } catch (e) {
      error = 'Failed to load battery data.';
      console.error('BatteryDetail fetch error:', e);
    } finally {
      loading = false;
    }
  }

  // ---- Polling ----

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 5000);
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

  // ---- Derived ----
  $: ringColor = data ? batteryColor(data.percentage) : 'var(--text-muted)';
  $: ringColorClass = data ? batteryColorClass(data.percentage) : '';
  $: healthClass = data ? healthColorClass(data.healthPercent) : '';
  $: cycleRatio = data ? Math.min(data.cycleCount / (data.maxCycleCount || 1), 1) : 0;
  $: sparkPts = sparklinePoints(history, 280, 52);
  $: sparkArea = sparklineArea(history, 280, 52);
  $: sparkColor = data ? batteryColor(data.percentage) : 'var(--text-muted)';
</script>

<div class="battery-detail">
  <!-- Header -->
  <div class="detail-header">
    <button class="back-btn" on:click={() => dispatch('back')}>
      <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
        <path d="M10 3L5 8L10 13" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
      Back
    </button>
    <div class="header-text">
      <h1>Battery</h1>
      {#if data}
        <span class="status-badge" class:charging={data.isCharging}>
          {data.status}
        </span>
      {/if}
    </div>
  </div>

  {#if loading}
    <div class="loading-state">Loading battery data...</div>
  {:else if error}
    <div class="error-state">{error}</div>
  {:else if data}
    <div class="content">

      <!-- Main ring + health row -->
      <div class="top-row">

        <!-- Percentage ring -->
        <div class="ring-card">
          <svg class="ring-svg" viewBox="0 0 200 200" width="200" height="200">
            <!-- Track -->
            <circle
              cx="100" cy="100" r="{RING_RADIUS}"
              fill="none"
              stroke="var(--bg-tertiary)"
              stroke-width="12"
            />
            <!-- Fill arc — starts at top (–90deg rotation) -->
            <circle
              cx="100" cy="100" r="{RING_RADIUS}"
              fill="none"
              stroke="{ringColor}"
              stroke-width="12"
              stroke-linecap="round"
              stroke-dasharray="{ringOffset(data.percentage)}"
              transform="rotate(-90 100 100)"
              style="transition: stroke-dasharray 0.6s ease, stroke 0.4s ease;"
            />
            <!-- Center percentage text -->
            <text x="100" y="96" text-anchor="middle" class="ring-pct-text" fill="{ringColor}">{data.percentage}%</text>
            <!-- Center sub-label -->
            {#if data.isCharging}
              <!-- Charging bolt icon (SVG path) -->
              <text x="100" y="120" text-anchor="middle" class="ring-sub-text" fill="var(--text-secondary)">Charging</text>
              <path d="M 97 108 L 93 118 L 100 114 L 100 124 L 104 114 L 97 118 Z"
                fill="{ringColor}" opacity="0.85"
                transform="translate(0, -4)"
              />
            {:else}
              <text x="100" y="120" text-anchor="middle" class="ring-sub-text" fill="var(--text-secondary)">{data.timeRemaining ? data.timeRemaining + ' left' : 'Calculating...'}</text>
            {/if}
          </svg>
        </div>

        <!-- Health + Cycle count stacked -->
        <div class="right-col">

          <!-- Battery Health -->
          <div class="section-card health-card">
            <div class="section-title">Battery Health</div>
            <div class="health-row">
              <span class="health-pct {healthClass}">{data.healthPercent}%</span>
              <span class="condition-badge" class:normal={data.condition === 'Normal'} class:service={data.condition !== 'Normal'}>
                {data.condition}
              </span>
            </div>
            <div class="bar-track">
              <div
                class="bar-fill {healthClass}"
                style="width: {data.healthPercent}%"
              ></div>
            </div>
            <div class="health-caption">
              Max capacity {data.maxCapacity}% of design capacity ({data.designCapacity} mAh)
            </div>
          </div>

          <!-- Cycle Count -->
          <div class="section-card cycle-card">
            <div class="section-title">Cycle Count</div>
            <div class="cycle-row">
              <span class="cycle-value">{data.cycleCount} <span class="cycle-max">/ {data.maxCycleCount}</span></span>
            </div>
            <div class="bar-track">
              <div
                class="bar-fill"
                class:green={cycleRatio < 0.5}
                class:yellow={cycleRatio >= 0.5 && cycleRatio < 0.8}
                class:red={cycleRatio >= 0.8}
                style="width: {cycleRatio * 100}%"
              ></div>
            </div>
            <div class="health-caption">
              {(data.maxCycleCount ?? 0) - (data.cycleCount ?? 0)} cycles remaining before service recommended
            </div>
          </div>

        </div>
      </div>

      <!-- Stats cards grid -->
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-label">Time Remaining</div>
          <div class="stat-value">{data.timeRemaining || '—'}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Power Source</div>
          <div class="stat-value">{data.powerSource}</div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Temperature</div>
          <div class="stat-value">{(data.temperature ?? 0).toFixed(1)}<span class="stat-unit"> C</span></div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Voltage</div>
          <div class="stat-value">{(data.voltage ?? 0).toFixed(1)}<span class="stat-unit"> V</span></div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Wattage</div>
          <div class="stat-value">{(data.wattage ?? 0).toFixed(1)}<span class="stat-unit"> W</span></div>
        </div>
        <div class="stat-card">
          <div class="stat-label">Max Capacity</div>
          <div class="stat-value">{data.maxCapacity}<span class="stat-unit"> %</span></div>
        </div>
      </div>

      <!-- Sparkline history -->
      <div class="section-card sparkline-card">
        <div class="sparkline-header">
          <span class="section-title">Charge History</span>
          <span class="sparkline-caption">{history.length} / {MAX_HISTORY} readings &middot; updates every 5s</span>
        </div>

        {#if history.length < 2}
          <div class="sparkline-empty">Collecting data — check back in a few seconds.</div>
        {:else}
          <svg class="sparkline-svg" viewBox="0 0 280 52" preserveAspectRatio="none" width="100%" height="52">
            <!-- Baseline -->
            <line x1="0" y1="52" x2="280" y2="52" stroke="var(--border)" stroke-width="1"/>
            <!-- 50% reference line -->
            <line x1="0" y1="26" x2="280" y2="26" stroke="var(--bg-tertiary)" stroke-width="1" stroke-dasharray="4 3"/>
            <!-- Area fill -->
            <path d="{sparkArea}" fill="{sparkColor}" opacity="0.12"/>
            <!-- Line -->
            <polyline
              points="{sparkPts}"
              fill="none"
              stroke="{sparkColor}"
              stroke-width="2"
              stroke-linejoin="round"
              stroke-linecap="round"
            />
            <!-- Latest dot -->
            {#if history.length > 0}
              {@const lastIdx = history.length - 1}
              {@const dotX = (lastIdx / (history.length - 1)) * 280}
              {@const dotY = 52 - (history[lastIdx].pct / 100) * 52}
              <circle cx="{dotX.toFixed(1)}" cy="{dotY.toFixed(1)}" r="3.5" fill="{sparkColor}"/>
            {/if}
          </svg>
          <div class="sparkline-axis">
            <span>Oldest</span>
            <span>Latest — {data.percentage}%</span>
          </div>
        {/if}
      </div>

    </div>
  {/if}
</div>

<style>
  .battery-detail {
    padding: 0 32px 32px;
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  /* ---- Header ---- */
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
    background: var(--bg-card);
    border: 1px solid var(--border);
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    padding: 6px 14px;
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

  .header-text {
    display: flex;
    align-items: baseline;
    gap: 12px;
  }

  .header-text h1 {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }

  .status-badge {
    font-size: 12px;
    font-weight: 500;
    padding: 3px 10px;
    border-radius: 20px;
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
  }

  .status-badge.charging {
    background: var(--green-dim);
    color: var(--green);
    border-color: color-mix(in srgb, var(--green) 30%, transparent);
  }

  /* ---- States ---- */
  .loading-state,
  .error-state {
    color: var(--text-secondary);
    padding: 60px 0;
    text-align: center;
    font-size: 14px;
  }

  .error-state {
    color: var(--red);
  }

  /* ---- Layout ---- */
  .content {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .top-row {
    display: flex;
    gap: 20px;
    align-items: stretch;
  }

  .right-col {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-width: 0;
  }

  /* ---- Ring card ---- */
  .ring-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .ring-svg {
    display: block;
  }

  .ring-pct-text {
    font-size: 34px;
    font-weight: 800;
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif;
    letter-spacing: -1px;
  }

  .ring-sub-text {
    font-size: 12px;
    font-family: -apple-system, BlinkMacSystemFont, 'SF Pro Display', system-ui, sans-serif;
    font-weight: 500;
  }

  /* ---- Shared section card ---- */
  .section-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px;
  }

  .section-title {
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.6px;
    color: var(--text-secondary);
    margin-bottom: 14px;
  }

  /* ---- Health card ---- */
  .health-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }

  .health-pct {
    font-size: 28px;
    font-weight: 700;
    line-height: 1;
  }

  .health-pct.green { color: var(--green); }
  .health-pct.yellow { color: var(--yellow); }
  .health-pct.red { color: var(--red); }

  .condition-badge {
    font-size: 11px;
    font-weight: 600;
    padding: 3px 10px;
    border-radius: 20px;
    letter-spacing: 0.2px;
  }

  .condition-badge.normal {
    background: var(--green-dim);
    color: var(--green);
    border: 1px solid color-mix(in srgb, var(--green) 25%, transparent);
  }

  .condition-badge.service {
    background: var(--yellow-dim);
    color: var(--yellow);
    border: 1px solid color-mix(in srgb, var(--yellow) 25%, transparent);
  }

  /* ---- Bar track ---- */
  .bar-track {
    height: 6px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 10px;
  }

  .bar-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.5s ease;
    background: var(--accent);
  }

  .bar-fill.green { background: var(--green); }
  .bar-fill.yellow { background: var(--yellow); }
  .bar-fill.red { background: var(--red); }

  .health-caption {
    font-size: 11px;
    color: var(--text-muted);
    line-height: 1.4;
  }

  /* ---- Cycle card ---- */
  .cycle-row {
    margin-bottom: 12px;
  }

  .cycle-value {
    font-size: 26px;
    font-weight: 700;
    color: var(--text-primary);
  }

  .cycle-max {
    font-size: 16px;
    font-weight: 500;
    color: var(--text-muted);
  }

  /* ---- Stats grid ---- */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 14px;
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
    letter-spacing: 0.5px;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .stat-value {
    font-size: 22px;
    font-weight: 700;
    color: var(--text-primary);
    line-height: 1;
  }

  .stat-unit {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-muted);
  }

  /* ---- Sparkline card ---- */
  .sparkline-card {
    padding: 20px;
  }

  .sparkline-header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .sparkline-header .section-title {
    margin-bottom: 0;
  }

  .sparkline-caption {
    font-size: 11px;
    color: var(--text-muted);
  }

  .sparkline-svg {
    display: block;
    width: 100%;
    border-radius: 4px;
  }

  .sparkline-axis {
    display: flex;
    justify-content: space-between;
    margin-top: 6px;
    font-size: 10px;
    color: var(--text-muted);
  }

  .sparkline-empty {
    font-size: 13px;
    color: var(--text-muted);
    text-align: center;
    padding: 20px 0;
  }

  /* ---- Responsive ---- */
  @media (max-width: 900px) {
    .top-row {
      flex-direction: column;
    }

    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }

  @media (max-width: 600px) {
    .stats-grid {
      grid-template-columns: 1fr 1fr;
    }
  }
</style>
