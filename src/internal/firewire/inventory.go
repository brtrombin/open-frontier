// Package firewire lists the IEEE 1394 controllers and devices that Windows
// has enumerated. It only reads the Plug and Play database; it sends nothing
// on the bus.
package firewire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Device is one Plug and Play device, as Win32_PnPEntity reports it.
type Device struct {
	Name          string   `json:"Name"`
	DeviceID      string   `json:"PNPDeviceID"`
	ClassGUID     string   `json:"ClassGuid"`
	Manufacturer  string   `json:"Manufacturer"`
	Service       string   `json:"Service"`
	Status        string   `json:"Status"`
	Problem       int      `json:"ConfigManagerErrorCode"`
	HardwareIDs   []string `json:"HardwareID"`
	CompatibleIDs []string `json:"CompatibleID"`
}

// Driver is the installed driver of a device, as Win32_PnPSignedDriver reports it.
type Driver struct {
	DeviceID string `json:"DeviceID"`
	Version  string `json:"DriverVersion"`
	Provider string `json:"DriverProviderName"`
	Inf      string `json:"InfName"`
}

// Inventory is what Collect gathers.
type Inventory struct {
	Devices []Device `json:"devices"`
	Drivers []Driver `json:"drivers"`
}

const hostControllerClass = "{6bdd1fc1-810f-11d0-bec7-08002be2092f}"

// Parse decodes the JSON that the PowerShell query prints.
func Parse(data []byte) (Inventory, error) {
	data = bytes.TrimSpace(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf")))
	var inv Inventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return inv, fmt.Errorf("unexpected PowerShell output: %v", err)
	}
	return inv, nil
}

var problems = map[int]string{
	1: "not configured", 3: "driver may be corrupted", 10: "cannot start",
	12: "not enough resources", 18: "drivers need reinstalling", 19: "registry problem",
	22: "disabled", 24: "not present", 28: "no driver installed",
	31: "not working properly", 37: "driver failed to start", 39: "driver missing or corrupted",
	43: "stopped after reporting problems", 45: "not connected", 48: "driver blocked",
	52: "driver signature not verified",
}

// Report prints the inventory and what it means for the project.
func Report(w io.Writer, inv Inventory) {
	drivers := map[string]Driver{}
	for _, d := range inv.Drivers {
		drivers[strings.ToUpper(d.DeviceID)] = d
	}
	var hosts, bus, other []Device
	for _, d := range inv.Devices {
		id := strings.ToUpper(d.DeviceID)
		switch {
		case strings.HasPrefix(id, `1394\`):
			bus = append(bus, d)
		case strings.EqualFold(d.ClassGUID, hostControllerClass):
			hosts = append(hosts, d)
		default:
			other = append(other, d)
		}
	}
	section(w, "IEEE 1394 (FireWire) controllers", hosts, drivers)
	section(w, "Devices on the FireWire bus", bus, drivers)
	section(w, "Virtual and protocol devices (V1394, SBP2, 61883, AVC)", other, drivers)

	fmt.Fprintln(w, "Summary:")
	working := 0
	for _, h := range hosts {
		if h.Problem == 0 {
			working++
		}
	}
	switch {
	case len(hosts) == 0:
		fmt.Fprintln(w, "- No FireWire controller found: this PC cannot reach the scanner as it is.")
	case working == len(hosts):
		fmt.Fprintf(w, "- FireWire controllers working: %d.\n", working)
	default:
		fmt.Fprintf(w, "- FireWire controllers: %d, of which %d report a problem (see above).\n", len(hosts), len(hosts)-working)
	}
	if len(bus) == 0 {
		fmt.Fprintln(w, "- No device on the FireWire bus (expected while the scanner is not connected).")
	} else {
		fmt.Fprintf(w, "- Devices on the bus: %d. Their hardware IDs come from each device's configuration ROM.\n", len(bus))
	}
	for _, d := range other {
		id := strings.ToUpper(d.DeviceID)
		if strings.HasPrefix(id, `SBP2\`) || strings.HasPrefix(id, `61883\`) || strings.HasPrefix(id, `AVC\`) {
			fmt.Fprintln(w, "- A Windows protocol driver is attached to a FireWire device (SBP2, 61883 or AVC).")
			fmt.Fprintln(w, "  If that device is the scanner, power everything down and report it before going on.")
			break
		}
	}
}

func section(w io.Writer, title string, list []Device, drivers map[string]Driver) {
	fmt.Fprintf(w, "%s:\n", title)
	if len(list) == 0 {
		fmt.Fprintln(w, "  (none)")
	}
	for _, d := range list {
		name := d.Name
		if name == "" {
			name = "(no name)"
		}
		fmt.Fprintf(w, "  %s\n", name)
		fmt.Fprintf(w, "    id          %s\n", d.DeviceID)
		status := d.Status
		if d.Problem != 0 {
			text := problems[d.Problem]
			if text == "" {
				text = "see Device Manager"
			}
			status = fmt.Sprintf("%s, problem code %d (%s)", d.Status, d.Problem, text)
		}
		fmt.Fprintf(w, "    status      %s\n", status)
		if drv, ok := drivers[strings.ToUpper(d.DeviceID)]; ok {
			fmt.Fprintf(w, "    driver      %s, %s %s (%s)\n", d.Service, drv.Provider, drv.Version, drv.Inf)
		} else if d.Service != "" {
			fmt.Fprintf(w, "    driver      %s\n", d.Service)
		}
		if d.Manufacturer != "" {
			fmt.Fprintf(w, "    maker       %s\n", d.Manufacturer)
		}
		list := func(label string, ids []string) {
			for i, id := range ids {
				if i == 0 {
					fmt.Fprintf(w, "    %-11s %s\n", label, id)
				} else {
					fmt.Fprintf(w, "                %s\n", id)
				}
			}
		}
		list("hardware", d.HardwareIDs)
		list("compatible", d.CompatibleIDs)
	}
	fmt.Fprintln(w)
}
