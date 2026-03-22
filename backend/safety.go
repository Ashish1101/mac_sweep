package backend

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	logDir := filepath.Join(homeDir, ".config", "macsweep")
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

	// Try silent move to ~/.Trash (works with FDA, no Finder sound)
	homeDir, _ := os.UserHomeDir()
	trashDir := filepath.Join(homeDir, ".Trash")
	trashDest := filepath.Join(trashDir, filepath.Base(path))

	// Handle name collision in Trash
	if _, err := os.Stat(trashDest); err == nil {
		trashDest = filepath.Join(trashDir, fmt.Sprintf("%s_%d_%s", filepath.Base(path), time.Now().UnixNano(), ""))
	}

	if renameErr := os.Rename(path, trashDest); renameErr == nil {
		s.logOperation("trash", path, size, "success")
		return nil
	}

	// Fallback: use Finder (plays system sound)
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

func (s *SafetyService) RestoreFromTrash(originalPath string) error {
	fileName := filepath.Base(originalPath)
	parentDir := filepath.Dir(originalPath)
	homeDir, _ := os.UserHomeDir()
	trashPath := filepath.Join(homeDir, ".Trash", fileName)

	// Ensure parent directory exists
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("cannot create parent directory: %w", err)
	}

	// Check if something already exists at the original path
	if _, err := os.Stat(originalPath); err == nil {
		return fmt.Errorf("file already exists at original location: %s", originalPath)
	}

	// Try direct rename first (works if app has FDA)
	renameErr := os.Rename(trashPath, originalPath)
	if renameErr == nil {
		s.logRestoreSuccess(originalPath)
		return nil
	}

	// Fallback: use osascript with shell mv (inherits app's FDA context)
	script := fmt.Sprintf(`do shell script "mv %q %q"`, trashPath, originalPath)
	cmd := exec.Command("osascript", "-e", script)
	if shellErr := cmd.Run(); shellErr == nil {
		s.logRestoreSuccess(originalPath)
		return nil
	}

	// Fallback: use Finder to move (works for non-hidden destinations)
	finderScript := fmt.Sprintf(`
tell application "Finder"
	set trashItems to items of trash
	repeat with anItem in trashItems
		if name of anItem is %q then
			move anItem to (POSIX file %q as alias)
			return "ok"
		end if
	end repeat
	return "not found"
end tell`, fileName, parentDir)
	cmd2 := exec.Command("osascript", "-e", finderScript)
	out, finderErr := cmd2.Output()
	if finderErr == nil && strings.TrimSpace(string(out)) == "ok" {
		s.logRestoreSuccess(originalPath)
		return nil
	}

	return fmt.Errorf("restore failed for '%s' — ensure Full Disk Access is granted to MacSweep in System Settings > Privacy > Full Disk Access (rename: %v)", fileName, renameErr)
}

func (s *SafetyService) logRestoreSuccess(originalPath string) {
	info, _ := os.Stat(originalPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
		if info.IsDir() {
			size = dirSize(originalPath)
		}
	}
	s.logOperation("restore", originalPath, size, "success")
}

type RestoreResult struct {
	Succeeded []string `json:"succeeded"`
	Failed    []string `json:"failed"`
	Errors    []string `json:"errors"`
}

func (s *SafetyService) RestoreAllFromTrash() RestoreResult {
	result := RestoreResult{}

	logs, err := s.GetOperationHistory(0)
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result
	}

	// Collect unique trash operations (most recent first, skip already restored)
	restored := make(map[string]bool)
	for _, log := range logs {
		if log.Operation == "restore" {
			restored[log.Path] = true
		}
	}

	for _, log := range logs {
		if log.Operation != "trash" || log.Status != "success" {
			continue
		}
		if restored[log.Path] {
			continue
		}

		if err := s.RestoreFromTrash(log.Path); err != nil {
			result.Failed = append(result.Failed, log.Path)
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.Succeeded = append(result.Succeeded, log.Path)
		}
		restored[log.Path] = true
	}

	return result
}

// GetTrashItems returns the list of filenames currently in the Trash.
// Tries direct file listing first (works with FDA), falls back to Finder AppleScript.
func (s *SafetyService) GetTrashItems() []string {
	homeDir, _ := os.UserHomeDir()
	trashDir := filepath.Join(homeDir, ".Trash")

	// Try direct access (works when app has FDA)
	entries, err := os.ReadDir(trashDir)
	if err == nil {
		var names []string
		for _, e := range entries {
			if !strings.HasPrefix(e.Name(), ".") {
				names = append(names, e.Name())
			}
		}
		return names
	}

	// Fallback to Finder AppleScript (can list but can't move without FDA)
	script := `tell application "Finder"
	set trashItems to items of trash
	set nameList to {}
	repeat with anItem in trashItems
		set end of nameList to name of anItem
	end repeat
	return nameList
end tell`
	cmd := exec.Command("osascript", "-e", script)
	out, errAS := cmd.Output()
	if errAS != nil {
		return []string{}
	}
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, ", ")
}

// CanAccessTrash checks if the app has direct access to ~/.Trash (requires FDA).
func (s *SafetyService) CanAccessTrash() bool {
	homeDir, _ := os.UserHomeDir()
	_, err := os.ReadDir(filepath.Join(homeDir, ".Trash"))
	return err == nil
}

func (s *SafetyService) EmptyTrash() error {
	script := `tell application "Finder" to empty trash`
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to empty trash: %w", err)
	}
	return nil
}

func (s *SafetyService) PlayTrashSound() {
	soundPath := "/System/Library/Components/CoreAudio.component/Contents/SharedSupport/SystemSounds/dock/drag to trash.aif"
	if _, err := os.Stat(soundPath); err != nil {
		soundPath = "/System/Library/Sounds/Funk.aiff"
	}
	exec.Command("afplay", soundPath).Start()
}

func (s *SafetyService) PlaySound(soundID string) {
	trashSound := "/System/Library/Components/CoreAudio.component/Contents/SharedSupport/SystemSounds/dock/drag to trash.aif"
	if _, err := os.Stat(trashSound); err != nil {
		trashSound = "/System/Library/Sounds/Funk.aiff"
	}

	switch soundID {
	case "default":
		exec.Command("afplay", trashSound).Start()
	case "faaah":
		// Played from frontend via HTML5 Audio
	case "funk":
		exec.Command("afplay", "/System/Library/Sounds/Funk.aiff").Start()
	case "glass":
		exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Start()
	case "pop":
		exec.Command("afplay", "/System/Library/Sounds/Pop.aiff").Start()
	case "basso":
		exec.Command("afplay", "/System/Library/Sounds/Basso.aiff").Start()
	case "hero":
		exec.Command("afplay", "/System/Library/Sounds/Hero.aiff").Start()
	case "sosumi":
		exec.Command("afplay", "/System/Library/Sounds/Sosumi.aiff").Start()
	case "none":
		// silent
	default:
		exec.Command("afplay", trashSound).Start()
	}
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
