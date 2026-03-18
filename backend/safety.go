package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type OperationLog struct {
	Timestamp string `json:"timestamp"`
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Size      int64  `json:"size"`
	Status    string `json:"status"`
}

type SafetyService struct {
	logDir string
}

func NewSafetyService() *SafetyService {
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".config", "mole-gui")
	os.MkdirAll(logDir, 0755)
	return &SafetyService{logDir: logDir}
}

func (s *SafetyService) MoveToTrash(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	size := info.Size()
	if info.IsDir() {
		size = dirSize(path)
	}

	// Use macOS Finder to move to Trash (recoverable)
	script := fmt.Sprintf(`tell application "Finder" to delete POSIX file "%s"`, path)
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to move to trash: %w", err)
	}

	s.logOperation("trash", path, size, "success")
	return nil
}

func (s *SafetyService) MoveMultipleToTrash(paths []string) ([]string, []string) {
	var succeeded, failed []string
	for _, path := range paths {
		if err := s.MoveToTrash(path); err != nil {
			failed = append(failed, path)
		} else {
			succeeded = append(succeeded, path)
		}
	}
	return succeeded, failed
}

func (s *SafetyService) GetOperationHistory(limit int) ([]OperationLog, error) {
	if limit <= 0 {
		limit = 50
	}

	logFile := filepath.Join(s.logDir, "audit.log")
	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []OperationLog{}, nil
		}
		return nil, err
	}

	var logs []OperationLog
	lines := splitLines(string(data))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var log OperationLog
		if err := json.Unmarshal([]byte(line), &log); err == nil {
			logs = append(logs, log)
		}
	}

	// Return most recent first
	if len(logs) > limit {
		logs = logs[len(logs)-limit:]
	}

	// Reverse
	for i, j := 0, len(logs)-1; i < j; i, j = i+1, j-1 {
		logs[i], logs[j] = logs[j], logs[i]
	}

	return logs, nil
}

func (s *SafetyService) PlayTrashSound() {
	exec.Command("afplay", "/System/Library/Components/CoreAudio.component/Contents/SharedSupport/SystemSounds/dock/drag to trash.aif").Start()
}

func (s *SafetyService) logOperation(op, path string, size int64, status string) {
	entry := OperationLog{
		Timestamp: time.Now().Format(time.RFC3339),
		Operation: op,
		Path:      path,
		Size:      size,
		Status:    status,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	logFile := filepath.Join(s.logDir, "audit.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.Write(data)
	f.WriteString("\n")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
