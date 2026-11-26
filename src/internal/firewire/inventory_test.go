package firewire

import (
	"bytes"
	"strings"
	"testing"
)

const sample = "\xef\xbb\xbf" + `{"drivers":[{"DeviceID":"PCI\\VEN_1106&DEV_3044&SUBSYS_30441106&REV_C0\\4&1C2B3A4&0&20F0","DriverVersion":"10.0.19041.1","DriverProviderName":"Microsoft","InfName":"1394.inf"}],
"devices":[
 {"Name":"Controlador de host compatível com OHCI 1394","PNPDeviceID":"PCI\\VEN_1106&DEV_3044&SUBSYS_30441106&REV_C0\\4&1C2B3A4&0&20F0","ClassGuid":"{6BDD1FC1-810F-11D0-BEC7-08002BE2092F}","Manufacturer":"VIA","Service":"1394ohci","Status":"OK","ConfigManagerErrorCode":0,"HardwareID":["PCI\\VEN_1106&DEV_3044"],"CompatibleID":null},
 {"Name":null,"PNPDeviceID":"1394\\FUJIFILM&SCANNER\\0814438C00017F2A","ClassGuid":null,"Manufacturer":null,"Service":null,"Status":"Error","ConfigManagerErrorCode":28,"HardwareID":["1394\\FUJIFILM&SCANNER","1394\\00A0B0&010203"],"CompatibleID":["1394\\00A0B0&010203"]}
]}`

func TestParseAndReport(t *testing.T) {
	inv, err := Parse([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Devices) != 2 || len(inv.Drivers) != 1 {
		t.Fatalf("parsed %+v", inv)
	}
	var out bytes.Buffer
	Report(&out, inv)
	text := out.String()
	for _, want := range []string{
		"compatível", "1394ohci, Microsoft 10.0.19041.1 (1394.inf)",
		"problem code 28 (no driver installed)", `1394\00A0B0&010203`,
		"FireWire controllers working: 1.", "Devices on the bus: 1.",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("report lacks %q:\n%s", want, text)
		}
	}
}

func TestEmptyInventory(t *testing.T) {
	inv, err := Parse([]byte(`{"devices":[],"drivers":[]}`))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	Report(&out, inv)
	if !strings.Contains(out.String(), "No FireWire controller found") {
		t.Fatal(out.String())
	}
	if _, err := Parse([]byte("PowerShell said no")); err == nil {
		t.Fatal("garbage accepted")
	}
}
