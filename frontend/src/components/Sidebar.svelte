<script>
  import { currentPage } from '../stores/navigation.js';

  const navItems = [
    { id: 'dashboard', label: 'Dashboard', icon: '&#9673;' },
    { id: 'clean', label: 'Clean', icon: '&#10026;' },
    { id: 'analyze', label: 'Analyze', icon: '&#9678;' },
    { id: 'monitor', label: 'Monitor', icon: '&#9632;' },
    { id: 'history', label: 'History', icon: '&#9201;' },
    { id: 'settings', label: 'Settings', icon: '&#9881;' },
  ];

  function navigate(id) {
    currentPage.set(id);
  }
</script>

<aside class="sidebar">
  <div class="titlebar-drag"></div>
  <div class="logo">
    <span class="logo-icon">&#9900;</span>
    <span class="logo-text">Mole</span>
  </div>
  <nav>
    {#each navItems as item}
      <button
        class="nav-item"
        class:active={$currentPage === item.id}
        on:click={() => navigate(item.id)}
      >
        <span class="nav-icon">{@html item.icon}</span>
        <span class="nav-label">{item.label}</span>
      </button>
    {/each}
  </nav>
  <div class="sidebar-footer">
    <div class="version">Mole GUI v0.1.0</div>
  </div>
</aside>

<style>
  .sidebar {
    width: var(--sidebar-width);
    height: 100vh;
    background: var(--bg-secondary);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    flex-shrink: 0;
    position: relative;
  }

  .titlebar-drag {
    height: var(--titlebar-height);
    -webkit-app-region: drag;
    flex-shrink: 0;
  }

  .logo {
    padding: 0 20px 20px;
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .logo-icon {
    font-size: 24px;
    color: var(--accent);
  }

  .logo-text {
    font-size: 20px;
    font-weight: 700;
    letter-spacing: -0.5px;
    color: var(--text-primary);
  }

  nav {
    flex: 1;
    padding: 0 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 14px;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--text-secondary);
    transition: all var(--transition);
    width: 100%;
    text-align: left;
  }

  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .nav-item.active {
    background: var(--accent-dim);
    color: var(--accent);
  }

  .nav-icon {
    font-size: 18px;
    width: 24px;
    text-align: center;
    flex-shrink: 0;
  }

  .nav-label {
    font-size: 14px;
    font-weight: 500;
  }

  .sidebar-footer {
    padding: 16px 20px;
    border-top: 1px solid var(--border);
  }

  .version {
    font-size: 11px;
    color: var(--text-muted);
  }
</style>
