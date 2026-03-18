<script>
  import Sidebar from './components/Sidebar.svelte';
  import Dashboard from './lib/Dashboard.svelte';
  import Clean from './lib/Clean.svelte';
  import Analyze from './lib/Analyze.svelte';
  import Monitor from './lib/Monitor.svelte';
  import History from './lib/History.svelte';
  import Settings from './lib/Settings.svelte';
  import { currentPage } from './stores/navigation.js';
</script>

<div class="app-layout">
  <Sidebar />
  <main class="main-content">
    <div class="titlebar-drag"></div>
    <div class="content-scroll">
      <div class="page-container" class:hidden={$currentPage !== 'dashboard'}>
        <Dashboard visible={$currentPage === 'dashboard'} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'clean'}>
        <Clean />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'analyze'}>
        <Analyze />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'monitor'}>
        <Monitor visible={$currentPage === 'monitor'} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'history'}>
        <History visible={$currentPage === 'history'} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'settings'}>
        <Settings />
      </div>
    </div>
  </main>
</div>

<style>
  .app-layout {
    display: flex;
    height: 100vh;
    background: var(--bg-primary);
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .titlebar-drag {
    height: var(--titlebar-height);
    -webkit-app-region: drag;
    flex-shrink: 0;
  }

  .content-scroll {
    flex: 1;
    overflow-y: auto;
    padding-top: 8px;
  }

  .page-container {
    height: 100%;
  }

  .page-container.hidden {
    display: none;
  }
</style>
