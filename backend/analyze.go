package backend

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// inodeID uniquely identifies a file across devices.
type inodeID struct {
	dev uint64
	ino uint64
}

// inodeFromInfo extracts the inode identity from an existing os.FileInfo
// without making an extra syscall.
func inodeFromInfo(info os.FileInfo) (inodeID, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return inodeID{}, false
	}
	return inodeID{dev: uint64(st.Dev), ino: st.Ino}, true
}

// seenAdd returns true if the inode was already seen (and should be skipped),
// false if it is new (and has been recorded).
func seenAdd(seen *sync.Map, id inodeID) bool {
	_, loaded := seen.LoadOrStore(id, struct{}{})
	return loaded
}

type FileEntry struct {
	Name     string       `json:"name"`
	Path     string       `json:"path"`
	Size     int64        `json:"size"`
	IsDir    bool         `json:"isDir"`
	Children []*FileEntry `json:"children,omitempty"`
	ModTime  int64        `json:"modTime"`
}

type ScanResult struct {
	Root       *FileEntry `json:"root"`
	TotalSize  int64      `json:"totalSize"`
	TotalFiles int        `json:"totalFiles"`
	TotalDirs  int        `json:"totalDirs"`
}

type AnalyzeProgress struct {
	RequestId   int    `json:"requestId"`
	CurrentFile string `json:"currentFile"`
	FilesFound  int    `json:"filesFound"`
	DirsFound   int    `json:"dirsFound"`
	SizeFound   int64  `json:"sizeFound"`
	Percentage  int    `json:"percentage"`
}

type AnalyzeService struct {
	ctx              context.Context
	mu               sync.Mutex
	scanCancel       context.CancelFunc
	drillCancel      context.CancelFunc
	prefetchCache    sync.Map // key: string (path), value: []*FileEntry
	lastProgressEmit int64    // unix nano timestamp
}

func NewAnalyzeService() *AnalyzeService {
	return &AnalyzeService{}
}

func (a *AnalyzeService) SetContext(ctx context.Context) {
	a.ctx = ctx
}

// ---------------------------------------------------------------------------
// StartScan — fire-and-forget async scan with progress events
// ---------------------------------------------------------------------------

func (a *AnalyzeService) StartScan(path string, maxDepth int, requestId int) {
	a.mu.Lock()
	// Cancel any previous scan
	if a.scanCancel != nil {
		a.scanCancel()
	}
	scanCtx, cancel := context.WithCancel(context.Background())
	a.scanCancel = cancel
	a.mu.Unlock()

	if maxDepth <= 0 {
		maxDepth = 3
	}

	go func() {
		defer cancel()

		info, err := os.Stat(path)
		if err != nil {
			return
		}

		root := &FileEntry{
			Name:  info.Name(),
			Path:  path,
			IsDir: info.IsDir(),
		}

		result := &ScanResult{Root: root}

		if info.IsDir() {
			var filesFound int64
			var dirsFound int64
			var sizeFound int64
			var seenInodes sync.Map

			// Start partial-tree emitter: snapshots root.Children every 500ms
			treeTicker := time.NewTicker(500 * time.Millisecond)
			treeDone := make(chan struct{})
			go func() {
				defer treeTicker.Stop()
				for {
					select {
					case <-treeTicker.C:
						if a.ctx != nil && root.Children != nil {
							// Snapshot children with current in-progress sizes
							snapshot := make([]map[string]interface{}, 0, len(root.Children))
							for _, c := range root.Children {
								snapshot = append(snapshot, map[string]interface{}{
									"name":  c.Name,
									"path":  c.Path,
									"size":  c.Size,
									"isDir": c.IsDir,
								})
							}
							wailsRuntime.EventsEmit(a.ctx, "analyze:partial-tree", map[string]interface{}{
								"requestId":  requestId,
								"items":      snapshot,
								"filesFound": int(atomic.LoadInt64(&filesFound)),
								"dirsFound":  int(atomic.LoadInt64(&dirsFound)),
								"sizeFound":  atomic.LoadInt64(&sizeFound),
							})
						}
					case <-treeDone:
						return
					case <-scanCtx.Done():
						return
					}
				}
			}()

			a.scanDirAsync(scanCtx, root, path, 0, maxDepth, result, requestId, &filesFound, &dirsFound, &sizeFound, &seenInodes)
			close(treeDone)
			result.TotalSize = root.Size
		} else {
			root.Size = info.Size()
			root.ModTime = info.ModTime().Unix()
			result.TotalSize = root.Size
			result.TotalFiles = 1
		}

		// Emit complete event
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "analyze:complete", map[string]interface{}{
				"requestId": requestId,
				"result":    result,
			})
		}
	}()
}

func (a *AnalyzeService) scanDirAsync(ctx context.Context, entry *FileEntry, path string, depth, maxDepth int, result *ScanResult, requestId int, filesFound, dirsFound, sizeFound *int64, seen *sync.Map) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if depth >= maxDepth {
		// Full recursive walk for accurate sizing (with cancellation support)
		entry.Size = dirSizeRecursive(ctx, path, seen)
		atomic.AddInt64(sizeFound, entry.Size)
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	atomic.AddInt64(dirsFound, 1)
	result.TotalDirs++

	var children []*FileEntry
	var totalSize int64

	// Use bounded parallelism for subdirectories at this level
	sem := make(chan struct{}, 16) // max 16 goroutines
	var wg sync.WaitGroup
	childSlice := make([]*FileEntry, 0, len(entries))
	var childMu sync.Mutex

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return
		default:
		}

		name := e.Name()
		fullPath := filepath.Join(path, name)
		info, err := e.Info()
		if err != nil {
			continue
		}

		child := &FileEntry{
			Name:    name,
			Path:    fullPath,
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Unix(),
		}

		if e.IsDir() {
			wg.Add(1)
			sem <- struct{}{}
			go func(c *FileEntry, fp string) {
				defer wg.Done()
				defer func() { <-sem }()
				a.scanDirAsync(ctx, c, fp, depth+1, maxDepth, result, requestId, filesFound, dirsFound, sizeFound, seen)
			}(child, fullPath)
		} else {
			if id, ok := inodeFromInfo(info); !ok || !seenAdd(seen, id) {
				child.Size = info.Size()
				atomic.AddInt64(filesFound, 1)
				atomic.AddInt64(sizeFound, child.Size)
				result.TotalFiles++
			}
		}

		childMu.Lock()
		childSlice = append(childSlice, child)
		if depth == 0 {
			entry.Children = childSlice
		}
		childMu.Unlock()
	}

	// Emit progress (throttled to every 100ms)
	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&a.lastProgressEmit)
	if now-last > 100_000_000 { // 100ms
		atomic.StoreInt64(&a.lastProgressEmit, now)
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "analyze:progress", AnalyzeProgress{
				RequestId:   requestId,
				CurrentFile: path,
				FilesFound:  int(atomic.LoadInt64(filesFound)),
				DirsFound:   int(atomic.LoadInt64(dirsFound)),
				SizeFound:   atomic.LoadInt64(sizeFound),
				Percentage:  0,
			})
		}
	}

	wg.Wait()

	// Emit progress after wait completes
	if a.ctx != nil {
		now = time.Now().UnixNano()
		last = atomic.LoadInt64(&a.lastProgressEmit)
		if now-last > 100_000_000 {
			atomic.StoreInt64(&a.lastProgressEmit, now)
			wailsRuntime.EventsEmit(a.ctx, "analyze:progress", AnalyzeProgress{
				RequestId:   requestId,
				CurrentFile: path,
				FilesFound:  int(atomic.LoadInt64(filesFound)),
				DirsFound:   int(atomic.LoadInt64(dirsFound)),
				SizeFound:   atomic.LoadInt64(sizeFound),
				Percentage:  0,
			})
		}
	}

	children = childSlice
	for _, c := range children {
		totalSize += c.Size
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	entry.Children = children
	entry.Size = totalSize
}

// CancelScan cancels any in-flight async scan.
func (a *AnalyzeService) CancelScan() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.scanCancel != nil {
		a.scanCancel()
		a.scanCancel = nil
	}
}

// ---------------------------------------------------------------------------
// StartDrill — fire-and-forget drill into a subdirectory
// ---------------------------------------------------------------------------

func (a *AnalyzeService) StartDrill(path string, requestId int) {
	// Cancel any previous drill
	if a.drillCancel != nil {
		a.drillCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.drillCancel = cancel

	go func() {
		defer cancel()
		items := a.drillInto(ctx, path)

		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "analyze:drill-complete", map[string]interface{}{
				"requestId": requestId,
				"items":     items,
			})
		}
	}()
}

// CancelDrill cancels any in-flight drill operation.
func (a *AnalyzeService) CancelDrill() {
	if a.drillCancel != nil {
		a.drillCancel()
		a.drillCancel = nil
	}
}

func (a *AnalyzeService) drillInto(ctx context.Context, path string) []*FileEntry {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var seen sync.Map
	const largeThreshold = 200

	if len(entries) > largeThreshold {
		// Shallow sizing for large directories
		items := make([]*FileEntry, 0, len(entries))
		for _, e := range entries {
			select {
			case <-ctx.Done():
				return items
			default:
			}

			name := e.Name()
			fullPath := filepath.Join(path, name)
			eInfo, err := e.Info()
			if err != nil {
				continue
			}

			child := &FileEntry{
				Name:    name,
				Path:    fullPath,
				IsDir:   e.IsDir(),
				ModTime: eInfo.ModTime().Unix(),
			}

			if e.IsDir() {
				child.Size = dirSizeRecursive(ctx, fullPath, &seen)
			} else {
				child.Size = eInfo.Size()
			}

			items = append(items, child)
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i].Size > items[j].Size
		})
		return items
	}

	// Concurrent sizing for smaller directories
	items := make([]*FileEntry, 0, len(entries))
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return items
		default:
		}

		name := e.Name()
		fullPath := filepath.Join(path, name)
		eInfo, err := e.Info()
		if err != nil {
			continue
		}

		child := &FileEntry{
			Name:    name,
			Path:    fullPath,
			IsDir:   e.IsDir(),
			ModTime: eInfo.ModTime().Unix(),
		}

		if e.IsDir() {
			wg.Add(1)
			sem <- struct{}{}
			go func(c *FileEntry, fp string) {
				defer wg.Done()
				defer func() { <-sem }()
				c.Size = dirSizeCtx(ctx, fp, &seen)
			}(child, fullPath)
		} else {
			child.Size = eInfo.Size()
		}

		mu.Lock()
		items = append(items, child)
		mu.Unlock()
	}

	wg.Wait()

	sort.Slice(items, func(i, j int) bool {
		return items[i].Size > items[j].Size
	})

	return items
}

// ---------------------------------------------------------------------------
// PreFetchChildren — background pre-fetch of subdirectory children
// ---------------------------------------------------------------------------

func (a *AnalyzeService) PreFetchChildren(paths []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	sem := make(chan struct{}, 8) // max 8 concurrent pre-fetches
	var wg sync.WaitGroup

	go func() {
		defer cancel()

		for _, p := range paths {
			select {
			case <-ctx.Done():
				return
			default:
			}

			wg.Add(1)
			sem <- struct{}{}
			go func(dirPath string) {
				defer wg.Done()
				defer func() { <-sem }()

				children := a.fetchChildrenWithSizes(ctx, dirPath)
				if children != nil {
					a.prefetchCache.Store(dirPath, children)
					if a.ctx != nil {
						wailsRuntime.EventsEmit(a.ctx, "analyze:prefetch-ready", dirPath)
					}
				}
			}(p)
		}

		wg.Wait()
	}()
}

func (a *AnalyzeService) fetchChildrenWithSizes(ctx context.Context, path string) []*FileEntry {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	children := make([]*FileEntry, 0, len(entries))
	sem := make(chan struct{}, 16)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var seen sync.Map

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		name := e.Name()
		fullPath := filepath.Join(path, name)
		info, err := e.Info()
		if err != nil {
			continue
		}

		child := &FileEntry{
			Name:    name,
			Path:    fullPath,
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Unix(),
		}

		if e.IsDir() {
			wg.Add(1)
			sem <- struct{}{}
			go func(c *FileEntry, fp string) {
				defer wg.Done()
				defer func() { <-sem }()
				c.Size = dirSizeCtx(ctx, fp, &seen)
			}(child, fullPath)
		} else {
			child.Size = info.Size()
		}

		mu.Lock()
		children = append(children, child)
		mu.Unlock()
	}

	wg.Wait()

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	return children
}

// GetCachedChildren returns pre-fetched children from the cache if available.
func (a *AnalyzeService) GetCachedChildren(path string) ([]*FileEntry, bool) {
	val, ok := a.prefetchCache.Load(path)
	if !ok {
		return nil, false
	}
	entries, ok := val.([]*FileEntry)
	return entries, ok
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func dirSizeCtx(ctx context.Context, path string, seen *sync.Map) int64 {
	return dirSizeRecursive(ctx, path, seen)
}

func dirSizeRecursive(ctx context.Context, path string, seen *sync.Map) int64 {
	select {
	case <-ctx.Done():
		return 0
	default:
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}

	var size int64
	for _, e := range entries {
		fp := filepath.Join(path, e.Name())
		if e.IsDir() {
			size += dirSizeRecursive(ctx, fp, seen)
		} else {
			info, err := e.Info()
			if err == nil {
				if id, ok := inodeFromInfo(info); !ok || !seenAdd(seen, id) {
					size += info.Size()
				}
			}
		}
	}
	return size
}

func dirSize(path string) int64 {
	var seen sync.Map
	return dirSizeRecursive(context.Background(), path, &seen)
}

func shallowDirSizeDeep(ctx context.Context, path string, levels int, seen *sync.Map) int64 {
	if levels <= 0 {
		return 0
	}
	select {
	case <-ctx.Done():
		return 0
	default:
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	var size int64
	for _, e := range entries {
		fp := filepath.Join(path, e.Name())
		if e.IsDir() {
			size += shallowDirSizeDeep(ctx, fp, levels-1, seen)
		} else {
			info, err := e.Info()
			if err == nil {
				if id, ok := inodeFromInfo(info); !ok || !seenAdd(seen, id) {
					size += info.Size()
				}
			}
		}
	}
	return size
}
