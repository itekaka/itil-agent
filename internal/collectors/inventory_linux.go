package collectors

import (
	"bufio"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func BuildInventoryPayload(agentID string) map[string]any {
	hostname, _ := os.Hostname()
	device := map[string]any{
		"hostname":     hostname,
		"os_name":      runtime.GOOS,
		"architecture": runtime.GOARCH,
		"logged_user":  os.Getenv("USER"),
	}

	// OS pretty name (Linux)
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				device["os_name"] = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
			}
			if strings.HasPrefix(line, "VERSION_ID=") {
				device["os_version"] = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			}
		}
	}

	hardware := map[string]any{
		"logical_cores": runtime.NumCPU(),
	}
	// MemTotal from /proc/meminfo
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						hardware["ram_total_bytes"] = kb * 1024
					}
				}
			}
		}
	}
	// CPU model
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					hardware["cpu_model"] = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	}

	// Simple root filesystem as one disk/partition
	disks := []map[string]any{}
	if st, err := os.Stat("/"); err == nil && st.IsDir() {
		// best-effort sizes via syscall would need golang.org/x/sys; keep placeholder partition mount
		disks = append(disks, map[string]any{
			"disk_index":     0,
			"model":          "rootfs",
			"interface_type": "virt",
			"health_status":  "OK",
			"partitions": []map[string]any{
				{
					"mount_point": "/",
					"filesystem":  "unknown",
				},
			},
		})
	}

	// Network interfaces
	nics := []map[string]any{}
	if ifaces, err := net.Interfaces(); err == nil {
		for _, iface := range ifaces {
			if iface.Flags&net.FlagLoopback != 0 {
				continue
			}
			row := map[string]any{
				"name":           iface.Name,
				"interface_type": "ethernet",
				"mac_address":    iface.HardwareAddr.String(),
			}
			addrs, _ := iface.Addrs()
			for _, a := range addrs {
				s := a.String()
				if strings.Contains(s, ".") && row["ipv4"] == nil {
					row["ipv4"] = strings.Split(s, "/")[0]
				}
			}
			nics = append(nics, row)
		}
	}

	// Minimal software list (optional empty on Linux Phase 1)
	software := []map[string]any{}

	_ = bufio.NewReader
	return map[string]any{
		"agent_id":          agentID,
		"inventory_version": 1,
		"collected_at":      time.Now().Format(time.RFC3339),
		"device":            device,
		"hardware":          hardware,
		"disks":             disks,
		"network_interfaces": nics,
		"software":          software,
	}
}
