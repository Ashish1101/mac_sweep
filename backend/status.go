package backend

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type SystemStatus struct {
	CPU     CPUInfo     `json:"cpu"`
	Memory  MemoryInfo  `json:"memory"`
	Disk    DiskInfo    `json:"disk"`
	Battery BatteryInfo `json:"battery"`
	Uptime  string      `json:"uptime"`
	Health  int         `json:"health"`
}

type CPUInfo struct {
	Usage    float64 `json:"usage"`
	Cores    int     `json:"cores"`
	Model    string  `json:"model"`
}

type MemoryInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
}

type DiskInfo struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
	MountPath string  `json:"mountPath"`
}

type BatteryInfo struct {
	Percentage int    `json:"percentage"`
	IsCharging bool   `json:"isCharging"`
	Status     string `json:"status"`
}

type ProcessInfo struct {
	Name      string  `json:"name"`
	PID       int     `json:"pid"`
	CPUUsage  float64 `json:"cpuUsage"`
	MemUsage  float64 `json:"memUsage"`
}

type KillResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type CPUDetail struct {
	Model string  `json:"model"`
	Cores int     `json:"cores"`
	Usage float64 `json:"usage"`
	User  float64 `json:"user"`
	Sys   float64 `json:"sys"`
	Idle  float64 `json:"idle"`
}

type MemoryDetail struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Active    uint64  `json:"active"`
	Inactive  uint64  `json:"inactive"`
	Wired     uint64  `json:"wired"`
	Free      uint64  `json:"free"`
	Usage     float64 `json:"usage"`
}

type DiskDetail struct {
	Total     uint64  `json:"total"`
	Used      uint64  `json:"used"`
	Available uint64  `json:"available"`
	Usage     float64 `json:"usage"`
	MountPath string  `json:"mountPath"`
	FSType    string  `json:"fsType"`
}

type BatteryDetail struct {
	Percentage    int    `json:"percentage"`
	IsCharging    bool   `json:"isCharging"`
	Status        string `json:"status"`
	CycleCount    int    `json:"cycleCount"`
	Condition     string `json:"condition"`
	TimeRemaining string `json:"timeRemaining"`
}

type StatusService struct{}

func NewStatusService() *StatusService {
	return &StatusService{}
}

func (s *StatusService) GetSystemStatus() (*SystemStatus, error) {
	status := &SystemStatus{}

	status.CPU = s.getCPUInfo()
	status.Memory = s.getMemoryInfo()
	status.Disk = s.getDiskInfo("/")
	status.Battery = s.getBatteryInfo()
	status.Uptime = s.getUptime()
	status.Health = s.calculateHealth(status)

	return status, nil
}

func (s *StatusService) getCPUInfo() CPUInfo {
	info := CPUInfo{
		Cores: runtime.NumCPU(),
	}

	// Get CPU model
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err == nil {
		info.Model = strings.TrimSpace(string(out))
	}

	// Get CPU usage via top
	out, err = exec.Command("top", "-l", "1", "-n", "0", "-stats", "cpu").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "CPU usage") {
				parts := strings.Fields(line)
				for i, p := range parts {
					if strings.Contains(p, "user") && i > 0 {
						val := strings.TrimSuffix(parts[i-1], "%")
						if v, err := strconv.ParseFloat(val, 64); err == nil {
							info.Usage += v
						}
					}
					if strings.Contains(p, "sys") && i > 0 {
						val := strings.TrimSuffix(parts[i-1], "%")
						if v, err := strconv.ParseFloat(val, 64); err == nil {
							info.Usage += v
						}
					}
				}
				break
			}
		}
	}

	return info
}

func (s *StatusService) getMemoryInfo() MemoryInfo {
	info := MemoryInfo{}

	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		if total, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err == nil {
			info.Total = total
		}
	}

	out, err = exec.Command("vm_stat").Output()
	if err == nil {
		pageSize := uint64(os.Getpagesize())
		lines := strings.Split(string(out), "\n")
		var free, active, inactive, speculative, wired uint64

		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "."))
			num, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				continue
			}

			switch key {
			case "Pages free":
				free = num * pageSize
			case "Pages active":
				active = num * pageSize
			case "Pages inactive":
				inactive = num * pageSize
			case "Pages speculative":
				speculative = num * pageSize
			case "Pages wired down":
				wired = num * pageSize
			}
		}

		info.Used = active + wired
		info.Available = free + inactive + speculative
		if info.Total > 0 {
			info.Usage = float64(info.Used) / float64(info.Total) * 100
		}
	}

	return info
}

func (s *StatusService) getDiskInfo(path string) DiskInfo {
	info := DiskInfo{MountPath: path}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err == nil {
		info.Total = stat.Blocks * uint64(stat.Bsize)
		info.Available = stat.Bavail * uint64(stat.Bsize)
		info.Used = info.Total - info.Available
		if info.Total > 0 {
			info.Usage = float64(info.Used) / float64(info.Total) * 100
		}
	}

	return info
}

func (s *StatusService) getBatteryInfo() BatteryInfo {
	info := BatteryInfo{Status: "Unknown"}

	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return info
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "%") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasSuffix(p, "%;") || strings.HasSuffix(p, "%") {
					val := strings.TrimSuffix(strings.TrimSuffix(p, ";"), "%")
					if v, err := strconv.Atoi(val); err == nil {
						info.Percentage = v
					}
				}
			}

			if strings.Contains(line, "charging") && !strings.Contains(line, "discharging") && !strings.Contains(line, "not charging") {
				info.IsCharging = true
				info.Status = "Charging"
			} else if strings.Contains(line, "discharging") {
				info.Status = "On Battery"
			} else if strings.Contains(line, "AC Power") || strings.Contains(line, "charged") {
				info.Status = "Plugged In"
			}
		}
	}

	return info
}

func (s *StatusService) getUptime() string {
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return "Unknown"
	}

	str := string(out)
	if idx := strings.Index(str, "sec = "); idx >= 0 {
		secStr := str[idx+6:]
		if endIdx := strings.Index(secStr, ","); endIdx >= 0 {
			secStr = secStr[:endIdx]
			if sec, err := strconv.ParseInt(secStr, 10, 64); err == nil {
				bootTime := time.Unix(sec, 0)
				uptime := time.Since(bootTime)
				days := int(uptime.Hours() / 24)
				hours := int(uptime.Hours()) % 24
				mins := int(uptime.Minutes()) % 60
				if days > 0 {
					return strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
				}
				return strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
			}
		}
	}

	return "Unknown"
}

func (s *StatusService) GetTopProcesses(limit int) ([]ProcessInfo, error) {
	if limit <= 0 {
		limit = 10
	}

	out, err := exec.Command("ps", "-Arco", "pid,pcpu,pmem,comm").Output()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[1], 64)
		mem, _ := strconv.ParseFloat(fields[2], 64)
		name := strings.Join(fields[3:], " ")

		processes = append(processes, ProcessInfo{
			Name:     name,
			PID:      pid,
			CPUUsage: cpu,
			MemUsage: mem,
		})

		if len(processes) >= limit {
			break
		}
	}

	return processes, nil
}

func (s *StatusService) calculateHealth(status *SystemStatus) int {
	score := 100

	// Disk usage penalty
	if status.Disk.Usage > 90 {
		score -= 30
	} else if status.Disk.Usage > 80 {
		score -= 15
	} else if status.Disk.Usage > 70 {
		score -= 5
	}

	// Memory usage penalty
	if status.Memory.Usage > 90 {
		score -= 25
	} else if status.Memory.Usage > 80 {
		score -= 10
	}

	// CPU usage penalty
	if status.CPU.Usage > 90 {
		score -= 20
	} else if status.CPU.Usage > 70 {
		score -= 10
	}

	// Battery penalty
	if status.Battery.Percentage > 0 && status.Battery.Percentage < 20 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *StatusService) KillProcess(pid int) KillResult {
	if pid <= 1 {
		return KillResult{Success: false, Message: "Cannot kill system process"}
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return KillResult{Success: false, Message: "Process not found: " + err.Error()}
	}
	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		err = proc.Signal(syscall.SIGKILL)
		if err != nil {
			return KillResult{Success: false, Message: "Failed to kill: " + err.Error()}
		}
	}
	return KillResult{Success: true, Message: "Process terminated"}
}

func (s *StatusService) GetAllProcesses() ([]ProcessInfo, error) {
	out, err := exec.Command("ps", "-Arco", "pid,pcpu,pmem,comm").Output()
	if err != nil {
		return nil, err
	}

	var processes []ProcessInfo
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpu, _ := strconv.ParseFloat(fields[1], 64)
		mem, _ := strconv.ParseFloat(fields[2], 64)
		name := strings.Join(fields[3:], " ")

		processes = append(processes, ProcessInfo{
			Name:     name,
			PID:      pid,
			CPUUsage: cpu,
			MemUsage: mem,
		})
	}

	return processes, nil
}

func (s *StatusService) GetCPUDetail() CPUDetail {
	detail := CPUDetail{
		Cores: runtime.NumCPU(),
	}

	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err == nil {
		detail.Model = strings.TrimSpace(string(out))
	}

	out, err = exec.Command("top", "-l", "1", "-n", "0", "-stats", "cpu").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "CPU usage") {
				parts := strings.Fields(line)
				for i, p := range parts {
					if strings.Contains(p, "user") && i > 0 {
						val := strings.TrimSuffix(parts[i-1], "%")
						if v, err := strconv.ParseFloat(val, 64); err == nil {
							detail.User = v
						}
					}
					if strings.Contains(p, "sys") && i > 0 {
						val := strings.TrimSuffix(parts[i-1], "%")
						if v, err := strconv.ParseFloat(val, 64); err == nil {
							detail.Sys = v
						}
					}
					if strings.Contains(p, "idle") && i > 0 {
						val := strings.TrimSuffix(parts[i-1], "%")
						if v, err := strconv.ParseFloat(val, 64); err == nil {
							detail.Idle = v
						}
					}
				}
				break
			}
		}
	}

	detail.Usage = detail.User + detail.Sys
	return detail
}

func (s *StatusService) GetMemoryDetail() MemoryDetail {
	detail := MemoryDetail{}

	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err == nil {
		if total, err := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64); err == nil {
			detail.Total = total
		}
	}

	out, err = exec.Command("vm_stat").Output()
	if err == nil {
		pageSize := uint64(os.Getpagesize())
		lines := strings.Split(string(out), "\n")

		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "."))
			num, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				continue
			}

			switch key {
			case "Pages free":
				detail.Free = num * pageSize
			case "Pages active":
				detail.Active = num * pageSize
			case "Pages inactive":
				detail.Inactive = num * pageSize
			case "Pages wired down":
				detail.Wired = num * pageSize
			}
		}

		detail.Used = detail.Active + detail.Wired
		detail.Available = detail.Free + detail.Inactive
		if detail.Total > 0 {
			detail.Usage = float64(detail.Used) / float64(detail.Total) * 100
		}
	}

	return detail
}

func (s *StatusService) GetDiskDetail() DiskDetail {
	detail := DiskDetail{MountPath: "/"}

	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		detail.Total = stat.Blocks * uint64(stat.Bsize)
		detail.Available = stat.Bavail * uint64(stat.Bsize)
		detail.Used = detail.Total - detail.Available
		if detail.Total > 0 {
			detail.Usage = float64(detail.Used) / float64(detail.Total) * 100
		}

		// Extract filesystem type from Fstypename
		fsType := make([]byte, 0, len(stat.Fstypename))
		for _, b := range stat.Fstypename {
			if b == 0 {
				break
			}
			fsType = append(fsType, byte(b))
		}
		detail.FSType = string(fsType)
	}

	return detail
}

func (s *StatusService) GetBatteryDetail() BatteryDetail {
	detail := BatteryDetail{Status: "Unknown"}

	// Basic info from pmset
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "%") {
				parts := strings.Fields(line)
				for _, p := range parts {
					if strings.HasSuffix(p, "%;") || strings.HasSuffix(p, "%") {
						val := strings.TrimSuffix(strings.TrimSuffix(p, ";"), "%")
						if v, err := strconv.Atoi(val); err == nil {
							detail.Percentage = v
						}
					}
				}

				if strings.Contains(line, "charging") && !strings.Contains(line, "discharging") && !strings.Contains(line, "not charging") {
					detail.IsCharging = true
					detail.Status = "Charging"
				} else if strings.Contains(line, "discharging") {
					detail.Status = "On Battery"
				} else if strings.Contains(line, "AC Power") || strings.Contains(line, "charged") {
					detail.Status = "Plugged In"
				}

				// Time remaining
				for i, p := range parts {
					if p == "remaining" && i > 0 {
						detail.TimeRemaining = parts[i-1]
					} else if strings.Contains(p, "no estimate") || strings.Contains(p, "(no") {
						detail.TimeRemaining = "Calculating..."
					}
				}
			}
		}
	}

	// Cycle count and condition from system_profiler
	out, err = exec.Command("system_profiler", "SPPowerDataType").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Cycle Count:") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "Cycle Count:"))
				if v, err := strconv.Atoi(val); err == nil {
					detail.CycleCount = v
				}
			} else if strings.HasPrefix(trimmed, "Condition:") {
				detail.Condition = strings.TrimSpace(strings.TrimPrefix(trimmed, "Condition:"))
			}
		}
	}

	return detail
}
