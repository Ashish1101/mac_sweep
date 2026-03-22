<script>
  import { GetOperationHistory, RestoreFromTrash, RestoreAllFromTrash, EmptyTrash, GetTrashItems, CanAccessTrash, MoveToTrash } from '../../wailsjs/go/main/App.js';
  import { settings } from '../stores/settings.js';
  import { playDeleteSound, playRestoreSound } from '../stores/sound.js';

  export let visible = true;

  let history = [];
  let trashNames = new Set();
  let hasTrashAccess = false;
  let loading = true;
  let loaded = false;
  let actionInProgress = null;
  let showRestoreAllDialog = false;
  let restoreAllInProgress = false;
  let showEmptyTrashDialog = false;
  let emptyTrashInProgress = false;
  let errorMsg = null;

  function showError(msg) {
    errorMsg = msg;
    setTimeout(() => { errorMsg = null; }, 5000);
  }

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function timeAgo(ts) {
    try {
      const now = Date.now();
      const then = new Date(ts).getTime();
      const diff = now - then;
      const mins = Math.floor(diff / 60000);
      if (mins < 1) return 'just now';
      if (mins < 60) return `${mins}m ago`;
      const hrs = Math.floor(mins / 60);
      if (hrs < 24) return `${hrs}h ago`;
      const days = Math.floor(hrs / 24);
      if (days < 30) return `${days}d ago`;
      return `${Math.floor(days / 30)}mo ago`;
    } catch {
      return '';
    }
  }

  function daysRemaining(ts, retentionDays) {
    if (!retentionDays || retentionDays <= 0) return null;
    try {
      const trashDate = new Date(ts).getTime();
      const expiresAt = trashDate + retentionDays * 86400000;
      const remaining = expiresAt - Date.now();
      if (remaining <= 0) return 'overdue';
      const days = Math.ceil(remaining / 86400000);
      if (days === 1) return '1 day left';
      return `${days} days left`;
    } catch {
      return null;
    }
  }

  function getFileName(path) {
    return path.split('/').pop() || path;
  }

  async function loadHistory() {
    loading = true;
    try {
      const [hist, items, access] = await Promise.all([
        GetOperationHistory(200),
        GetTrashItems(),
        CanAccessTrash()
      ]);
      history = hist;
      trashNames = new Set(items || []);
      hasTrashAccess = access;
    } catch (e) {
      console.error('Failed to load history:', e);
    }
    loading = false;
    loaded = true;
  }

  $: dedupedHistory = (() => {
    const seen = new Map();
    for (const entry of history) {
      if (entry.status !== 'success') continue;
      if (entry.operation !== 'trash' && entry.operation !== 'restore') continue;
      if (!seen.has(entry.path)) {
        seen.set(entry.path, entry);
      }
    }
    return [...seen.values()];
  })();

  $: restorableCount = dedupedHistory.filter(e =>
    e.operation === 'trash' && trashNames.has(getFileName(e.path))
  ).length;

  function getState(entry) {
    const inTrash = trashNames.has(getFileName(entry.path));
    if (entry.operation === 'restore') return 'restored';
    if (entry.operation === 'trash' && inTrash) return 'in-trash';
    if (entry.operation === 'trash' && !inTrash) return 'gone';
    return 'unknown';
  }

  async function restoreItem(path) {
    actionInProgress = path;
    try {
      await RestoreFromTrash(path);
      playRestoreSound();
      await loadHistory();
    } catch (e) {
      showError('Restore failed: ' + (e?.message || e));
    }
    actionInProgress = null;
  }

  async function deleteItem(path) {
    actionInProgress = path;
    try {
      await MoveToTrash(path);
      playDeleteSound();
      await loadHistory();
    } catch (e) {
      showError('Delete failed: ' + (e?.message || e));
    }
    actionInProgress = null;
  }

  async function confirmRestoreAll() {
    restoreAllInProgress = true;
    try {
      await RestoreAllFromTrash();
      showRestoreAllDialog = false;
      await loadHistory();
    } catch (e) {
      showError('Restore all failed: ' + (e?.message || e));
    }
    restoreAllInProgress = false;
  }

  async function confirmEmptyTrash() {
    emptyTrashInProgress = true;
    try {
      await EmptyTrash();
      showEmptyTrashDialog = false;
      await loadHistory();
    } catch (e) {
      showError('Empty trash failed: ' + (e?.message || e));
    }
    emptyTrashInProgress = false;
  }

  $: if (visible) { loadHistory(); }
</script>

<div class="history-page">
  <div class="page-header">
    <div class="header-row">
      <div>
        <h1>Operation History</h1>
        <p class="subtitle">Audit trail of all operations</p>
      </div>
      <div class="header-actions">
        {#if hasTrashAccess && restorableCount > 0}
          <button class="btn-header btn-header-restore" on:click={() => showRestoreAllDialog = true}>
            &#8634; Restore All ({restorableCount})
          </button>
        {/if}
        <button class="btn-header btn-header-empty" on:click={() => showEmptyTrashDialog = true}>
          &#128465; Empty Trash
        </button>
      </div>
    </div>
  </div>

  {#if errorMsg}
    <div class="toast toast-error">
      <span>&#10007;</span>
      <span class="toast-msg">{errorMsg}</span>
      <button class="toast-close" on:click={() => errorMsg = null}>&#10005;</button>
    </div>
  {/if}

  {#if !hasTrashAccess && !loading && dedupedHistory.some(e => getState(e) === 'in-trash')}
    <div class="toast toast-warn">
      <span>&#9888;</span>
      <span class="toast-msg">Full Disk Access required to restore items. Grant in <strong>System Settings &gt; Privacy &gt; Full Disk Access</strong>.</span>
    </div>
  {/if}

  {#if loading}
    <div class="loading">Loading history...</div>
  {:else if dedupedHistory.length === 0}
    <div class="empty-state">
      <div class="empty-icon">&#9201;</div>
      <p>No operations recorded yet</p>
      <p class="empty-sub">Actions you take will appear here</p>
    </div>
  {:else}
    <div class="history-list">
      {#each dedupedHistory as entry}
        {@const state = getState(entry)}
        {@const retention = $settings.trashRetention}
        {@const countdown = state === 'in-trash' ? daysRemaining(entry.timestamp, retention) : null}
        <div class="history-item" class:dimmed={state === 'gone'}>
          <div class="item-icon" class:icon-trash={state === 'in-trash'} class:icon-restored={state === 'restored'} class:icon-gone={state === 'gone'}>
            {#if state === 'restored'}
              {'\u21A9'}
            {:else if state === 'gone'}
              {'\u2717'}
            {:else}
              {'\u2713'}
            {/if}
          </div>

          <div class="item-name-col">
            <span class="item-name" title={entry.path}>{getFileName(entry.path)}</span>
            <span class="item-path" title={entry.path}>{entry.path}</span>
          </div>

          <div class="item-status-col">
            {#if state === 'restored'}
              <span class="badge badge-restored">Restored</span>
            {:else if state === 'in-trash'}
              <span class="badge badge-deleted">Deleted</span>
            {:else}
              <span class="badge badge-gone">Permanently Deleted</span>
            {/if}
          </div>

          <div class="item-size-col">{formatBytes(entry.size)}</div>

          <div class="item-time-col">
            <span class="item-time">{timeAgo(entry.timestamp)}</span>
            {#if countdown}
              <span class="item-countdown" class:urgent={countdown === 'overdue' || countdown === '1 day left'}>
                {countdown === 'overdue' ? 'Overdue' : countdown}
              </span>
            {/if}
          </div>

          <div class="item-action-col">
            {#if state === 'restored'}
              <button
                class="btn-action btn-action-delete"
                on:click={() => deleteItem(entry.path)}
                disabled={actionInProgress === entry.path}
                title="Move to Trash"
              >
                {#if actionInProgress === entry.path}
                  &#8987;
                {:else}
                  &#128465;
                {/if}
              </button>
            {:else if state === 'in-trash' && hasTrashAccess}
              <button
                class="btn-action btn-action-restore"
                on:click={() => restoreItem(entry.path)}
                disabled={actionInProgress === entry.path}
                title="Restore to original location"
              >
                {#if actionInProgress === entry.path}
                  &#8987;
                {:else}
                  &#8634;
                {/if}
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if showRestoreAllDialog}
  <div class="dialog-overlay" on:click={() => showRestoreAllDialog = false}>
    <div class="dialog" on:click|stopPropagation>
      <div class="dialog-icon">&#8634;</div>
      <h2>Restore All Items</h2>
      <p>This will restore <strong>{restorableCount}</strong> {restorableCount === 1 ? 'item' : 'items'} still in Trash to {restorableCount === 1 ? 'its' : 'their'} original {restorableCount === 1 ? 'location' : 'locations'}.</p>
      <div class="dialog-actions">
        <button class="btn-cancel" on:click={() => showRestoreAllDialog = false} disabled={restoreAllInProgress}>Cancel</button>
        <button class="btn-confirm" on:click={confirmRestoreAll} disabled={restoreAllInProgress}>
          {restoreAllInProgress ? 'Restoring...' : 'Restore All'}
        </button>
      </div>
    </div>
  </div>
{/if}

{#if showEmptyTrashDialog}
  <div class="dialog-overlay" on:click={() => showEmptyTrashDialog = false}>
    <div class="dialog" on:click|stopPropagation>
      <div class="dialog-icon danger">&#128465;</div>
      <h2>Empty Trash</h2>
      <p>This will <strong>permanently delete</strong> all items in the Trash immediately. This action cannot be undone.</p>
      <p class="dialog-warn">All trashed files will be removed and cannot be recovered.</p>
      <div class="dialog-actions">
        <button class="btn-cancel" on:click={() => showEmptyTrashDialog = false} disabled={emptyTrashInProgress}>Cancel</button>
        <button class="btn-danger" on:click={confirmEmptyTrash} disabled={emptyTrashInProgress}>
          {emptyTrashInProgress ? 'Emptying...' : 'Empty Trash'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .history-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 24px; }
  .header-row { display: flex; align-items: flex-start; justify-content: space-between; }
  .header-actions { display: flex; gap: 8px; flex-shrink: 0; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }
  .loading { color: var(--text-secondary); padding: 40px 0; text-align: center; }

  .toast {
    display: flex; align-items: center; gap: 10px;
    padding: 10px 14px; border-radius: var(--radius-sm);
    margin-bottom: 12px; font-size: 13px; color: var(--text-primary);
  }
  .toast-error { background: var(--red-dim); border: 1px solid var(--red); }
  .toast-warn { background: var(--yellow-dim); border: 1px solid var(--yellow); }
  .toast-msg { flex: 1; }
  .toast-close {
    background: none; color: var(--text-muted); font-size: 14px;
    padding: 2px 6px; border-radius: 4px; flex-shrink: 0;
  }
  .toast-close:hover { color: var(--text-primary); background: var(--bg-hover); }

  .empty-state { text-align: center; padding: 60px 0; }
  .empty-icon { font-size: 48px; color: var(--text-muted); margin-bottom: 16px; }
  .empty-state p { color: var(--text-secondary); }
  .empty-sub { font-size: 12px; color: var(--text-muted); margin-top: 8px; }

  .history-list {
    display: flex; flex-direction: column; gap: 4px;
    max-height: calc(100vh - 220px); overflow-y: auto;
  }

  .history-item {
    display: flex; align-items: center; gap: 16px;
    padding: 12px 16px;
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    transition: background var(--transition);
  }
  .history-item:hover { background: var(--bg-hover); }
  .history-item.dimmed { opacity: 0.4; }

  /* Col 1: Icon */
  .item-icon {
    width: 34px; height: 34px; border-radius: 50%;
    display: flex; align-items: center; justify-content: center;
    font-size: 14px; flex-shrink: 0;
  }
  .icon-trash { background: var(--yellow-dim); color: var(--yellow); }
  .icon-restored { background: var(--green-dim); color: var(--green); }
  .icon-gone { background: var(--red-dim); color: var(--red); }

  /* Col 2: Name + path */
  .item-name-col {
    flex: 1; min-width: 0;
    display: flex; flex-direction: column; gap: 2px;
  }
  .item-name {
    font-weight: 600; font-size: 13px;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .item-path {
    font-size: 11px; color: var(--text-muted);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }

  /* Col 3: Status badge */
  .item-status-col {
    flex-shrink: 0; width: 130px; text-align: center;
  }
  .badge {
    font-size: 11px; font-weight: 600; padding: 3px 10px;
    border-radius: 10px; white-space: nowrap;
    display: inline-block;
  }
  .badge-deleted { background: var(--yellow-dim); color: var(--yellow); }
  .badge-restored { background: var(--green-dim); color: var(--green); }
  .badge-gone { background: var(--red-dim); color: var(--red); }

  /* Col 4: Size */
  .item-size-col {
    flex-shrink: 0; width: 70px; text-align: right;
    font-size: 13px; font-weight: 500; color: var(--text-secondary);
  }

  /* Col 5: Time + countdown */
  .item-time-col {
    flex-shrink: 0; width: 100px; text-align: right;
    display: flex; flex-direction: column; align-items: flex-end; gap: 2px;
  }
  .item-time { font-size: 12px; color: var(--text-muted); }
  .item-countdown {
    font-size: 10px; color: var(--text-muted);
    padding: 1px 6px; border-radius: 3px;
    background: var(--bg-tertiary); white-space: nowrap;
  }
  .item-countdown.urgent { background: var(--red-dim); color: var(--red); }

  /* Col 6: Action */
  .item-action-col {
    flex-shrink: 0; width: 36px;
    display: flex; justify-content: center;
  }
  .btn-action {
    width: 32px; height: 32px; font-size: 16px; border-radius: 50%;
    border: 1px solid transparent; cursor: pointer;
    transition: all var(--transition);
    display: flex; align-items: center; justify-content: center; padding: 0;
  }
  .btn-action:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn-action-restore { background: var(--accent-dim); color: var(--accent); }
  .btn-action-restore:hover:not(:disabled) {
    background: var(--accent); color: var(--bg-primary); border-color: var(--accent);
  }
  .btn-action-delete { background: var(--red-dim); color: var(--red); }
  .btn-action-delete:hover:not(:disabled) {
    background: var(--red); color: var(--bg-primary); border-color: var(--red);
  }

  /* Header buttons */
  .btn-header {
    padding: 8px 16px; font-size: 13px; font-weight: 600;
    border-radius: var(--radius-sm); cursor: pointer;
    transition: all var(--transition); white-space: nowrap;
  }
  .btn-header-restore {
    background: var(--accent-dim); color: var(--accent); border: 1px solid var(--accent);
  }
  .btn-header-restore:hover { background: var(--accent); color: var(--bg-primary); }
  .btn-header-empty {
    background: var(--red-dim); color: var(--red); border: 1px solid var(--red);
  }
  .btn-header-empty:hover { background: var(--red); color: var(--bg-primary); }

  /* Dialog */
  .dialog-overlay {
    position: fixed; inset: 0; background: rgba(0,0,0,0.6);
    display: flex; align-items: center; justify-content: center;
    z-index: 1000; backdrop-filter: blur(4px);
  }
  .dialog {
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: 12px; padding: 32px; max-width: 420px;
    width: 90%; text-align: center;
  }
  .dialog-icon { font-size: 36px; color: var(--accent); margin-bottom: 12px; }
  .dialog-icon.danger { color: var(--red); }
  .dialog h2 { font-size: 18px; font-weight: 700; margin-bottom: 12px; }
  .dialog p { font-size: 13px; color: var(--text-secondary); line-height: 1.5; margin-bottom: 8px; }
  .dialog-warn { font-size: 12px; color: var(--text-muted); }
  .dialog-actions { display: flex; gap: 10px; margin-top: 20px; justify-content: center; }
  .btn-cancel {
    padding: 8px 20px; font-size: 13px; font-weight: 500;
    border-radius: var(--radius-sm); background: var(--bg-hover);
    color: var(--text-secondary); border: 1px solid var(--border); cursor: pointer;
  }
  .btn-cancel:hover:not(:disabled) { background: var(--bg-secondary); }
  .btn-confirm {
    padding: 8px 20px; font-size: 13px; font-weight: 600;
    border-radius: var(--radius-sm); background: var(--accent);
    color: var(--bg-primary); border: none; cursor: pointer;
  }
  .btn-confirm:hover:not(:disabled) { opacity: 0.9; }
  .btn-danger {
    padding: 8px 20px; font-size: 13px; font-weight: 600;
    border-radius: var(--radius-sm); background: var(--red);
    color: var(--bg-primary); border: none; cursor: pointer;
  }
  .btn-danger:hover:not(:disabled) { opacity: 0.9; }
  .btn-confirm:disabled, .btn-cancel:disabled, .btn-danger:disabled {
    opacity: 0.5; cursor: not-allowed;
  }
</style>
