package metrics

import (
	"math"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
)

// GPUInfo holds metrics for one GPU device.
type GPUInfo struct {
	Name         string
	DeviceUtil   float64 // overall device utilization %
	RendererUtil float64 // renderer pipeline %
	TilerUtil    float64 // tiler %
	MemInUse     uint64  // bytes currently in use
}

// Snapshot holds all system metrics captured at a point in time.
type Snapshot struct {
	Timestamp time.Time

	// CPU
	CPUTotal  float64
	CPUCores  []float64
	CPUModel  string
	IsAppleSi bool
	NumPCores int
	NumECores int

	// GPU
	GPUs []GPUInfo

	// Memory
	MemTotal    uint64
	MemUsed     uint64
	MemPercent  float64
	SwapTotal   uint64
	SwapUsed    uint64
	SwapPercent float64

	// Disk I/O (bytes/sec, aggregated across all disks)
	DiskReadPS  float64
	DiskWritePS float64

	// Network (bytes/sec, aggregated across all interfaces)
	NetUpPS   float64
	NetDownPS float64

	// Load averages
	Load1  float64
	Load5  float64
	Load15 float64
}

// Collector gathers system metrics and computes per-second rates between samples.
type Collector struct {
	mu sync.Mutex

	cpuModel  string
	isAppleSi bool
	numPCores int
	numECores int

	prevDisk map[string]disk.IOCountersStat
	prevNet  []psnet.IOCountersStat
	prevTime time.Time
	seeded   bool
}

func New() *Collector {
	c := &Collector{}
	c.detectHardware()
	return c
}

func (c *Collector) detectHardware() {
	if infos, err := cpu.Info(); err == nil && len(infos) > 0 {
		c.cpuModel = infos[0].ModelName
	}

	out, err := exec.Command("sysctl", "-n", "hw.optional.arm64").Output()
	if err != nil || strings.TrimSpace(string(out)) != "1" {
		return
	}
	c.isAppleSi = true

	if out, err = exec.Command("sysctl", "-n", "hw.perflevel0.physicalcpu").Output(); err == nil {
		c.numPCores, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
	if out, err = exec.Command("sysctl", "-n", "hw.perflevel1.physicalcpu").Output(); err == nil {
		c.numECores, _ = strconv.Atoi(strings.TrimSpace(string(out)))
	}
}

// Collect gathers a new snapshot. On the first call disk/network rates are 0.
func (c *Collector) Collect() (*Snapshot, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	s := &Snapshot{
		Timestamp: now,
		CPUModel:  c.cpuModel,
		IsAppleSi: c.isAppleSi,
		NumPCores: c.numPCores,
		NumECores: c.numECores,
	}

	// CPU — gopsutil tracks previous times internally when interval=0
	if perCore, err := cpu.Percent(0, true); err == nil {
		s.CPUCores = perCore
		var sum float64
		for _, p := range perCore {
			sum += p
		}
		if len(perCore) > 0 {
			s.CPUTotal = sum / float64(len(perCore))
		}
	}

	// Memory
	if vm, err := mem.VirtualMemory(); err == nil {
		s.MemTotal = vm.Total
		s.MemUsed = vm.Used
		s.MemPercent = vm.UsedPercent
	}
	if sw, err := mem.SwapMemory(); err == nil {
		s.SwapTotal = sw.Total
		s.SwapUsed = sw.Used
		s.SwapPercent = sw.UsedPercent
	}

	// Load averages
	if avg, err := load.Avg(); err == nil {
		s.Load1 = avg.Load1
		s.Load5 = avg.Load5
		s.Load15 = avg.Load15
	}

	// Disk I/O rates
	if diskIO, err := disk.IOCounters(); err == nil {
		if c.seeded {
			elapsed := now.Sub(c.prevTime).Seconds()
			if elapsed > 0 {
				for name, cur := range diskIO {
					if prev, ok := c.prevDisk[name]; ok {
						s.DiskReadPS += math.Max(0, float64(cur.ReadBytes-prev.ReadBytes)/elapsed)
						s.DiskWritePS += math.Max(0, float64(cur.WriteBytes-prev.WriteBytes)/elapsed)
					}
				}
			}
		}
		c.prevDisk = diskIO
	}

	// Network I/O rates (aggregate, pernic=false)
	if nets, err := psnet.IOCounters(false); err == nil {
		if c.seeded && len(nets) > 0 && len(c.prevNet) > 0 {
			elapsed := now.Sub(c.prevTime).Seconds()
			if elapsed > 0 {
				for i, cur := range nets {
					if i >= len(c.prevNet) {
						break
					}
					prev := c.prevNet[i]
					if cur.BytesSent >= prev.BytesSent {
						s.NetUpPS += float64(cur.BytesSent-prev.BytesSent) / elapsed
					}
					if cur.BytesRecv >= prev.BytesRecv {
						s.NetDownPS += float64(cur.BytesRecv-prev.BytesRecv) / elapsed
					}
				}
			}
		}
		c.prevNet = nets
	}

	s.GPUs = c.collectGPUs()

	c.prevTime = now
	c.seeded = true
	return s, nil
}

// collectGPUs reads GPU performance statistics from ioreg (no sudo required).
func (c *Collector) collectGPUs() []GPUInfo {
	out, err := exec.Command("ioreg", "-r", "-c", "IOAccelerator").Output()
	if err != nil {
		return nil
	}

	const (
		perfKey  = `"PerformanceStatistics"`
		classKey = `"IOClass" = "`
	)

	lines := strings.Split(string(out), "\n")
	seen := make(map[string]bool)
	var gpus []GPUInfo

	for i, line := range lines {
		if !strings.Contains(line, perfKey) || seen[line] {
			continue
		}
		seen[line] = true

		// IOClass appears in the next few lines after PerformanceStatistics.
		className := ""
		for j := i + 1; j < min(i+10, len(lines)); j++ {
			if idx := strings.Index(lines[j], classKey); idx >= 0 {
				rest := lines[j][idx+len(classKey):]
				if end := strings.Index(rest, `"`); end > 0 {
					className = rest[:end]
				}
				break
			}
		}

		g := GPUInfo{
			Name:         gpuFriendlyName(className, c.isAppleSi, c.cpuModel),
			DeviceUtil:   float64(ioregStat(line, "Device Utilization %")),
			RendererUtil: float64(ioregStat(line, "Renderer Utilization %")),
			TilerUtil:    float64(ioregStat(line, "Tiler Utilization %")),
			MemInUse:     uint64(ioregStat(line, "In use system memory")),
		}
		gpus = append(gpus, g)
	}
	return gpus
}

// ioregStat extracts an integer value for key from an ioreg property line.
// It matches `"key"=<digits>` exactly, so "In use system memory" won't
// accidentally match "In use system memory (driver)".
func ioregStat(line, key string) int64 {
	search := `"` + key + `"=`
	idx := strings.Index(line, search)
	if idx < 0 {
		return 0
	}
	rest := line[idx+len(search):]
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0
	}
	v, _ := strconv.ParseInt(rest[:end], 10, 64)
	return v
}

func gpuFriendlyName(ioClass string, isAppleSi bool, cpuModel string) string {
	if isAppleSi || strings.HasPrefix(ioClass, "AGX") {
		if cpuModel != "" {
			return cpuModel + " GPU"
		}
		return "Apple GPU"
	}
	switch {
	case strings.Contains(ioClass, "ATI") || strings.Contains(ioClass, "AMD"):
		return "AMD GPU"
	case strings.Contains(ioClass, "Intel"):
		return "Intel GPU"
	default:
		if ioClass != "" {
			return ioClass
		}
		return "GPU"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
