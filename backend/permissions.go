package backend

import (
	"os"
	"os/exec"
	"path/filepath"
)

type FDAStatus struct {
	HasFullDiskAccess bool `json:"hasFullDiskAccess"`
}

type PermissionsService struct{}

func NewPermissionsService() *PermissionsService {
	return &PermissionsService{}
}

// CheckFullDiskAccess probes known macOS privacy-protected directories
// to determine if Full Disk Access has been granted.
func (p *PermissionsService) CheckFullDiskAccess() FDAStatus {
	home, err := os.UserHomeDir()
	if err != nil {
		return FDAStatus{HasFullDiskAccess: false}
	}

	protectedPaths := []string{
		filepath.Join(home, "Library", "Safari"),
		filepath.Join(home, "Library", "Mail"),
		filepath.Join(home, "Library", "Containers", "com.apple.Safari"),
		filepath.Join(home, "Library", "Application Support", "com.apple.TCC"),
	}

	for _, path := range protectedPaths {
		_, err := os.ReadDir(path)
		if err == nil {
			return FDAStatus{HasFullDiskAccess: true}
		}
		if os.IsNotExist(err) {
			continue
		}
		// Permission denied — FDA not granted
	}

	return FDAStatus{HasFullDiskAccess: false}
}

// OpenFullDiskAccessSettings opens System Settings to the Full Disk Access pane.
func (p *PermissionsService) OpenFullDiskAccessSettings() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles").Run()
}
