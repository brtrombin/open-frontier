# SP-3000 FireWire protocol notes

Everything here is derived by the project, in our own words. No Fujifilm material is copied. Each fact carries its evidence and a confidence level. Empty sections mean "not known yet".

Status (2026-11-25): **nothing observed yet.** Fill in during Phases 0, 1 and 2.

## 1. Physical topology

| Link | From | To | Connector | Same bus as | Evidence | Confidence |
|---|---|---|---|---|---|---|
| FW-A | Scanner PC port ? | MCU1 | ? | ? | Owner report, Reddit 1m6cqib (journal 2026-11-26) | medium |
| FW-B | Scanner PC port ? | MCU2 | ? | ? | Same | medium |
| FW-? | Scanner PC port ? | Power unit (triggers scanner power-up when the host is on) | ? | ? | Chauvel guide; probably one of the two links above | medium |
| FW-? | Scanner PC port ? | Image controller on the scan head (main data link) | ? | ? | Chauvel guide; probably the other link | medium |

Notes: in Chauvel's original setup the two links used two controllers; one two-port card also worked, so one bus is enough (medium confidence).

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

## 9. Server side: InSpool, OutSpool and the order database

Not FireWire, but the other interface of the scanner PC, and the one existing open projects already use. Facts below come from reading two public repositories (see journal 2026-11-26, "standalone projects"), described in our own words. No code copied.

- **Shares**: the scanner side writes each scanned frame into an `InSpool` share and the server side uses an `OutSpool` share, over SMB with the factory account FRONTIER/FRONTIER (fujisan mounts both from a Mac). On an MS01 setup, `D:\Inspool` lives on the front-end PC (photolaundry's hot previewer). In the fujisan setup both shares and the database were on the Toshiba. Where they live probably depends on the software version (medium confidence).
- **Order database**: Microsoft SQL Server (MSDE/SQL 2005) database `FDIA_DB`. Tables used: `OrderTable` (FDIAManageID, OrderID, Status, InfFileName), `OtherTable` (FilePath, which points to a `CdOrder.INF`), `InputImageTable` (ImageID, InfFileName), `OutputImageTable` (ImageID, FileName), plus `ImageTable`, `OutputEzTable`, `Old_convertInformationTable`. High confidence that these names exist on that setup.
- **Order status codes** (per fujisan's comments): 0 = error, 2 = waiting to convert, 4 = waiting to write, 5 = completed. A replacement front end picks up status 2, exports, sets 5, then deletes the order rows and its spool folders.
- **RAW image files in InSpool**: plain RGB pixel data after a 32-byte header that starts with the ASCII bytes `DR`; width and height are the 5th and 6th little-endian 16-bit words of the header; pixels are read as 8-bit RGB in fujisan (medium confidence on bit depth: TIFF/16-bit setups may differ). photolaundry also describes them as "plain RGB data with custom headers".
- **Per-frame `.INF` files**: fujisan reads a 6-byte frame name (the number printed on the film) at offset `0x3CC` and 8 bytes of rotation data at offset `0x400` (low-medium confidence; exploratory script).
- **`CdOrder.INF`**: text file with lines starting with `Frame`, giving per-image index, a rotation code and the film frame number. fujisan maps rotation codes 0..4 to 0, 270, 0, 180, 0 degrees (low-medium confidence).

## 10. Open questions

- Does the second FireWire link (MCU to power supply) carry power/light commands?
- Asynchronous or isochronous image data?
- Do the GPA23 DSPs deliver processed or raw data? Can raw be requested?
- What does the .tag file contain?
- High-level or low-level commands?
- Is per-machine calibration data uploaded by the host at init? (A reinstall reportedly requires loading the scanner's own backup, which suggests yes; medium confidence.)
- Where does Fujifilm's "Image Intelligence" color processing run: in MS11 on the PC, or in the image box (GPA23 DSPs)? If it is in the PC software, static analysis can study it; if it is in the DSPs, raw capture bypasses it and the look must be rebuilt. Unknown.
