<script>
  import { onDestroy } from 'svelte';
  import {
    GetSystemStatus,
    GetTopProcesses,
    GetAllProcesses,
    KillProcess,
    GetCPUDetail,
    GetMemoryDetail,
    GetDiskDetail,
    GetBatteryDetail
  } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  // ---- State ----
  let status = null;
  let processes = [];
  let cpuHistory = [];
  let memHistory = [];
  let interval;
  const MAX_HISTORY = 30;

  // Expanded detail panel
  let expandedCard = null; // 'cpu' | 'memory' | 'disk' | 'battery' | null
  let cpuDetail = null;
  let memoryDetail = null;
  let diskDetail = null;
  let batteryDetail = null;

  // Top processes sort
  let topSortKey = 'cpuUsage';
  let topSortAsc = false;

  // All processes overlay
  let showAllProcesses = false;
  let allProcesses = [];
  let allProcSearch = '';
  let allSortKey = 'cpuUsage';
  let allSortAsc = false;

  // Kill modal
  let killTarget = null; // { name, pid }

  // ---- Helpers ----
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

  function sortProcesses(procs, key, asc) {
    return [...procs].sort((a, b) => {
      let av = a[key];
      let bv = b[key];
      if (typeof av === 'string') {
        av = av.toLowerCase();
        bv = bv.toLowerCase();
      }
      if (av < bv) return asc ? -1 : 1;
      if (av > bv) return asc ? 1 : -1;
      return 0;
    });
  }

  function sortArrow(currentKey, sortKey, sortAsc) {
    if (currentKey !== sortKey) return '';
    return sortAsc ? '\u25B2' : '\u25BC';
  }

  // ---- Sorting handlers ----
  function toggleTopSort(key) {
    if (topSortKey === key) {
      topSortAsc = !topSortAsc;
    } else {
      topSortKey = key;
      topSortAsc = false;
    }
  }

  function toggleAllSort(key) {
    if (allSortKey === key) {
      allSortAsc = !allSortAsc;
    } else {
      allSortKey = key;
      allSortAsc = false;
    }
  }

  // ---- Derived sorted data ----
  $: sortedTopProcesses = sortProcesses(processes, topSortKey, topSortAsc);

  $: filteredAllProcesses = allProcesses.filter(p => {
    if (!allProcSearch) return true;
    const q = allProcSearch.toLowerCase();
    return p.name.toLowerCase().includes(q) || String(p.pid).includes(q);
  });

  $: sortedAllProcesses = sortProcesses(filteredAllProcesses, allSortKey, allSortAsc);

  // ---- Data fetching ----
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

  async function fetchDetail(card) {
    try {
      if (card === 'cpu') cpuDetail = await GetCPUDetail();
      if (card === 'memory') memoryDetail = await GetMemoryDetail();
      if (card === 'disk') diskDetail = await GetDiskDetail();
      if (card === 'battery') batteryDetail = await GetBatteryDetail();
    } catch (e) {
      console.error('Detail fetch error:', e);
    }
  }

  async function openAllProcesses() {
    showAllProcesses = true;
    allProcSearch = '';
    try {
      allProcesses = await GetAllProcesses();
    } catch (e) {
      console.error('All processes fetch error:', e);
    }
  }

  async function refreshAllProcesses() {
    try {
      allProcesses = await GetAllProcesses();
    } catch (e) {
      console.error('Refresh all processes error:', e);
    }
  }

  function closeAllProcesses() {
    showAllProcesses = false;
  }

  // ---- Card expansion ----
  function toggleCard(card) {
    if (expandedCard === card) {
      expandedCard = null;
    } else {
      expandedCard = card;
      fetchDetail(card);
    }
  }

  // ---- Kill process ----
  function requestKill(proc) {
    killTarget = { name: proc.name, pid: proc.pid };
  }

  function cancelKill() {
    killTarget = null;
  }

  async function confirmKill() {
    if (!killTarget) return;
    try {
      const result = await KillProcess(killTarget.pid);
      if (result.success) {
        // Refresh data after kill
        await fetchData();
        if (showAllProcesses) await refreshAllProcesses();
      } else {
        console.error('Kill failed:', result.message);
      }
    } catch (e) {
      console.error('Kill error:', e);
    }
    killTarget = null;
  }

  // ---- Polling ----
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
</script>

<div class="monitor-page">
  <div class="page-header">
    <h1>System Monitor</h1>
    <p class="subtitle">Real-time system performance</p>
  </div>

  {#if status}
    <!-- Gauge Cards -->
    <div class="gauges">
      <div
        class="gauge-card"
        class:expanded={expandedCard === 'cpu'}
        on:click={() => toggleCard('cpu')}
        on:keydown={(e) => e.key === 'Enter' && toggleCard('cpu')}
        tabindex="0"
        role="button"
      >
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

      <div
        class="gauge-card"
        class:expanded={expandedCard === 'memory'}
        on:click={() => toggleCard('memory')}
        on:keydown={(e) => e.key === 'Enter' && toggleCard('memory')}
        tabindex="0"
        role="button"
      >
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

      <div
        class="gauge-card"
        class:expanded={expandedCard === 'disk'}
        on:click={() => toggleCard('disk')}
        on:keydown={(e) => e.key === 'Enter' && toggleCard('disk')}
        tabindex="0"
        role="button"
      >
        <div class="gauge-header">
          <span class="gauge-label">Disk</span>
          <span class="gauge-value">{status.disk.usage.toFixed(1)}%</span>
        </div>
        <div class="gauge-bar">
          <div class="gauge-fill" class:yellow={status.disk.usage > 70} class:red={status.disk.usage > 90} class:accent={status.disk.usage <= 70} style="width: {status.disk.usage}%"></div>
        </div>
        <div class="gauge-detail">{formatBytes(status.disk.used)} / {formatBytes(status.disk.total)}</div>
      </div>

      <div
        class="gauge-card"
        class:expanded={expandedCard === 'battery'}
        on:click={() => toggleCard('battery')}
        on:keydown={(e) => e.key === 'Enter' && toggleCard('battery')}
        tabindex="0"
        role="button"
      >
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

    <!-- Detail Panel (below all gauge cards) -->
    {#if expandedCard === 'cpu' && cpuDetail}
      <div class="detail-panel">
        <div class="detail-title">CPU Details</div>
        <div class="detail-grid">
          <div class="detail-item full-width">
            <span class="detail-label">Model</span>
            <span class="detail-value">{cpuDetail.model}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Cores</span>
            <span class="detail-value">{cpuDetail.cores}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Total Usage</span>
            <span class="detail-value">{cpuDetail.usage.toFixed(1)}%</span>
          </div>
        </div>
        <div class="detail-bars">
          <div class="detail-bar-row">
            <span class="bar-label">User</span>
            <div class="bar-track">
              <div class="bar-fill blue" style="width: {cpuDetail.user}%"></div>
            </div>
            <span class="bar-value">{cpuDetail.user.toFixed(1)}%</span>
          </div>
          <div class="detail-bar-row">
            <span class="bar-label">System</span>
            <div class="bar-track">
              <div class="bar-fill accent" style="width: {cpuDetail.sys}%"></div>
            </div>
            <span class="bar-value">{cpuDetail.sys.toFixed(1)}%</span>
          </div>
          <div class="detail-bar-row">
            <span class="bar-label">Idle</span>
            <div class="bar-track">
              <div class="bar-fill idle" style="width: {cpuDetail.idle}%"></div>
            </div>
            <span class="bar-value">{cpuDetail.idle.toFixed(1)}%</span>
          </div>
        </div>
      </div>
    {/if}

    {#if expandedCard === 'memory' && memoryDetail}
      <div class="detail-panel">
        <div class="detail-title">Memory Details</div>
        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-label">Total</span>
            <span class="detail-value">{formatBytes(memoryDetail.total)}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Used</span>
            <span class="detail-value">{formatBytes(memoryDetail.used)}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Available</span>
            <span class="detail-value">{formatBytes(memoryDetail.available)}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Usage</span>
            <span class="detail-value">{memoryDetail.usage.toFixed(1)}%</span>
          </div>
        </div>
        <div class="detail-subtitle">Memory Composition</div>
        <div class="mem-stacked-bar">
          {#if memoryDetail.total > 0}
            <div class="mem-segment active" style="width: {(memoryDetail.active / memoryDetail.total * 100).toFixed(1)}%" title="Active: {formatBytes(memoryDetail.active)}"></div>
            <div class="mem-segment wired" style="width: {(memoryDetail.wired / memoryDetail.total * 100).toFixed(1)}%" title="Wired: {formatBytes(memoryDetail.wired)}"></div>
            <div class="mem-segment inactive" style="width: {(memoryDetail.inactive / memoryDetail.total * 100).toFixed(1)}%" title="Inactive: {formatBytes(memoryDetail.inactive)}"></div>
            <div class="mem-segment free" style="width: {(memoryDetail.free / memoryDetail.total * 100).toFixed(1)}%" title="Free: {formatBytes(memoryDetail.free)}"></div>
          {/if}
        </div>
        <div class="mem-legend">
          <span class="legend-item"><span class="legend-dot active"></span> Active {formatBytes(memoryDetail.active)}</span>
          <span class="legend-item"><span class="legend-dot wired"></span> Wired {formatBytes(memoryDetail.wired)}</span>
          <span class="legend-item"><span class="legend-dot inactive"></span> Inactive {formatBytes(memoryDetail.inactive)}</span>
          <span class="legend-item"><span class="legend-dot free"></span> Free {formatBytes(memoryDetail.free)}</span>
        </div>
      </div>
    {/if}

    {#if expandedCard === 'disk' && diskDetail}
      <div class="detail-panel">
        <div class="detail-title">Disk Details</div>
        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-label">Mount Path</span>
            <span class="detail-value mono">{diskDetail.mountPath}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Filesystem</span>
            <span class="detail-value mono">{diskDetail.fsType}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Total</span>
            <span class="detail-value">{formatBytes(diskDetail.total)}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Used</span>
            <span class="detail-value">{formatBytes(diskDetail.used)}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Available</span>
            <span class="detail-value">{formatBytes(diskDetail.available)}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Usage</span>
            <span class="detail-value">{diskDetail.usage.toFixed(1)}%</span>
          </div>
        </div>
        <div class="disk-usage-visual">
          <div class="disk-ring-container">
            <svg viewBox="0 0 120 120" class="disk-ring">
              <circle cx="60" cy="60" r="50" fill="none" stroke="var(--bg-tertiary)" stroke-width="10" />
              <circle cx="60" cy="60" r="50" fill="none" stroke="{diskDetail.usage > 90 ? 'var(--red)' : diskDetail.usage > 70 ? 'var(--yellow)' : 'var(--accent)'}" stroke-width="10" stroke-linecap="round" stroke-dasharray="{diskDetail.usage * 3.14} 314" transform="rotate(-90 60 60)" />
            </svg>
            <span class="disk-ring-label">{diskDetail.usage.toFixed(0)}%</span>
          </div>
        </div>
      </div>
    {/if}

    {#if expandedCard === 'battery' && batteryDetail}
      <div class="detail-panel">
        <div class="detail-title">Battery Details</div>
        <div class="battery-hero">
          <div class="battery-percent" class:charging={batteryDetail.isCharging} class:low={batteryDetail.percentage < 20}>
            {batteryDetail.percentage}%
          </div>
          <div class="battery-status">{batteryDetail.status}{batteryDetail.isCharging ? ' (Charging)' : ''}</div>
        </div>
        <div class="detail-grid">
          {#if batteryDetail.timeRemaining}
            <div class="detail-item">
              <span class="detail-label">Time Remaining</span>
              <span class="detail-value">{batteryDetail.timeRemaining}</span>
            </div>
          {/if}
          <div class="detail-item">
            <span class="detail-label">Cycle Count</span>
            <span class="detail-value">{batteryDetail.cycleCount || 'N/A'}</span>
          </div>
          <div class="detail-item">
            <span class="detail-label">Condition</span>
            <span class="detail-value">{batteryDetail.condition || 'N/A'}</span>
          </div>
        </div>
      </div>
    {/if}

    <!-- Top Processes -->
    <div class="processes-section">
      <div class="processes-header">
        <h2>Top Processes</h2>
        <button class="btn-all-processes" on:click={openAllProcesses}>All Processes</button>
      </div>
      <div class="process-table">
        <div class="process-header">
          <span class="col-name sortable" on:click={() => toggleTopSort('name')} on:keydown={(e) => e.key === 'Enter' && toggleTopSort('name')} tabindex="0" role="columnheader">
            Process {sortArrow('name', topSortKey, topSortAsc)}
          </span>
          <span class="col-cpu sortable" on:click={() => toggleTopSort('cpuUsage')} on:keydown={(e) => e.key === 'Enter' && toggleTopSort('cpuUsage')} tabindex="0" role="columnheader">
            CPU % {sortArrow('cpuUsage', topSortKey, topSortAsc)}
          </span>
          <span class="col-mem sortable" on:click={() => toggleTopSort('memUsage')} on:keydown={(e) => e.key === 'Enter' && toggleTopSort('memUsage')} tabindex="0" role="columnheader">
            MEM % {sortArrow('memUsage', topSortKey, topSortAsc)}
          </span>
          <span class="col-pid sortable" on:click={() => toggleTopSort('pid')} on:keydown={(e) => e.key === 'Enter' && toggleTopSort('pid')} tabindex="0" role="columnheader">
            PID {sortArrow('pid', topSortKey, topSortAsc)}
          </span>
          <span class="col-kill"></span>
        </div>
        {#each sortedTopProcesses as proc (proc.pid)}
          <div class="process-row">
            <span class="col-name" title={proc.name}>{proc.name}</span>
            <span class="col-cpu">
              <span class="proc-bar-bg">
                <span class="proc-bar blue" style="width: {Math.max(proc.cpuUsage > 0 ? 2 : 0, Math.min(proc.cpuUsage, 100))}%"></span>
              </span>
              <span class="proc-value">{proc.cpuUsage.toFixed(1)}</span>
            </span>
            <span class="col-mem">
              <span class="proc-bar-bg">
                <span class="proc-bar green" style="width: {Math.max(proc.memUsage > 0 ? 2 : 0, Math.min(proc.memUsage, 100))}%"></span>
              </span>
              <span class="proc-value">{proc.memUsage.toFixed(1)}</span>
            </span>
            <span class="col-pid">{proc.pid}</span>
            <span class="col-kill">
              <button class="kill-btn" on:click|stopPropagation={() => requestKill(proc)} title="Kill process">&#10005;</button>
            </span>
          </div>
        {/each}
      </div>
    </div>

    <!-- Health Row -->
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

  <!-- All Processes Overlay -->
  {#if showAllProcesses}
    <div class="overlay-backdrop" on:click={closeAllProcesses} on:keydown={(e) => e.key === 'Escape' && closeAllProcesses()} tabindex="-1" role="dialog">
      <div class="overlay-panel" on:click|stopPropagation on:keydown|stopPropagation role="document">
        <div class="overlay-header">
          <button class="overlay-back" on:click={closeAllProcesses}>&#8592; Back</button>
          <h2>All Processes</h2>
          <span class="proc-count">{sortedAllProcesses.length} processes</span>
          <button class="overlay-refresh" on:click={refreshAllProcesses}>Refresh</button>
        </div>
        <div class="overlay-search">
          <input
            type="text"
            placeholder="Search by name or PID..."
            bind:value={allProcSearch}
          />
        </div>
        <div class="overlay-table-wrap">
          <div class="process-table overlay-table">
            <div class="process-header">
              <span class="col-name sortable" on:click={() => toggleAllSort('name')} on:keydown={(e) => e.key === 'Enter' && toggleAllSort('name')} tabindex="0" role="columnheader">
                Process {sortArrow('name', allSortKey, allSortAsc)}
              </span>
              <span class="col-cpu sortable" on:click={() => toggleAllSort('cpuUsage')} on:keydown={(e) => e.key === 'Enter' && toggleAllSort('cpuUsage')} tabindex="0" role="columnheader">
                CPU % {sortArrow('cpuUsage', allSortKey, allSortAsc)}
              </span>
              <span class="col-mem sortable" on:click={() => toggleAllSort('memUsage')} on:keydown={(e) => e.key === 'Enter' && toggleAllSort('memUsage')} tabindex="0" role="columnheader">
                MEM % {sortArrow('memUsage', allSortKey, allSortAsc)}
              </span>
              <span class="col-pid sortable" on:click={() => toggleAllSort('pid')} on:keydown={(e) => e.key === 'Enter' && toggleAllSort('pid')} tabindex="0" role="columnheader">
                PID {sortArrow('pid', allSortKey, allSortAsc)}
              </span>
              <span class="col-kill"></span>
            </div>
            {#each sortedAllProcesses as proc (proc.pid)}
              <div class="process-row">
                <span class="col-name" title={proc.name}>{proc.name}</span>
                <span class="col-cpu">
                  <span class="proc-bar-bg">
                    <span class="proc-bar blue" style="width: {Math.max(proc.cpuUsage > 0 ? 2 : 0, Math.min(proc.cpuUsage, 100))}%"></span>
                  </span>
                  <span class="proc-value">{proc.cpuUsage.toFixed(1)}</span>
                </span>
                <span class="col-mem">
                  <span class="proc-bar-bg">
                    <span class="proc-bar green" style="width: {Math.max(proc.memUsage > 0 ? 2 : 0, Math.min(proc.memUsage, 100))}%"></span>
                  </span>
                  <span class="proc-value">{proc.memUsage.toFixed(1)}</span>
                </span>
                <span class="col-pid">{proc.pid}</span>
                <span class="col-kill">
                  <button class="kill-btn" on:click|stopPropagation={() => requestKill(proc)} title="Kill process">&#10005;</button>
                </span>
              </div>
            {/each}
          </div>
        </div>
      </div>
    </div>
  {/if}

  <!-- Kill Confirmation Modal -->
  {#if killTarget}
    <div class="modal-backdrop" on:click={cancelKill} on:keydown={(e) => e.key === 'Escape' && cancelKill()} tabindex="-1" role="dialog">
      <div class="modal" on:click|stopPropagation on:keydown|stopPropagation role="alertdialog">
        <div class="modal-title">Kill Process?</div>
        <div class="modal-body">
          <div class="modal-proc-name">{killTarget.name}</div>
          <div class="modal-proc-pid">PID: {killTarget.pid}</div>
          <div class="modal-warning">This will terminate the process</div>
        </div>
        <div class="modal-actions">
          <button class="modal-btn cancel" on:click={cancelKill}>Cancel</button>
          <button class="modal-btn danger" on:click={confirmKill}>Kill</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .monitor-page {
    padding: 0 32px 32px;
    position: relative;
  }

  .page-header { margin-bottom: 24px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }
  .loading { color: var(--text-secondary); padding: 40px 0; text-align: center; }

  /* ---- Gauge Cards Grid ---- */
  .gauges {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 16px;
  }

  .gauge-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 20px;
    cursor: pointer;
    transition: border-color var(--transition), box-shadow var(--transition), transform var(--transition);
  }

  .gauge-card:hover {
    border-color: var(--text-muted);
    transform: translateY(-1px);
  }

  .gauge-card.expanded {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent), 0 4px 16px rgba(124, 92, 252, 0.12);
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

  /* ---- Detail Panel ---- */
  .detail-panel {
    background: var(--bg-card);
    border: 1px solid var(--accent);
    border-radius: var(--radius);
    padding: 24px;
    margin-bottom: 20px;
    animation: slideDown 0.2s ease;
  }

  @keyframes slideDown {
    from { opacity: 0; transform: translateY(-8px); }
    to { opacity: 1; transform: translateY(0); }
  }

  .detail-title {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 16px;
    color: var(--text-primary);
  }

  .detail-subtitle {
    font-size: 13px;
    font-weight: 500;
    color: var(--text-secondary);
    margin: 16px 0 8px;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 12px;
    margin-bottom: 4px;
  }

  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .detail-item.full-width {
    grid-column: 1 / -1;
  }

  .detail-label {
    font-size: 11px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .detail-value {
    font-size: 14px;
    color: var(--text-primary);
    font-weight: 500;
  }

  .detail-value.mono {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 13px;
  }

  /* CPU detail bars */
  .detail-bars {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 16px;
  }

  .detail-bar-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .bar-label {
    font-size: 12px;
    color: var(--text-secondary);
    width: 56px;
    flex-shrink: 0;
  }

  .bar-track {
    flex: 1;
    height: 8px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    overflow: hidden;
  }

  .bar-fill {
    height: 100%;
    border-radius: 4px;
    transition: width 0.5s ease;
  }

  .bar-fill.blue { background: var(--blue); }
  .bar-fill.accent { background: var(--accent); }
  .bar-fill.idle { background: var(--text-muted); opacity: 0.4; }

  .bar-value {
    font-size: 12px;
    color: var(--text-secondary);
    width: 48px;
    text-align: right;
    flex-shrink: 0;
  }

  /* Memory stacked bar */
  .mem-stacked-bar {
    display: flex;
    height: 14px;
    border-radius: 7px;
    overflow: hidden;
    background: var(--bg-tertiary);
  }

  .mem-segment {
    height: 100%;
    transition: width 0.5s ease;
    min-width: 0;
  }

  .mem-segment.active { background: var(--blue); }
  .mem-segment.wired { background: var(--accent); }
  .mem-segment.inactive { background: var(--yellow); }
  .mem-segment.free { background: var(--green); opacity: 0.5; }

  .mem-legend {
    display: flex;
    gap: 16px;
    margin-top: 10px;
    flex-wrap: wrap;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .legend-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .legend-dot.active { background: var(--blue); }
  .legend-dot.wired { background: var(--accent); }
  .legend-dot.inactive { background: var(--yellow); }
  .legend-dot.free { background: var(--green); opacity: 0.5; }

  /* Disk ring */
  .disk-usage-visual {
    display: flex;
    justify-content: center;
    margin-top: 16px;
  }

  .disk-ring-container {
    position: relative;
    width: 120px;
    height: 120px;
  }

  .disk-ring {
    width: 100%;
    height: 100%;
  }

  .disk-ring-label {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    font-size: 22px;
    font-weight: 700;
    color: var(--text-primary);
  }

  /* Battery hero */
  .battery-hero {
    text-align: center;
    margin-bottom: 20px;
  }

  .battery-percent {
    font-size: 48px;
    font-weight: 700;
    color: var(--green);
    line-height: 1;
    margin-bottom: 6px;
  }

  .battery-percent.charging { color: var(--blue); }
  .battery-percent.low { color: var(--red); }

  .battery-status {
    font-size: 14px;
    color: var(--text-secondary);
  }

  /* ---- Processes Section ---- */
  .processes-section {
    margin-bottom: 24px;
  }

  .processes-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .processes-header h2 {
    font-size: 16px;
    font-weight: 600;
  }

  .btn-all-processes {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    padding: 6px 14px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 500;
    transition: background var(--transition), color var(--transition);
  }

  .btn-all-processes:hover {
    background: var(--accent-dim);
    color: var(--accent);
  }

  .process-table {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    overflow: hidden;
  }

  .process-header, .process-row {
    display: grid;
    grid-template-columns: 1fr 130px 130px 70px 36px;
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
    transition: background var(--transition);
  }

  .process-row:hover {
    background: var(--bg-hover);
  }

  .sortable {
    cursor: pointer;
    user-select: none;
    transition: color var(--transition);
  }

  .sortable:hover {
    color: var(--text-primary);
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
    transition: width 0.3s ease;
  }

  .proc-bar.blue { background: var(--blue); }
  .proc-bar.green { background: var(--green); }

  .proc-value {
    font-size: 12px;
    color: var(--text-secondary);
    min-width: 32px;
    text-align: right;
    flex-shrink: 0;
  }

  .col-kill {
    display: flex;
    justify-content: center;
    align-items: center;
  }

  .kill-btn {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    background: transparent;
    color: var(--text-muted);
    font-size: 12px;
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    transition: opacity var(--transition), background var(--transition), color var(--transition);
  }

  .process-row:hover .kill-btn {
    opacity: 1;
  }

  .kill-btn:hover {
    background: var(--red-dim);
    color: var(--red);
  }

  /* ---- Health Row ---- */
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

  /* ---- All Processes Overlay ---- */
  .overlay-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: fadeIn 0.15s ease;
  }

  @keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
  }

  .overlay-panel {
    width: 90%;
    max-width: 840px;
    max-height: 85vh;
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    animation: scaleIn 0.15s ease;
  }

  @keyframes scaleIn {
    from { transform: scale(0.96); opacity: 0; }
    to { transform: scale(1); opacity: 1; }
  }

  .overlay-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .overlay-header h2 {
    font-size: 16px;
    font-weight: 600;
  }

  .proc-count {
    font-size: 12px;
    color: var(--text-muted);
    margin-left: auto;
  }

  .overlay-back {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    padding: 6px 12px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 500;
    transition: background var(--transition), color var(--transition);
  }

  .overlay-back:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .overlay-refresh {
    background: var(--accent-dim);
    color: var(--accent);
    padding: 6px 14px;
    border-radius: var(--radius-sm);
    font-size: 12px;
    font-weight: 500;
    transition: background var(--transition);
  }

  .overlay-refresh:hover {
    background: var(--accent);
    color: #fff;
  }

  .overlay-search {
    padding: 12px 20px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .overlay-search input {
    width: 100%;
    padding: 10px 14px;
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-radius: var(--radius-sm);
    font-size: 13px;
  }

  .overlay-search input::placeholder {
    color: var(--text-muted);
  }

  .overlay-table-wrap {
    overflow-y: auto;
    flex: 1;
  }

  .overlay-table {
    border: none;
    border-radius: 0;
  }

  .overlay-table .process-header {
    position: sticky;
    top: 0;
    z-index: 1;
  }

  /* ---- Kill Modal ---- */
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: fadeIn 0.1s ease;
  }

  .modal {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 28px;
    width: 360px;
    max-width: 90vw;
    animation: scaleIn 0.12s ease;
  }

  .modal-title {
    font-size: 18px;
    font-weight: 700;
    margin-bottom: 16px;
    color: var(--text-primary);
  }

  .modal-body {
    margin-bottom: 24px;
  }

  .modal-proc-name {
    font-size: 15px;
    font-weight: 600;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .modal-proc-pid {
    font-size: 13px;
    color: var(--text-muted);
    font-family: 'SF Mono', 'Fira Code', monospace;
    margin-bottom: 12px;
  }

  .modal-warning {
    font-size: 13px;
    color: var(--red);
    padding: 8px 12px;
    background: var(--red-dim);
    border-radius: var(--radius-sm);
  }

  .modal-actions {
    display: flex;
    gap: 10px;
    justify-content: flex-end;
  }

  .modal-btn {
    padding: 8px 20px;
    border-radius: var(--radius-sm);
    font-size: 13px;
    font-weight: 600;
    transition: background var(--transition), color var(--transition);
  }

  .modal-btn.cancel {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
  }

  .modal-btn.cancel:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .modal-btn.danger {
    background: var(--red);
    color: #fff;
  }

  .modal-btn.danger:hover {
    background: #ef4444;
  }
</style>
