<script>
  import { onMount } from 'svelte';
  import { ScanDirectory, GetDirectoryChildren, MoveToTrash, GetHomeDir } from '../../wailsjs/go/main/App.js';

  let scanResult = null;
  let scanning = false;
  let currentPath = '';
  let breadcrumbs = [];
  let children = [];
  let selectedPaths = new Set();
  let showDeleteConfirm = false;

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function getBarWidth(size, maxSize) {
    if (!maxSize) return 0;
    return Math.max(2, (size / maxSize) * 100);
  }

  function getBarColor(entry) {
    if (!entry.isDir) return 'var(--blue)';
    const age = Date.now() / 1000 - entry.modTime;
    const days = age / 86400;
    if (days > 365) return 'var(--red)';
    if (days > 90) return 'var(--yellow)';
    return 'var(--accent)';
  }

  async function startScan(path) {
    scanning = true;
    selectedPaths = new Set();
    try {
      scanResult = await ScanDirectory(path, 3);
      currentPath = path;
      updateBreadcrumbs(path);
      if (scanResult && scanResult.root && scanResult.root.children) {
        children = scanResult.root.children;
      } else {
        children = [];
      }
    } catch (e) {
      console.error('Scan failed:', e);
    }
    scanning = false;
  }

  async function drillDown(entry) {
    if (!entry.isDir) return;
    scanning = true;
    try {
      const items = await GetDirectoryChildren(entry.path);
      children = items || [];
      currentPath = entry.path;
      updateBreadcrumbs(entry.path);
    } catch (e) {
      console.error('Failed to load directory:', e);
    }
    scanning = false;
  }

  function updateBreadcrumbs(path) {
    const parts = path.split('/').filter(Boolean);
    breadcrumbs = parts.map((part, i) => ({
      name: part,
      path: '/' + parts.slice(0, i + 1).join('/')
    }));
  }

  async function navigateBreadcrumb(path) {
    await drillDown({ path, isDir: true });
  }

  function toggleSelect(path) {
    if (selectedPaths.has(path)) {
      selectedPaths.delete(path);
    } else {
      selectedPaths.add(path);
    }
    selectedPaths = selectedPaths;
  }

  async function deleteSelected() {
    showDeleteConfirm = false;
    for (const path of selectedPaths) {
      try {
        await MoveToTrash(path);
      } catch (e) {
        console.error('Failed to trash:', path, e);
      }
    }
    selectedPaths = new Set();
    // Refresh
    if (currentPath) {
      await drillDown({ path: currentPath, isDir: true });
    }
  }

  onMount(async () => {
    const home = await GetHomeDir();
    currentPath = home;
  });
</script>

<div class="analyze-page">
  <div class="page-header">
    <div class="header-row">
      <div>
        <h1>Disk Analysis</h1>
        <p class="subtitle">Visualize and manage disk usage</p>
      </div>
      <button class="btn-primary" on:click={() => startScan(currentPath)} disabled={scanning}>
        {scanning ? 'Scanning...' : 'Scan'}
      </button>
    </div>
  </div>

  {#if scanResult}
    <div class="scan-summary">
      <div class="summary-item">
        <span class="summary-value">{formatBytes(scanResult.totalSize)}</span>
        <span class="summary-label">Total Size</span>
      </div>
      <div class="summary-item">
        <span class="summary-value">{scanResult.totalFiles.toLocaleString()}</span>
        <span class="summary-label">Files</span>
      </div>
      <div class="summary-item">
        <span class="summary-value">{scanResult.totalDirs.toLocaleString()}</span>
        <span class="summary-label">Folders</span>
      </div>
    </div>
  {/if}

  <div class="breadcrumbs">
    <button class="crumb" on:click={() => navigateBreadcrumb('/')}>/</button>
    {#each breadcrumbs as crumb}
      <span class="crumb-sep">/</span>
      <button class="crumb" on:click={() => navigateBreadcrumb(crumb.path)}>{crumb.name}</button>
    {/each}
  </div>

  {#if selectedPaths.size > 0}
    <div class="selection-bar">
      <span>{selectedPaths.size} item(s) selected</span>
      <button class="btn-danger" on:click={() => showDeleteConfirm = true}>Move to Trash</button>
      <button class="btn-secondary-sm" on:click={() => selectedPaths = new Set()}>Deselect All</button>
    </div>
  {/if}

  <div class="file-list">
    {#if children.length === 0 && !scanning}
      <div class="empty">No items found. Click Scan to start.</div>
    {/if}

    {#each children as entry}
      {@const maxSize = children.length > 0 ? children[0].size : 1}
      <div class="file-row" class:is-dir={entry.isDir} class:selected={selectedPaths.has(entry.path)}>
        <button class="file-check" on:click|stopPropagation={() => toggleSelect(entry.path)}>
          {#if selectedPaths.has(entry.path)}
            <span class="check-on">&#10003;</span>
          {:else}
            <span class="check-off">&#9675;</span>
          {/if}
        </button>
        <button class="file-info" on:click={() => entry.isDir ? drillDown(entry) : null}>
          <span class="file-icon">{entry.isDir ? '&#128193;' : '&#128196;'}</span>
          <div class="file-details">
            <div class="file-name">{entry.name}</div>
            <div class="file-bar-container">
              <div class="file-bar" style="width: {getBarWidth(entry.size, maxSize)}%; background: {getBarColor(entry)}"></div>
            </div>
          </div>
          <span class="file-size">{formatBytes(entry.size)}</span>
        </button>
      </div>
    {/each}
  </div>

  {#if showDeleteConfirm}
    <div class="modal-overlay" on:click={() => showDeleteConfirm = false}>
      <div class="modal" on:click|stopPropagation>
        <h3>Confirm Delete</h3>
        <p class="modal-desc">Move {selectedPaths.size} item(s) to Trash?</p>
        <p class="modal-note">Items can be recovered from Trash</p>
        <div class="modal-actions">
          <button class="btn-secondary" on:click={() => showDeleteConfirm = false}>Cancel</button>
          <button class="btn-danger" on:click={deleteSelected}>Move to Trash</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .analyze-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 24px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }

  .header-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
  }

  .btn-primary {
    background: var(--accent);
    color: white;
    padding: 10px 24px;
    border-radius: var(--radius-sm);
    font-weight: 600;
    transition: all var(--transition);
  }
  .btn-primary:hover { background: var(--accent-hover); }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-secondary {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    padding: 10px 24px;
    border-radius: var(--radius-sm);
    font-weight: 500;
  }

  .btn-secondary-sm {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    padding: 6px 14px;
    border-radius: var(--radius-sm);
    font-size: 12px;
  }

  .btn-danger {
    background: var(--red-dim);
    color: var(--red);
    padding: 10px 24px;
    border-radius: var(--radius-sm);
    font-weight: 600;
    transition: all var(--transition);
  }
  .btn-danger:hover { background: rgba(248, 113, 113, 0.25); }

  .scan-summary {
    display: flex;
    gap: 24px;
    margin-bottom: 20px;
    padding: 16px 20px;
    background: var(--bg-card);
    border-radius: var(--radius);
    border: 1px solid var(--border);
  }

  .summary-item { display: flex; flex-direction: column; }
  .summary-value { font-size: 20px; font-weight: 700; }
  .summary-label { font-size: 12px; color: var(--text-secondary); margin-top: 2px; }

  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 10px 0;
    margin-bottom: 12px;
    overflow-x: auto;
    white-space: nowrap;
  }

  .crumb {
    background: none;
    color: var(--text-secondary);
    font-size: 13px;
    padding: 2px 4px;
    border-radius: 4px;
  }

  .crumb:hover { color: var(--accent); background: var(--accent-dim); }
  .crumb-sep { color: var(--text-muted); font-size: 12px; }

  .selection-bar {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    background: var(--accent-dim);
    border-radius: var(--radius-sm);
    margin-bottom: 12px;
    font-size: 13px;
    color: var(--accent);
  }

  .file-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: calc(100vh - 350px);
    overflow-y: auto;
  }

  .empty {
    text-align: center;
    padding: 40px;
    color: var(--text-secondary);
  }

  .file-row {
    display: flex;
    align-items: center;
    border-radius: var(--radius-sm);
    transition: background var(--transition);
  }

  .file-row:hover { background: var(--bg-hover); }
  .file-row.selected { background: var(--accent-dim); }

  .file-check {
    background: none;
    padding: 12px 8px 12px 12px;
    font-size: 16px;
    flex-shrink: 0;
  }

  .check-on { color: var(--accent); }
  .check-off { color: var(--text-muted); }

  .file-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 12px 10px 0;
    background: none;
    color: var(--text-primary);
    text-align: left;
  }

  .is-dir .file-info { cursor: pointer; }

  .file-icon { font-size: 18px; flex-shrink: 0; }
  .file-details { flex: 1; min-width: 0; }
  .file-name {
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .file-bar-container {
    height: 3px;
    background: var(--bg-tertiary);
    border-radius: 2px;
    margin-top: 6px;
    overflow: hidden;
  }

  .file-bar {
    height: 100%;
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .file-size {
    font-size: 12px;
    color: var(--text-muted);
    font-weight: 500;
    flex-shrink: 0;
    min-width: 70px;
    text-align: right;
  }

  .modal-overlay {
    position: fixed; top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex; align-items: center; justify-content: center;
    z-index: 100;
  }

  .modal {
    background: var(--bg-secondary);
    border-radius: var(--radius);
    padding: 24px;
    max-width: 400px;
    width: 90%;
    border: 1px solid var(--border);
  }

  .modal h3 { font-size: 18px; margin-bottom: 12px; }
  .modal-desc { color: var(--text-secondary); margin-bottom: 8px; }
  .modal-note { font-size: 12px; color: var(--green); margin-bottom: 20px; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 12px; }
</style>
