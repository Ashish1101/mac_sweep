<script>
  import { onMount, onDestroy } from 'svelte';
  import Sidebar from './components/Sidebar.svelte';
  import Dashboard from './lib/Dashboard.svelte';
  import Clean from './lib/Clean.svelte';
  import Analyze from './lib/Analyze.svelte';
  import Monitor from './lib/Monitor.svelte';
  import History from './lib/History.svelte';
  import Settings from './lib/Settings.svelte';
  import CpuDetail from './lib/CpuDetail.svelte';
  import MemoryDetail from './lib/MemoryDetail.svelte';
  import DiskDetail from './lib/DiskDetail.svelte';
  import BatteryDetail from './lib/BatteryDetail.svelte';
  import NetworkDetail from './lib/NetworkDetail.svelte';
  import WiFiDetail from './lib/WiFiDetail.svelte';
  import { currentPage } from './stores/navigation.js';
  import { fdaStatus } from './stores/permissions.js';
  import { settings } from './stores/settings.js';
  import { CheckFullDiskAccess } from '../wailsjs/go/main/App.js';

  function applyTheme(theme) {
    const root = document.documentElement;
    if (theme === 'system') {
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      root.className = prefersDark ? '' : 'light';
    } else {
      root.className = theme === 'light' ? 'light' : '';
    }
  }

  $: applyTheme($settings.theme);

  function onVisibilityChange() {
    if (!document.hidden) {
      CheckFullDiskAccess().then(result => {
        fdaStatus.set(result);
      }).catch(() => {});
    }
  }

  onMount(() => {
    document.addEventListener('visibilitychange', onVisibilityChange);
    // Listen for system theme changes when using 'system' theme
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
      if ($settings.theme === 'system') applyTheme('system');
    });
  });

  onDestroy(() => {
    document.removeEventListener('visibilitychange', onVisibilityChange);
  });
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
      <div class="page-container" class:hidden={$currentPage !== 'cpu-detail'}>
        <CpuDetail visible={$currentPage === 'cpu-detail'} on:back={() => currentPage.set('dashboard')} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'memory-detail'}>
        <MemoryDetail visible={$currentPage === 'memory-detail'} on:back={() => currentPage.set('dashboard')} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'disk-detail'}>
        <DiskDetail visible={$currentPage === 'disk-detail'} on:back={() => currentPage.set('dashboard')} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'battery-detail'}>
        <BatteryDetail visible={$currentPage === 'battery-detail'} on:back={() => currentPage.set('dashboard')} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'network-detail'}>
        <NetworkDetail visible={$currentPage === 'network-detail'} on:back={() => currentPage.set('dashboard')} />
      </div>
      <div class="page-container" class:hidden={$currentPage !== 'wifi-detail'}>
        <WiFiDetail visible={$currentPage === 'wifi-detail'} on:back={() => currentPage.set('dashboard')} />
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
