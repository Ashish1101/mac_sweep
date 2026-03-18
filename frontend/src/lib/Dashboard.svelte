<script>
  import { onMount, onDestroy } from 'svelte';
  import { GetSystemStatus } from '../../wailsjs/go/main/App.js';
  import { currentPage } from '../stores/navigation.js';

  export let visible = true;

  let status = null;
  let loading = true;
  let interval;

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  async function fetchStatus() {
    try {
      status = await GetSystemStatus();
    } catch (e) {
      console.error('Failed to get status:', e);
    }
    loading = false;
  }

  function startPolling() {
    if (interval) clearInterval(interval);
    fetchStatus();
    interval = setInterval(fetchStatus, 5000);
  }

  function stopPolling() {
    if (interval) { clearInterval(interval); interval = null; }
  }

  // Pause/resume polling based on visibility
  $: if (visible) { startPolling(); } else { stopPolling(); }

  onDestroy(() => stopPolling());
</script>

<div class="dashboard">
  <div class="page-header">
    <h1>Dashboard</h1>
    <p class="subtitle">System overview at a glance</p>
  </div>

  {#if loading}
    <div class="loading">Loading system status...</div>
  {:else if status}
    <div class="health-banner" class:good={status.health >= 80} class:warn={status.health >= 50 && status.health < 80} class:bad={status.health < 50}>
      <div class="health-score">{status.health}</div>
      <div class="health-info">
        <div class="health-label">Health Score</div>
        <div class="health-desc">
          {#if status.health >= 80}
            Your system is healthy
          {:else if status.health >= 50}
            Some attention needed
          {:else}
            Action recommended
          {/if}
        </div>
      </div>
      <div class="health-uptime">Uptime: {status.uptime}</div>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-icon blue">&#9670;</span>
          <span class="stat-label">CPU</span>
        </div>
        <div class="stat-value">{status.cpu.usage.toFixed(1)}%</div>
        <div class="stat-bar">
          <div class="stat-bar-fill blue" style="width: {status.cpu.usage}%"></div>
        </div>
        <div class="stat-detail">{status.cpu.cores} cores &middot; {status.cpu.model.split(' ').slice(0, 3).join(' ')}</div>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-icon green">&#9632;</span>
          <span class="stat-label">Memory</span>
        </div>
        <div class="stat-value">{status.memory.usage.toFixed(1)}%</div>
        <div class="stat-bar">
          <div class="stat-bar-fill green" style="width: {status.memory.usage}%"></div>
        </div>
        <div class="stat-detail">{formatBytes(status.memory.used)} / {formatBytes(status.memory.total)}</div>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-icon" class:yellow={status.disk.usage > 70} class:red={status.disk.usage > 90} class:accent={status.disk.usage <= 70}>&#9679;</span>
          <span class="stat-label">Disk</span>
        </div>
        <div class="stat-value">{status.disk.usage.toFixed(1)}%</div>
        <div class="stat-bar">
          <div class="stat-bar-fill" class:yellow={status.disk.usage > 70} class:red={status.disk.usage > 90} class:accent={status.disk.usage <= 70} style="width: {status.disk.usage}%"></div>
        </div>
        <div class="stat-detail">{formatBytes(status.disk.used)} / {formatBytes(status.disk.total)}</div>
      </div>

      <div class="stat-card">
        <div class="stat-header">
          <span class="stat-icon green">&#9889;</span>
          <span class="stat-label">Battery</span>
        </div>
        <div class="stat-value">{status.battery.percentage}%</div>
        <div class="stat-bar">
          <div class="stat-bar-fill green" style="width: {status.battery.percentage}%"></div>
        </div>
        <div class="stat-detail">{status.battery.status}</div>
      </div>
    </div>

    <div class="quick-actions">
      <h2>Quick Actions</h2>
      <div class="actions-grid">
        <button class="action-card" on:click={() => currentPage.set('clean')}>
          <span class="action-icon">&#10026;</span>
          <span class="action-label">Quick Clean</span>
          <span class="action-desc">Scan and remove junk files</span>
        </button>
        <button class="action-card" on:click={() => currentPage.set('analyze')}>
          <span class="action-icon">&#9678;</span>
          <span class="action-label">Scan Disk</span>
          <span class="action-desc">Visualize disk usage</span>
        </button>
        <button class="action-card" on:click={() => currentPage.set('monitor')}>
          <span class="action-icon">&#9632;</span>
          <span class="action-label">Monitor</span>
          <span class="action-desc">Real-time system stats</span>
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .dashboard {
    padding: 0 32px 32px;
  }

  .page-header {
    margin-bottom: 24px;
  }

  .page-header h1 {
    font-size: 24px;
    font-weight: 700;
    letter-spacing: -0.5px;
  }

  .subtitle {
    color: var(--text-secondary);
    margin-top: 4px;
  }

  .loading {
    color: var(--text-secondary);
    padding: 40px 0;
    text-align: center;
  }

  .health-banner {
    display: flex;
    align-items: center;
    gap: 20px;
    padding: 20px 24px;
    border-radius: var(--radius);
    margin-bottom: 24px;
  }

  .health-banner.good { background: var(--green-dim); }
  .health-banner.warn { background: var(--yellow-dim); }
  .health-banner.bad { background: var(--red-dim); }

  .health-score {
    font-size: 40px;
    font-weight: 800;
    line-height: 1;
  }

  .good .health-score { color: var(--green); }
  .warn .health-score { color: var(--yellow); }
  .bad .health-score { color: var(--red); }

  .health-info { flex: 1; }

  .health-label {
    font-size: 13px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .health-desc {
    font-size: 16px;
    font-weight: 600;
    margin-top: 2px;
  }

  .health-uptime {
    font-size: 13px;
    color: var(--text-secondary);
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 32px;
  }

  .stat-card {
    background: var(--bg-card);
    border-radius: var(--radius);
    padding: 20px;
    border: 1px solid var(--border);
  }

  .stat-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 12px;
  }

  .stat-icon {
    font-size: 16px;
  }

  .stat-icon.blue { color: var(--blue); }
  .stat-icon.green { color: var(--green); }
  .stat-icon.yellow { color: var(--yellow); }
  .stat-icon.red { color: var(--red); }
  .stat-icon.accent { color: var(--accent); }

  .stat-label {
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 500;
  }

  .stat-value {
    font-size: 28px;
    font-weight: 700;
    margin-bottom: 12px;
  }

  .stat-bar {
    height: 4px;
    background: var(--bg-tertiary);
    border-radius: 2px;
    overflow: hidden;
    margin-bottom: 10px;
  }

  .stat-bar-fill {
    height: 100%;
    border-radius: 2px;
    transition: width 0.5s ease;
  }

  .stat-bar-fill.blue { background: var(--blue); }
  .stat-bar-fill.green { background: var(--green); }
  .stat-bar-fill.yellow { background: var(--yellow); }
  .stat-bar-fill.red { background: var(--red); }
  .stat-bar-fill.accent { background: var(--accent); }

  .stat-detail {
    font-size: 12px;
    color: var(--text-muted);
  }

  .quick-actions h2 {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 16px;
  }

  .actions-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
  }

  .action-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 24px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    color: var(--text-primary);
    transition: all var(--transition);
    text-align: center;
  }

  .action-card:hover {
    background: var(--bg-hover);
    border-color: var(--accent);
    transform: translateY(-2px);
  }

  .action-icon {
    font-size: 28px;
    color: var(--accent);
  }

  .action-label {
    font-size: 15px;
    font-weight: 600;
  }

  .action-desc {
    font-size: 12px;
    color: var(--text-secondary);
  }

  @media (max-width: 1000px) {
    .stats-grid {
      grid-template-columns: repeat(2, 1fr);
    }
  }
</style>
