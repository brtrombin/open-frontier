# Hardware shopping list

Checked against the Linux kernel tree (master, 7.4-rc4) on 2026-11-25. Always confirm the chip on the actual card: listings often lie, and the same product code can ship with different chips. The PCI ID shown by `lspci -nn` is the final word.

## 1. Linux station (main PC)

| Item | Spec | Why |
|---|---|---|
| Platform | **Intel**, desktop, 12th to 14th gen Core i5 or similar | Avoids the VT630x + ASM1083 reboot quirk the kernel documents on AMD Ryzen (high confidence on the quirk) |
| Motherboard | At least **2 free PCIe slots** (x1 or wider), electrically connected to the CPU or chipset, not shared with M.2 | One FireWire card plus a spare or a second bus |
| RAM | 16 GB | Pixel-shift assembly of 8-position scans and whole-roll TIFFs |
| Storage | 1 TB NVMe | Raw captures are large |
| Network | Gigabit Ethernet | Web UI and file transfer |
| OS | Debian 13 (kernel 6.12 LTS) or Ubuntu 24.04 LTS | Pinned LTS kernel |

## 2. FireWire host card for the station (buy 2)

| Option | Chip | `lspci -nn` ID | Notes |
|---|---|---|---|
| **Preferred** | VIA **VT6315N**, native PCIe | `1106:3403` | Native PCIe, no bridge. Kernel applies `QUIRK_NO_MSI`, which is handled automatically. Example product: StarTech **PEX1394A2V2** (2x 6-pin FW400). Listings disagree on its chip (VT6315N vs TI), so verify before buying |
| Alternative | TI **XIO2213B** | `104c:823f` | FireWire 800 (1394b) controller. Cards usually have 9-pin ports, and some add one 6-pin port. A 9-pin to 6-pin bilingual cable connects to 1394a devices |
| VM path only | VIA VT6306/6307/6308 | `1106:3044` | The only chip validated in Chauvel's ESXi passthrough guide. On PCIe cards it sits behind a PCIe-to-PCI bridge. Do not use on AMD Ryzen with an ASM1083/1085 bridge |

Avoid: cards that only list "1394 OHCI" without a chip name, and Ricoh/JMicron/O2 combo cards (the kernel has quirks for them, so they are less predictable).

## 3. Bus sniffer for nosy (buy 1, 2 if you find them)

The kernel's nosy driver works **only** with the TI PCILynx link controller TSB12LV21, TSB12LV21A or TSB12LV21B. It is not OHCI. Source: `drivers/firewire/Kconfig`, option `FIREWIRE_NOSY`, and the PCI ID table in `drivers/firewire/nosy.c`, which matches only `104c:8000` (high confidence).

| Card | Bus | Notes |
|---|---|---|
| **Unibrain Fireboard 400 Lynx-2** | PCI (conventional) | TSB12LV21B link + TSB41AB3 PHY, 3 ports (2 external, 1 internal). Do **not** buy the "FireBoard 400-OHCI", which uses TSB12LV23/26 and does not work with nosy |
| **IOI IOI-1394TT** | PCI (conventional) | 3-port PCILynx-2. Appears to be the same hardware as the Fireboard 400 Lynx-2 (medium confidence, matching specs) |
| Newer Technology FireWire 2 Go | CardBus | Needs an old laptop with a CardBus slot |
| Apple Power Mac G3 blue & white, G4 with PCI graphics | onboard | Would need Linux on PowerPC; last resort |

Check after install: `lspci -nn` must show `104c:8000`, and `modinfo nosy` must succeed. The user-space tool `nosy-dump` is in the kernel sources under `tools/firewire/`. Build it from the source tree matching your kernel.

## 4. Sniffer PC (and Windows 2000 spare)

| Item | Spec | Why |
|---|---|---|
| Used desktop | Intel H61, B75 or Q77 chipset, with at least one **conventional PCI** slot | Hosts the PCILynx card natively, no adapter. The same class of machine (H61, 4 cores, 8 GB, SATA SSD) was cited as a native Windows 2000 replacement for the Toshiba |
| Spare SATA SSD | 120 GB or more | So the Windows 2000 spare and the Linux sniffer use separate disks |

If you really must use a modern PC, a PCIe-to-PCI adapter card can host the PCILynx, but that path is untested here (low confidence that nosy works through every bridge).

## 5. Cables

| Item | Qty | Notes |
|---|---|---|
| FireWire 400, 6-pin to 6-pin, 1.8 m | 4 | Confirm in Phase 0 that the scanner side is 6-pin (photograph it) |
| FireWire 9-pin to 6-pin (bilingual), 1.8 m | 2 | Only if you use the TI XIO2213B card |

## 6. Backup and preservation

| Item | Qty | Notes |
|---|---|---|
| External drive, 2 TB | 2 | Disk images and captures, stored in two different places |
| USB stick, 8 GB or more | 2 | Clonezilla Live **i686** (32-bit) and a FAT32 stick for the collection kit |
| IDE/SATA to USB 3.0 adapter | 1 | Lets you image the Toshiba disk on another PC (the Toshiba may only have USB 1.1). Check whether its disk is IDE or SATA first |
| Spare disk of the same size or larger | 1 | For the restore test, the only real proof the backup works |

## 7. Electrical

| Item | Qty | Notes |
|---|---|---|
| Grounded power strip with surge protection | 1 | Scanner PC, Linux PC and sniffer on the **same** strip. Check that it handles the scanner's load (up to 10 A at 100 V, about 4.2 A at 240 V), or keep the scanner on its own circuit and only the PCs on the strip, sharing the same ground |
