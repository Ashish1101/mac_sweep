package backend

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type CleanCategory struct {
	Name     string      `json:"name"`
	Risk     string      `json:"risk"`
	Items    []CleanItem `json:"items"`
	Size     int64       `json:"size"`
	Selected bool        `json:"selected"`
}

type CleanItem struct {
	Path     string      `json:"path"`
	Size     int64       `json:"size"`
	Name     string      `json:"name"`
	IsDir    bool        `json:"isDir"`
	Children []CleanItem `json:"children,omitempty"`
}

type CleanResult struct {
	Categories []CleanCategory `json:"categories"`
	TotalSize  int64           `json:"totalSize"`
	TotalFiles int             `json:"totalFiles"`
}

type ScanProgress struct {
	Phase       string `json:"phase"`
	Category    string `json:"category"`
	CurrentFile string `json:"currentFile"`
	FilesFound  int    `json:"filesFound"`
	SizeFound   int64  `json:"sizeFound"`
	Percentage  int    `json:"percentage"`
}

type CleanService struct {
	safety     *SafetyService
	ctx        context.Context
	drillCancel context.CancelFunc
}

func NewCleanService(safety *SafetyService) *CleanService {
	return &CleanService{safety: safety}
}

func (c *CleanService) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *CleanService) emitProgress(progress ScanProgress) {
	if c.ctx != nil {
		wailsRuntime.EventsEmit(c.ctx, "clean:progress", progress)
	}
}

func (c *CleanService) DryRun() (*CleanResult, error) {
	homeDir, _ := os.UserHomeDir()
	result := &CleanResult{}

	categories := []struct {
		name  string
		risk  string
		paths []string
	}{
		{
			name: "Browser Caches",
			risk: "low",
			paths: []string{
				filepath.Join(homeDir, "Library/Caches/Google/Chrome"),
				filepath.Join(homeDir, "Library/Caches/com.apple.Safari"),
				filepath.Join(homeDir, "Library/Caches/Firefox"),
				filepath.Join(homeDir, "Library/Caches/com.microsoft.edgemac"),
				filepath.Join(homeDir, "Library/Caches/com.brave.Browser"),
			},
		},
		{
			name: "Application Caches",
			risk: "low",
			paths: []string{
				filepath.Join(homeDir, "Library/Caches/com.spotify.client"),
				filepath.Join(homeDir, "Library/Caches/com.docker.docker"),
				filepath.Join(homeDir, "Library/Caches/com.apple.dt.Xcode"),
				filepath.Join(homeDir, "Library/Caches/com.microsoft.VSCode"),
				filepath.Join(homeDir, "Library/Caches/Slack"),
			},
		},
		{
			name: "System Logs",
			risk: "medium",
			paths: []string{
				filepath.Join(homeDir, "Library/Logs"),
				"/private/var/log",
			},
		},
		{
			name: "Temporary Files",
			risk: "low",
			paths: []string{
				os.TempDir(),
				"/private/var/folders",
			},
		},
		{
			name: "Developer Caches",
			risk: "low",
			paths: []string{
				filepath.Join(homeDir, "Library/Caches/CocoaPods"),
				filepath.Join(homeDir, "Library/Caches/pip"),
				filepath.Join(homeDir, "Library/Caches/Homebrew"),
				filepath.Join(homeDir, ".npm/_cacache"),
				filepath.Join(homeDir, ".gradle/caches"),
			},
		},
		{
			name: "Download History & Crash Reports",
			risk: "medium",
			paths: []string{
				filepath.Join(homeDir, "Library/Application Support/CrashReporter"),
				"/Library/Logs/DiagnosticReports",
				filepath.Join(homeDir, "Library/Logs/DiagnosticReports"),
			},
		},
	}

	totalCategories := len(categories)
	globalFilesFound := 0
	var globalSizeFound int64

	for catIdx, cat := range categories {
		category := CleanCategory{
			Name:     cat.name,
			Risk:     cat.risk,
			Selected: cat.risk != "high",
		}

		c.emitProgress(ScanProgress{
			Phase:      "scanning",
			Category:   cat.name,
			Percentage: (catIdx * 100) / totalCategories,
			FilesFound: globalFilesFound,
			SizeFound:  globalSizeFound,
		})

		for _, p := range cat.paths {
			info, err := os.Stat(p)
			if err != nil {
				continue
			}

			var size int64
			var children []CleanItem

			if info.IsDir() {
				size, children = c.scanDirWithProgress(p, cat.name, homeDir, &globalFilesFound, &globalSizeFound, catIdx, totalCategories)
			} else {
				size = info.Size()
				globalFilesFound++
				globalSizeFound += size
			}

			if size > 0 {
				name := filepath.Base(p)
				if strings.HasPrefix(p, homeDir) {
					name = "~" + strings.TrimPrefix(p, homeDir)
				}

				item := CleanItem{
					Path:     p,
					Size:     size,
					Name:     name,
					IsDir:    info.IsDir(),
					Children: children,
				}

				category.Items = append(category.Items, item)
				category.Size += size
				result.TotalFiles++
			}
		}

		// Sort items within category by size descending
		sort.Slice(category.Items, func(i, j int) bool {
			return category.Items[i].Size > category.Items[j].Size
		})

		if len(category.Items) > 0 {
			result.TotalSize += category.Size
			result.Categories = append(result.Categories, category)
		}
	}

	c.emitProgress(ScanProgress{
		Phase:      "complete",
		Percentage: 100,
		FilesFound: globalFilesFound,
		SizeFound:  globalSizeFound,
	})

	return result, nil
}

// scanDirWithProgress scans a directory one level deep, collecting immediate children
// with their total sizes, while emitting progress events for each file encountered.
func (c *CleanService) scanDirWithProgress(dirPath, categoryName, homeDir string, filesFound *int, sizeFound *int64, catIdx, totalCats int) (int64, []CleanItem) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, nil
	}

	var totalSize int64
	var children []CleanItem

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dirPath, name)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		var childSize int64
		var grandchildren []CleanItem

		if entry.IsDir() {
			childSize, grandchildren = c.scanSubDirWithProgress(fullPath, categoryName, homeDir, filesFound, sizeFound, catIdx, totalCats)
		} else {
			childSize = info.Size()
			*filesFound++
			*sizeFound += childSize

			c.emitProgress(ScanProgress{
				Phase:       "scanning",
				Category:    categoryName,
				CurrentFile: shortenPath(fullPath, homeDir),
				FilesFound:  *filesFound,
				SizeFound:   *sizeFound,
				Percentage:  (catIdx * 100) / totalCats,
			})
		}

		if childSize > 0 {
			displayName := name
			if strings.HasPrefix(fullPath, homeDir) {
				displayName = "~" + strings.TrimPrefix(fullPath, homeDir)
			}
			children = append(children, CleanItem{
				Path:     fullPath,
				Size:     childSize,
				Name:     displayName,
				IsDir:    entry.IsDir(),
				Children: grandchildren,
			})
			totalSize += childSize
		}
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	return totalSize, children
}

// scanSubDirWithProgress scans deeper directories, collecting children down to 2 extra levels.
func (c *CleanService) scanSubDirWithProgress(dirPath, categoryName, homeDir string, filesFound *int, sizeFound *int64, catIdx, totalCats int) (int64, []CleanItem) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, nil
	}

	var totalSize int64
	var children []CleanItem

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dirPath, name)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		var childSize int64

		if entry.IsDir() {
			childSize = dirSize(fullPath)
		} else {
			childSize = info.Size()
		}

		*filesFound++
		*sizeFound += childSize

		// Emit progress every 20 files to avoid flooding events
		if *filesFound%20 == 0 {
			c.emitProgress(ScanProgress{
				Phase:       "scanning",
				Category:    categoryName,
				CurrentFile: shortenPath(fullPath, homeDir),
				FilesFound:  *filesFound,
				SizeFound:   *sizeFound,
				Percentage:  (catIdx * 100) / totalCats,
			})
		}

		if childSize > 0 {
			children = append(children, CleanItem{
				Path:  fullPath,
				Size:  childSize,
				Name:  name,
				IsDir: entry.IsDir(),
			})
			totalSize += childSize
		}
	}

	sort.Slice(children, func(i, j int) bool {
		return children[i].Size > children[j].Size
	})

	// Keep top 50 children to avoid huge payloads
	if len(children) > 50 {
		children = children[:50]
	}

	return totalSize, children
}

// StartDrill kicks off an async directory scan in a goroutine.
// Results are sent via "drill:complete" event. Never blocks the UI.
// requestId lets the frontend ignore stale results.
func (c *CleanService) StartDrill(path string, requestId int) {
	// Cancel any previous drill
	if c.drillCancel != nil {
		c.drillCancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	c.drillCancel = cancel

	go func() {
		defer cancel()
		items := c.drillAsync(path, ctx)

		// Emit result — frontend checks requestId to ignore stale results
		if c.ctx != nil {
			wailsRuntime.EventsEmit(c.ctx, "drill:complete", map[string]interface{}{
				"requestId": requestId,
				"items":     items,
			})
		}
	}()
}

// CancelDrill cancels any in-flight drill operation.
func (c *CleanService) CancelDrill() {
	if c.drillCancel != nil {
		c.drillCancel()
		c.drillCancel = nil
	}
}

func (c *CleanService) drillAsync(path string, ctx context.Context) []CleanItem {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil
	}

	homeDir, _ := os.UserHomeDir()
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	// Cap entries to avoid massive payloads
	const maxEntries = 200
	if len(entries) > maxEntries {
		// Quick partial: just list entries with file sizes, skip deep dir sizing
		items := make([]CleanItem, 0, maxEntries)
		for _, entry := range entries {
			if len(items) >= maxEntries {
				break
			}
			select {
			case <-ctx.Done():
				return items
			default:
			}

			name := entry.Name()
			fullPath := filepath.Join(path, name)
			eInfo, err := entry.Info()
			if err != nil {
				continue
			}
			displayName := name
			if strings.HasPrefix(fullPath, homeDir) {
				displayName = "~" + strings.TrimPrefix(fullPath, homeDir)
			}
			var sz int64
			if !entry.IsDir() {
				sz = eInfo.Size()
			} else {
				var seen sync.Map
				sz = dirSizeRecursive(ctx, fullPath, &seen)
			}
			items = append(items, CleanItem{
				Path:  fullPath,
				Name:  displayName,
				IsDir: entry.IsDir(),
				Size:  sz,
			})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].Size > items[j].Size })
		return items
	}

	// Small-to-medium directory: size all children concurrently
	type sizeResult struct {
		index int
		size  int64
	}

	items := make([]CleanItem, 0, len(entries))
	var pendingIndices []int

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			return items
		default:
		}

		name := entry.Name()
		fullPath := filepath.Join(path, name)
		eInfo, err := entry.Info()
		if err != nil {
			continue
		}

		displayName := name
		if strings.HasPrefix(fullPath, homeDir) {
			displayName = "~" + strings.TrimPrefix(fullPath, homeDir)
		}

		item := CleanItem{Path: fullPath, Name: displayName, IsDir: entry.IsDir()}
		if !entry.IsDir() {
			item.Size = eInfo.Size()
		} else {
			pendingIndices = append(pendingIndices, len(items))
		}
		items = append(items, item)
	}

	// Concurrent dir sizing with bounded parallelism
	if len(pendingIndices) > 0 {
		ch := make(chan sizeResult, len(pendingIndices))
		sem := make(chan struct{}, 8) // max 8 concurrent walkers

		for _, idx := range pendingIndices {
			sem <- struct{}{}
			go func(i int, p string) {
				defer func() { <-sem }()
				var sz int64
				filepath.Walk(p, func(_ string, fi os.FileInfo, err error) error {
					if err != nil {
						return nil
					}
					select {
					case <-ctx.Done():
						return filepath.SkipAll
					default:
					}
					if !fi.IsDir() {
						sz += fi.Size()
					}
					return nil
				})
				ch <- sizeResult{index: i, size: sz}
			}(idx, items[idx].Path)
		}

		for received := 0; received < len(pendingIndices); received++ {
			select {
			case res := <-ch:
				items[res.index].Size = res.size
			case <-ctx.Done():
				goto done
			}
		}
	done:
	}

	// Filter and sort
	filtered := items[:0]
	for _, item := range items {
		if item.Size > 0 || item.IsDir {
			filtered = append(filtered, item)
		}
	}
	sort.Slice(filtered, func(i, j int) bool { return filtered[i].Size > filtered[j].Size })

	// Cap output
	if len(filtered) > 200 {
		filtered = filtered[:200]
	}
	return filtered
}

// shallowDirSize gets a fast size estimate by only looking at immediate children (no recursion).
func shallowDirSize(path string) int64 {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	var total int64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if !info.IsDir() {
			total += info.Size()
		}
	}
	return total
}

type CleanExecResult struct {
	Freed        int64    `json:"freed"`
	DeletedCount int      `json:"deletedCount"`
	DeletedPaths []string `json:"deletedPaths"`
	FailedPaths  []string `json:"failedPaths"`
}

func (c *CleanService) ExecuteClean(paths []string) (*CleanExecResult, error) {
	res := &CleanExecResult{}

	for i, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			res.FailedPaths = append(res.FailedPaths, path)
			continue
		}

		// Use file size directly; skip expensive recursive dir sizing
		// (frontend already tracks sizes from scan data)
		var size int64
		if !info.IsDir() {
			size = info.Size()
		}

		if c.ctx != nil {
			wailsRuntime.EventsEmit(c.ctx, "clean:delete-progress", map[string]interface{}{
				"current": i + 1,
				"total":   len(paths),
				"path":    path,
			})
		}

		if err := c.safety.MoveToTrash(path); err == nil {
			res.Freed += size
			res.DeletedCount++
			res.DeletedPaths = append(res.DeletedPaths, path)
		} else {
			res.FailedPaths = append(res.FailedPaths, path)
		}
	}

	return res, nil
}

func shortenPath(path, homeDir string) string {
	if strings.HasPrefix(path, homeDir) {
		return "~" + strings.TrimPrefix(path, homeDir)
	}
	// Truncate long paths
	if len(path) > 60 {
		return "..." + path[len(path)-57:]
	}
	return path
}
