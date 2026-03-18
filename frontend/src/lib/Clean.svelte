<script>
  import { onMount, onDestroy } from 'svelte';
  import { CleanDryRun, ExecuteClean, StartDrill, CancelDrill, MoveToTrash, PlayTrashSound } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

  // --- State ---
  let result = null;
  let scanning = false;
  let cleaning = false;
  let showConfirm = false;
  let cleanResult = null;

  // Progress
  let scanProgress = null;

  // Drill-down navigation
  let breadcrumbs = [];
  let currentView = 'root';
  let currentCatIndex = -1;
  let currentItems = [];
  let drillLoading = false;
  let drillAborted = false;

  // Reactivity trigger — bump this whenever nav state changes
  let navVersion = 0;

  // Sort control for file list
  let sortBy = 'size';

  // Selection: map of path -> {size, name}
  let selectedPaths = {};

  // Single-item delete (inline trash icon)
  let singleDeletePath = null;
  let singleDeleteInfo = null;

  // Heatmap hover
  let hoveredSegment = null;

  let unsubProgress = null;

  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function riskColor(risk) {
    switch (risk) {
      case 'low': return 'green';
      case 'medium': return 'yellow';
      case 'high': return 'red';
      default: return 'blue';
    }
  }

  // --- Sunburst heatmap helpers ---
  const COLORS = [
    '#7c5cfc', '#60a5fa', '#4ade80', '#fbbf24', '#f87171',
    '#a78bfa', '#34d399', '#fb923c', '#f472b6', '#38bdf8',
    '#818cf8', '#2dd4bf', '#facc15', '#fb7185', '#c084fc'
  ];

  function getHeatmapData(_v) {
    if (!result) return [];
    if (currentView === 'root') {
      return result.categories.map((cat, i) => ({
        label: cat.name,
        size: cat.size,
        color: COLORS[i % COLORS.length],
        index: i,
        type: 'category',
        risk: cat.risk,
        children: cat.items.map(item => ({ label: item.name.split('/').pop() || item.name, size: item.size, path: item.path, isDir: item.isDir })),
      }));
    }
    if (currentView === 'category' && currentCatIndex >= 0) {
      const cat = result.categories[currentCatIndex];
      if (!cat) return [];
      return (cat.items || []).map((item, i) => ({
        label: item.name.split('/').pop() || item.name,
        size: item.size,
        color: COLORS[i % COLORS.length],
        index: i,
        type: 'item',
        path: item.path,
        isDir: item.isDir,
        children: (item.children || []).map(c => ({ label: c.name?.split('/').pop() || c.name, size: c.size, path: c.path, isDir: c.isDir })),
      }));
    }
    if (currentView === 'item') {
      return currentItems.map((item, i) => ({
        label: item.name.split('/').pop() || item.name,
        size: item.size,
        color: COLORS[i % COLORS.length],
        index: i,
        type: 'leaf',
        path: item.path,
        isDir: item.isDir,
        children: (item.children || []).map(c => ({ label: c.name?.split('/').pop() || c.name, size: c.size, path: c.path, isDir: c.isDir })),
      }));
    }
    return [];
  }

  function buildInnerArcs(data) {
    const total = data.reduce((s, d) => s + d.size, 0);
    if (total === 0) return [];
    let angle = 0;
    return data.map((d, i) => {
      const sweep = (d.size / total) * 360;
      const arc = { ...d, startAngle: angle, sweepAngle: sweep, index: i, ring: 'inner' };
      angle += sweep;
      return arc;
    });
  }

  function buildOuterArcs(innerArcs) {
    const outerArcs = [];
    for (const inner of innerArcs) {
      if (!inner.children || inner.children.length === 0) continue;
      const childTotal = inner.children.reduce((s, c) => s + (c.size || 0), 0);
      if (childTotal === 0) continue;
      let angle = inner.startAngle;
      for (const child of inner.children) {
        const sweep = (child.size / childTotal) * inner.sweepAngle;
        const relSize = child.size / childTotal;
        const opacity = 0.5 + relSize * 0.4;
        outerArcs.push({
          label: child.label || child.name,
          path: child.path,
          size: child.size,
          isDir: child.isDir,
          color: inner.color,
          opacity: Math.min(0.9, Math.max(0.5, opacity)),
          startAngle: angle,
          sweepAngle: sweep,
          parentIndex: inner.index,
          ring: 'outer',
        });
        angle += sweep;
      }
    }
    return outerArcs;
  }

  function describeArc(cx, cy, r, innerR, startAngle, sweepAngle) {
    if (sweepAngle >= 359.99) sweepAngle = 359.99;
    const startRad = (startAngle - 90) * Math.PI / 180;
    const endRad = (startAngle + sweepAngle - 90) * Math.PI / 180;
    const largeArc = sweepAngle > 180 ? 1 : 0;

    const x1 = cx + r * Math.cos(startRad);
    const y1 = cy + r * Math.sin(startRad);
    const x2 = cx + r * Math.cos(endRad);
    const y2 = cy + r * Math.sin(endRad);
    const x3 = cx + innerR * Math.cos(endRad);
    const y3 = cy + innerR * Math.sin(endRad);
    const x4 = cx + innerR * Math.cos(startRad);
    const y4 = cy + innerR * Math.sin(startRad);

    return `M ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2} L ${x3} ${y3} A ${innerR} ${innerR} 0 ${largeArc} 0 ${x4} ${y4} Z`;
  }

  // --- Bump navVersion to trigger reactive updates ---
  function bumpNav() {
    navVersion++;
  }

  // --- Scanning ---
  async function scan() {
    scanning = true;
    cleanResult = null;
    result = null;
    scanProgress = { phase: 'starting', category: '', currentFile: '', filesFound: 0, sizeFound: 0, percentage: 0 };
    breadcrumbs = [];
    currentView = 'root';
    currentCatIndex = -1;
    currentItems = [];
    selectedPaths = {};
    bumpNav();

    try {
      result = await CleanDryRun();
    } catch (e) {
      console.error('Scan failed:', e);
    }
    scanning = false;
    scanProgress = null;
    bumpNav();
  }

  // --- Navigation ---
  function goToRoot() {
    drillRequestId++;
    CancelDrill();
    drillLoading = false;
    currentView = 'root';
    currentCatIndex = -1;
    currentItems = [];
    breadcrumbs = [];
    bumpNav();
  }

  function drillIntoCategory(catIndex) {
    drillRequestId++;
    CancelDrill();
    drillLoading = false;
    const cat = result.categories[catIndex];
    if (!cat) { goToRoot(); return; }
    currentView = 'category';
    currentCatIndex = catIndex;
    currentItems = cat.items || [];
    breadcrumbs = [{ label: cat.name, type: 'category', catIndex }];
    bumpNav();
  }

  // Drill request counter — stale results are ignored
  let drillRequestId = 0;

  function drillIntoItem(item) {
    if (!item.isDir) return;

    const myId = ++drillRequestId;
    drillAborted = false;

    // Update breadcrumbs and view immediately — UI never blocks
    currentView = 'item';
    const shortName = item.name.split('/').pop() || item.name;
    breadcrumbs = [...breadcrumbs, { label: shortName, type: 'item', path: item.path }];

    // If we already have children from scan data, show them instantly
    if (item.children && item.children.length > 0) {
      currentItems = item.children;
      drillLoading = false;
      bumpNav();
      return;
    }

    // Fire-and-forget: Go runs the scan in a background goroutine
    // Result arrives via "drill:complete" event — handled by onDrillComplete
    currentItems = [];
    drillLoading = true;
    bumpNav();
    StartDrill(item.path, myId);
  }

  // Called when the Go goroutine finishes scanning
  function onDrillComplete(data) {
    // Ignore stale results from a cancelled/superseded drill
    if (!data || data.requestId !== drillRequestId) return;

    currentItems = data.items || [];
    drillLoading = false;
    bumpNav();
  }

  function cancelDrill() {
    drillRequestId++; // invalidate pending result
    CancelDrill();    // tell Go to stop the goroutine
    drillLoading = false;
    drillAborted = true;

    // Navigate back to previous level
    if (breadcrumbs.length > 1) {
      breadcrumbs = breadcrumbs.slice(0, -1);
      const lastCrumb = breadcrumbs[breadcrumbs.length - 1];
      if (lastCrumb.type === 'category') {
        currentView = 'category';
        currentCatIndex = lastCrumb.catIndex;
        currentItems = result.categories[lastCrumb.catIndex]?.items || [];
      } else {
        currentItems = [];
      }
    } else if (breadcrumbs.length === 1) {
      const lastCrumb = breadcrumbs[0];
      if (lastCrumb.type === 'category') {
        currentView = 'category';
        currentCatIndex = lastCrumb.catIndex;
        currentItems = result.categories[lastCrumb.catIndex]?.items || [];
      } else {
        goToRoot();
      }
    } else {
      goToRoot();
    }
    bumpNav();
  }

  // Called when user clicks a list row body (NOT the checkbox)
  // Dirs: drill down. Files: do nothing (use checkbox to select).
  function handleListClick(item) {
    if (item.isCategoryRow) {
      drillIntoCategory(item.catIndex);
    } else if (item.isDir) {
      drillIntoItem(item);
    }
    // Files: no action on row click — user must use checkbox explicitly
  }

  // Called when user clicks a heatmap arc or legend item
  function handleHeatmapClick(segment) {
    if (segment.type === 'category') {
      drillIntoCategory(segment.index);
    } else if (segment.isDir) {
      // Drill into directory — try index-based lookup first, then fallback to path
      let item;
      if (segment.type === 'item' && currentCatIndex >= 0) {
        item = result.categories[currentCatIndex].items[segment.index];
      } else if (segment.type === 'leaf') {
        item = currentItems[segment.index];
      }

      if (item) {
        drillIntoItem(item);
      } else if (segment.path) {
        // Outer ring arcs or untyped segments — drill using path directly
        drillIntoItem({ path: segment.path, isDir: true, name: segment.label || segment.path.split('/').pop(), size: segment.size, children: null });
      }
    } else if (segment.path) {
      // File: toggle selection via checkbox
      toggleSelect(segment.path, segment.size, segment.label);
    }
  }

  function navigateBreadcrumb(index) {
    if (index < 0) {
      goToRoot();
      return;
    }

    // Skip navigation if clicking the already-active (last) breadcrumb
    if (index === breadcrumbs.length - 1) {
      return;
    }

    // Cancel any pending drill
    drillRequestId++;
    CancelDrill();
    drillLoading = false;

    const crumb = breadcrumbs[index];
    breadcrumbs = breadcrumbs.slice(0, index + 1);

    if (crumb.type === 'category') {
      const cat = result.categories[crumb.catIndex];
      if (cat) {
        currentView = 'category';
        currentCatIndex = crumb.catIndex;
        currentItems = cat.items || [];
      } else {
        goToRoot();
        return;
      }
    } else if (crumb.type === 'item' && crumb.path) {
      // Preserve currentCatIndex from the first breadcrumb if it's a category
      if (breadcrumbs.length > 0 && breadcrumbs[0].type === 'category') {
        currentCatIndex = breadcrumbs[0].catIndex;
      }
      // Fire-and-forget drill — result arrives via event
      const myId = ++drillRequestId;
      currentView = 'item';
      currentItems = [];
      drillLoading = true;
      StartDrill(crumb.path, myId);
    }
    bumpNav();
  }

  // --- Selection ---
  function toggleSelect(path, size, name) {
    if (selectedPaths[path]) {
      delete selectedPaths[path];
    } else {
      selectedPaths[path] = { size, name };
    }
    selectedPaths = selectedPaths;
  }

  // Select all visible items (skipping category virtual rows)
  function selectAllVisible() {
    const items = getListItems(navVersion, sortBy);
    for (const item of items) {
      if (!item.isCategoryRow) {
        selectedPaths[item.path] = { size: item.size, name: item.name };
      }
    }
    selectedPaths = selectedPaths;
  }

  // Select ALL files across all categories recursively
  function selectAllDeep() {
    if (!result) return;
    for (const cat of result.categories) {
      for (const item of cat.items) {
        selectedPaths[item.path] = { size: item.size, name: item.name };
        // Also add children if available
        if (item.children) {
          for (const child of item.children) {
            selectedPaths[child.path] = { size: child.size, name: child.name };
          }
        }
      }
    }
    selectedPaths = selectedPaths;
  }

  function deselectAll() {
    selectedPaths = {};
  }

  function getSelectedSize() {
    return Object.values(selectedPaths).reduce((s, v) => s + v.size, 0);
  }

  function getSelectedCount() {
    return Object.keys(selectedPaths).length;
  }

  // --- List items for current view ---
  function sortItems(items, sort) {
    if (!items || items.length === 0) return items;
    const sorted = [...items];
    switch (sort) {
      case 'name':
        sorted.sort((a, b) => {
          const nameA = (a.name.split('/').pop() || a.name).toLowerCase();
          const nameB = (b.name.split('/').pop() || b.name).toLowerCase();
          return nameA.localeCompare(nameB);
        });
        break;
      case 'date':
        if (sorted.some(item => item.modTime)) {
          sorted.sort((a, b) => (b.modTime || 0) - (a.modTime || 0));
        } else {
          sorted.sort((a, b) => (b.size || 0) - (a.size || 0));
        }
        break;
      case 'size':
      default:
        sorted.sort((a, b) => (b.size || 0) - (a.size || 0));
        break;
    }
    return sorted;
  }

  function getListItems(_v, _sort) {
    if (currentView === 'root') {
      const mapped = (result?.categories || []).map((cat, i) => ({
        name: cat.name,
        size: cat.size,
        path: '__cat__' + i,
        isDir: true,
        risk: cat.risk,
        itemCount: cat.items.length,
        isCategoryRow: true,
        catIndex: i,
      }));
      return sortItems(mapped, sortBy);
    }
    if (currentView === 'category' && currentCatIndex >= 0) {
      const cat = result.categories[currentCatIndex];
      if (!cat) return [];
      return sortItems(cat.items || [], sortBy);
    }
    return sortItems(currentItems, sortBy);
  }

  // --- Cleanup ---
  function openDeleteConfirm(path, info) {
    // Reset cleaning flag so a previous in-flight delete doesn't block the new modal
    cleaning = false;
    if (path) {
      singleDeletePath = path;
      singleDeleteInfo = info;
    }
    showConfirm = true;
  }

  async function executeClean() {
    showConfirm = false;
    cleaning = true;

    try {
      let pathsToDelete;

      if (singleDeletePath) {
        // Single-item delete from inline trash icon
        pathsToDelete = [singleDeletePath];
      } else {
        // Multi-select delete
        pathsToDelete = Object.keys(selectedPaths).filter(p => !p.startsWith('__cat__'));
      }

      if (pathsToDelete.length === 0) {
        cleaning = false;
        return;
      }

      const execResult = await ExecuteClean(pathsToDelete);

      // Play macOS trash sound
      if (execResult && execResult.deletedCount > 0) {
        PlayTrashSound();
      }

      // Remove deleted paths from the result data so list refreshes
      if (execResult && execResult.deletedPaths) {
        const deletedSet = new Set(execResult.deletedPaths);
        purgeDeletedFromResult(deletedSet);
      }

      // Clear selection of deleted items
      if (execResult && execResult.deletedPaths) {
        for (const dp of execResult.deletedPaths) {
          delete selectedPaths[dp];
        }
      }
      selectedPaths = selectedPaths;

      // Show result banner briefly, then return to live view
      cleanResult = {
        freed: execResult?.freed || 0,
        count: execResult?.deletedCount || 0,
        failed: execResult?.failedPaths?.length || 0,
      };

    } catch (e) {
      console.error('Clean failed:', e);
    }

    singleDeletePath = null;
    singleDeleteInfo = null;
    cleaning = false;
  }

  // Remove deleted paths from the in-memory scan result
  function purgeDeletedFromResult(deletedSet) {
    if (!result) return;

    // Track the category name we're currently viewing (before indices shift)
    const currentCatName = (currentView === 'category' && currentCatIndex >= 0 && result.categories[currentCatIndex])
      ? result.categories[currentCatIndex].name
      : null;

    for (const cat of result.categories) {
      cat.items = cat.items.filter(item => {
        if (deletedSet.has(item.path)) {
          cat.size -= item.size;
          result.totalSize -= item.size;
          result.totalFiles--;
          return false;
        }
        // Also purge from children
        if (item.children) {
          item.children = item.children.filter(child => {
            if (deletedSet.has(child.path)) {
              item.size -= child.size;
              cat.size -= child.size;
              result.totalSize -= child.size;
              return false;
            }
            return true;
          });
        }
        return true;
      });
    }

    // Remove empty categories
    result.categories = result.categories.filter(cat => cat.items.length > 0);

    // Also purge from currentItems if in drill-down view
    currentItems = currentItems.filter(item => !deletedSet.has(item.path));

    // Fix stale navigation: if the category we were viewing got removed or emptied,
    // navigate back to root to avoid accessing undefined indices
    if (currentView === 'category') {
      if (currentCatName) {
        // Find the category by name (index may have shifted after filter)
        const newIndex = result.categories.findIndex(c => c.name === currentCatName);
        if (newIndex === -1) {
          // Category was removed entirely — go back to root
          currentView = 'root';
          currentCatIndex = -1;
          currentItems = [];
          breadcrumbs = [];
        } else {
          // Category still exists but index may have shifted
          currentCatIndex = newIndex;
          currentItems = result.categories[newIndex].items || [];
          if (breadcrumbs.length > 0) {
            breadcrumbs[0] = { ...breadcrumbs[0], catIndex: newIndex };
          }
        }
      } else {
        // Safety fallback
        currentView = 'root';
        currentCatIndex = -1;
        currentItems = [];
        breadcrumbs = [];
      }
    } else if (currentView === 'item' && currentItems.length === 0) {
      // All items in the drilled view got deleted — go back one level
      if (breadcrumbs.length > 1) {
        breadcrumbs = breadcrumbs.slice(0, -1);
        const lastCrumb = breadcrumbs[breadcrumbs.length - 1];
        if (lastCrumb.type === 'category' && lastCrumb.catIndex < result.categories.length) {
          currentView = 'category';
          currentCatIndex = lastCrumb.catIndex;
          currentItems = result.categories[lastCrumb.catIndex].items || [];
        } else {
          currentView = 'root';
          currentCatIndex = -1;
          currentItems = [];
          breadcrumbs = [];
        }
      } else {
        currentView = 'root';
        currentCatIndex = -1;
        currentItems = [];
        breadcrumbs = [];
      }
    }

    // If no categories left at all, keep result but with empty array
    if (result.categories.length === 0) {
      currentView = 'root';
      currentCatIndex = -1;
      currentItems = [];
      breadcrumbs = [];
    }

    result = result; // trigger reactivity
    bumpNav();
  }

  let unsubDrill = null;

  onMount(() => {
    unsubProgress = EventsOn('clean:progress', (progress) => {
      scanProgress = progress;
    });
    unsubDrill = EventsOn('drill:complete', onDrillComplete);
  });

  onDestroy(() => {
    if (unsubProgress) unsubProgress();
    if (unsubDrill) unsubDrill();
    CancelDrill();
  });

  // FIXED: include navVersion as dependency so these update on navigation
  $: heatmapData = result ? getHeatmapData(navVersion) : [];
  $: innerArcs = buildInnerArcs(heatmapData);
  $: outerArcs = buildOuterArcs(innerArcs);
  $: listItems = result ? getListItems(navVersion, sortBy) : [];
  $: totalCurrentSize = heatmapData.reduce((s, d) => s + d.size, 0);
  // Reactive selected count/size for selection bar visibility
  $: selectedCount = Object.keys(selectedPaths).length;
  $: selectedSize = Object.values(selectedPaths).reduce((s, v) => s + v.size, 0);
</script>

<div class="clean-page">
  <div class="page-header">
    <h1>Clean</h1>
    <p class="subtitle">Scan and remove junk files safely</p>
  </div>

  <!-- Scan progress overlay -->
  {#if scanning}
    <div class="scan-progress-card">
      <div class="progress-top">
        <div class="progress-spinner"></div>
        <div class="progress-info">
          <div class="progress-title">Scanning your system...</div>
          {#if scanProgress}
            <div class="progress-category">
              {scanProgress.category || 'Initializing'}
            </div>
            <div class="progress-file">{scanProgress.currentFile || ''}</div>
            <div class="progress-stats">
              <span>{scanProgress.filesFound.toLocaleString()} files found</span>
              <span class="dot">&middot;</span>
              <span>{formatBytes(scanProgress.sizeFound)} reclaimable</span>
            </div>
          {/if}
        </div>
      </div>
      {#if scanProgress}
        <div class="progress-bar-track">
          <div class="progress-bar-fill" style="width: {Math.max(scanProgress.percentage, 5)}%"></div>
        </div>
      {/if}
    </div>

  <!-- Empty state -->
  {:else if !result}
    <div class="empty-state">
      <div class="empty-icon">&#10026;</div>
      <p>Scan your system to find reclaimable space</p>
      <button class="btn-primary" on:click={scan} disabled={scanning}>Start Scan</button>
    </div>

  <!-- Scan results: split view -->
  {:else}
    <!-- Cleanup result banner (shown inline, list stays visible) -->
    {#if cleanResult}
      <div class="result-banner">
        <span class="result-icon">&#10004;</span>
        <div>
          <div class="result-title">Cleanup Complete</div>
          <div class="result-detail">
            Freed {formatBytes(cleanResult.freed)} across {cleanResult.count} items (moved to Trash)
            {#if cleanResult.failed > 0}
              &middot; {cleanResult.failed} failed
            {/if}
          </div>
        </div>
        <button class="btn-secondary-sm" on:click={() => cleanResult = null}>Dismiss</button>
      </div>
    {/if}

    <!-- Selection bar -->
    {#if selectedCount > 0}
      <div class="selection-bar">
        <span class="sel-count">{selectedCount} selected</span>
        <span class="sel-size">{formatBytes(selectedSize)}</span>
        <div class="sel-actions">
          <button class="btn-sm-ghost" on:click={deselectAll}>Deselect All</button>
          <button class="btn-sm-danger" on:click={() => openDeleteConfirm(null, null)}>Delete All Selected</button>
        </div>
      </div>
    {/if}

    <!-- Breadcrumbs + actions -->
    <div class="nav-row">
      <div class="breadcrumbs">
        <button class="crumb" class:active={currentView === 'root'} on:click={goToRoot}>All Categories</button>
        {#each breadcrumbs as crumb, i}
          <span class="crumb-sep">&#9654;</span>
          <button class="crumb" class:active={i === breadcrumbs.length - 1} on:click={() => navigateBreadcrumb(i)}>
            {crumb.label}
          </button>
        {/each}
      </div>
      <div class="nav-actions">
        {#if currentView === 'root'}
          <button class="btn-sm-accent" on:click={selectAllDeep}>Select All Files</button>
        {:else}
          <button class="btn-sm-ghost" on:click={selectAllVisible}>Select All</button>
        {/if}
      </div>
    </div>

    <div class="split-view">
      <!-- LEFT PANEL: file list -->
      <div class="left-panel">
        <div class="panel-header">
          <div class="panel-header-left">
            {#if currentView !== 'root'}
              <button class="btn-back" on:click={() => {
                if (breadcrumbs.length > 1) {
                  const parentIndex = breadcrumbs.length - 2;
                  navigateBreadcrumb(parentIndex);
                } else {
                  goToRoot();
                }
              }}>&#8592; Back</button>
            {/if}
            <span class="panel-title">
              {#if currentView === 'root'}
                {result.categories.length} Categories &middot; {formatBytes(result.totalSize)}
              {:else if currentView === 'category' && result.categories[currentCatIndex]}
                {listItems.length} items in {result.categories[currentCatIndex].name}
              {:else}
                {listItems.length} items
              {/if}
            </span>
          </div>
          <select class="sort-select" bind:value={sortBy}>
            <option value="size">Size</option>
            <option value="name">Name</option>
            <option value="date">Date</option>
          </select>
        </div>

        <div class="file-list">
          {#if drillLoading}
            <div class="list-loading">
              <div class="mini-spinner"></div>
              <span>Loading contents...</span>
              <button class="btn-cancel" on:click={cancelDrill}>Cancel</button>
            </div>
          {:else}
            {#each listItems as item, i}
              {@const maxSize = listItems.length > 0 ? listItems[0].size : 1}
              <div
                class="file-row"
                class:selected={selectedPaths[item.path]}
                class:is-category={item.isCategoryRow}
              >
                <!-- Checkbox -->
                {#if !item.isCategoryRow}
                  <button class="file-check" on:click|stopPropagation={() => toggleSelect(item.path, item.size, item.name)}>
                    {#if selectedPaths[item.path]}
                      <span class="check-on">&#10003;</span>
                    {:else}
                      <span class="check-off">&#9675;</span>
                    {/if}
                  </button>
                {:else}
                  <div class="file-check-spacer"></div>
                {/if}

                <!-- Clickable row -->
                <button class="file-info" on:click={() => handleListClick(item)}>
                  <div class="file-left">
                    <span class="file-icon">
                      {#if item.isCategoryRow}
                        &#128451;
                      {:else if item.isDir}
                        &#128193;
                      {:else}
                        &#128196;
                      {/if}
                    </span>
                    <div class="file-details">
                      <div class="file-name">
                        {item.isCategoryRow ? item.name : (item.name.split('/').pop() || item.name)}
                      </div>
                      <div class="file-bar-track">
                        <div class="file-bar-fill" style="width: {Math.max(2, (item.size / maxSize) * 100)}%; background: {COLORS[i % COLORS.length]}"></div>
                      </div>
                    </div>
                  </div>

                  <div class="file-right">
                    {#if item.risk}
                      <span class="risk-badge {riskColor(item.risk)}">{item.risk.toUpperCase()}</span>
                    {/if}
                    {#if item.itemCount}
                      <span class="item-count">{item.itemCount} items</span>
                    {/if}
                    <span class="file-size">{formatBytes(item.size)}</span>
                    {#if item.isDir}
                      <span class="drill-arrow">&#9654;</span>
                    {/if}
                  </div>
                </button>

                <!-- Inline delete button for non-category items -->
                {#if !item.isCategoryRow}
                  <button
                    class="inline-delete"
                    title="Move to Trash"
                    on:click|stopPropagation={() => {
                      openDeleteConfirm(item.path, { size: item.size, name: item.name });
                    }}
                  >&#128465;</button>
                {/if}
              </div>
            {/each}

            {#if listItems.length === 0}
              <div class="list-empty">No items found</div>
            {/if}
          {/if}
        </div>

        <!-- Footer -->
        <div class="panel-footer">
          <button class="btn-secondary-sm" on:click={() => { result = null; selectedPaths = {}; }}>Rescan</button>
          <span class="footer-total">{formatBytes(totalCurrentSize)} total</span>
        </div>
      </div>

      <!-- RIGHT PANEL: sunburst heatmap -->
      <div class="right-panel">
        <div class="panel-header">
          <span class="panel-title">Size Map</span>
          {#if currentView !== 'root'}
            <span class="panel-subtitle">{breadcrumbs.map(b => b.label).join(' / ')}</span>
          {/if}
        </div>

        <div class="heatmap-container">
          {#if innerArcs.length > 0}
            <svg viewBox="0 0 300 300" class="sunburst">
              <!-- Inner ring: radius 55-100 -->
              {#each innerArcs as arc, i}
                <path
                  d={describeArc(150, 150, 100, 55, arc.startAngle, arc.sweepAngle)}
                  fill={arc.color}
                  fill-opacity={hoveredSegment && hoveredSegment.ring === 'inner' && hoveredSegment.index === i ? 1 : 0.75}
                  stroke="var(--bg-primary)"
                  stroke-width="2"
                  class="arc-segment"
                  on:mouseenter={() => hoveredSegment = { ring: 'inner', index: i, label: arc.label, size: arc.size, isDir: arc.isDir !== undefined ? arc.isDir : true }}
                  on:mouseleave={() => hoveredSegment = null}
                  on:click={() => handleHeatmapClick(arc)}
                />
              {/each}

              <!-- Outer ring: radius 105-135 -->
              {#each outerArcs as arc, i}
                <path
                  d={describeArc(150, 150, 135, 105, arc.startAngle, arc.sweepAngle)}
                  fill={arc.color}
                  fill-opacity={hoveredSegment && hoveredSegment.ring === 'outer' && hoveredSegment.outerIndex === i ? 0.95 : arc.opacity}
                  stroke="var(--bg-primary)"
                  stroke-width="1"
                  class="arc-segment"
                  on:mouseenter={() => hoveredSegment = { ring: 'outer', outerIndex: i, label: arc.label, size: arc.size, isDir: arc.isDir }}
                  on:mouseleave={() => hoveredSegment = null}
                  on:click={() => handleHeatmapClick(arc)}
                />
              {/each}

              <!-- Center text -->
              <text x="150" y="140" text-anchor="middle" fill="var(--text-primary)" font-size="16" font-weight="700">
                {hoveredSegment ? formatBytes(hoveredSegment.size) : formatBytes(totalCurrentSize)}
              </text>
              <text x="150" y="160" text-anchor="middle" fill="var(--text-secondary)" font-size="11">
                {hoveredSegment ? hoveredSegment.label : (heatmapData.length + ' items')}
              </text>
              {#if hoveredSegment}
                <text x="150" y="176" text-anchor="middle" fill="var(--text-muted)" font-size="9">
                  {hoveredSegment.isDir ? 'Click to open' : 'Click to select'}
                </text>
              {/if}
            </svg>

            <!-- Legend -->
            <div class="heatmap-legend">
              {#each heatmapData.slice(0, 10) as segment, i}
                <button
                  class="legend-item"
                  class:hovered={hoveredSegment && hoveredSegment.ring === 'inner' && hoveredSegment.index === i}
                  class:is-selected={selectedPaths[segment.path]}
                  on:mouseenter={() => hoveredSegment = { ring: 'inner', index: i, label: segment.label, size: segment.size, isDir: segment.isDir !== undefined ? segment.isDir : true }}
                  on:mouseleave={() => hoveredSegment = null}
                  on:click={() => handleHeatmapClick(segment)}
                >
                  <span class="legend-dot" style="background: {segment.color}"></span>
                  <span class="legend-label">{segment.label.split('/').pop() || segment.label}</span>
                  <span class="legend-size">{formatBytes(segment.size)}</span>
                  {#if segment.isDir}
                    <span class="legend-arrow">&#9654;</span>
                  {:else}
                    {#if selectedPaths[segment.path]}
                      <span class="legend-check">&#10003;</span>
                    {/if}
                  {/if}
                </button>
              {/each}
              {#if heatmapData.length > 10}
                <div class="legend-more">+{heatmapData.length - 10} more</div>
              {/if}
            </div>
          {:else}
            <div class="heatmap-empty">No data to visualize</div>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Confirm modal -->
  {#if showConfirm}
    {@const isSingle = singleDeletePath !== null}
    {@const modalItems = isSingle
      ? [[singleDeletePath, singleDeleteInfo]]
      : Object.entries(selectedPaths).filter(([p]) => !p.startsWith('__cat__'))
    }
    {@const modalTotal = isSingle
      ? (singleDeleteInfo?.size || 0)
      : modalItems.reduce((s, [, info]) => s + info.size, 0)
    }
    <div class="modal-overlay" on:click={() => { showConfirm = false; singleDeletePath = null; singleDeleteInfo = null; }}>
      <div class="modal" on:click|stopPropagation>
        <h3>Confirm Cleanup</h3>
        <div class="modal-body">
          <p>Move {modalItems.length} item(s) to Trash:</p>
          <div class="modal-items">
            {#each modalItems.slice(0, 10) as [path, info]}
              <div class="modal-item">
                <span class="modal-item-name">{(info.name || path).split('/').pop()}</span>
                <span class="modal-item-size">{formatBytes(info.size)}</span>
              </div>
            {/each}
            {#if modalItems.length > 10}
              <div class="modal-item-more">...and {modalItems.length - 10} more</div>
            {/if}
          </div>
          <div class="modal-total">Total: {formatBytes(modalTotal)}</div>
          <p class="modal-note">Items can be recovered from Trash</p>
        </div>
        <div class="modal-actions">
          <button class="btn-secondary" on:click={() => { showConfirm = false; singleDeletePath = null; singleDeleteInfo = null; }}>Cancel</button>
          <button class="btn-danger" on:click={executeClean} disabled={cleaning}>
            {cleaning ? 'Cleaning...' : 'Move to Trash'}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .clean-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 20px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }

  /* --- Scan progress --- */
  .scan-progress-card {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    padding: 24px;
    margin-bottom: 24px;
  }
  .progress-top { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
  .progress-spinner {
    width: 32px; height: 32px;
    border: 3px solid var(--bg-tertiary);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    flex-shrink: 0; margin-top: 2px;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .progress-info { flex: 1; min-width: 0; }
  .progress-title { font-size: 16px; font-weight: 600; margin-bottom: 6px; }
  .progress-category { font-size: 13px; color: var(--accent); font-weight: 500; margin-bottom: 4px; }
  .progress-file {
    font-size: 12px; color: var(--text-muted);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    margin-bottom: 8px;
    font-family: 'SF Mono', 'Fira Code', monospace;
    min-height: 16px;
  }
  .progress-stats { display: flex; gap: 8px; font-size: 13px; color: var(--text-secondary); }
  .progress-stats .dot { color: var(--text-muted); }
  .progress-bar-track { height: 4px; background: var(--bg-tertiary); border-radius: 2px; overflow: hidden; }
  .progress-bar-fill { height: 100%; background: var(--accent); border-radius: 2px; transition: width 0.3s ease; }

  /* --- Empty state --- */
  .empty-state { text-align: center; padding: 60px 0; }
  .empty-icon { font-size: 48px; color: var(--accent); margin-bottom: 16px; }
  .empty-state p { color: var(--text-secondary); margin-bottom: 24px; }

  /* --- Selection bar --- */
  .selection-bar {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 16px;
    background: var(--accent-dim);
    border-radius: var(--radius-sm);
    margin-bottom: 12px;
  }
  .sel-count { font-size: 13px; font-weight: 600; color: var(--accent); }
  .sel-size { font-size: 13px; color: var(--text-secondary); }
  .sel-actions { margin-left: auto; display: flex; gap: 8px; }

  /* --- Nav row (breadcrumbs + actions) --- */
  .nav-row {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 14px;
  }
  .nav-actions { flex-shrink: 0; }
  .breadcrumbs { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .crumb {
    background: none; color: var(--text-secondary); font-size: 13px;
    padding: 4px 8px; border-radius: 6px; transition: all var(--transition);
  }
  .crumb:hover { color: var(--accent); background: var(--accent-dim); }
  .crumb.active { color: var(--text-primary); font-weight: 600; }
  .crumb-sep { color: var(--text-muted); font-size: 10px; }

  /* --- Split view --- */
  .split-view {
    display: grid;
    grid-template-columns: 1fr 340px;
    gap: 16px;
    height: calc(100vh - 280px);
  }

  .left-panel, .right-panel {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    display: flex; flex-direction: column; overflow: hidden;
  }

  .panel-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 12px 16px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  .panel-header-left {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }
  .panel-title {
    font-size: 12px; font-weight: 600; color: var(--text-secondary);
    text-transform: uppercase; letter-spacing: 0.5px;
  }
  .panel-subtitle {
    font-size: 11px; color: var(--text-muted);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    max-width: 180px;
  }

  .btn-back {
    background: none; color: var(--text-secondary);
    font-size: 12px; padding: 4px 10px; border-radius: 6px;
    transition: all var(--transition);
  }
  .btn-back:hover { background: var(--bg-hover); color: var(--text-primary); }

  .sort-select {
    background: var(--bg-tertiary);
    color: var(--text-secondary);
    border: 1px solid var(--border);
    border-radius: 6px;
    padding: 4px 8px;
    font-size: 11px;
    outline: none;
    cursor: pointer;
  }
  .sort-select:hover { border-color: var(--text-muted); }

  /* --- File list --- */
  .file-list { flex: 1; overflow-y: auto; padding: 4px 0; }

  .list-loading {
    display: flex; align-items: center; justify-content: center;
    gap: 10px; padding: 32px; color: var(--text-muted); font-size: 13px;
  }
  .mini-spinner {
    width: 16px; height: 16px;
    border: 2px solid var(--bg-tertiary); border-top-color: var(--accent);
    border-radius: 50%; animation: spin 0.8s linear infinite;
  }
  .btn-cancel {
    background: var(--bg-tertiary); color: var(--text-secondary);
    padding: 4px 12px; border-radius: 6px; font-size: 12px;
    margin-left: 8px; transition: all var(--transition);
  }
  .btn-cancel:hover { background: var(--red-dim); color: var(--red); }
  .list-empty { text-align: center; padding: 32px; color: var(--text-muted); font-size: 13px; }

  .file-row {
    display: flex; align-items: center;
    margin: 0 6px; border-radius: var(--radius-sm);
    transition: background var(--transition);
  }
  .file-row:hover { background: var(--bg-hover); }
  .file-row.selected { background: var(--accent-dim); }
  .file-row.is-category { border-bottom: 1px solid var(--border); margin-bottom: 2px; }

  .file-check {
    background: none; padding: 10px 6px 10px 10px;
    font-size: 16px; flex-shrink: 0;
  }
  .file-check-spacer { width: 32px; flex-shrink: 0; }
  .check-on { color: var(--accent); }
  .check-off { color: var(--text-muted); }

  .file-info {
    flex: 1; display: flex; align-items: center; justify-content: space-between;
    gap: 8px; padding: 10px 8px 10px 0;
    background: none; color: var(--text-primary); text-align: left;
    min-width: 0; cursor: pointer;
  }

  .file-left { display: flex; align-items: center; gap: 10px; flex: 1; min-width: 0; }
  .file-icon { font-size: 18px; flex-shrink: 0; }
  .file-details { flex: 1; min-width: 0; }
  .file-name {
    font-size: 13px; white-space: nowrap; overflow: hidden;
    text-overflow: ellipsis; margin-bottom: 4px;
  }
  .file-bar-track { height: 3px; background: var(--bg-tertiary); border-radius: 2px; overflow: hidden; }
  .file-bar-fill { height: 100%; border-radius: 2px; transition: width 0.3s ease; min-width: 2px; }

  .file-right { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
  .file-size { font-size: 12px; color: var(--text-muted); font-weight: 500; min-width: 60px; text-align: right; }
  .item-count { font-size: 11px; color: var(--text-muted); }
  .drill-arrow { font-size: 10px; color: var(--text-muted); }

  .inline-delete {
    background: none; color: var(--red);
    font-size: 15px; padding: 6px 8px; border-radius: 6px;
    opacity: 0; transition: all var(--transition);
    flex-shrink: 0;
  }
  .file-row:hover .inline-delete { opacity: 0.7; }
  .inline-delete:hover { opacity: 1; color: #ff4444; background: var(--red-dim); }

  .risk-badge {
    font-size: 9px; font-weight: 700; padding: 2px 6px;
    border-radius: 4px; letter-spacing: 0.5px;
  }
  .risk-badge.green { background: var(--green-dim); color: var(--green); }
  .risk-badge.yellow { background: var(--yellow-dim); color: var(--yellow); }
  .risk-badge.red { background: var(--red-dim); color: var(--red); }

  .panel-footer {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 16px; border-top: 1px solid var(--border); flex-shrink: 0;
  }
  .footer-total { font-size: 12px; color: var(--text-muted); }

  /* --- Sunburst heatmap --- */
  .heatmap-container {
    flex: 1; display: flex; flex-direction: column;
    align-items: center; padding: 16px; overflow-y: auto;
  }
  .sunburst { width: 100%; max-width: 280px; margin-bottom: 16px; }
  .arc-segment { cursor: pointer; transition: fill-opacity 0.15s ease; }

  .heatmap-legend { width: 100%; display: flex; flex-direction: column; gap: 2px; }
  .legend-item {
    display: flex; align-items: center; gap: 8px;
    padding: 6px 8px; border-radius: 6px;
    background: none; color: var(--text-primary); text-align: left;
    transition: background var(--transition); width: 100%;
  }
  .legend-item:hover, .legend-item.hovered { background: var(--bg-hover); }
  .legend-item.is-selected { background: var(--accent-dim); }
  .legend-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .legend-label { flex: 1; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .legend-size { font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
  .legend-arrow { font-size: 9px; color: var(--text-muted); }
  .legend-check { font-size: 11px; color: var(--accent); }
  .legend-more { font-size: 11px; color: var(--text-muted); text-align: center; padding: 4px; }
  .heatmap-empty { color: var(--text-muted); font-size: 13px; text-align: center; padding: 40px 16px; }

  /* --- Buttons --- */
  .btn-primary {
    background: var(--accent); color: white; padding: 10px 24px;
    border-radius: var(--radius-sm); font-weight: 600; transition: all var(--transition);
  }
  .btn-primary:hover { background: var(--accent-hover); }
  .btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }

  .btn-secondary {
    background: var(--bg-tertiary); color: var(--text-primary); padding: 10px 24px;
    border-radius: var(--radius-sm); font-weight: 500; transition: all var(--transition);
  }
  .btn-secondary:hover { background: var(--bg-hover); }

  .btn-secondary-sm {
    background: var(--bg-tertiary); color: var(--text-secondary);
    padding: 6px 14px; border-radius: 6px; font-size: 12px; transition: all var(--transition);
  }
  .btn-secondary-sm:hover { background: var(--bg-hover); }

  .btn-sm-ghost {
    background: none; color: var(--text-secondary);
    padding: 6px 12px; border-radius: 6px; font-size: 12px; transition: all var(--transition);
  }
  .btn-sm-ghost:hover { background: var(--bg-hover); color: var(--text-primary); }

  .btn-sm-accent {
    background: var(--accent-dim); color: var(--accent);
    padding: 6px 14px; border-radius: 6px; font-size: 12px;
    font-weight: 600; transition: all var(--transition);
  }
  .btn-sm-accent:hover { background: var(--accent); color: white; }

  .btn-sm-danger {
    background: var(--red-dim); color: var(--red);
    padding: 6px 14px; border-radius: 6px; font-size: 12px;
    font-weight: 600; transition: all var(--transition);
  }
  .btn-sm-danger:hover { background: rgba(248, 113, 113, 0.25); }

  .btn-danger {
    background: var(--red-dim); color: var(--red); padding: 10px 24px;
    border-radius: var(--radius-sm); font-weight: 600; transition: all var(--transition);
  }
  .btn-danger:hover { background: rgba(248, 113, 113, 0.25); }

  /* --- Result banner --- */
  .result-banner {
    display: flex; align-items: center; gap: 16px;
    padding: 16px 20px; background: var(--green-dim); border-radius: var(--radius);
    margin-bottom: 12px;
  }
  .result-icon { font-size: 24px; color: var(--green); }
  .result-title { font-weight: 600; font-size: 16px; }
  .result-detail { font-size: 13px; color: var(--text-secondary); margin-top: 4px; }

  /* --- Modal --- */
  .modal-overlay {
    position: fixed; top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex; align-items: center; justify-content: center; z-index: 100;
  }
  .modal {
    background: var(--bg-secondary); border-radius: var(--radius); padding: 24px;
    max-width: 480px; width: 90%;
    border: 1px solid var(--border); box-shadow: var(--shadow);
  }
  .modal h3 { font-size: 18px; margin-bottom: 16px; }
  .modal-body { margin-bottom: 20px; }
  .modal-body p { color: var(--text-secondary); margin-bottom: 12px; }
  .modal-items { max-height: 200px; overflow-y: auto; margin-bottom: 12px; }
  .modal-item {
    display: flex; justify-content: space-between; padding: 6px 0;
    font-size: 13px; border-bottom: 1px solid var(--border);
  }
  .modal-item-name {
    color: var(--text-primary); overflow: hidden; text-overflow: ellipsis;
    white-space: nowrap; flex: 1; margin-right: 12px;
  }
  .modal-item-size { color: var(--text-muted); flex-shrink: 0; }
  .modal-item-more { font-size: 12px; color: var(--text-muted); padding: 8px 0 0; }
  .modal-total { font-size: 15px; font-weight: 600; color: var(--accent); margin-bottom: 8px; }
  .modal-note { font-size: 12px; color: var(--green); }
  .modal-actions { display: flex; justify-content: flex-end; gap: 12px; }
</style>
