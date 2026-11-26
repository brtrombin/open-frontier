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

Not FireWire, but the other interface of the scanner PC, and the one existing open projects already use. Facts below come from reading two public repositories (journal 2026-11-26, "standalone projects") and the plain-text documentation and support tools shipped with Sprocket 1.25.1 (journal 2026-11-26, "Sprocket docs"), described in our own words. No code copied.

### 9.1 Spool folders and shares

- **Spool path is a registry key** on each Fuji PC: `HKEY_LOCAL_MACHINE\SOFTWARE\FUJIFILM\FDIA\InputSpool` and `...\OutputSpool`. The value is either a local path (`D:\InSpool`, meaning the scanner writes to its own disk) or a network path (`\\SCANSTATION\InSpool`, meaning it writes to the export PC). Editing these keys and restarting the Fuji PC changes where it writes. Verified on FDIA V5.5 clients (high confidence from Sprocket docs). The SP-500 uses `SOFTWARE\FUJIFILM\FE\KIOSK\SPOOLPATH` instead (medium confidence).
- **InSpool / OutSpool split**: in some setups the .INF lives beside the .RAW in the same InSpool folder (SSI-era: `000007.RAW` / `000007.INF`). In other setups (MS01-era, Fuji's own software) the scanner writes the .RAW into InSpool as `00000007.RAW` (8-digit padding) and the .INF into OutSpool as `000007.INF` (6-digit padding). The split depends on the F-DIA software version, not the scanner model (medium confidence from Sprocket docs).
- **SMB1**: the Fuji PC speaks SMB1 only. Any replacement host must accept SMB1 connections (high confidence).
- **Factory credentials**: the factory SMB account is FRONTIER/FRONTIER (high confidence, from fujisan and Sprocket docs).

### 9.2 RAW image files

- 32-byte header starting with ASCII `DR`, followed by `width * height * 3` bytes of 8-bit RGB pixel data. Width and height are the 5th and 6th little-endian 16-bit words of the header (medium confidence from fujisan). Sprocket verifies file completeness by computing `32 + width * height * 3` from the .INF's dimensions and comparing against the actual file size, verified on 335 real archived frames (high confidence from Sprocket docs).
- **The RAW already contains the Fuji color rendering.** Sprocket's documentation states: "the SP-3000 bakes its color rendering into the RAW, so the default decode already is the Frontier look" and that applying a 3D LUT is only needed to get a *different* grade. This means the "Image Intelligence" processing happens before the data reaches InSpool, not in the export software (high confidence for the RAW content, medium confidence for where the processing runs, since it could be in the DSPs, in the scanner PC software, or both).
- Typical sizes: about 26 MB for a 2503x3637 normal scan, about 57 MB for a 5444x3649 large scan, about 40 MB per frame average (high confidence from Sprocket docs).

### 9.3 Per-frame .INF files (DIPP format)

The .INF is a Fuji "DIPP" (Digital Imaging Processing Protocol) record: a sequence of typed key/value entries. Layout (all little-endian): an ASCII key (NUL-terminated), then a 2-byte record size, a 2-byte type code, a 4-byte count or length, and the value bytes. Type codes: 0x03 = scalar u32, 0x13/0x1013 = u32 array, 0x1F = UTF-16LE string, 0x40 = Windows FILETIME (8 bytes). High confidence from Sprocket's `inf_parser.py`.

Known fields (key names are the ASCII strings in the file):

| Key | Type | Meaning |
|---|---|---|
| `00_MODEL_NAME` | string | Scanner model |
| `00_SERIAL_NO` | string | Scanner serial number |
| `00_SYSTEM_VERSION` | string | System software version |
| `00_DIPP_VERSION` | string | DIPP protocol version |
| `00_VENDOR_ID` | string | Vendor identifier |
| `02_FRAME_NUMBER` | u32 | Numeric frame index |
| `02_FRAME_NUMBER_STRING` | string | Film frame designation as printed on the film (e.g. "01A") |
| `02_ROTATION_FOR_PROPER_RECTANGLE` | u32 | Rotation for correct orientation |
| `01_FILMKIND` | u32 | Film type code |
| `01_SORT_NUMBER` | u32 | Sort/sequence number |
| `02_DX` | u32 | DX code from the film canister |
| `02_MASKKIND` | u32 | Mask/carrier type |
| `02_FILE_SOURCE` | u32 | Source identifier |
| `02_SCENE_TYPE` | u32 | Scene type |
| `02_SOURCE_SIZE` | u32 array | Source dimensions |
| `02_SCAN_SIZE` | u32 array | Scan dimensions |
| `02_FILE_SIZE` | u32 array | Pixel dimensions as (H, W) |
| `00_REVISION_DATE` | FILETIME | Revision date |
| `02_SCANNING_DATE` | FILETIME | Date and time of scan |

This supersedes the earlier fujisan finding that read the frame name at offset 0x3CC and rotation at 0x400. Those fixed offsets were an approximation; the DIPP format is key-based and positions vary between firmware versions (medium-high confidence).

### 9.4 CdOrder.INF

Text file with lines starting with `Frame`, giving per-image index, a rotation code and the film frame number. fujisan maps rotation codes 0..4 to 0, 270, 0, 180, 0 degrees (low-medium confidence). On the SP-500, `CdOrder.INF` ends with `EndOrder` and is the signal that an order is complete (high confidence from Sprocket docs).

### 9.5 Order database (FDIA_DB)

Microsoft SQL Server (MSDE/SQL Server Express), database name `FDIA_DB`. One database per scanner (high confidence). The Fuji software reaches it through named pipes by default; TCP/IP is off and must be enabled with `SVRNETCN.exe` for network access (high confidence from Sprocket docs). There is one database per independent scanner system; folder counters repeat across systems, and two scanners sharing one database get cross-contaminated order names (high confidence from Sprocket docs).

**Tables** (column names from public repositories and Sprocket's database setup documentation, in our own words):

| Table | Key columns | Purpose |
|---|---|---|
| `OrderTable` | FDIAManageID (PK), OrderID, InputDevice, OutputDevice, Date, Time, Priority, Status, InputServiceNo | One row per order |
| `OtherTable` | FileID, FDIAManageID, FilePath (points to CdOrder.INF), Status | Supplementary files per order |
| `ImageTable` | ImageID + FDIAManageID (PK), FilePath, FileName, INFPath, OrgPixelH, OrgPixelW, Para_D/C/M/Y, Step_dc/dm/dy/c/m/y, AeHigh/AeMiddle/AeLow, AwbRed/AwbGreen/AweBlue, Aepaecn1-8, UsmParam, ColorSpace, FileFormat, Status | Per-frame image data, including CMYD color parameters and auto-exposure |
| `ImageInfTable` | FDIAManageID, ImageID, ThumbnailPath | Thumbnail references |
| `DeviceTable` | DeviceName, DeviceAttribute, FileFormat, ColorSpace, Service, Status | Registered devices (menu entries) |
| `DeviceInfTable` | DeviceName (PK), InfFileName | Device configuration file paths |
| `DeviceStatusTable` | DeviceName (PK), StsFileName | Device status file paths |
| `ServiceTable` | DeviceName, ServiceNo, ServiceName, SetNumber | Service menu entries |
| `SizeTable` | SizeName, DeviceName, Hmm, Wmm, Hpixel, Wpixel, SetNumber, Quality, Resize | Output dimensions per format and device |
| `CombinationTable` | InDeviceColor, InDeviceFormat, OutDeviceColor, OutDeviceFormat | Valid input/output format combinations |
| `ColorSpace` | ColorSpaceID, ColorSpace | Color space names |
| `FileFormat` | FileFormatID, FileFormat | Format names: 1=BMP, 2=Exif, 3=FPX, 4=F-RAW, 5=JPEG, 6=TIFF, 7=PSD, 8=FEO |
| `DESCRIPTOR` | FDIAManageIDMAX | Order counter (recycles; MaxFdiaNo = 10000 on every system examined) |
| `LogTable` | FDIAManageID, OrderID, Date, Time, Status | Order log |
| `OrderInfTable` | OrderNo + DeviceName (PK), InfFileName | Per-order device configuration |
| `SubOrderTable` | FDIAManageID, LockInfo, LockDevice | Order locking |

**ImageTable CMYD columns**: `Para_D`, `Para_C`, `Para_M`, `Para_Y` (smallint) are per-frame density/color adjustment parameters. `Step_dc`, `Step_dm`, `Step_dy`, `Step_c`, `Step_m`, `Step_y` (smallint) are step adjustments. `AeHigh`, `AeMiddle`, `AeLow` (real) are auto-exposure levels. `AwbRed`, `AwbGreen`, `AweBlue` (real, note the typo "Awe" not "Awb" for blue) are auto white balance. `Aepaecn1` through `Aepaecn8` (real) and `UsmParam` (int, unsharp mask) round out the per-frame processing parameters. High confidence on column names from Sprocket's schema.

**Order status codes**: 0 = error, 2 = scan finished / ready to import, 4 = waiting to write, 5 = completed (high confidence, confirmed by both fujisan and Sprocket docs).

**Standard devices** (from seed data): PIC_SCAN, PIC_TIFF (both output attribute 2, FileFormat 4 = F-RAW), PRINT, SP3000_F1, SP3000_F2, SP3000_input (all input attribute 1), SP3000sRGB (output attribute 2, service 1). High confidence from Sprocket seed tables.

### 9.6 Scanner output dimensions

From the SizeTable seed data shipped with Sprocket (high confidence). Width is always along the film strip (305 mm physical for 35mm film).

| Size name | Format | H pixels | W pixels | H mm | W mm |
|---|---|---|---|---|---|
| DSC | PIC_SCAN / PIC_TIFF | 1074 | 1430 | 0 | 0 |
| 135N | SP3000sRGB | 2504 | 3638 | 209 | 305 |
| 135H | SP3000sRGB | 2736 | 3638 | 228.7 | 305 |
| 135L | SP3000sRGB | 5444 | 3650 | 457 | 305 |
| 135S | SP3000sRGB | 1346 | 2004 | 111 | 166.6 |
| 645N | SP3000sRGB | 4030 | 3048 | 337.2 | 254 |
| 645L | SP3000sRGB | 4948 | 3650 | 415 | 305 |
| 6X6N | SP3000sRGB | 3036 | 3036 | 254 | 254 |
| 6X6L | SP3000sRGB | 3638 | 3638 | 305 | 305 |
| 6X7N | SP3000sRGB | 3716 | 3048 | 310.6 | 254 |
| 6X7L | SP3000sRGB | 4452 | 3650 | 373 | 305 |
| 6X9L | SP3000sRGB | 5444 | 3650 | 457 | 305 |
| 110N | SP3000sRGB | 3814 | 2930 | 319 | 244 |
| 110 | SP3000sRGB | 4760 | 3650 | 399 | 305 |
| APS-P | SP3000sRGB | 2098 | 3638 | 174.7 | 305 |
| 35XL | SP3000sRGB | 8110 | 5428 | 682.7 | 455.6 |

Note: the L (large) sizes at 3650 W pixels vs N (normal) at 3638 suggest different scan modes, possibly related to pixel shift. Adre Lanine's medium output of 5573 x 3722 is larger than any single-pass size here, confirming pixel-shift assembly on the PC side (medium confidence).

### 9.7 F-DIA client library dialects

Two versions of the F-DIA client library exist: dialect "41" (SSI-era) and dialect "45" (MS01-era, Fuji's own software). The scanner's SQL queries are fixed inside its client library, so any database emulator must present the schema layout the client expects (medium confidence from Sprocket docs).

### 9.8 SP-500 differences

The SP-500 writes finished JPEG or TIFF files plus a `.li` sidecar per frame into `ImgExchange` (not InSpool). There is no .RAW decode step. `CdOrder.INF` ending with `EndOrder` signals order completion. Order names come from the `.li` file, not the database. No FDIA_DB is needed for naming. The SP-500 uses `SOFTWARE\FUJIFILM\FE\KIOSK\SPOOLPATH` for its spool path. High confidence from Sprocket docs.

### 9.9 Safety rules for the server side

These are confirmed hazards, not hypothetical. Any tool that touches InSpool or FDIA_DB must respect them.

1. **Never open InSpool files before Status == 2.** Reading files while the scanner is still writing causes a sharing violation that aborts `ExecLSeqObj.exe` ("abnormal program termination") and wedges the scanner on "Printing External File". On Windows, files must be opened with `FILE_SHARE_DELETE` (`FILE_SHARE_ALL`) to avoid blocking the Fuji software's own rename/delete (high confidence from Sprocket docs, confirmed as an observed failure).
2. **Never delete spool folders.** Deleting an order folder the scanner still owns is the suspected cause of real scanner hangs. A missing folder produces Fuji error W-5302 ("insufficient disk space") when the operator clicks Continue, because the error really means "the folder is gone" (high confidence from Sprocket docs).
3. **Never write to FDIA_DB.** Sprocket only reads; fujisan writes Status=5 and deletes rows, but that is risky. The scanner itself sets Status=2 and clears "Printing External File" with no importer running (verified by scanning with SSI stopped, high confidence from Sprocket docs).
4. **Never run the collection kit during a scan.** Even read-only access to InSpool can trigger the sharing violation above.
5. **The scanner releases itself.** The SP-3000 finishes its own write, sets Status=2 and clears "Printing External File" by itself. Nothing an importer does to InSpool frees a scanner that has not already freed itself (high confidence from Sprocket docs, verified experimentally).

## 10. Open questions

- Does the second FireWire link (MCU to power supply) carry power/light commands?
- Asynchronous or isochronous image data?
- Do the GPA23 DSPs deliver processed or raw data? Can raw be requested?
- What does the .tag file contain?
- High-level or low-level commands?
- Is per-machine calibration data uploaded by the host at init? (A reinstall reportedly requires loading the scanner's own backup, which suggests yes; medium confidence.)
- ~~Where does Fujifilm's "Image Intelligence" color processing run?~~ **Partially answered (2026-11-26)**: the RAW files in InSpool already contain the Fuji color rendering ("the SP-3000 bakes its color rendering into the RAW"). This means processing happens before the data reaches InSpool. It could be in the GPA23 DSPs, in the scanner PC software (ExecLSeqObj.exe or similar), or split between both. For our project: if we capture the same data the scanner PC receives over FireWire, it may still be pre-rendered (which would be simpler for us), or we may get truly raw CCD data that requires the full pipeline. Static analysis and comparing FireWire captures against InSpool .RAW files will answer this. Medium confidence.
- What do the ImageTable CMYD parameters (`Para_D/C/M/Y`, `Step_*`, `Ae*`, `Awb*`, `Aepaecn*`, `UsmParam`) represent numerically? Are they adjustments applied by the operator at the scanner, or computed by the software? They could be the key to reproducing fine color adjustments.
- The two F-DIA client library dialects (41 vs 45): which does our scanner use, and what are the protocol differences?
