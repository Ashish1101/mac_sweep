<script>
  import { onMount } from 'svelte';
  import { CheckFullDiskAccess, OpenFullDiskAccessSettings } from '../../wailsjs/go/main/App.js';
  import { fdaStatus } from '../stores/permissions.js';
  import { settings } from '../stores/settings.js';
  import { previewSound } from '../stores/sound.js';

  let activeTab = 'general';
  let fdaChecking = false;
  let fdaJustChecked = false;

  async function checkFDA() {
    fdaChecking = true;
    try {
      const result = await CheckFullDiskAccess();
      fdaStatus.set(result);
    } catch (e) {
      console.error('Failed to check FDA:', e);
    }
    fdaChecking = false;
    fdaJustChecked = true;
    setTimeout(() => { fdaJustChecked = false; }, 1500);
  }

  async function openFDASettings() {
    try {
      await OpenFullDiskAccessSettings();
    } catch (e) {
      console.error('Failed to open settings:', e);
    }
  }

  function toggleSetting(key) {
    settings.setSetting(key, !$settings[key]);
  }

  onMount(() => { checkFDA(); });
</script>

<div class="settings-page">
  <div class="page-header">
    <h1>Settings</h1>
    <p class="subtitle">Configure MacSweep preferences</p>
  </div>

  <div class="settings-layout">
    <div class="settings-tabs">
      <button class="tab" class:active={activeTab === 'general'} on:click={() => activeTab = 'general'}>General</button>
      <button class="tab" class:active={activeTab === 'safety'} on:click={() => activeTab = 'safety'}>Safety</button>
      <button class="tab" class:active={activeTab === 'about'} on:click={() => activeTab = 'about'}>About</button>
    </div>

    <div class="settings-content">
      {#if activeTab === 'general'}
        <div class="setting-group">
          <h3>Permissions</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Full Disk Access</div>
              <div class="setting-desc">Required for scanning privacy-protected directories</div>
            </div>
            {#if $fdaStatus && $fdaStatus.hasFullDiskAccess}
              <span class="badge granted">Granted</span>
            {:else}
              <div class="fda-row-actions">
                <span class="badge not-granted" class:flash={fdaJustChecked}>
                  {fdaChecking ? 'Checking...' : 'Not Granted'}
                </span>
                <button class="setting-btn" on:click={openFDASettings}>Open Settings</button>
                <button class="setting-btn subtle" on:click={checkFDA} disabled={fdaChecking}>
                  {fdaChecking ? 'Checking...' : 'Refresh'}
                </button>
              </div>
            {/if}
          </div>
        </div>

        <div class="setting-group">
          <h3>Appearance</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Theme</div>
              <div class="setting-desc">Choose application theme</div>
            </div>
            <select class="setting-select" value={$settings.theme} on:change={(e) => settings.setSetting('theme', e.target.value)}>
              <option value="dark">Dark</option>
              <option value="light">Light</option>
              <option value="system">System</option>
            </select>
          </div>
        </div>

        <div class="setting-group">
          <h3>Monitoring</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Refresh Interval</div>
              <div class="setting-desc">How often to update system metrics</div>
            </div>
            <select class="setting-select" value={$settings.refreshInterval} on:change={(e) => settings.setSetting('refreshInterval', Number(e.target.value))}>
              <option value={2}>2 seconds</option>
              <option value={5}>5 seconds</option>
              <option value={10}>10 seconds</option>
            </select>
          </div>
        </div>

        <div class="setting-group">
          <h3>Sounds</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Delete Sound</div>
              <div class="setting-desc">Sound to play when moving items to Trash</div>
            </div>
            <div class="sound-picker">
              <select class="setting-select" value={$settings.deleteSound} on:change={(e) => settings.setSetting('deleteSound', e.target.value)}>
                <option value="default">macOS Trash (Default)</option>
                <optgroup label="Meme Sounds">
                  <option value="faaah">FAAAH</option>
                  <option value="he-knew">He Knew He F'd Up</option>
                  <option value="really-nigga">Really Nigga</option>
                  <option value="oh-my-god">Ohhh My God</option>
                  <option value="wait-a-minute">Wait A Minute (Kazoo)</option>
                </optgroup>
                <optgroup label="System Sounds">
                  <option value="funk">Funk</option>
                  <option value="glass">Glass</option>
                  <option value="pop">Pop</option>
                  <option value="basso">Basso</option>
                  <option value="hero">Hero</option>
                  <option value="sosumi">Sosumi</option>
                </optgroup>
                <option value="none">None (Silent)</option>
              </select>
              <button class="btn-preview" on:click={() => previewSound($settings.deleteSound)} title="Preview sound">&#9654;</button>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Restore Sound</div>
              <div class="setting-desc">Sound to play when restoring items from Trash</div>
            </div>
            <div class="sound-picker">
              <select class="setting-select" value={$settings.restoreSound} on:change={(e) => settings.setSetting('restoreSound', e.target.value)}>
                <option value="default">macOS Trash (Default)</option>
                <optgroup label="Meme Sounds">
                  <option value="wait-a-minute">Wait A Minute (Kazoo)</option>
                  <option value="faaah">FAAAH</option>
                  <option value="he-knew">He Knew He F'd Up</option>
                  <option value="really-nigga">Really Nigga</option>
                  <option value="oh-my-god">Ohhh My God</option>
                </optgroup>
                <optgroup label="System Sounds">
                  <option value="hero">Hero</option>
                  <option value="glass">Glass</option>
                  <option value="pop">Pop</option>
                  <option value="funk">Funk</option>
                  <option value="sosumi">Sosumi</option>
                </optgroup>
                <option value="none">None (Silent)</option>
              </select>
              <button class="btn-preview" on:click={() => previewSound($settings.restoreSound)} title="Preview sound">&#9654;</button>
            </div>
          </div>
        </div>

      {:else if activeTab === 'safety'}
        <div class="setting-group">
          <h3>Deletion Safety</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Always use Trash</div>
              <div class="setting-desc">Move files to Trash instead of permanent deletion</div>
            </div>
            <div class="toggle" class:on={$settings.alwaysUseTrash} on:click={() => toggleSetting('alwaysUseTrash')}>
              <div class="toggle-knob"></div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Require confirmation</div>
              <div class="setting-desc">Show confirmation dialog before destructive operations</div>
            </div>
            <div class="toggle" class:on={$settings.requireConfirmation} on:click={() => toggleSetting('requireConfirmation')}>
              <div class="toggle-knob"></div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Trash retention</div>
              <div class="setting-desc">Days before auto-emptying Trash items</div>
            </div>
            <select class="setting-select" value={$settings.trashRetention} on:change={(e) => settings.setSetting('trashRetention', Number(e.target.value))}>
              <option value={30}>30 days</option>
              <option value={7}>7 days</option>
              <option value={0}>Never</option>
            </select>
          </div>
        </div>

        <div class="setting-group">
          <h3>Audit</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Operation logging</div>
              <div class="setting-desc">Log all operations for audit trail</div>
            </div>
            <div class="toggle" class:on={$settings.operationLogging} on:click={() => toggleSetting('operationLogging')}>
              <div class="toggle-knob"></div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'about'}
        <div class="about-section">
          <div class="about-logo">&#9900;</div>
          <h2>MacSweep</h2>
          <p class="about-version">Version 0.1.0</p>
          <p class="about-desc">
            A safe, visual disk cleaner for macOS.
            Features trash-based deletion, dry-run previews, and risk classification.
          </p>
          <div class="about-links">
            <div class="about-link">
              <span class="link-label">License</span>
              <span class="link-value">MIT</span>
            </div>
            <div class="about-link">
              <span class="link-label">Built with</span>
              <span class="link-value">Wails + Svelte + Go</span>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .settings-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 24px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }

  .settings-layout {
    display: flex;
    gap: 24px;
  }

  .settings-tabs {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 140px;
  }

  .tab {
    padding: 10px 14px;
    background: transparent;
    color: var(--text-secondary);
    border-radius: var(--radius-sm);
    text-align: left;
    font-weight: 500;
    transition: all var(--transition);
  }

  .tab:hover { background: var(--bg-hover); color: var(--text-primary); }
  .tab.active { background: var(--accent-dim); color: var(--accent); }

  .settings-content {
    flex: 1;
  }

  .setting-group {
    margin-bottom: 28px;
  }

  .setting-group h3 {
    font-size: 13px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 12px;
  }

  .setting-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 16px;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    margin-bottom: 6px;
  }

  .setting-info { flex: 1; }
  .setting-name { font-weight: 500; font-size: 14px; }
  .setting-desc { font-size: 12px; color: var(--text-muted); margin-top: 2px; }

  .setting-select {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 6px 12px;
    font-size: 13px;
    cursor: pointer;
  }

  .sound-picker {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .btn-preview {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    background: var(--accent-dim);
    color: var(--accent);
    border: 1px solid transparent;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    transition: all var(--transition);
    flex-shrink: 0;
  }

  .btn-preview:hover {
    background: var(--accent);
    color: var(--bg-primary);
    border-color: var(--accent);
  }

  .toggle {
    width: 44px;
    height: 24px;
    border-radius: 12px;
    background: var(--bg-tertiary);
    position: relative;
    cursor: pointer;
    transition: background var(--transition);
    flex-shrink: 0;
  }

  .toggle.on {
    background: var(--accent);
  }

  .toggle-knob {
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: white;
    position: absolute;
    top: 2px;
    left: 2px;
    transition: left var(--transition);
  }

  .toggle.on .toggle-knob {
    left: 22px;
  }

  .badge {
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
  }

  .badge.granted {
    background: var(--green-dim);
    color: var(--green);
  }

  .badge.not-granted {
    background: var(--yellow-dim);
    color: var(--yellow);
    transition: all 0.3s ease;
  }

  .badge.not-granted.flash {
    animation: badge-flash 0.6s ease;
  }

  @keyframes badge-flash {
    0% { transform: scale(1); }
    30% { transform: scale(1.1); background: var(--yellow); color: var(--bg-primary); }
    100% { transform: scale(1); }
  }

  .fda-row-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .setting-btn {
    padding: 5px 12px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    background: var(--accent);
    color: white;
    border: none;
    cursor: pointer;
    transition: opacity var(--transition);
    white-space: nowrap;
  }

  .setting-btn:hover { opacity: 0.9; }

  .setting-btn.subtle {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border: 1px solid var(--border);
  }

  .setting-btn.subtle:hover { background: var(--bg-hover); }

  .about-section {
    text-align: center;
    padding: 40px 0;
  }

  .about-logo {
    font-size: 48px;
    color: var(--accent);
    margin-bottom: 12px;
  }

  .about-section h2 {
    font-size: 24px;
    margin-bottom: 4px;
  }

  .about-version {
    color: var(--text-muted);
    font-size: 13px;
    margin-bottom: 16px;
  }

  .about-desc {
    color: var(--text-secondary);
    max-width: 400px;
    margin: 0 auto 24px;
    line-height: 1.6;
  }

  .about-links {
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 300px;
    margin: 0 auto;
  }

  .about-link {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    border-bottom: 1px solid var(--border);
  }

  .link-label {
    color: var(--text-muted);
    font-size: 13px;
  }

  .link-value {
    color: var(--text-secondary);
    font-size: 13px;
  }
</style>
