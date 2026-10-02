//go:build windows

package collectors

import (
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows/registry"
)

type winOS struct {
	Caption string
	Version string
	CSName  string
}

type winCPU struct {
	Name                      string
	NumberOfCores             uint32
	NumberOfLogicalProcessors uint32
}

type winCS struct {
	TotalPhysicalMemory uint64
	Manufacturer        string
	Model               string
	UserName            string
}

type winBIOS struct {
	SMBIOSBIOSVersion string
	SerialNumber      string
	Manufacturer      string
}

type winDisk struct {
	Index         uint32
	Model         string
	SerialNumber  string
	Size          uint64
	InterfaceType string
	Status        string
}

type winLogical struct {
	DeviceID   string
	FileSystem string
	Size       uint64
	FreeSpace  uint64
}

type winNIC struct {
	Description          string
	MACAddress           string
	IPAddress            []string
	DefaultIPGateway     []string
	DNSServerSearchOrder []string
	DHCPEnabled          bool
}

func BuildInventoryPayload(agentID string) map[string]any {
	hostname, _ := os.Hostname()

	device := map[string]any{
		"hostname":     hostname,
		"architecture": runtime.GOARCH,
		"logged_user":  os.Getenv("USERNAME"),
	}

	var osRows []winOS
	_ = wmi.Query("SELECT Caption, Version, CSName FROM Win32_OperatingSystem", &osRows)
	if len(osRows) > 0 {
		device["os_name"] = osRows[0].Caption
		device["os_version"] = osRows[0].Version
		if osRows[0].CSName != "" {
			device["hostname"] = osRows[0].CSName
		}
	}

	var csRows []winCS
	_ = wmi.Query("SELECT TotalPhysicalMemory, Manufacturer, Model, UserName FROM Win32_ComputerSystem", &csRows)
	if len(csRows) > 0 {
		device["manufacturer"] = csRows[0].Manufacturer
		device["model"] = csRows[0].Model
		if csRows[0].UserName != "" {
			device["logged_user"] = csRows[0].UserName
		}
	}

	hardware := map[string]any{}
	var cpuRows []winCPU
	_ = wmi.Query("SELECT Name, NumberOfCores, NumberOfLogicalProcessors FROM Win32_Processor", &cpuRows)
	if len(cpuRows) > 0 {
		hardware["cpu_model"] = cpuRows[0].Name
		hardware["physical_cores"] = int(cpuRows[0].NumberOfCores)
		hardware["logical_cores"] = int(cpuRows[0].NumberOfLogicalProcessors)
	}
	if len(csRows) > 0 {
		hardware["ram_total_bytes"] = int64(csRows[0].TotalPhysicalMemory)
	}

	var biosRows []winBIOS
	_ = wmi.Query("SELECT SMBIOSBIOSVersion, SerialNumber, Manufacturer FROM Win32_BIOS", &biosRows)
	if len(biosRows) > 0 {
		hardware["bios_version"] = biosRows[0].SMBIOSBIOSVersion
		hardware["bios_serial"] = biosRows[0].SerialNumber
		device["serial_number"] = biosRows[0].SerialNumber
		hardware["motherboard"] = biosRows[0].Manufacturer
	}

	var diskRows []winDisk
	_ = wmi.Query("SELECT Index, Model, SerialNumber, Size, InterfaceType, Status FROM Win32_DiskDrive", &diskRows)

	var logicalRows []winLogical
	_ = wmi.Query("SELECT DeviceID, FileSystem, Size, FreeSpace FROM Win32_LogicalDisk WHERE DriveType=3", &logicalRows)

	disks := make([]map[string]any, 0, len(diskRows))
	for i, d := range diskRows {
		parts := []map[string]any{}
		// Phase 1: attach fixed drives only to first physical disk to avoid duplicate rows
		if i == 0 {
			for _, l := range logicalRows {
				total := int64(l.Size)
				free := int64(l.FreeSpace)
				used := total - free
				var pct float64
				if total > 0 {
					pct = float64(used) * 100.0 / float64(total)
				}
				parts = append(parts, map[string]any{
					"mount_point":   l.DeviceID + "\\",
					"filesystem":    l.FileSystem,
					"total_bytes":   total,
					"used_bytes":    used,
					"free_bytes":    free,
					"usage_percent": pct,
				})
			}
		}
		disks = append(disks, map[string]any{
			"disk_index":     int(d.Index),
			"model":          d.Model,
			"serial_number":  strings.TrimSpace(d.SerialNumber),
			"capacity_bytes": int64(d.Size),
			"interface_type": d.InterfaceType,
			"health_status":  d.Status,
			"partitions":     parts,
		})
	}

	var nicRows []winNIC
	_ = wmi.Query("SELECT Description, MACAddress, IPAddress, DefaultIPGateway, DNSServerSearchOrder, DHCPEnabled FROM Win32_NetworkAdapterConfiguration WHERE IPEnabled=True", &nicRows)

	nics := make([]map[string]any, 0, len(nicRows))
	for _, n := range nicRows {
		row := map[string]any{
			"name":           n.Description,
			"interface_type": "Ethernet",
			"mac_address":    n.MACAddress,
			"dhcp_enabled":   n.DHCPEnabled,
		}
		if len(n.IPAddress) > 0 {
			row["ipv4"] = n.IPAddress[0]
			if len(n.IPAddress) > 1 {
				row["ipv6"] = n.IPAddress[1]
			}
		}
		if len(n.DefaultIPGateway) > 0 {
			row["gateway"] = n.DefaultIPGateway[0]
		}
		if len(n.DNSServerSearchOrder) > 0 {
			row["dns"] = strings.Join(n.DNSServerSearchOrder, ",")
		}
		nics = append(nics, row)
	}

	return map[string]any{
		"agent_id":           agentID,
		"inventory_version":  1,
		"collected_at":       time.Now().Format(time.RFC3339),
		"device":             device,
		"hardware":           hardware,
		"disks":              disks,
		"network_interfaces": nics,
		"software":           collectInstalledSoftware(),
	}
}

func collectInstalledSoftware() []map[string]any {
	paths := []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	}
	seen := map[string]bool{}
	out := []map[string]any{}

	for _, p := range paths {
		k, err := registry.OpenKey(registry.LOCAL_MACHINE, p, registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		names, _ := k.ReadSubKeyNames(-1)
		for _, name := range names {
			sk, err := registry.OpenKey(k, name, registry.QUERY_VALUE)
			if err != nil {
				continue
			}
			display, _, _ := sk.GetStringValue("DisplayName")
			if strings.TrimSpace(display) == "" {
				sk.Close()
				continue
			}
			version, _, _ := sk.GetStringValue("DisplayVersion")
			publisher, _, _ := sk.GetStringValue("Publisher")
			loc, _, _ := sk.GetStringValue("InstallLocation")
			installDateRaw, _, _ := sk.GetStringValue("InstallDate")
			var installDate any
			// Registry sering format YYYYMMDD
			if len(installDateRaw) == 8 {
				installDate = installDateRaw[0:4] + "-" + installDateRaw[4:6] + "-" + installDateRaw[6:8]
			}
			key := display + "|" + version
			if seen[key] {
				sk.Close()
				continue
			}
			seen[key] = true
			item := map[string]any{
				"name":             display,
				"publisher":        publisher,
				"version":          version,
				"architecture":     runtime.GOARCH,
				"install_location": loc,
			}
			if installDate != nil {
				item["install_date"] = installDate
			}
			out = append(out, item)
			sk.Close()
		}
		k.Close()
	}
	return out
}
