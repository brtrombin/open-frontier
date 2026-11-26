# SP-3000 FireWire protocol notes

Everything here is derived by the project, in our own words. No Fujifilm material is copied. Each fact carries its evidence and a confidence level. Empty sections mean "not known yet".

Status (2026-11-26): **nothing observed on FireWire yet.** Server-side protocol substantially documented from public docs. Spool file decoding is implemented in `src/` (our own code), not yet checked on real files. Fill in during Phases 0, 1 and 2.

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
- What bus ID does VDEntry assign, and does the software depend on it? Hypothesis (low-medium): VDEntry adds a Windows 1394 "virtual device", so the Fuji driver addresses the scanner node directly instead of binding to its unit directory (`docs/plan-review.md`, point 15).

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
- **InSpool / OutSpool split**: in some setups the .INF lives beside the .RAW in the same InSpool folder (SSI-era: `000007.RAW` / `000007.INF`). In other setups (MS01-era, Fuji's own software) the scanner writes the .RAW into InSpool as `00000007.RAW` (8-digit padding) and the .INF into OutSpool as `000007.INF` (6-digit padding). The split depends on the F-DIA software version, not the scanner model (medium confidence from Sprocket docs). Without the .INF, pictures come out rotated wrong and incremental export cannot start, because the .INF carries rotation and dimensions needed before decode (high confidence from Sprocket docs). Sprocket auto-discovers OutSpool: if a folder called OutSpool exists beside InSpool, it finds it automatically (medium confidence).
- **The scanner writes the .INF after the .RAW.** A frame is "done" once its RAW file has been unchanged for `frame_settle_seconds` AND its .INF exists. Important for file-watching logic (high confidence from Sprocket docs).
- **SMB1**: the Fuji PC speaks SMB1 only. Any replacement host must accept SMB1 connections (high confidence).
- **Factory credentials**: the factory SMB account is FRONTIER/FRONTIER (high confidence, from fujisan and Sprocket docs).

### 9.2 RAW image files

- 32-byte header starting with ASCII `DR`, followed by `width * height * 3` bytes of 8-bit RGB pixel data. Width and height are the 5th and 6th little-endian 16-bit words of the header (medium confidence from fujisan). Sprocket verifies file completeness by computing `32 + width * height * 3` from the .INF's dimensions and comparing against the actual file size, verified on 335 real archived frames (high confidence from Sprocket docs).
- **The RAW already contains the Fuji color rendering.** Sprocket's documentation states: "the SP-3000 bakes its color rendering into the RAW, so the default decode already is the Frontier look" and that applying a 3D LUT is only needed to get a *different* grade. This means the "Image Intelligence" processing happens before the data reaches InSpool, not in the export software (high confidence for the RAW content, medium confidence for where the processing runs, since it could be in the DSPs, in the scanner PC software, or both).
- Typical sizes: about 26 MB for a 2503x3637 normal scan, about 57 MB for a 5444x3649 large scan, about 40 MB per frame average (high confidence from Sprocket docs).
- Not confirmed on real files yet: which header words hold width and height, whether the pixels are interleaved (RGBRGB...) or in three planes, and the channel order. `openfrontier inspect` prints all 16 header words and ranks the possible layouts by the difference between samples one image row apart (the true row length gives the smallest); `openfrontier compare` matches a frame against the Fuji software's output in all eight orientations and both channel orders.

### 9.3 Per-frame .INF files (DIPP format)

The .INF is a Fuji "DIPP" (Digital Imaging Processing Protocol) record: a sequence of typed key/value entries. Layout (all little-endian): an ASCII key (NUL-terminated), then a 2-byte record size, a 2-byte type code, a 4-byte count or length, and the value bytes. Type codes: 0x03 = scalar u32, 0x13/0x1013 = u32 array, 0x1F = UTF-16LE string, 0x40 = Windows FILETIME (8 bytes). High confidence from Sprocket's `inf_parser.py`. That file is not consistent about 0x13: its description calls it an array, its decoder reads it as a scalar, and it also mentions 0x08 as a u8. Our decoder reads 0x13 as a u32 array for the size keys (`02_SOURCE_SIZE`, `02_SCAN_SIZE`, `02_FILE_SIZE`) and as a scalar otherwise, and lists every key it finds, known or not, so a real file settles it (medium confidence until then).

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

**Connection**: MSDE/SQL Server Express ships with TCP/IP disabled. The Fuji software reaches its database through **named pipes**, not TCP/IP, so no site ever needed to enable TCP/IP. `SVRNETCN.exe` enables it; restart MSSQLSERVER service afterward (high confidence from Sprocket docs). SQL Server **default instance** is required: the scanner's software looks for the server by its plain computer name, and a named instance (`COMPUTERNAME\SQLEXPRESS`) cannot be found without extra configuration (high confidence). Mixed Mode authentication is required (the scanner logs in with a SQL login, not Windows auth). `CHECK_POLICY = OFF` on the SQL login: without it, modern password policies reject the scanner's password in a way that looks unrelated (high confidence). The login gets `db_owner` role on the database (high confidence).

**Factory database password**: `radiogaga`, the same across Fuji's own software, treated as a default, not a secret (high confidence from Sprocket docs).

**Tables** (column names from public repositories and Sprocket's database setup documentation, in our own words):

| Table | Key columns | Purpose |
|---|---|---|
| `OrderTable` | FDIAManageID (int, PK), OrderID (nvarchar 17), InputDevice (nvarchar 32), OutputDevice (nvarchar 32), Date (nvarchar 11), Time (nvarchar 9), Priority (int), Status (int), InputServiceNo (int), FreeString (nvarchar 65), OrderSize (float), DeleteDate (nvarchar 11), DeleteTime (nvarchar 9), HoldStatus (int) | One row per order. Date/Time stored as strings, not datetime |
| `OtherTable` | FileID, FDIAManageID, FilePath (points to CdOrder.INF), Status | Supplementary files per order |
| `ImageTable` | ImageID + FDIAManageID (PK), FilePath, FileName, INFPath, OrgPixelH, OrgPixelW, Para_D/C/M/Y, Step_dc/dm/dy/c/m/y, AeHigh/AeMiddle/AeLow, AwbRed/AwbGreen/AweBlue, Aepaecn1-8, UsmParam, ColorSpace, FileFormat, Status, Attribute, BackPrint1/2 (nvarchar 41), Border, Resize, Upper, SourceServiceAttribute, ColorTableFlag, ColorTableNo, FreeString (nvarchar 65), Size (nvarchar 33), Sheets, FileSize (float), ErrorCode, OriginalFileName (nvarchar 256), ImageSize, Quality, ConvertFlg | Per-frame image data, including CMYD color parameters, auto-exposure and additional processing flags |
| `ImageInfTable` | FDIAManageID, ImageID, ThumbnailPath | Thumbnail references |
| `DeviceTable` | DeviceName, DeviceAttribute, FileFormat, ColorSpace, Service, Status | Registered devices (menu entries) |
| `DeviceInfTable` | DeviceName (PK), InfFileName | Device configuration file paths |
| `DeviceStatusTable` | DeviceName (PK), StsFileName | Device status file paths |
| `ServiceTable` | DeviceName, ServiceNo, ServiceName, SetNumber | Service menu entries |
| `SizeTable` | SizeName, DeviceName, Hmm, Wmm, Hpixel, Wpixel, SetNumber, Quality, Resize | Output dimensions per format and device |
| `CombinationTable` | InDeviceColor, InDeviceFormat, OutDeviceColor, OutDeviceFormat | Valid input/output format combinations |
| `ColorSpace` | ColorSpaceID, ColorSpace | Color space names |
| `FileFormat` | FileFormatID, FileFormat | Format names: 1=BMP, 2=Exif, 3=FPX, 4=F-RAW, 5=JPEG, 6=TIFF, 7=PSD, 8=FEO |
| `DESCRIPTOR` | FDIAManageIDMAX | Order counter (starts at 0, first order becomes 1; recycles at MaxFdiaNo = 10000 on every system examined) |
| `LogTable` | FDIAManageID, OrderID, InputDevice (nvarchar 32), OutputDevice (nvarchar 32), Date, Time, Status, FreeString (nvarchar 65), Priority, OrderSize (float), DeleteDate, DeleteTime, InputServiceNo, HoldStatus | Order log (mirrors OrderTable, appears to be an audit/history table) |
| `OrderInfTable` | OrderNo + DeviceName (PK), InfFileName | Per-order device configuration |
| `SubOrderTable` | FDIAManageID (int), LockInfo (int), LockDevice (nvarchar 32) | Order locking |

**Multi-scanner configuration**: each scanner needs its own database name AND its own login name. Convention: `FDIA_DB_EZPC00`/`FDIA_DB_EZPC01` for database names, `fdiaclient_ezpc00`/`fdiaclient_ezpc01` for logins (high confidence from Sprocket docs).

**ImageTable CMYD columns**: `Para_D`, `Para_C`, `Para_M`, `Para_Y` (smallint) are per-frame density/color adjustment parameters. `Step_dc`, `Step_dm`, `Step_dy`, `Step_c`, `Step_m`, `Step_y` (smallint) are step adjustments. `AeHigh`, `AeMiddle`, `AeLow` (real) are auto-exposure levels. `AwbRed`, `AwbGreen`, `AweBlue` (real, note the typo "Awe" not "Awb" for blue) are auto white balance. `Aepaecn1` through `Aepaecn8` (real) and `UsmParam` (int, unsharp mask) round out the per-frame processing parameters. High confidence on column names from Sprocket's schema.

**Order status codes**: 0 = error, 2 = scan finished / ready to import, 4 = waiting to write, 5 = completed (high confidence, confirmed by both fujisan and Sprocket docs). When `finalize_on_status` is enabled, Status=2 is the signal an order is done (more accurate than the quiescence timer, because a pause mid-roll cannot be mistaken for the end). `status_stale_minutes = 30`: if the DB shows in-progress but nothing has changed for 30 min, finalize anyway. Status < 2 blocks the quiescence timer, preventing publication of a partial roll during a scanning pause (high confidence from Sprocket docs).

**Standard devices** (from seed data): PIC_SCAN, PIC_TIFF (both output attribute 2, FileFormat 4 = F-RAW), PRINT, SP3000_F1, SP3000_F2, SP3000_input (all input attribute 1), SP3000sRGB (output attribute 2, service 1). High confidence from Sprocket seed tables.

**Device configuration files**: device menu entries are stored as files (e.g. `DEV00003.INF`, `DEV00002.INF`), served from a network path the scanner can reach. The scanner queries `DeviceInfTable.InfFileName` to find them. The file format is length-prefixed structures. Standard mappings: `PIC_SCAN=DEV00003.INF`, `PIC_TIFF=DEV00002.INF`. Without these files: Fuji error E-1775 "no service menu" (high confidence from Sprocket docs). DI (C4/C5) export software offers JPG, FPX, BMP and FUJICOLORCD as writing-service formats, with no TIFF option (high confidence).

**Order registration handshake**: the scanner registers orders via a handshake file in addition to the database. "Connection to the Imaging controller failed" at order registration is scanner-to-database/handshake communication, not related to the importer (medium-high confidence from Sprocket docs).

### 9.6 Scanner output dimensions

From the SizeTable seed data shipped with Sprocket (high confidence). The millimetre columns are output (print) sizes at about 300 pixels per inch (305 mm is 12 inches), not film dimensions; a 35 mm frame is 36 x 24 mm. (Corrected 2026-11-26: an earlier version of this note read 305 mm as the film.)

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

Note: the L (large) sizes at 3650 W pixels vs N (normal) at 3638 suggest different scan modes, possibly related to pixel shift (low-medium confidence). Adre Lanine's medium output of 5573 x 3722 is about the size of the scanner's own 135L (5444 x 3650), and 35XL is larger still, so his output does not by itself prove pixel-shift assembly on the PC side (medium confidence; corrected 2026-11-26, an earlier version said it did). His "2x" compares against the usual medium export (2740 x 1830).

### 9.7 F-DIA client library dialects

Two versions of the F-DIA client library exist: dialect "41" (SSI-era) and dialect "45" (MS01-era, Fuji's own software). The scanner's SQL queries are fixed inside its client library, so any database emulator must present the schema layout the client expects (medium confidence from Sprocket docs).

### 9.8 SP-500 differences

The SP-500 writes finished JPEG or TIFF files plus a `.li` sidecar per frame into `ImgExchange` (not InSpool). The file is **bit-identical** to what the scanner produced, copied byte-for-byte with no re-encoding. There is no .RAW decode step. `CdOrder.INF` ending with `EndOrder` signals order completion. Order names come from the `.li` file, not the database. No FDIA_DB is needed for naming. The SP-500 uses `SOFTWARE\FUJIFILM\FE\KIOSK\SPOOLPATH` for its spool path. High confidence from Sprocket docs.

Additional SP-500 details (high confidence from Sprocket docs):
- **ImgExchange folder structure**: `ImgExchange/<session>-1-4/<order>/`, with a session-level directory above order directories.
- **Housekeeping folders**: `_exfDelReserved` and `_exfFolderProperty` in ImgExchange are Fuji's own housekeeping and are correctly skipped. So 4 folders in ImgExchange is usually 2 orders.
- **ImgExchange is temporary staging** that the Fuji software sweeps. Once swept, files are gone; the archive is the only copy.
- **Order numbering**: 5 digits (Fuji pads them), with a machine letter prepended when registered, e.g. `g01125`. The letter is the "Fuji Order Identifier", set on the scanner itself ('a' for the first scanner on a controller, 'b' for the second).
- **`.li` file contents**: carries the order number the operator typed and the film-frame number. Also contains a Fuji roll number (`fuji_roll`), always '01' on every roll seen so far; how it increments for a second roll is not confirmed (medium confidence).
- **SP-500 files are already finished**: no LUT or mono processing applies. Choosing TIFF/BMP is a transcode of a compressed source, not a lossless master. If you want lossless, set the scanner to TIFF (high confidence).

### 9.9 Safety rules for the server side

These are confirmed hazards, not hypothetical. Any tool that touches InSpool or FDIA_DB must respect them.

1. **Never open InSpool files before Status == 2.** Reading files while the scanner is still writing causes a sharing violation that aborts `ExecLSeqObj.exe` ("abnormal program termination") and wedges the scanner on "Printing External File". On Windows, files must be opened with `FILE_SHARE_DELETE` (`FILE_SHARE_ALL`) to avoid blocking the Fuji software's own rename/delete (high confidence from Sprocket docs, confirmed as an observed failure).
2. **Never delete spool folders.** Deleting an order folder the scanner still owns is the suspected cause of real scanner hangs. A missing folder produces Fuji error W-5302 ("insufficient disk space") when the operator clicks Continue, because the error really means "the folder is gone" (high confidence from Sprocket docs).
3. **Never write to FDIA_DB.** Sprocket only reads; fujisan writes Status=5 and deletes rows, but that is risky. The scanner itself sets Status=2 and clears "Printing External File" with no importer running (verified by scanning with SSI stopped, high confidence from Sprocket docs).
4. **Never run the collection kit during a scan.** Even read-only access to InSpool can trigger the sharing violation above.
5. **The scanner releases itself.** The SP-3000 finishes its own write, sets Status=2 and clears "Printing External File" by itself. Nothing an importer does to InSpool frees a scanner that has not already freed itself (high confidence from Sprocket docs, verified experimentally).

### 9.10 F-DIA service emulation (TDS protocol)

The most significant finding from the Sprocket documentation for the open-frontier project. The SP-3000's server-side interface is entirely **TDS (Tabular Data Stream)**, the standard SQL Server wire protocol. The scanner does not use a proprietary protocol on the server side: it makes SQL queries over TCP port 1433 (or named pipes on Windows), and any server that speaks TDS and returns the expected schema can replace the original MSDE/SQL Server (high confidence from Sprocket docs).

Key facts:
- **No authentication in the protocol.** The scanner's F-DIA client does not send a password the server can check. Access control is IP-based only (high confidence).
- **Transport**: TCP port 1433 (standard SQL Server port) or named pipes (Windows only). The scanner connects to a SQL Server alias configured by VDEntry or a client alias script (high confidence).
- **LOGIN7 packet**: the scanner sends a TDS LOGIN7 packet. An older dialect uses a different record layout that "has never been decoded" (medium confidence on the older dialect's specifics).
- **Dialect auto-detection**: the server can start with dialect "41" (SSI-era) and work out from the scanner's first few SQL requests whether it is dialect "45" (MS01-era). If dialect 4.5 is detected, the database layout is rebuilt, but only while the DB is still empty, never once it holds an order (high confidence).
- **One database cannot serve scanners of both dialect kinds** (41 and 45) (high confidence).
- **Port conflict is fatal by design**: if the port cannot be bound (usually because a real SQL Server still holds it), the emulator refuses to start, to avoid the scanner talking to the wrong server while everything looks healthy (high confidence).
- **The scanner's SQL queries are fixed inside its client library.** Any database emulator must present the schema layout the client expects (high confidence).

**Implication for open-frontier** (medium-high confidence): to replace the Windows server completely, we need to serve TDS on port 1433, respond to the scanner's fixed SQL queries, and serve the device INF files from a network path the scanner can reach. This is a known, documented protocol. No proprietary wire format on the server side.

### 9.11 SMB mount requirements for Linux

Required mount options for a Linux host accessing the scanner PC's spool shares (high confidence from Sprocket docs):

- **Read-write mount.** The importer clears processed order folders out of the spool, exactly as the original importer did. A read-only mount silently lets the scanner's own disk fill up.
- **`actimeo=1`**: SMB attribute cache default (30s) can serve stale file sizes, making a still-writing frame look settled. Must mount with `actimeo=1`.
- **`soft` mount**: a hard mount wedges the worker on scanner reboot.
- **autofs recommended** over manual mounts: a service that starts before the share is up polls an empty directory forever without erroring. autofs mounts on first access.
- **systemd `RequiresMountsFor=`** in the service unit prevents starting against an empty mountpoint, but only for the path written into the unit.
- **SMB1 negotiation on macOS**: macOS won't negotiate down to SMB1 on its own for XP/2003-era PCs. Needs `/etc/nsmb.conf` with `protocol_vers_map=1` and `signing_required=no`.
- **Keep desktop file managers off these shares.** macOS writes `.DS_Store` and `._*` files; junk written into a scanner's spool is there for good.
- **SMB1 read performance**: about 8 MB/s over 100 Mbit. Latency-bound. Streams are largely independent so concurrency helps: 4 workers use about 25% of a 1 Gb NIC, 8 workers about 50% (high confidence, measured values).

### 9.12 Timing and performance

Scanner-side and importer timing values (high confidence from Sprocket docs):

| Parameter | Value | Meaning |
|---|---|---|
| poll_interval | 5 s | Between looks at the spool folder |
| quiescence_seconds | 150 s | How long an order must stop changing before it counts as finished (live mode) |
| validate_quiescence_seconds | 30 s | Same for validate mode (intentionally short) |
| resolve_grace_seconds | 180 s | Wait up to 3 min for order ID to appear in the map before falling back to folder number |
| frame_settle_seconds | 5 s | Fallback timer for frames whose size cannot be verified |
| status_stale_minutes | 30 min | If DB shows in-progress but nothing has changed, finalize anyway |
| Dashboard refresh | 15 s | Web dashboard polling interval |
| Heartbeat line | about 60 s | Log heartbeat frequency |
| Worker restart delay | about 15 s | Crash recovery time |
| Stuck frame retry | 10 min | Automatic retry interval for frames that failed to process |
| SMB1 throughput | about 8 MB/s | Per stream over 100 Mbit (latency-bound) |
| Frame read time | about 5 s | Per 40 MB frame per stream |

**Processing order**: FIFO by lowest FDIAManageID (earliest scanned), matching SSI behavior. In incremental mode, `stream_frames_per_pass` limits frames per order per poll pass, so one big roll cannot hog the whole pass and starve others (high confidence).

### 9.13 B&W handling and color notes

- **B&W sepia cast**: B&W film goes down the color-negative path in the scanner, producing a tinted grayscale rather than neutral (high confidence from Sprocket docs).
- **Neutralization**: Rec.709 luma collapse, the same as an editor's destructive "Desaturate" (high confidence).
- **Channel correlation measurements on real scans**: B&W frames 0.99937 to 0.99977 (nearly identical channels), color frames 0.860 to 0.994. The 0.99 threshold sits clear of every B&W frame seen. An overcast color frame scores 0.994, which is why chroma alone cannot trigger desaturation (would hit fog, snow, dusk frames) (high confidence).
- **3D LUT format**: `.cube` files. These can be made in Resolve, Lightroom, or fit from reference images (high confidence).
- **The SP-3000 never writes JPEGs itself.** It writes only .RAW + .INF. The JPEG is always produced by the export software (high confidence from Sprocket docs).
- **Re-export from TIFF scanner output**: if the scanner rendered as JPEG, asking for TIFF "decodes the JPEG and re-encodes it. That result is a transcode of a compressed source, not a lossless master." If you want lossless, set the scanner to TIFF (high confidence).

### 9.14 SP-3000 order numbering

- The order number field is 10 characters that the operator fills however the lab likes. Convention varies by lab (2 letters + 4 digits + 4 letters at one lab, all-numeric at another). `OrderID` max length is 17 characters in the database (`nvarchar(17)`) (high confidence from Sprocket docs).

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
- What does the "handshake file" at order registration contain, and where does it live?
- What is the exact TDS LOGIN7 record layout the scanner sends? (The older dialect's layout "has never been decoded".)
- Files referenced by Sprocket but not shipped: `PORTING.md`, `LINUX_INSTALL.md`, `SSI_REPLACEMENT_SPEC.md`. These likely contain additional protocol or deployment facts.
- RAW layout on real files: header word positions, interleaved or planar pixels, channel order (see 9.2).
- What do the values of `02_ROTATION_FOR_PROPER_RECTANGLE` mean, and in which direction? Our converter provisionally reads 0, 90, 180 and 270 as degrees clockwise and leaves other values unrotated; `openfrontier compare` tells which rotation really matches the Fuji output.
