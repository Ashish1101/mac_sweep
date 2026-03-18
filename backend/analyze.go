package backend

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

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
	ctx           context.Context
	mu            sync.Mutex
	cancelScan    chan struct{}
	isScanning    bool
	scanCancel    context.CancelFunc
	drillCancel   context.CancelFunc
	prefetchCache sync.Map // key: string (path), value: []*FileEntry
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

			a.scanDirAsync(scanCtx, root, path, 0, maxDepth, result, requestId, &filesFound, &dirsFound, &sizeFound)
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

func (a *AnalyzeService) scanDirAsync(ctx context.Context, entry *FileEntry, path string, depth, maxDepth int, result *ScanResult, requestId int, filesFound, dirsFound, sizeFound *int64) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	if depth >= maxDepth {
		// Use bounded parallelism for sizing at leaf depth
		size := dirSizeCtx(ctx, path)
		entry.Size = size
		atomic.AddInt64(sizeFound, size)
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
	type indexedChild struct {
		index int
		child *FileEntry
	}

	sem := make(chan struct{}, 8) // max 8 goroutines
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
		if strings.HasPrefix(name, ".") {
			continue
		}

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
				a.scanDirAsync(ctx, c, fp, depth+1, maxDepth, result, requestId, filesFound, dirsFound, sizeFound)
			}(child, fullPath)
		} else {
			child.Size = info.Size()
			atomic.AddInt64(filesFound, 1)
			atomic.AddInt64(sizeFound, child.Size)
			result.TotalFiles++
		}

		childMu.Lock()
		childSlice = append(childSlice, child)
		childMu.Unlock()
	}

	wg.Wait()

	children = childSlice
	for _, c := range children {
		totalSize += c.Size
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	entry.Children = children
	entry.Size = totalSize
	result.TotalSize += totalSize

	// Emit progress
	if a.ctx != nil && depth <= 1 {
		wailsRuntime.EventsEmit(a.ctx, "analyze:progress", AnalyzeProgress{
			RequestId:   requestId,
			CurrentFile: path,
			FilesFound:  int(atomic.LoadInt64(filesFound)),
			DirsFound:   int(atomic.LoadInt64(dirsFound)),
			SizeFound:   atomic.LoadInt64(sizeFound),
			Percentage:  0, // percentage is hard to estimate without a pre-pass
		})
	}
}

// CancelScan cancels any in-flight async scan.
func (a *AnalyzeService) CancelScan() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.scanCancel != nil {
		a.scanCancel()
		a.scanCancel = nil
	}
	// Also cancel legacy channel-based scan
	if a.isScanning && a.cancelScan != nil {
		close(a.cancelScan)
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
			if strings.HasPrefix(name, ".") {
				continue
			}

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
				// Shallow: just estimate from immediate children
				child.Size = shallowDirSize(fullPath)
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
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return items
		default:
		}

		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

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
				c.Size = dirSizeCtx(ctx, fp)
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

	sem := make(chan struct{}, 4) // max 4 concurrent pre-fetches
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
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

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
				c.Size = dirSizeCtx(ctx, fp)
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
// Legacy synchronous methods (kept for backward compatibility)
// ---------------------------------------------------------------------------

func (a *AnalyzeService) ScanDirectory(path string, maxDepth int) (*ScanResult, error) {
	a.mu.Lock()
	if a.isScanning {
		a.mu.Unlock()
		return nil, nil
	}
	a.isScanning = true
	a.cancelScan = make(chan struct{})
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isScanning = false
		a.mu.Unlock()
	}()

	if maxDepth <= 0 {
		maxDepth = 3
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	root := &FileEntry{
		Name:  info.Name(),
		Path:  path,
		IsDir: info.IsDir(),
	}

	result := &ScanResult{Root: root}

	if info.IsDir() {
		a.scanDir(root, path, 0, maxDepth, result)
	} else {
		root.Size = info.Size()
		root.ModTime = info.ModTime().Unix()
		result.TotalSize = root.Size
		result.TotalFiles = 1
	}

	return result, nil
}

func (a *AnalyzeService) scanDir(entry *FileEntry, path string, depth, maxDepth int, result *ScanResult) {
	select {
	case <-a.cancelScan:
		return
	default:
	}

	if depth >= maxDepth {
		size := dirSize(path)
		entry.Size = size
		return
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return
	}

	result.TotalDirs++
	var children []*FileEntry
	var totalSize int64

	for _, e := range entries {
		select {
		case <-a.cancelScan:
			return
		default:
		}

		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

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
			a.scanDir(child, fullPath, depth+1, maxDepth, result)
		} else {
			child.Size = info.Size()
			result.TotalFiles++
		}

		totalSize += child.Size
		children = append(children, child)
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	entry.Children = children
	entry.Size = totalSize
	result.TotalSize += totalSize
}

func (a *AnalyzeService) GetDirectoryChildren(path string) ([]*FileEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var children []*FileEntry
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}

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
			child.Size = dirSize(fullPath)
		} else {
			child.Size = info.Size()
		}

		children = append(children, child)
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	return children, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func dirSize(path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

func dirSizeCtx(ctx context.Context, path string) int64 {
	var size int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return filepath.SkipAll
		default:
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size
}

