<script>
  import { GetOperationHistory } from '../../wailsjs/go/main/App.js';

  export let visible = true;

  let history = [];
  let loading = true;
  let loaded = false;

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function formatDate(ts) {
    try {
      const d = new Date(ts);
      return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    } catch {
      return ts;
    }
  }

  async function loadHistory() {
    loading = true;
    try {
      history = await GetOperationHistory(50);
    } catch (e) {
      console.error('Failed to load history:', e);
    }
    loading = false;
    loaded = true;
  }

  // Refresh history each time the tab becomes visible
  $: if (visible) { loadHistory(); }
</script>

<div class="history-page">
  <div class="page-header">
    <h1>Operation History</h1>
    <p class="subtitle">Audit trail of all operations</p>
  </div>

  {#if loading}
    <div class="loading">Loading history...</div>
  {:else if history.length === 0}
    <div class="empty-state">
      <div class="empty-icon">&#9201;</div>
      <p>No operations recorded yet</p>
      <p class="empty-sub">Actions you take will appear here</p>
    </div>
  {:else}
    <div class="history-list">
      {#each history as entry}
        <div class="history-item">
          <div class="history-icon" class:success={entry.status === 'success'} class:error={entry.status !== 'success'}>
            {entry.status === 'success' ? '&#10003;' : '&#10007;'}
          </div>
          <div class="history-info">
            <div class="history-op">{entry.operation}</div>
            <div class="history-path" title={entry.path}>{entry.path}</div>
          </div>
          <div class="history-meta">
            <div class="history-size">{formatBytes(entry.size)}</div>
            <div class="history-time">{formatDate(entry.timestamp)}</div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .history-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 24px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }
  .loading { color: var(--text-secondary); padding: 40px 0; text-align: center; }

  .empty-state {
    text-align: center;
    padding: 60px 0;
  }
  .empty-icon { font-size: 48px; color: var(--text-muted); margin-bottom: 16px; }
  .empty-state p { color: var(--text-secondary); }
  .empty-sub { font-size: 12px; color: var(--text-muted); margin-top: 8px; }

  .history-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: calc(100vh - 200px);
    overflow-y: auto;
  }

  .history-item {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 14px 16px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
  }

  .history-item:hover {
    background: var(--bg-hover);
  }

  .history-icon {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 14px;
    flex-shrink: 0;
  }

  .history-icon.success {
    background: var(--green-dim);
    color: var(--green);
  }

  .history-icon.error {
    background: var(--red-dim);
    color: var(--red);
  }

  .history-info {
    flex: 1;
    min-width: 0;
  }

  .history-op {
    font-weight: 600;
    text-transform: capitalize;
    font-size: 13px;
  }

  .history-path {
    font-size: 12px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    margin-top: 2px;
  }

  .history-meta {
    text-align: right;
    flex-shrink: 0;
  }

  .history-size {
    font-size: 13px;
    font-weight: 500;
  }

  .history-time {
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 2px;
  }
</style>
