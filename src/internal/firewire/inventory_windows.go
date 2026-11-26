//go:build windows

package firewire

import (
	"fmt"
	"os/exec"
	"strings"
)

// One line, single quotes only: it travels as one command-line argument.
const query = `try { [Console]::OutputEncoding = New-Object System.Text.UTF8Encoding $false } catch {}; ` +
	`$ErrorActionPreference = 'Stop'; ` +
	`$hc = '` + hostControllerClass + `'; ` +
	`$d = @(Get-CimInstance Win32_PnPEntity | Where-Object { ($_.PNPDeviceID -match '^(1394|V1394|SBP2|61883|AVC)\\') -or ($_.ClassGuid -eq $hc) } | ` +
	`Select-Object Name, PNPDeviceID, ClassGuid, Manufacturer, Service, Status, ConfigManagerErrorCode, HardwareID, CompatibleID); ` +
	`$ids = @($d | ForEach-Object { $_.PNPDeviceID }); ` +
	`$v = @(Get-CimInstance Win32_PnPSignedDriver | Where-Object { $ids -contains $_.DeviceID } | ` +
	`Select-Object DeviceID, DriverVersion, DriverProviderName, InfName); ` +
	`ConvertTo-Json -InputObject @{ devices = $d; drivers = $v } -Depth 4 -Compress`

// Collect asks Windows, through PowerShell, for its 1394 devices and drivers.
func Collect() (Inventory, error) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", query).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return Inventory{}, fmt.Errorf("PowerShell query failed: %v: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return Inventory{}, fmt.Errorf("could not run PowerShell: %v", err)
	}
	return Parse(out)
}
