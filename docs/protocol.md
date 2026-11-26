# SP-3000 FireWire protocol notes

Everything here is derived by the project, in our own words. No Fujifilm material is copied. Each fact carries its evidence and a confidence level. Empty sections mean "not known yet".

Status (2026-11-25): **nothing observed yet.** Fill in during Phases 0, 1 and 2.

## 1. Physical topology

| Link | From | To | Connector | Same bus as | Evidence | Confidence |
|---|---|---|---|---|---|---|
| FW-A | Scanner PC port ? | Image box / head | ? | ? | | |
| FW-B | MCU | Power supply unit | ? | ? | | |

Questions:
- How many 1394 controllers does the scanner PC have (one bus or two)?
- Which PCI controller (vendor:device ID) and driver?
- What bus ID does VDEntry assign, and does the software depend on it?

## 2. Nodes on the bus

One entry per node seen in self-ID / sysfs.

| Node | GUID | Vendor ID (OUI) | Vendor name | Model | Link speed | Power class | Role (root/IRM/BM) | Evidence |
|---|---|---|---|---|---|---|---|---|

## 3. Config ROM

Per node: bus info block, root directory, unit directories (specifier ID, version), text leaves. Keep the hexdump of our own capture in a private place; record only the decoded fields here.

## 4. Address space

CSR and vendor-specific addresses used by the host, with direction, size and meaning.

| Address | R/W/Lock | Size | Meaning | Evidence | Confidence |
|---|---|---|---|---|---|

## 5. Initialization sequence

Ordered list of transactions from bus reset to "scanner ready", with timing.

## 6. Commands

| Command | Transaction | Address | Payload layout | Response | Evidence | Confidence |
|---|---|---|---|---|---|---|

## 7. Image data transfer

- Asynchronous block reads, or isochronous stream? Channel, packet size, rate?
- Block layout, relation to the menu 2241 .bin/.tag dumps.
- Pixel-shift positions per frame, IR channel.

## 8. Error handling and abort

- How the host cancels a scan.
- How the scanner reports errors.

## 9. Open questions

- Does the second FireWire link (MCU to power supply) carry power/light commands?
- Asynchronous or isochronous image data?
- Do the GPA23 DSPs deliver processed or raw data? Can raw be requested?
- What does the .tag file contain?
- High-level or low-level commands?
- Is per-machine calibration data uploaded by the host at init? (A reinstall reportedly requires loading the scanner's own backup, which suggests yes; medium confidence.)
- Where does Fujifilm's "Image Intelligence" color processing run: in MS11 on the PC, or in the image box (GPA23 DSPs)? If it is in the PC software, static analysis can study it; if it is in the DSPs, raw capture bypasses it and the look must be rebuilt. Unknown.
