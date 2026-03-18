package backend

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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

type AnalyzeService struct {
	mu          sync.Mutex
	cancelScan  chan struct{}
	isScanning  bool
}

func NewAnalyzeService() *AnalyzeService {
	return &AnalyzeService{}
}

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

func (a *AnalyzeService) CancelScan() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.isScanning && a.cancelScan != nil {
		close(a.cancelScan)
	}
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
