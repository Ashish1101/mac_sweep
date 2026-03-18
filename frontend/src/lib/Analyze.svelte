<script>
  import { onMount, onDestroy } from 'svelte';
  import { StartAnalyzeScan, CancelAnalyzeScan, StartAnalyzeDrill, CancelAnalyzeDrill, PreFetchChildren, GetCachedChildren, GetHomeDir, MoveToTrash, PlayTrashSound } from '../../wailsjs/go/main/App.js';
  import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js';

  // --- State ---
  let scanResult = null;
  let scanning = false;
  let scanProgress = null;

  // Navigation
  let currentPath = '';
  let breadcrumbs = [];
  let children = [];

  // Selection
  let selectedPaths = new Set();

  // Drill
  let drillLoading = false;
  let drillRequestId = 0;

  // Sort
  let sortBy = 'size';

  // Heatmap
  let hoveredSegment = null;
  let legendExpanded = false;

  // Delete
  let showDeleteConfirm = false;
  let singleDeletePath = null;
  let singleDeleteInfo = null;
  let deleting = false;

  // Scan input
  let scanPath = '';

  // Reactivity trigger
  let navVersion = 0;

  // Event unsubscribers
  let unsubProgress = null;
  let unsubComplete = null;
  let unsubDrill = null;
  let unsubPrefetch = null;

  function bumpNav() {
    navVersion++;
  }

  // --- Utilities ---
  function formatBytes(bytes) {
    if (!bytes) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  const COLORS = [
    '#7c5cfc', '#60a5fa', '#4ade80', '#fbbf24', '#f87171',
    '#a78bfa', '#34d399', '#fb923c', '#f472b6', '#38bdf8',
    '#818cf8', '#2dd4bf', '#facc15', '#fb7185', '#c084fc'
  ];

  // --- Sunburst helpers ---
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

  function buildInnerArcs(_v) {
    if (!children || children.length === 0) return [];
    const total = children.reduce((s, c) => s + (c.size || 0), 0);
    if (total === 0) return [];
    let angle = 0;
    return children.map((entry, i) => {
      const sweep = (entry.size / total) * 360;
      const arc = {
        label: entry.name,
        path: entry.path,
        size: entry.size,
        isDir: entry.isDir,
        color: COLORS[i % COLORS.length],
        startAngle: angle,
        sweepAngle: sweep,
        index: i,
        ring: 'inner',
        children: entry.children || [],
      };
      angle += sweep;
      return arc;
    });
  }

  function buildOuterArcs(innerArcs) {
    const outerArcs = [];
    for (const inner of innerArcs) {
      if (!inner.isDir || !inner.children || inner.children.length === 0) continue;
      const childTotal = inner.children.reduce((s, c) => s + (c.size || 0), 0);
      if (childTotal === 0) continue;
      let angle = inner.startAngle;
      for (const child of inner.children) {
        const sweep = (child.size / childTotal) * inner.sweepAngle;
        // Opacity based on relative size within parent
        const relSize = child.size / childTotal;
        const opacity = 0.5 + relSize * 0.4; // 0.5 to 0.9
        outerArcs.push({
          label: child.name,
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

  // --- Sorting ---
  function sortItems(items, sort) {
    if (!items || items.length === 0) return items;
    const sorted = [...items];
    switch (sort) {
      case 'name':
        sorted.sort((a, b) => {
          const nameA = (a.name || '').toLowerCase();
          const nameB = (b.name || '').toLowerCase();
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

  function getSortedChildren(_v, _sort) {
    return sortItems(children, sortBy);
  }

  // --- Scanning ---
  function startScan() {
    const path = scanPath.trim() || currentPath || '/';
    scanning = true;
    scanResult = null;
    children = [];
    selectedPaths = new Set();
    breadcrumbs = [];
    scanProgress = { currentFile: '', filesFound: 0, dirsFound: 0, sizeFound: 0, percentage: 0 };

    const reqId = ++drillRequestId;
    currentPath = path;
    updateBreadcrumbs(path);
    bumpNav();

    StartAnalyzeScan(path, 3, reqId);
  }

  function onScanProgress(data) {
    if (scanning) {
      scanProgress = data;
    }
  }

  function onScanComplete(data) {
    if (!data) return;
    scanning = false;
    scanProgress = null;
    scanResult = data.result || null;

    if (scanResult && scanResult.root && scanResult.root.children) {
      children = scanResult.root.children;
    } else {
      children = [];
    }

    currentPath = scanPath.trim() || currentPath;
    updateBreadcrumbs(currentPath);
    bumpNav();

    // Pre-fetch top 10 largest directories
    prefetchTopDirs(children);
  }

  function prefetchTopDirs(items) {
    if (!items || items.length === 0) return;
    const dirs = items
      .filter(e => e.isDir)
      .sort((a, b) => (b.size || 0) - (a.size || 0))
      .slice(0, 10)
      .map(e => e.path);
    if (dirs.length > 0) {
      PreFetchChildren(dirs);
    }
  }

  // --- Breadcrumbs ---
  function updateBreadcrumbs(path) {
    const parts = path.split('/').filter(Boolean);
    breadcrumbs = parts.map((part, i) => ({
      name: part,
      path: '/' + parts.slice(0, i + 1).join('/')
    }));
  }

  // --- Navigation ---
  async function drillDown(entry) {
    if (!entry.isDir) return;

    const myId = ++drillRequestId;

    currentPath = entry.path;
    updateBreadcrumbs(entry.path);
    selectedPaths = new Set();

    // Check cache first
    try {
      const cached = await GetCachedChildren(entry.path);
      if (cached && myId === drillRequestId) {
        children = cached;
        drillLoading = false;
        bumpNav();
        prefetchTopDirs(cached);
        return;
      }
    } catch (e) {
      // Cache miss, proceed to drill
    }

    // If entry has inline children, use them
    if (entry.children && entry.children.length > 0 && myId === drillRequestId) {
      children = entry.children;
      drillLoading = false;
      bumpNav();
      prefetchTopDirs(entry.children);
      return;
    }

    // Fire-and-forget drill
    children = [];
    drillLoading = true;
    bumpNav();
    StartAnalyzeDrill(entry.path, myId);
  }

  function onDrillComplete(data) {
    if (!data || data.requestId !== drillRequestId) return;

    children = data.items || [];
    drillLoading = false;
    bumpNav();

    prefetchTopDirs(children);
  }

  function cancelDrill() {
    drillRequestId++;
    CancelAnalyzeDrill();
    drillLoading = false;

    // Go back one level
    if (breadcrumbs.length > 1) {
      const parentCrumb = breadcrumbs[breadcrumbs.length - 2];
      navigateBreadcrumb(parentCrumb.path);
    } else {
      // Go to root of scan
      navigateBreadcrumb(breadcrumbs.length > 0 ? breadcrumbs[0].path : '/');
    }
  }

  function navigateBreadcrumb(path) {
    // Don't increment drillRequestId here — let drillDown manage it
    CancelAnalyzeDrill();
    drillLoading = false;
    drillDown({ path, isDir: true, children: null });
  }

  function goBack() {
    if (breadcrumbs.length > 1) {
      const parentCrumb = breadcrumbs[breadcrumbs.length - 2];
      CancelAnalyzeDrill();
      drillLoading = false;
      drillDown({ path: parentCrumb.path, isDir: true, children: null });
    }
  }

  // --- Heatmap click ---
  function handleHeatmapClick(arc) {
    if (arc.isDir) {
      drillDown({ path: arc.path, isDir: true, children: arc.children || null });
    }
  }

  // --- Selection ---
  function toggleSelect(path, size, name) {
    if (selectedPaths.has(path)) {
      selectedPaths.delete(path);
    } else {
      selectedPaths.add(path);
    }
    selectedPaths = selectedPaths;
  }

  function deselectAll() {
    selectedPaths = new Set();
  }

  function selectAllVisible() {
    const items = getSortedChildren(navVersion, sortBy);
    for (const item of items) {
      selectedPaths.add(item.path);
    }
    selectedPaths = selectedPaths;
  }

  // --- Delete ---
  async function executeDelete() {
    showDeleteConfirm = false;
    deleting = true;

    const pathsToDelete = singleDeletePath ? [singleDeletePath] : [...selectedPaths];

    if (pathsToDelete.length === 0) {
      deleting = false;
      return;
    }

    // Collect sizes BEFORE removing from children array
    const deletedSet = new Set(pathsToDelete);
    let freedSize = 0;
    let freedFiles = 0;
    let freedDirs = 0;
    for (const child of children) {
      if (deletedSet.has(child.path)) {
        freedSize += child.size || 0;
        if (child.isDir) freedDirs++;
        else freedFiles++;
      }
    }

    let deletedCount = 0;
    for (const path of pathsToDelete) {
      try {
        await MoveToTrash(path);
        deletedCount++;
      } catch (e) {
        console.error('Failed to trash:', path, e);
      }
    }

    if (deletedCount > 0) {
      PlayTrashSound();
    }

    // Remove deleted items from children
    children = children.filter(c => !deletedSet.has(c.path));

    // Update scan result totals with pre-collected sizes
    if (scanResult && deletedCount > 0) {
      scanResult.totalSize = Math.max(0, (scanResult.totalSize || 0) - freedSize);
      scanResult.totalFiles = Math.max(0, (scanResult.totalFiles || 0) - freedFiles);
      scanResult.totalDirs = Math.max(0, (scanResult.totalDirs || 0) - freedDirs);
      scanResult = scanResult; // trigger reactivity
    }

    // Clear selection
    for (const path of pathsToDelete) {
      selectedPaths.delete(path);
    }
    selectedPaths = selectedPaths;
    singleDeletePath = null;
    singleDeleteInfo = null;
    deleting = false;
    bumpNav();
  }

  // --- Lifecycle ---
  onMount(async () => {
    const home = await GetHomeDir();
    currentPath = home;
    scanPath = home;

    unsubProgress = EventsOn('analyze:progress', onScanProgress);
    unsubComplete = EventsOn('analyze:complete', onScanComplete);
    unsubDrill = EventsOn('analyze:drill-complete', onDrillComplete);
    unsubPrefetch = EventsOn('analyze:prefetch-ready', (data) => {
      // A path is now cached; no action needed — next drill will hit cache
    });
  });

  onDestroy(() => {
    if (unsubProgress) unsubProgress();
    if (unsubComplete) unsubComplete();
    if (unsubDrill) unsubDrill();
    if (unsubPrefetch) unsubPrefetch();
    CancelAnalyzeScan();
    CancelAnalyzeDrill();
  });

  // --- Reactive ---
  $: innerArcs = buildInnerArcs(navVersion);
  $: outerArcs = buildOuterArcs(innerArcs);
  $: listItems = getSortedChildren(navVersion, sortBy);
  $: totalCurrentSize = children.reduce((s, c) => s + (c.size || 0), 0);
  $: selectedCount = selectedPaths.size;
  $: selectedSize = (() => {
    let total = 0;
    for (const path of selectedPaths) {
      const item = children.find(c => c.path === path);
      if (item) total += item.size || 0;
    }
    return total;
  })();
</script>

<div class="analyze-page">
  <div class="page-header">
    <h1>Disk Analysis</h1>
    <p class="subtitle">Visualize and manage disk usage</p>
  </div>

  <!-- Scan path input -->
  <div class="scan-input-row">
    <input
      class="scan-input"
      type="text"
      bind:value={scanPath}
      placeholder="Enter path to scan..."
      on:keydown={(e) => e.key === 'Enter' && startScan()}
      disabled={scanning}
    />
    <button class="btn-primary" on:click={startScan} disabled={scanning}>
      {scanning ? 'Scanning...' : 'Scan'}
    </button>
  </div>

  <!-- Scan progress -->
  {#if scanning}
    <div class="scan-progress-card">
      <div class="progress-top">
        <div class="progress-spinner"></div>
        <div class="progress-info">
          <div class="progress-title">Scanning your system...</div>
          {#if scanProgress}
            <div class="progress-file">{scanProgress.currentFile || ''}</div>
            <div class="progress-stats">
              <span>{(scanProgress.filesFound || 0).toLocaleString()} files</span>
              <span class="dot">&middot;</span>
              <span>{(scanProgress.dirsFound || 0).toLocaleString()} dirs</span>
              <span class="dot">&middot;</span>
              <span>{formatBytes(scanProgress.sizeFound || 0)}</span>
            </div>
          {/if}
        </div>
      </div>
      {#if scanProgress}
        <div class="progress-bar-track">
          <div class="progress-bar-fill" style="width: {Math.max(scanProgress.percentage || 0, 5)}%"></div>
        </div>
      {/if}
    </div>

  <!-- Empty state -->
  {:else if !scanResult}
    <div class="empty-state">
      <div class="empty-icon">&#128269;</div>
      <p>Enter a path and click Scan to analyze disk usage</p>
    </div>

  <!-- Scan results -->
  {:else}
    <!-- Summary -->
    <div class="scan-summary">
      <div class="summary-item">
        <span class="summary-value">{formatBytes(scanResult.totalSize)}</span>
        <span class="summary-label">Total Size</span>
      </div>
      <div class="summary-item">
        <span class="summary-value">{(scanResult.totalFiles || 0).toLocaleString()}</span>
        <span class="summary-label">Files</span>
      </div>
      <div class="summary-item">
        <span class="summary-value">{(scanResult.totalDirs || 0).toLocaleString()}</span>
        <span class="summary-label">Folders</span>
      </div>
    </div>

    <!-- Selection bar -->
    {#if selectedCount > 0}
      <div class="selection-bar">
        <span class="sel-count">{selectedCount} selected</span>
        <span class="sel-size">{formatBytes(selectedSize)}</span>
        <div class="sel-actions">
          <button class="btn-sm-ghost" on:click={deselectAll}>Deselect All</button>
          <button class="btn-sm-danger" on:click={() => showDeleteConfirm = true}>Move to Trash</button>
        </div>
      </div>
    {/if}

    <!-- Breadcrumbs -->
    <div class="nav-row">
      <div class="breadcrumbs">
        <button class="crumb" on:click={() => navigateBreadcrumb('/')}>/</button>
        {#each breadcrumbs as crumb, i}
          <span class="crumb-sep">&#9654;</span>
          <button
            class="crumb"
            class:active={i === breadcrumbs.length - 1}
            on:click={() => navigateBreadcrumb(crumb.path)}
          >{crumb.name}</button>
        {/each}
      </div>
      <div class="nav-actions">
        <button class="btn-sm-ghost" on:click={selectAllVisible}>Select All</button>
      </div>
    </div>

    <!-- Split view -->
    <div class="split-view">
      <!-- LEFT PANEL: file list -->
      <div class="left-panel">
        <div class="panel-header">
          <div class="panel-header-left">
            {#if breadcrumbs.length > 1}
              <button class="btn-back" on:click={goBack}>&#8592; Back</button>
            {/if}
            <span class="panel-title">
              {listItems.length} items &middot; {formatBytes(totalCurrentSize)}
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
            {#each listItems as entry, i}
              {@const maxSize = listItems.length > 0 ? listItems[0].size : 1}
              <div
                class="file-row"
                class:selected={selectedPaths.has(entry.path)}
                class:is-dir={entry.isDir}
              >
                <!-- Checkbox -->
                <button class="file-check" on:click|stopPropagation={() => toggleSelect(entry.path, entry.size, entry.name)}>
                  {#if selectedPaths.has(entry.path)}
                    <span class="check-on">&#10003;</span>
                  {:else}
                    <span class="check-off">&#9675;</span>
                  {/if}
                </button>

                <!-- Clickable row -->
                <button class="file-info" on:click={() => entry.isDir ? drillDown(entry) : null}>
                  <div class="file-left">
                    <span class="file-icon">
                      {#if entry.isDir}
                        &#128193;
                      {:else}
                        &#128196;
                      {/if}
                    </span>
                    <div class="file-details">
                      <div class="file-name">{entry.name}</div>
                      <div class="file-bar-track">
                        <div class="file-bar-fill" style="width: {Math.max(2, (entry.size / maxSize) * 100)}%; background: {COLORS[i % COLORS.length]}"></div>
                      </div>
                    </div>
                  </div>

                  <div class="file-right">
                    <span class="file-size">{formatBytes(entry.size)}</span>
                    {#if entry.isDir}
                      <span class="drill-arrow">&#9654;</span>
                    {/if}
                  </div>
                </button>

                <!-- Inline delete -->
                <button
                  class="inline-delete"
                  title="Move to Trash"
                  on:click|stopPropagation={() => {
                    singleDeletePath = entry.path;
                    singleDeleteInfo = { size: entry.size, name: entry.name };
                    showDeleteConfirm = true;
                  }}
                >&#128465;</button>
              </div>
            {/each}

            {#if listItems.length === 0}
              <div class="list-empty">No items found</div>
            {/if}
          {/if}
        </div>

        <!-- Footer -->
        <div class="panel-footer">
          <button class="btn-secondary-sm" on:click={() => { scanResult = null; children = []; selectedPaths = new Set(); }}>Rescan</button>
          <span class="footer-total">{formatBytes(totalCurrentSize)} total</span>
        </div>
      </div>

      <!-- RIGHT PANEL: sunburst heatmap -->
      <div class="right-panel">
        <div class="panel-header">
          <span class="panel-title">Size Map</span>
          <span class="panel-subtitle">{breadcrumbs.map(b => b.name).join(' / ')}</span>
        </div>

        <div class="heatmap-container">
          {#if innerArcs.length > 0}
            <svg viewBox="0 0 400 400" class="sunburst">
              <!-- Inner ring -->
              {#each innerArcs as arc, i}
                <path
                  d={describeArc(200, 200, 140, 70, arc.startAngle, arc.sweepAngle)}
                  fill={arc.color}
                  fill-opacity={hoveredSegment && hoveredSegment.ring === 'inner' && hoveredSegment.index === i ? 1 : 0.75}
                  stroke="var(--bg-primary)"
                  stroke-width="2"
                  class="arc-segment"
                  on:mouseenter={() => hoveredSegment = { ring: 'inner', index: i, label: arc.label, size: arc.size, isDir: arc.isDir }}
                  on:mouseleave={() => hoveredSegment = null}
                  on:click={() => handleHeatmapClick(arc)}
                />
              {/each}

              <!-- Outer ring -->
              {#each outerArcs as arc, i}
                <path
                  d={describeArc(200, 200, 185, 148, arc.startAngle, arc.sweepAngle)}
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
              <text x="200" y="192" text-anchor="middle" fill="var(--text-primary)" font-size="20" font-weight="700">
                {hoveredSegment ? formatBytes(hoveredSegment.size) : formatBytes(totalCurrentSize)}
              </text>
              <text x="200" y="214" text-anchor="middle" fill="var(--text-secondary)" font-size="13">
                {hoveredSegment ? hoveredSegment.label : (children.length + ' items')}
              </text>
              {#if hoveredSegment}
                <text x="200" y="232" text-anchor="middle" fill="var(--text-muted)" font-size="10">
                  {hoveredSegment.isDir ? 'Click to open' : ''}
                </text>
              {/if}
            </svg>

            <!-- Collapsible legend -->
            <div class="legend-section">
              <button class="legend-toggle" on:click={() => legendExpanded = !legendExpanded}>
                <span>{innerArcs.length} items</span>
                <span class="legend-toggle-icon">{legendExpanded ? '&#9660;' : '&#9654;'} {legendExpanded ? 'Hide' : 'Show'} legend</span>
              </button>
              {#if legendExpanded}
                <div class="heatmap-legend">
                  {#each innerArcs as segment, i}
                    <button
                      class="legend-item"
                      class:hovered={hoveredSegment && hoveredSegment.ring === 'inner' && hoveredSegment.index === i}
                      class:is-selected={selectedPaths.has(segment.path)}
                      on:mouseenter={() => hoveredSegment = { ring: 'inner', index: i, label: segment.label, size: segment.size, isDir: segment.isDir }}
                      on:mouseleave={() => hoveredSegment = null}
                      on:click={() => handleHeatmapClick(segment)}
                    >
                      <span class="legend-dot" style="background: {segment.color}"></span>
                      <span class="legend-label">{segment.label}</span>
                      <span class="legend-size">{formatBytes(segment.size)}</span>
                      {#if segment.isDir}
                        <span class="legend-arrow">&#9654;</span>
                      {/if}
                    </button>
                  {/each}
                </div>
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
  {#if showDeleteConfirm}
    {@const isSingle = singleDeletePath !== null}
    {@const modalPaths = isSingle ? [singleDeletePath] : [...selectedPaths]}
    {@const modalItems = modalPaths.map(p => {
      if (isSingle && singleDeleteInfo) return { path: p, name: singleDeleteInfo.name, size: singleDeleteInfo.size };
      const entry = children.find(c => c.path === p);
      return entry ? { path: p, name: entry.name, size: entry.size } : { path: p, name: p.split('/').pop(), size: 0 };
    })}
    {@const modalTotal = modalItems.reduce((s, item) => s + (item.size || 0), 0)}
    <div class="modal-overlay" on:click={() => { showDeleteConfirm = false; singleDeletePath = null; singleDeleteInfo = null; }}>
      <div class="modal" on:click|stopPropagation>
        <h3>Confirm Delete</h3>
        <div class="modal-body">
          <p>Move {modalItems.length} item(s) to Trash:</p>
          <div class="modal-items">
            {#each modalItems.slice(0, 10) as item}
              <div class="modal-item">
                <span class="modal-item-name">{item.name}</span>
                <span class="modal-item-size">{formatBytes(item.size)}</span>
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
          <button class="btn-secondary" on:click={() => { showDeleteConfirm = false; singleDeletePath = null; singleDeleteInfo = null; }}>Cancel</button>
          <button class="btn-danger" on:click={executeDelete} disabled={deleting}>
            {deleting ? 'Deleting...' : 'Move to Trash'}
          </button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .analyze-page { padding: 0 32px 32px; }
  .page-header { margin-bottom: 20px; }
  .page-header h1 { font-size: 24px; font-weight: 700; letter-spacing: -0.5px; }
  .subtitle { color: var(--text-secondary); margin-top: 4px; }

  /* --- Scan input --- */
  .scan-input-row {
    display: flex; gap: 10px; margin-bottom: 20px; align-items: center;
  }
  .scan-input {
    flex: 1; background: var(--bg-tertiary); color: var(--text-primary);
    border: 1px solid var(--border); border-radius: var(--radius-sm);
    padding: 10px 14px; font-size: 13px;
    font-family: 'SF Mono', 'Fira Code', monospace;
    outline: none; transition: border-color var(--transition);
  }
  .scan-input:focus { border-color: var(--accent); }
  .scan-input:disabled { opacity: 0.5; }

  /* --- Scan progress --- */
  .scan-progress-card {
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: var(--radius); padding: 24px; margin-bottom: 24px;
  }
  .progress-top { display: flex; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
  .progress-spinner {
    width: 32px; height: 32px;
    border: 3px solid var(--bg-tertiary); border-top-color: var(--accent);
    border-radius: 50%; animation: spin 0.8s linear infinite;
    flex-shrink: 0; margin-top: 2px;
  }
  @keyframes spin { to { transform: rotate(360deg); } }
  .progress-info { flex: 1; min-width: 0; }
  .progress-title { font-size: 16px; font-weight: 600; margin-bottom: 6px; }
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

  /* --- Scan summary --- */
  .scan-summary {
    display: flex; gap: 24px; margin-bottom: 16px;
    padding: 16px 20px; background: var(--bg-card);
    border-radius: var(--radius); border: 1px solid var(--border);
  }
  .summary-item { display: flex; flex-direction: column; }
  .summary-value { font-size: 20px; font-weight: 700; }
  .summary-label { font-size: 12px; color: var(--text-secondary); margin-top: 2px; }

  /* --- Selection bar --- */
  .selection-bar {
    display: flex; align-items: center; gap: 12px;
    padding: 10px 16px; background: var(--accent-dim);
    border-radius: var(--radius-sm); margin-bottom: 12px;
  }
  .sel-count { font-size: 13px; font-weight: 600; color: var(--accent); }
  .sel-size { font-size: 13px; color: var(--text-secondary); }
  .sel-actions { margin-left: auto; display: flex; gap: 8px; }

  /* --- Nav row --- */
  .nav-row {
    display: flex; justify-content: space-between; align-items: center;
    margin-bottom: 14px;
  }
  .nav-actions { flex-shrink: 0; }
  .breadcrumbs { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; overflow-x: auto; }
  .crumb {
    background: none; color: var(--text-secondary); font-size: 13px;
    padding: 4px 8px; border-radius: 6px; transition: all var(--transition);
    white-space: nowrap;
  }
  .crumb:hover { color: var(--accent); background: var(--accent-dim); }
  .crumb.active { color: var(--text-primary); font-weight: 600; }
  .crumb-sep { color: var(--text-muted); font-size: 10px; }

  /* --- Split view --- */
  .split-view {
    display: grid; grid-template-columns: 1fr 420px;
    gap: 16px; height: calc(100vh - 340px);
  }
  .left-panel, .right-panel {
    background: var(--bg-card); border: 1px solid var(--border);
    border-radius: var(--radius); display: flex; flex-direction: column; overflow: hidden;
  }
  .panel-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 12px 16px; border-bottom: 1px solid var(--border); flex-shrink: 0; gap: 8px;
  }
  .panel-title {
    font-size: 12px; font-weight: 600; color: var(--text-secondary);
    text-transform: uppercase; letter-spacing: 0.5px;
  }
  .panel-header-left {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }
  .panel-subtitle {
    font-size: 11px; color: var(--text-muted);
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px;
  }

  .sort-select {
    background: var(--bg-tertiary); color: var(--text-secondary);
    border: 1px solid var(--border); border-radius: 6px;
    padding: 4px 8px; font-size: 11px;
  }

  .btn-back {
    background: none; color: var(--text-secondary);
    font-size: 12px; padding: 4px 10px; border-radius: 6px;
    transition: all var(--transition); white-space: nowrap;
  }
  .btn-back:hover { background: var(--bg-hover); color: var(--text-primary); }

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

  .file-check {
    background: none; padding: 10px 6px 10px 10px;
    font-size: 16px; flex-shrink: 0;
  }
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
  .drill-arrow { font-size: 10px; color: var(--text-muted); }

  .inline-delete {
    background: none; color: var(--red);
    font-size: 15px; padding: 6px 8px; border-radius: 6px;
    opacity: 0; transition: all var(--transition); flex-shrink: 0;
  }
  .file-row:hover .inline-delete { opacity: 0.7; }
  .inline-delete:hover { opacity: 1; color: #ff4444; background: var(--red-dim); }

  .panel-footer {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 16px; border-top: 1px solid var(--border); flex-shrink: 0;
  }
  .footer-total { font-size: 12px; color: var(--text-muted); }

  /* --- Sunburst heatmap --- */
  .heatmap-container {
    flex: 1; display: flex; flex-direction: column;
    align-items: center; padding: 12px; overflow-y: auto;
  }
  .sunburst { width: 100%; margin-bottom: 8px; }
  .arc-segment { cursor: pointer; transition: fill-opacity 0.15s ease; }

  .legend-section {
    width: 100%; border-top: 1px solid var(--border);
    margin-top: 4px;
  }
  .legend-toggle {
    display: flex; justify-content: space-between; align-items: center;
    width: 100%; padding: 8px 8px; background: none;
    color: var(--text-secondary); font-size: 12px;
    transition: all var(--transition);
  }
  .legend-toggle:hover { color: var(--text-primary); }
  .legend-toggle-icon { font-size: 11px; color: var(--text-muted); }
  .heatmap-legend {
    width: 100%; display: flex; flex-direction: column; gap: 2px;
    max-height: 200px; overflow-y: auto; padding: 0 0 4px;
  }
  .legend-item {
    display: flex; align-items: center; gap: 8px;
    padding: 5px 8px; border-radius: 6px;
    background: none; color: var(--text-primary); text-align: left;
    transition: background var(--transition); width: 100%;
  }
  .legend-item:hover, .legend-item.hovered { background: var(--bg-hover); }
  .legend-item.is-selected { background: var(--accent-dim); }
  .legend-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .legend-label { flex: 1; font-size: 12px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .legend-size { font-size: 11px; color: var(--text-muted); flex-shrink: 0; }
  .legend-arrow { font-size: 9px; color: var(--text-muted); }
  .heatmap-empty { color: var(--text-muted); font-size: 13px; text-align: center; padding: 40px 16px; }

  /* --- Buttons --- */
  .btn-primary {
    background: var(--accent); color: white; padding: 10px 24px;
    border-radius: var(--radius-sm); font-weight: 600; transition: all var(--transition);
    white-space: nowrap;
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
