<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetSystemStatus, GetTopProcesses } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  let status = null;
  let processes = [];
  let cpuHistory = [];
  let memHistory = [];
  let interval;
  const MAX_HISTORY = 30;

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function sparklinePath(data, width, height) {
    if (data.length < 2) return '';
    const max = Math.max(...data, 100);
    const step = width / (data.length - 1);
    return data.map((v, i) => {
      const x = i * step;
      const y = height - (v / max) * height;
      return `${i === 0 ? 'M' : 'L'} ${x} ${y}`;
    }).join(' ');
  }

  async function fetchData() {
    try {
      status = await GetSystemStatus();
      processes = await GetTopProcesses(8);

      cpuHistory = [...cpuHistory, status.cpu.usage].slice(-MAX_HISTORY);
      memHistory = [...memHistory, status.memory.usage].slice(-MAX_HISTORY);
    } catch (e) {
      console.error('Monitor fetch error:', e);
    }
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchData();
    interval = setInterval(fetchData, 2000);
  }

  function stopPolling() {
    if (interval) { clearInterval(interval); interval = null; }
  }

  // Pause/resume polling based on visibility
  $: if (visible) { startPolling(); } else { stopPolling(); }

  onDestroy(() => stopPolling());
</script>

<div class="monitor-page">
  <div class="page-header">
    <h1>System Monitor</h1>
    <p class="subtitle">Real-time system performance</p>
  </div>

  {#if status}
    <div class="gauges">
      <div class="gauge-card">
        <div class="gauge-header">
          <span class="gauge-label">CPU</span>
          <span class="gauge-value">{status.cpu.usage.toFixed(1)}%</span>
        </div>
        <div class="gauge-bar">
          <div class="gauge-fill blue" style="width: {status.cpu.usage}%"></div>
        </div>
        <div class="gauge-detail">{status.cpu.cores} cores</div>
        <svg class="sparkline" viewBox="0 0 200 40">
          <path d={sparklinePath(cpuHistory, 200, 40)} fill="none" stroke="var(--blue)" stroke-width="1.5" />
        </svg>
      </div>

      <div class="gauge-card">
        <div class="gauge-header">
          <span class="gauge-label">Memory</span>
          <span class="gauge-value">{status.memory.usage.toFixed(1)}%</span>
        </div>
        <div class="gauge-bar">
          <div class="gauge-fill green" style="width: {status.memory.usage}%"></div>
        </div>
        <div class="gauge-detail">{formatBytes(status.memory.used)} / {formatBytes(status.memory.total)}</div>
        <svg class="sparkline" viewBox="0 0 200 40">
          <path d={sparklinePath(memHistory, 200, 40)} fill="none" stroke="var(--green)" stroke-width="1.5" />
        </svg>
      </div>

      <div class="gauge-card">
        <div class="gauge-header">
          <span class="gauge-label">Disk</span>
          <span class="gauge-value">{status.disk.usage.toFixed(1)}%</span>
        </div>
        <div class="gauge-bar">
          <div class="gauge-fill" class:yellow={status.disk.usage > 70} class:red={status.disk.usage > 90} class:accent={status.disk.usage <= 70} style="width: {status.disk.usage}%"></div>
        </div>
        <div class="gauge-detail">{formatBytes(status.disk.used)} / {formatBytes(status.disk.total)}</div>
      </div>

      <div class="gauge-card">
        <div class="gauge-header">
          <span class="gauge-label">Battery</span>
          <span class="gauge-value">{status.battery.percentage}%</span>
        </div>
        <div class="gauge-bar">
          <div class="gauge-fill green" style="width: {status.battery.percentage}%"></div>
        </div>
        <div class="gauge-detail">{status.battery.status}</div>
      </div>
    </div>

    <div class="processes-section">
      <h2>Top Processes</h2>
      <div class="process-table">
        <div class="process-header">
          <span class="col-name">Process</span>
          <span class="col-cpu">CPU %</span>
          <span class="col-mem">MEM %</span>
          <span class="col-pid">PID</span>
        </div>
        {#each processes as proc}
          <div class="process-row">
            <span class="col-name">{proc.name}</span>
            <span class="col-cpu">
              <span class="proc-bar-bg">
                <span class="proc-bar blue" style="width: {Math.min(proc.cpuUsage, 100)}%"></span>
              </span>
              {proc.cpuUsage.toFixed(1)}
            </span>
            <span class="col-mem">
              <span class="proc-bar-bg">
                <span class="proc-bar green" style="width: {Math.min(proc.memUsage, 100)}%"></span>
              </span>
              {proc.memUsage.toFixed(1)}
            </span>
            <span class="col-pid">{proc.pid}</span>
          </div>
        {/each}
      </div>
    </div>

    <div class="health-row">
      <span class="health-label">System Health:</span>
      <span class="health-score" class:good={status.health >= 80} class:warn={status.health >= 50 && status.health < 80} class:bad={status.health < 50}>
        {status.health}/100
      </span>
      <span class="health-uptime">Uptime: {status.uptime}</span>
    </div>
  {:else}
    <div class="loading">Loading system metrics...</div>
  {/if}
</div>

<style>
  .monitor-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 24px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }
  .loading { color: var(--text-secondary); padding: 40px 0; text-align: center; }

  .gauges {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 28px;
  }

  .gauge-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px;
  }

  .gauge-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 12px;
  }

  .gauge-label {
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .gauge-value {
    font-size: 22px;
    font-weight: 700;
  }

  .gauge-bar {
    height: 6px;
    background: var(--bg-tertiary);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 8px;
  }

  .gauge-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.5s ease;
  }

  .gauge-fill.blue { background: var(--blue); }
  .gauge-fill.green { background: var(--green); }
  .gauge-fill.yellow { background: var(--yellow); }
  .gauge-fill.red { background: var(--red); }
  .gauge-fill.accent { background: var(--accent); }

  .gauge-detail {
    font-size: 12px;
    color: var(--text-muted);
    margin-bottom: 12px;
  }

  .sparkline {
    width: 100%;
    height: 40px;
  }

  .processes-section {
    margin-bottom: 24px;
  }

  .processes-section h2 {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 12px;
  }

  .process-table {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .process-header, .process-row {
    display: grid;
    grid-template-columns: 1fr 150px 150px 80px;
    padding: 10px 16px;
    align-items: center;
    gap: 8px;
  }

  .process-header {
    background: var(--bg-tertiary);
    font-size: 12px;
    color: var(--text-muted);
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .process-row {
    border-top: 1px solid var(--border);
    font-size: 13px;
  }

  .process-row:hover {
    background: var(--bg-hover);
  }

  .col-name {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .col-cpu, .col-mem {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .col-pid {
    color: var(--text-muted);
    font-size: 12px;
    text-align: right;
  }

  .proc-bar-bg {
    flex: 1;
    height: 4px;
    background: var(--bg-tertiary);
    border-radius: 2px;
    overflow: hidden;
  }

  .proc-bar {
    height: 100%;
    border-radius: 2px;
  }

  .proc-bar.blue { background: var(--blue); }
  .proc-bar.green { background: var(--green); }

  .health-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 20px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
  }

  .health-label {
    font-size: 14px;
    color: var(--text-secondary);
  }

  .health-score {
    font-size: 18px;
    font-weight: 700;
  }

  .health-score.good { color: var(--green); }
  .health-score.warn { color: var(--yellow); }
  .health-score.bad { color: var(--red); }

  .health-uptime {
    margin-left: auto;
    font-size: 13px;
    color: var(--text-muted);
  }
</style>
