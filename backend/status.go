package backend

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Cached WiFi interface name and service name
var (
	wifiInterfaceOnce sync.Once
	wifiInterfaceName string // e.g. "en0", "en1"
	wifiServiceName   string // e.g. "Wi-Fi"
)

// getWiFiInterface returns the device name for the WiFi hardware port (e.g. "en0").
// Falls back to "en0" if detection fails.
func getWiFiInterface() string {
	wifiInterfaceOnce.Do(func() {
		out, err := exec.Command("networksetup", "-listallhardwareports").Output()
		if err != nil {
			wifiInterfaceName = "en0"
			wifiServiceName = "Wi-Fi"
			return
		}
		lines := strings.Split(string(out), "\n")
		for i, line := range lines {
			if strings.Contains(line, "Wi-Fi") || strings.Contains(line, "AirPort") {
				// Extract service name from this line
				if strings.HasPrefix(strings.TrimSpace(line), "Hardware Port:") {
					parts := strings.SplitN(line, ":", 2)
					if len(parts) == 2 {
						wifiServiceName = strings.TrimSpace(parts[1])
					}
				}
				// Look for "Device:" in the next few lines
				for j := i + 1; j < len(lines) && j <= i+3; j++ {
					if strings.Contains(lines[j], "Device:") {
						parts := strings.SplitN(lines[j], ":", 2)
						if len(parts) == 2 {
							wifiInterfaceName = strings.TrimSpace(parts[1])
						}
						break
					}
				}
				break
			}
		}
		if wifiInterfaceName == "" {
			wifiInterfaceName = "en0"
		}
		if wifiServiceName == "" {
			wifiServiceName = "Wi-Fi"
		}
	})
	return wifiInterfaceName
}

// getWiFiServiceName returns the network service name for WiFi (e.g. "Wi-Fi").
func getWiFiServiceName() string {
	getWiFiInterface() // ensure initialized
	return wifiServiceName
}

type SystemStatus struct {
	CPU     CPUInfo     `json:"cpu"`
	Memory  MemoryInfo  `json:"memory"`
	Disk    DiskInfo    `json:"disk"`
	Battery BatteryInfo `json:"battery"`
	Network NetworkInfo `json:"network"`
	Uptime  string      `json:"uptime"`
	Health  int         `json:"health"`
}

type NetworkInfo struct {
	Status    string `json:"status"`
	IPAddress string `json:"ipAddress"`
	WiFi      string `json:"wifi"`
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
	Model        string  `json:"model"`
	Cores        int     `json:"cores"`
	Architecture string  `json:"architecture"`
	Usage        float64 `json:"usage"`
	User         float64 `json:"user"`
	Sys          float64 `json:"sys"`
	Idle         float64 `json:"idle"`
	LoadAvg1     float64 `json:"loadAvg1"`
	LoadAvg5     float64 `json:"loadAvg5"`
	LoadAvg15    float64 `json:"loadAvg15"`
	ProcessCount int     `json:"processCount"`
	ThreadCount  int     `json:"threadCount"`
	Uptime       string  `json:"uptime"`
}

type MemoryDetail struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Available   uint64  `json:"available"`
	Active      uint64  `json:"active"`
	Inactive    uint64  `json:"inactive"`
	Wired       uint64  `json:"wired"`
	Compressed  uint64  `json:"compressed"`
	Free        uint64  `json:"free"`
	Usage       float64 `json:"usage"`
	SwapTotal   uint64  `json:"swapTotal"`
	SwapUsed    uint64  `json:"swapUsed"`
	AppMemory   uint64  `json:"appMemory"`
	CachedFiles uint64  `json:"cachedFiles"`
	Pressure    string  `json:"pressure"`
}

type WiFiDetail struct {
	PowerOn          bool             `json:"powerOn"`
	Connected        bool             `json:"connected"`
	SSID             string           `json:"ssid"`
	BSSID            string           `json:"bssid"`
	Channel          string           `json:"channel"`
	Band             string           `json:"band"`
	PHYMode          string           `json:"phyMode"`
	Security         string           `json:"security"`
	SignalStrength   int              `json:"signalStrength"`
	NoiseLevel       int              `json:"noiseLevel"`
	TransmitRate     string           `json:"transmitRate"`
	MACAddress       string           `json:"macAddress"`
	CardType         string           `json:"cardType"`
	FirmwareVersion  string           `json:"firmwareVersion"`
	CountryCode      string           `json:"countryCode"`
	SupportedPHY     string           `json:"supportedPHY"`
	AirDrop          string           `json:"airdrop"`
	AutoUnlock       string           `json:"autoUnlock"`
	SavedNetworks    []SavedNetwork   `json:"savedNetworks"`
	NearbyNetworks   []NearbyNetwork  `json:"nearbyNetworks"`
}

type SavedNetwork struct {
	Name string `json:"name"`
}

type NearbyNetwork struct {
	Name     string `json:"name"`
	Signal   string `json:"signal"`
	Channel  string `json:"channel"`
	Security string `json:"security"`
	PHYMode  string `json:"phyMode"`
}

type NetworkDetail struct {
	Interface    string `json:"interface"`
	Status       string `json:"status"`
	IPAddress    string `json:"ipAddress"`
	SubnetMask   string `json:"subnetMask"`
	Router       string `json:"router"`
	MACAddress   string `json:"macAddress"`
	IPv6Address  string `json:"ipv6Address"`
	DNS          []string `json:"dns"`
	WiFiNetwork  string `json:"wifiNetwork"`
	WiFiSecurity string `json:"wifiSecurity"`
	ConfigMethod string `json:"configMethod"`
	BytesSent    uint64 `json:"bytesSent"`
	BytesRecv    uint64 `json:"bytesRecv"`
	PacketsSent  uint64 `json:"packetsSent"`
	PacketsRecv  uint64 `json:"packetsRecv"`
	LinkSpeed    string `json:"linkSpeed"`
	Hostname     string `json:"hostname"`
	ExternalIP   string `json:"externalIP"`
}

type DiskDetail struct {
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Available  uint64  `json:"available"`
	Usage      float64 `json:"usage"`
	MountPath  string  `json:"mountPath"`
	FSType     string  `json:"fsType"`
	VolumeName string  `json:"volumeName"`
	DiskType   string  `json:"diskType"`
	ReadOps    uint64  `json:"readOps"`
	WriteOps   uint64  `json:"writeOps"`
}

type BatteryDetail struct {
	Percentage    int    `json:"percentage"`
	IsCharging    bool   `json:"isCharging"`
	Status        string `json:"status"`
	CycleCount    int    `json:"cycleCount"`
	MaxCycleCount int    `json:"maxCycleCount"`
	Condition     string `json:"condition"`
	TimeRemaining string `json:"timeRemaining"`
	MaxCapacity   int    `json:"maxCapacity"`
	DesignCapacity int   `json:"designCapacity"`
	Temperature   float64 `json:"temperature"`
	Voltage       float64 `json:"voltage"`
	Wattage       float64 `json:"wattage"`
	PowerSource   string  `json:"powerSource"`
	HealthPercent float64 `json:"healthPercent"`
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
	status.Network = s.getNetworkInfo()
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
	if info.Model == "" {
		if runtime.GOARCH == "arm64" {
			info.Model = "Apple Silicon"
		} else {
			info.Model = "Intel"
		}
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
			case "Pages active", "Pages app":
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

func (s *StatusService) getNetworkInfo() NetworkInfo {
	info := NetworkInfo{Status: "Inactive"}

	iface := getWiFiInterface()
	svc := getWiFiServiceName()

	out, err := exec.Command("ifconfig", iface).Output()
	if err == nil {
		if strings.Contains(string(out), "status: active") {
			info.Status = "Active"
		}
	}

	out, err = exec.Command("networksetup", "-getinfo", svc).Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == "IP address" {
				info.IPAddress = strings.TrimSpace(parts[1])
			}
		}
	}

	out, err = exec.Command("networksetup", "-getairportnetwork", iface).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if strings.Contains(raw, ": ") && !strings.Contains(raw, "not associated") {
			info.WiFi = strings.TrimSpace(strings.SplitN(raw, ": ", 2)[1])
		}
	}

	// If connected but SSID hidden (macOS privacy), show connected
	if info.Status == "Active" && info.WiFi == "" {
		info.WiFi = "Connected"
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
		Cores:        runtime.NumCPU(),
		Architecture: runtime.GOARCH,
	}

	out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output()
	if err == nil {
		detail.Model = strings.TrimSpace(string(out))
	}
	if detail.Model == "" {
		out2, err2 := exec.Command("sysctl", "-n", "machdep.cpu.brand").Output()
		if err2 == nil {
			detail.Model = strings.TrimSpace(string(out2))
		}
	}
	if detail.Model == "" {
		if runtime.GOARCH == "arm64" {
			detail.Model = "Apple Silicon"
		} else {
			detail.Model = "Intel"
		}
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
			if strings.Contains(line, "Processes:") {
				parts := strings.Fields(line)
				for i, p := range parts {
					clean := strings.TrimRight(p, ".,;:")
					if clean == "total" && i > 0 {
						num := strings.TrimRight(parts[i-1], ".,;:")
						if v, err := strconv.Atoi(num); err == nil {
							detail.ProcessCount = v
						}
					}
					if clean == "threads" && i > 0 {
						num := strings.TrimRight(parts[i-1], ".,;:")
						if v, err := strconv.Atoi(num); err == nil {
							detail.ThreadCount = v
						}
					}
				}
			}
		}
	}

	// Load averages
	out, err = exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err == nil {
		parts := strings.Fields(strings.Trim(strings.TrimSpace(string(out)), "{}"))
		if len(parts) >= 3 {
			if v, err := strconv.ParseFloat(parts[0], 64); err == nil { detail.LoadAvg1 = v }
			if v, err := strconv.ParseFloat(parts[1], 64); err == nil { detail.LoadAvg5 = v }
			if v, err := strconv.ParseFloat(parts[2], 64); err == nil { detail.LoadAvg15 = v }
		}
	}

	// Uptime
	out, err = exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if idx := strings.Index(raw, "sec = "); idx >= 0 {
			secStr := raw[idx+6:]
			if end := strings.Index(secStr, ","); end >= 0 {
				secStr = secStr[:end]
			}
			if sec, err := strconv.ParseInt(secStr, 10, 64); err == nil {
				bootTime := time.Unix(sec, 0)
				dur := time.Since(bootTime)
				days := int(dur.Hours()) / 24
				hours := int(dur.Hours()) % 24
				mins := int(dur.Minutes()) % 60
				if days > 0 {
					detail.Uptime = strconv.Itoa(days) + "d " + strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
				} else {
					detail.Uptime = strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
				}
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

	var pagesOccupiedByCompressor uint64
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
			case "Pages active", "Pages app":
				detail.Active = num * pageSize
			case "Pages inactive":
				detail.Inactive = num * pageSize
			case "Pages wired down":
				detail.Wired = num * pageSize
			case "Pages occupied by compressor":
				detail.Compressed = num * pageSize
				pagesOccupiedByCompressor = num * pageSize
			case "File-backed pages":
				detail.CachedFiles = num * pageSize
			}
		}

		detail.Used = detail.Active + detail.Wired + pagesOccupiedByCompressor
		detail.AppMemory = detail.Active
		detail.Available = detail.Free + detail.Inactive
		if detail.Total > 0 {
			detail.Usage = float64(detail.Used) / float64(detail.Total) * 100
		}
	}

	// Swap info
	out, err = exec.Command("sysctl", "-n", "vm.swapusage").Output()
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(string(out)))
		for i, p := range parts {
			if p == "total" && i > 0 {
				detail.SwapTotal = parseMemValue(parts[i-1])
			}
			if p == "used" && i > 0 {
				detail.SwapUsed = parseMemValue(parts[i-1])
			}
		}
	}

	// Memory pressure
	out, err = exec.Command("sysctl", "-n", "kern.memorystatus_vm_pressure_level").Output()
	if err == nil {
		level := strings.TrimSpace(string(out))
		switch level {
		case "1":
			detail.Pressure = "Normal"
		case "2":
			detail.Pressure = "Warning"
		case "4":
			detail.Pressure = "Critical"
		default:
			detail.Pressure = "Normal"
		}
	}

	return detail
}

func parseMemValue(s string) uint64 {
	s = strings.TrimSpace(s)
	multiplier := uint64(1)
	if strings.HasSuffix(s, "G") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "G")
	} else if strings.HasSuffix(s, "M") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "M")
	} else if strings.HasSuffix(s, "K") {
		multiplier = 1024
		s = strings.TrimSuffix(s, "K")
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil {
		return uint64(v * float64(multiplier))
	}
	return 0
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

		fsType := make([]byte, 0, len(stat.Fstypename))
		for _, b := range stat.Fstypename {
			if b == 0 {
				break
			}
			fsType = append(fsType, byte(b))
		}
		detail.FSType = string(fsType)
	}

	// Volume name
	out, err := exec.Command("diskutil", "info", "/").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Volume Name:") {
				detail.VolumeName = strings.TrimSpace(strings.TrimPrefix(trimmed, "Volume Name:"))
			}
			if strings.HasPrefix(trimmed, "Solid State:") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "Solid State:"))
				if val == "Yes" {
					detail.DiskType = "SSD"
				} else {
					detail.DiskType = "HDD"
				}
			}
			if strings.HasPrefix(trimmed, "Device / Media Name:") && detail.DiskType == "" {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "Device / Media Name:"))
				if strings.Contains(strings.ToLower(val), "ssd") || strings.Contains(strings.ToLower(val), "solid") {
					detail.DiskType = "SSD"
				}
			}
		}
	}
	if detail.DiskType == "" {
		detail.DiskType = "SSD" // Modern Macs are all SSD
	}

	// IO stats from ioreg — reliable across all macOS versions
	out, err = exec.Command("ioreg", "-c", "IOBlockStorageDriver", "-r", "-w0").Output()
	if err == nil {
		text := string(out)
		// Sum Operations (Read) and Operations (Write) across all drivers
		for _, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "Statistics") {
				// Parse "Operations (Read)"=NNN
				if idx := strings.Index(line, `"Operations (Read)"=`); idx >= 0 {
					valStr := line[idx+len(`"Operations (Read)"=`):]
					if end := strings.IndexAny(valStr, ",}"); end >= 0 {
						valStr = valStr[:end]
					}
					if v, err := strconv.ParseUint(strings.TrimSpace(valStr), 10, 64); err == nil {
						detail.ReadOps += v
					}
				}
				// Parse "Operations (Write)"=NNN
				if idx := strings.Index(line, `"Operations (Write)"=`); idx >= 0 {
					valStr := line[idx+len(`"Operations (Write)"=`):]
					if end := strings.IndexAny(valStr, ",}"); end >= 0 {
						valStr = valStr[:end]
					}
					if v, err := strconv.ParseUint(strings.TrimSpace(valStr), 10, 64); err == nil {
						detail.WriteOps += v
					}
				}
			}
		}
	}

	return detail
}

func (s *StatusService) GetBatteryDetail() BatteryDetail {
	detail := BatteryDetail{Status: "Unknown", MaxCycleCount: 1000}

	// Basic info from pmset
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "AC Power") {
				detail.PowerSource = "AC Power"
			} else if strings.Contains(line, "Battery Power") {
				detail.PowerSource = "Battery"
			}
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
				} else if strings.Contains(line, "charged") {
					detail.Status = "Fully Charged"
				} else if strings.Contains(line, "not charging") {
					detail.Status = "Not Charging"
				}

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

	// Detailed info from system_profiler
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
			} else if strings.HasPrefix(trimmed, "Maximum Capacity:") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "Maximum Capacity:"))
				val = strings.TrimSuffix(val, "%")
				if v, err := strconv.Atoi(val); err == nil {
					detail.MaxCapacity = v
				}
			} else if strings.HasPrefix(trimmed, "State of Charge") && strings.Contains(trimmed, ":") {
				// Already have percentage
			} else if strings.HasPrefix(trimmed, "Voltage") && strings.Contains(trimmed, "(mV):") {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, "Voltage (mV):"))
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					detail.Voltage = v / 1000.0
				}
			} else if strings.HasPrefix(trimmed, "Wattage") {
				val := strings.TrimSpace(trimmed[strings.Index(trimmed, ":")+1:])
				if v, err := strconv.ParseFloat(val, 64); err == nil {
					detail.Wattage = v
				}
			}
		}
	}

	// Temperature from ioreg
	out, err = exec.Command("ioreg", "-r", "-n", "AppleSmartBattery", "-w0").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "\"Temperature\"") {
				parts := strings.Split(trimmed, "=")
				if len(parts) == 2 {
					val := strings.TrimSpace(parts[1])
					if v, err := strconv.ParseFloat(val, 64); err == nil {
						detail.Temperature = v / 100.0 // Stored as centidegrees
					}
				}
			}
			if strings.Contains(trimmed, "\"DesignCapacity\"") {
				parts := strings.Split(trimmed, "=")
				if len(parts) == 2 {
					val := strings.TrimSpace(parts[1])
					if v, err := strconv.Atoi(val); err == nil {
						detail.DesignCapacity = v
					}
				}
			}
		}
	}

	// Battery health percent
	if detail.MaxCapacity > 0 {
		detail.HealthPercent = float64(detail.MaxCapacity)
	} else if detail.DesignCapacity > 0 && detail.MaxCapacity > 0 {
		detail.HealthPercent = float64(detail.MaxCapacity) / float64(detail.DesignCapacity) * 100
	}

	return detail
}

func (s *StatusService) GetNetworkDetail() NetworkDetail {
	iface := getWiFiInterface()
	svc := getWiFiServiceName()
	detail := NetworkDetail{Interface: iface, Status: "Inactive"}

	// Hostname
	if h, err := os.Hostname(); err == nil {
		detail.Hostname = h
	}

	// IP, subnet, router, config method from networksetup
	out, err := exec.Command("networksetup", "-getinfo", svc).Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			switch key {
			case "IP address":
				detail.IPAddress = val
			case "Subnet mask":
				detail.SubnetMask = val
			case "Router":
				detail.Router = val
			case "Wi-Fi ID":
				detail.MACAddress = val
			}
			if strings.Contains(key, "Configuration") && !strings.Contains(key, "IPv6") {
				detail.ConfigMethod = val
			}
		}
	}

	// WiFi network name
	out, err = exec.Command("networksetup", "-getairportnetwork", iface).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if strings.Contains(raw, ": ") {
			detail.WiFiNetwork = strings.TrimSpace(strings.SplitN(raw, ": ", 2)[1])
		}
	}

	// DNS servers
	out, err = exec.Command("networksetup", "-getdnsservers", svc).Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.Contains(trimmed, "aren't") {
				detail.DNS = append(detail.DNS, trimmed)
			}
		}
	}

	// IPv6 from ifconfig
	out, err = exec.Command("ifconfig", iface).Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "inet6") && !strings.Contains(trimmed, "fe80") {
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 {
					detail.IPv6Address = parts[1]
					break
				}
			}
			if strings.Contains(trimmed, "status: active") {
				detail.Status = "Active"
			}
		}
	}

	// Traffic stats from netstat
	out, err = exec.Command("netstat", "-ib").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 11 && fields[0] == iface && strings.Contains(fields[2], ".") {
				if v, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
					detail.PacketsRecv = v
				}
				if v, err := strconv.ParseUint(fields[6], 10, 64); err == nil {
					detail.BytesRecv = v
				}
				if v, err := strconv.ParseUint(fields[7], 10, 64); err == nil {
					detail.PacketsSent = v
				}
				if v, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
					detail.BytesSent = v
				}
				break
			}
		}
	}

	// Link speed from system_profiler (can be slow, skip if needed)
	out, err = exec.Command("networksetup", "-getMedia", svc).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if raw != "" {
			detail.LinkSpeed = raw
		}
	}

	return detail
}

func (s *StatusService) GetWiFiDetail() WiFiDetail {
	detail := WiFiDetail{}
	iface := getWiFiInterface()

	// Power status
	out, err := exec.Command("networksetup", "-getairportpower", iface).Output()
	if err == nil {
		detail.PowerOn = strings.Contains(string(out), "On")
	}

	// Saved networks
	out, err = exec.Command("networksetup", "-listpreferredwirelessnetworks", iface).Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines[1:] { // skip header
			name := strings.TrimSpace(line)
			if name != "" {
				detail.SavedNetworks = append(detail.SavedNetworks, SavedNetwork{Name: name})
			}
		}
	}

	// Detailed info from system_profiler
	out, err = exec.Command("system_profiler", "SPAirPortDataType").Output()
	if err == nil {
		text := string(out)
		lines := strings.Split(text, "\n")

		// Parse card info
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			kv := strings.SplitN(trimmed, ":", 2)
			if len(kv) != 2 {
				continue
			}
			key := strings.TrimSpace(kv[0])
			val := strings.TrimSpace(kv[1])

			switch key {
			case "Card Type":
				detail.CardType = val
			case "Firmware Version":
				detail.FirmwareVersion = val
			case "MAC Address":
				detail.MACAddress = val
			case "Country Code":
				detail.CountryCode = val
			case "Supported PHY Modes":
				detail.SupportedPHY = val
			case "AirDrop":
				detail.AirDrop = val
			case "Auto Unlock":
				detail.AutoUnlock = val
			case "Status":
				detail.Connected = val == "Connected"
			case "PHY Mode":
				if detail.PHYMode == "" { // first occurrence is current
					detail.PHYMode = val
				}
			case "Channel":
				if detail.Channel == "" {
					detail.Channel = val
					if strings.Contains(val, "2GHz") {
						detail.Band = "2.4 GHz"
					} else if strings.Contains(val, "5GHz") {
						detail.Band = "5 GHz"
					} else if strings.Contains(val, "6GHz") {
						detail.Band = "6 GHz"
					}
				}
			case "Security":
				if detail.Security == "" {
					detail.Security = val
				}
			case "Transmit Rate":
				if detail.TransmitRate == "" {
					detail.TransmitRate = val
				}
			}

			// Signal/Noise
			if key == "Signal / Noise" && detail.SignalStrength == 0 {
				parts := strings.Split(val, "/")
				if len(parts) == 2 {
					sigStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[0]), "dBm"))
					noiseStr := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(parts[1]), "dBm"))
					if v, err := strconv.Atoi(strings.TrimSpace(sigStr)); err == nil {
						detail.SignalStrength = v
					}
					if v, err := strconv.Atoi(strings.TrimSpace(noiseStr)); err == nil {
						detail.NoiseLevel = v
					}
				}
			}
		}

		// Parse nearby networks
		inOtherNetworks := false
		var currentNearby *NearbyNetwork
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "Other Local Wi-Fi Networks") {
				inOtherNetworks = true
				continue
			}
			if !inOtherNetworks {
				continue
			}
			// A network name line ends with ":"
			if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") || (strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, "PHY") && !strings.Contains(trimmed, "Channel") && !strings.Contains(trimmed, "Network") && !strings.Contains(trimmed, "Security") && !strings.Contains(trimmed, "Signal") && !strings.Contains(trimmed, "Transmit")) {
				name := strings.TrimSuffix(trimmed, ":")
				if name == "<redacted>" {
					continue // Skip redacted entries
				}
				n := NearbyNetwork{Name: name}
				detail.NearbyNetworks = append(detail.NearbyNetworks, n)
				currentNearby = &detail.NearbyNetworks[len(detail.NearbyNetworks)-1]
				continue
			}
			if currentNearby != nil {
				kv := strings.SplitN(trimmed, ":", 2)
				if len(kv) == 2 {
					k := strings.TrimSpace(kv[0])
					v := strings.TrimSpace(kv[1])
					switch k {
					case "Channel":
						currentNearby.Channel = v
					case "Security":
						currentNearby.Security = v
					case "Signal / Noise":
						currentNearby.Signal = v
					case "PHY Mode":
						currentNearby.PHYMode = v
					}
				}
			}
		}
	}

	// Get SSID — try multiple methods since macOS may redact it
	out, err = exec.Command("networksetup", "-getairportnetwork", iface).Output()
	if err == nil {
		raw := strings.TrimSpace(string(out))
		if strings.Contains(raw, ": ") && !strings.Contains(raw, "not associated") {
			detail.SSID = strings.TrimSpace(strings.SplitN(raw, ": ", 2)[1])
		}
	}

	// If SSID still empty but connected, mark as connected with hidden SSID
	if detail.SSID == "" && detail.Connected {
		detail.SSID = "Connected (SSID hidden)"
	}

	// Detect connection from ifconfig as fallback
	if !detail.Connected {
		out, err = exec.Command("ifconfig", iface).Output()
		if err == nil && strings.Contains(string(out), "status: active") {
			detail.Connected = true
			if detail.SSID == "" {
				detail.SSID = "Connected (SSID hidden)"
			}
		}
	}

	return detail
}

func (s *StatusService) GetWiFiPassword(networkName string) (string, error) {
	cmd := exec.Command("security", "find-generic-password", "-ga", networkName, "-w")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not retrieve password — keychain access denied or not found")
	}
	return strings.TrimSpace(string(out)), nil
}
