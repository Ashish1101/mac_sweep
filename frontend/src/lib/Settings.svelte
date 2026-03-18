<script>
  let activeTab = 'general';
</script>

<div class="settings-page">
  <div class="page-header">
    <h1>Settings</h1>
    <p class="subtitle">Configure Mole GUI preferences</p>
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
          <h3>Appearance</h3>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Theme</div>
              <div class="setting-desc">Choose application theme</div>
            </div>
            <select class="setting-select" disabled>
              <option>Dark</option>
              <option>Light</option>
              <option>System</option>
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
            <select class="setting-select" disabled>
              <option>2 seconds</option>
              <option>5 seconds</option>
              <option>10 seconds</option>
            </select>
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
            <div class="toggle on">
              <div class="toggle-knob"></div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Require confirmation</div>
              <div class="setting-desc">Show confirmation dialog before destructive operations</div>
            </div>
            <div class="toggle on">
              <div class="toggle-knob"></div>
            </div>
          </div>
          <div class="setting-row">
            <div class="setting-info">
              <div class="setting-name">Trash retention</div>
              <div class="setting-desc">Days before auto-emptying Trash items</div>
            </div>
            <select class="setting-select" disabled>
              <option>30 days</option>
              <option>7 days</option>
              <option>Never</option>
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
            <div class="toggle on">
              <div class="toggle-knob"></div>
            </div>
          </div>
        </div>

      {:else if activeTab === 'about'}
        <div class="about-section">
          <div class="about-logo">&#9900;</div>
          <h2>Mole GUI</h2>
          <p class="about-version">Version 0.1.0</p>
          <p class="about-desc">
            A safe, visual interface for the Mole system maintenance tool.
            Wraps the Mole CLI with safety features including trash-based deletion,
            dry-run previews, and risk classification.
          </p>
          <div class="about-links">
            <div class="about-link">
              <span class="link-label">Mole CLI</span>
              <span class="link-value">github.com/tw93/Mole</span>
            </div>
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
  }

  .toggle {
    width: 44px;
    height: 24px;
    border-radius: 12px;
    background: var(--bg-tertiary);
    position: relative;
    cursor: pointer;
    transition: background var(--transition);
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
