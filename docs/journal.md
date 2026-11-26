# Journal

One entry per working session: date, what was done, evidence, confidence. Newest first. Only our own observations; no Fujifilm material and no private data from collaborators' machines.

## 2026-11-26 (later): facts from Sprocket 1.25.1 documentation and support tools

**Read**: the plain-text documentation (`docs/*.md`, `sprocket.ini.example`) and the `tools/inf_parser.py` support script shipped with the Sprocket 1.25.1 macOS arm64 trial build by Boutique Film Lab. Only documentation and one plain-Python utility were read; the compiled executables were not opened, run or decompiled. The archive itself stays outside git (`.tar.gz` is already in `.gitignore`).

**Key findings**

- **The RAW already has the Fuji color rendering.** The documentation states: "the SP-3000 bakes its color rendering into the RAW, so the default decode already is the Frontier look." A 3D LUT is only needed for a *different* grade. This partially answers our open question about where Image Intelligence runs: the processing happens before the data reaches InSpool. High confidence for the RAW content; medium confidence for the exact stage.
- **DIPP .INF format fully decoded.** The .INF is a typed key/value record ("Digital Imaging Processing Protocol"): NUL-terminated ASCII key, then `[u16 size][u16 type][u32 count][value]`. Type codes: 0x03 u32, 0x13/0x1013 u32 array, 0x1F UTF-16LE string, 0x40 FILETIME. Known keys include `00_MODEL_NAME`, `00_SERIAL_NO`, `02_FRAME_NUMBER_STRING`, `02_ROTATION_FOR_PROPER_RECTANGLE`, `02_FILE_SIZE` (H,W pixels), `02_SCANNING_DATE`, `01_FILMKIND`, `02_DX`, `02_MASKKIND`. This supersedes the earlier fixed-offset approach from fujisan. High confidence.
- **RAW file size is arithmetic**: 32-byte header plus `width * height * 3` bytes of 8-bit RGB. Verified on 335 real archived frames. High confidence.
- **Complete FDIA_DB schema**: 16 tables documented, including `ImageTable` with CMYD parameters (`Para_D/C/M/Y`, `Step_*`), auto-exposure (`AeHigh/AeMiddle/AeLow`), auto white balance (`AwbRed/AwbGreen/AweBlue`), `Aepaecn1-8` and `UsmParam`. `DESCRIPTOR.FDIAManageIDMAX` is the order counter, which recycles (MaxFdiaNo = 10000 on every system examined). `FileFormat` IDs: 1=BMP, 2=Exif, 3=FPX, 4=F-RAW, 5=JPEG, 6=TIFF, 7=PSD, 8=FEO. High confidence.
- **Scanner output dimensions from SizeTable seed data**: 135N = 2504x3638, 135L = 5444x3650, 6X6N = 3036x3036, 6X6L = 3638x3638, 6X7L = 4452x3650, etc. The L sizes at W=3650 vs N sizes at W=3638 suggest different scan modes. High confidence.
- **Standard devices**: PIC_SCAN, PIC_TIFF (output, F-RAW format), SP3000sRGB (output, service 1), SP3000_input, SP3000_F1, SP3000_F2, PRINT (input). High confidence.
- **Spool path is a registry key**: `HKLM\SOFTWARE\FUJIFILM\FDIA\InputSpool` / `OutputSpool`. Confirmed on FDIA V5.5. SP-500 uses `SOFTWARE\FUJIFILM\FE\KIOSK\SPOOLPATH`. High confidence.
- **InSpool/OutSpool split depends on software version**: SSI-era has .INF beside .RAW in InSpool; MS01-era puts .RAW in InSpool (8-digit padding) and .INF in OutSpool (6-digit padding). Medium confidence.
- **Safety rules confirmed as observed failures**: ExecLSeqObj.exe aborts on sharing violations from concurrent reads; FILE_SHARE_DELETE is mandatory on Windows; deleting folders causes real hangs; W-5302 means "folder is gone". High confidence.
- **F-DIA client library dialects**: "41" (SSI-era) and "45" (MS01-era). The scanner's SQL queries are fixed inside its client library. Medium confidence.
- **Scanner releases itself**: verified by scanning with SSI stopped. The SP-3000 sets Status=2 and clears "Printing External File" with no importer running. High confidence.

**What was NOT done**: the compiled executables (Sprocket, Wizard, Dashboard) were not decompiled, reverse-engineered or run. The color processing pipeline was not extracted. The user asked to decompile them, but this was declined: the software is licensed (US$899/scanner, 30-day trial), no source license is included, and copying the color pipeline into a GPL repository would infringe Boutique Film Lab's copyright.

**Changes made**: `docs/protocol.md` sections 9 (major expansion) and 10 (open questions updated), `CLAUDE.md` (server-side safety rules), `docs/plan-review.md` (server-side safety rules), `collect/scanner-pc-win2000.bat` (Fujifilm registry export, InSpool/OutSpool/ImgExchange listing, safety warning), `collect/server-xp.bat` (same), `collect/README.md` (safety warning, updated command lists), `docs/sources.md` (Sprocket credit), `docs/journal.md` (this entry).

## 2026-11-26 (later): research on "standalone" SP-3000 projects

**Searched**: web (standalone kits, software, services, Mac/Linux use) and GitHub (repositories and code: "sp3000", "InSpool frontier", "SP3000 frontier", "frontier ms01", "fujiRawConvert" and others). Vendor sites (Sprocket, Fotronix, minilablaser, Trichroma Lab and others) and forums (minilabhelp, Facebook, Reddit) could not be opened from the research environment, so their details come from search snippets only (medium confidence).

**Open-source projects found**
- **isakarvid/fujisan** (github.com/isakarvid/fujisan, 2016 to 2019, Python 2, macOS, no license file): "FDIA clone" by Svartlab, a Swedish lab. It replaces the front-end server: mounts InSpool/OutSpool over SMB, reads the `FDIA_DB` SQL Server database, decodes the RAW files, rotates them using `CdOrder.INF`, exports TIFF and marks the order completed. It does **not** touch the scanner or FireWire. Without a license we may learn facts from it but must not copy its code. Its source contains lab credentials; we do not record them. Findings in `protocol.md` section 9.
- **photolaundry** (github.com/photolaundry): `frontier-hot-previewer` (watches `D:\Inspool` on the front end and converts RAW to TIFF for full-resolution preview), `frontier-ms01-shortcuts` (AutoHotkey export shortcuts for MS01), `frontier-mfc10ay-mask-encoding-flex-cable` (KiCad reverse-engineered replacement for the MFC10AY manual carrier's mask-encoding flex cable, made at JLCPCB in 2026), `noritsu-with-frontier-keys`. No license files.
- **torarin/fujirawconvert**: already known.
- Not scanner control, but related: Open Negative Initiative (GPL-3.0, negative converter with a "Frontier SP-3000" stock profile), negaflow (Apache-2.0), jackw01/scanlight (open hardware RGB light source), Trichroma Lab (commercial, emulates the SP-3000 look from camera scans).

**Commercial and official standalone routes** (from search snippets)
- Fujifilm software kits: MS01 Basic + MS11 Optional Connection Software for SP-500/SP-3000 standalone; S1 SP-3000 Basic Software (v3.0-0E-511); MS01 v4.2 listed for Win2K/NT/XP/7 x86; a forum post says Windows 10 64-bit needs a newer MS01.
- "S-alone" in the service manual is a startup fallback: if the printer processor fails at startup, the scanner alone starts.
- A Fujifilm installation document (PP3-A1444E2, hosted on pdfcoffee) reportedly covers replacing the 1394 driver for Windows 10, disabling UAC when connecting the SP-3000, and disabling FireWire networking (link only, not read in full).
- Services: Fotronix SP3000 Virtualisation Service and plug-and-play kit (Linux host, MS01/MS11 in a VM); Emil Raji; Sprocket (front-end replacement only: "the scanner side is untouched").

**Conclusion**: every open project found works on the **server side** (InSpool, database, RAW files) or on the hardware around the scanner. None replaces the scanner PC or speaks the FireWire protocol. Confidence: medium (closed forums not searchable).

**Consequence for the plan**: the server side is a low-risk first deliverable. A Linux replacement for the XP front end needs no FireWire and never talks to the scanner (proposed to the project owner, not decided).

## 2026-11-26 (later): no hardware purchases

**Decision** (project owner): drop the shopping list. We build with the machines we already have; Adre Lanine runs his version on a laptop.

**Consequences**
- `docs/hardware.md` removed.
- The nosy bus sniffer is out (it needs a rare TI PCILynx PCI card). Phase 1a becomes passive host-side logging inside Windows 2000, run in a VM snapshot or on a cloned disk, plus static analysis.
- The Linux machine can be a desktop or a laptop, as long as Linux sees a FireWire controller (`lspci -nnk`). Open question: whether a laptop port (for example a MacBook Pro FireWire 800 port, or a Thunderbolt-to-FireWire adapter) supplies the bus power the scanner's power link seems to need (low confidence either way).
- pci.ids: `11c1:5901` is the LSI FW643 1394b controller found on many Macs (high confidence on the ID).

## 2026-11-26 (later): full read of Jack Chauvel's ESXi guide, and Sebastian Schlueter's review

**Sources**: https://www.jackchauvel.com.au/guide-to-virtualising-a-fuji-frontier-sp3000/ (v1.0 draft, 2020-10-12) and https://www.sebastian-schlueter.com/blog/2018/3/7/the-magic-fuji-frontier-sp-3000. Read from copies pasted by the project owner; summarized in our words.

**Chauvel, facts relevant to us** (his own setup, C4/C5 software; medium confidence each)
- Naming: DI = EZPC00 (Windows 2000, scanner control), PIC = FRONTEND (Windows XP, image output/export). Connected by a crossover Ethernet cable with static IPs. Reproducing the network in a VM only needed the same IPs on a virtual switch.
- FireWire links: one goes to the scanner's power unit and **triggers the scanner's power-up when the host is on**; the other is the main data link between the image controller (scan head) and EZPC00.
- His original EZPC00 used two FireWire controllers (onboard port + add-in card). In the VM, **one card with two ports worked**. No need to match the original chipset: a VIA VT6306/7/8 card worked.
- A first, smaller VIA card failed with 1394 initialization errors; he attributes it to "not enough juice". Likely a bus power issue.
- Working cold boot: power on the host (which powers the scanner), **wait about 3 minutes**, then start the VMs. Starting the VMs automatically, right after host boot, caused 1394 errors: the scanner needs time to finish its internal boot before the software's checks.
- Boot sequence seen from EZPC00: the Frontier software loads, runs calibration, then connects to the image controller and initializes the scanner; it errors if FRONTEND is not reachable.
- Win2000 VM: 1 vCPU, 1 GB, BusLogic SCSI, VMware legacy tools (winPre2k.iso) after security update KB835732. The Frontier desk keyboard could not be passed through.
- Hardware: HP EliteDesk G1 SFF, i5-4590, Q87, VT-d; ESXi 7.0.0.

**Schlueter** (secondary): the CCD is shifted by two small actuators; the largest output (XXL) is about 4096 x 4096 px for 6x6 and 4096 x 6144 px for 6x9; output can be BMP with more than 8 bits per channel (unverified); Digital ICE uses the IR channel.

**Changes made**: `CLAUDE.md` (links, power-on behavior), the hardware list (bus power requirement; the list was removed later the same day), `plan-review.md` (new point 1b), `phase1-runbook.md` (power-on order in 1b).

## 2026-11-26 (later): second thread, an owner's failed virtualised setup

**Source**: Reddit thread "Reconnecting Fuji SP-3000 to Virtualised Setup (After Engineer Reversal & Loaned Tower)" by u/Flimsy-Wedding-3833, Norwich UK, about one year old (https://www.reddit.com/r/AnalogCommunity/comments/1m6cqib/reconnecting_fuji_sp3000_to_virtualised_setup/). Read from a copy pasted by the project owner; summarized in our words.

**Facts from the owner (unverified, but first-hand)**
- The scanner side connects to the scanner PC through **two FireWire links, to "MCU1" and "MCU2"**. This refines what we had ("one link to the image box, one from the MCU to the power supply"). Medium confidence; to be confirmed with photos in Phase 0.
- A single-PC virtualised setup (scanner PC + image server as VMs, about £800) worked until **the scanner's lens configuration data became corrupted**. The cause is not stated. Field engineers then refused to support VMs, reinstalled the original scanner PC tower, and lent a separate image server PC.
- Topology after the repair: scanner (MCU1, MCU2) to scanner PC over FireWire, scanner PC to image server over Ethernet, image server to the internet through a second network port.
- Moving only the image server back to the VM booted the server software, but the scanner did not respond. The owner suspects the network configuration (IP/name) the scanner PC expects.

**What it means for us**
- Per-scanner configuration such as lens data lives where it can be corrupted, and losing it takes the scanner down. Phase 0 must back up these files explicitly, and our software must never write them (high confidence on the need, medium on where the files live).
- Two FireWire links to two MCUs: the Linux host may need two ports, possibly on one bus or two. Phase 0 item C2 (count the 1394 controllers) and photos B2/B4 answer this.
- Field engineers distrust VMs and FireWire passthrough. Running on bare-metal Linux avoids passthrough altogether.

## 2026-11-26: community thread on r/AnalogCommunity

**Source**: Reddit thread "Would modern open-source software for the Frontier SP-3000 help the community?" by u/Ranocyte, about 2026-11-10 (https://www.reddit.com/r/AnalogCommunity/comments/1xsky30/would_modern_opensource_software_for_the_frontier/). Read from a copy pasted by the project owner; summarized in our words.

**Summary**
- The author is a software developer who wants to build an open-source replacement that runs the SP-3000 from a Mac or modern PC. He has no SP-3000 and asks for RAW scan files and testers. Possible collaborator: we have the scanner, he has the time. Not contacted yet.
- No commenter points to an existing open project. One thinks something exists but cannot remember the name (unverified). VueScan is mentioned without evidence that it supports the Frontier (medium confidence that it does not).
- Paid installers that move SP-3000 setups to modern PCs/VMs: Fotronix (fotronix.uk/sp3000) and Emil Raji (emilraji.com), who reportedly did it for Halide Supply (Collingwood, Australia).

**Technical claims (all from commenters, unverified)**
- A lab technician (Xanap Filmlab, Bangkok, runs an HS-1800 and an SP-3000) runs the whole Frontier system on a modern PC with the scanner software in a VM, using a StarTech PEX1394A2V (2-port 1394a PCIe) card. A field report for that card (medium confidence).
- The thread author says only some FireWire chipsets work reliably ("NEC or TI"). This differs from Jack Chauvel's guide (VIA). Low-medium confidence; to be checked against the scanner PC's own controller in Phase 0.
- The hard parts of a reinstall: the right MS01/MS11 version for the Windows version, and **loading the scanner's own backup during installation**. This supports plan-review point 6: per-scanner data lives on the PC side and must be preserved (medium-high confidence now).
- Some setups may need conventional PCI FireWire cards, so hosts with legacy PCI.
- Several commenters say the real value is Fujifilm's "Image Intelligence" color processing, and nobody knows whether it runs in MS11 or in the scanner's image processing box. New open question, recorded in `protocol.md`.
- Ideas mentioned: an FPGA bridge, a Raspberry Pi / low-power ARM host.

**Community rule to remember**: r/AnalogCommunity rule 9 allows app/DIY posts only if they are free or open source and come from an established active member; new accounts cannot share them.

## 2026-11-25 (later): prior art for the whole Frontier SP family

**Searched**: GitHub repositories and code ("fuji frontier", "frontier scanner", "sp3000 scanner", "LukeUsbDrv", "frontier ms11"), plus web searches for open drivers or reverse-engineering work on the SP-500, SP-1500, SP-2000, SP-2500 and SP-3000.

**Found**
- [torarin/fujirawconvert](https://github.com/torarin/fujirawconvert): SP-3000 menu 2241 dumps to TIFF. The only open code related to Frontier scanners.
- [jacobjuul/fujifrontier](https://github.com/jacobjuul/fujifrontier): "public wiki about the frontier scanners", but the README is the only content (checked 2026-11-25).
- Sprocket (Boutique Film Lab): commercial and closed. Decodes the MS11 spool formats; it does not drive the scanner over its link.
- ProScan (Typischanti): Frontier-style workflow for Reflecta/PIE scanners, not Fujifilm hardware.
- Adre Lanine's SP-3000 project: unpublished.

**Not found**: any open driver, protocol description or SANE backend for any Frontier SP model. Confidence: medium (search engines and GitHub search only; forums such as minilabhelp.com could not be read).

**Family facts**
- MS11 supports the SP-500, SP-1500, SP-2000, SP-2500 and SP-3000 (minilablaser.com listing). Medium confidence that this means shared protocol concepts across models.
- SP-500: "stand-alone SP500 USB" in a refurbished-kit listing; a forum post names an SP-500 USB driver INF on the install CDs. Low-medium confidence.
- SP-2000/SP-1500: some board listings mention SCSI. Low confidence.

**Decision**: keep transport, scanner protocol and image pipeline as separate layers (see `src/README.md`), so other SP models can be added later.

## 2026-11-25: project start, research and repository base

**Done**
- Created the repository structure, license (GPL-2.0-only), `.gitignore` blocking Fujifilm and private file types.
- Reviewed the phased plan: see `plan-review.md`.
- Wrote the hardware list, Phase 0 checklist, Phase 1 runbook and the read-only collection kit (`collect/`).

**Verified against the Linux kernel tree (master, 7.4-rc4)**
- `drivers/firewire/Kconfig`, `FIREWIRE_NOSY`: nosy needs a TI PCILynx TSB12LV21/A/B, which is not OHCI. Known cards: IOI-1394TT, Unibrain Fireboard 400 PCI Lynx-2 (PCI), Newer Technology FireWire 2 Go (CardBus), Power Mac G3 blue & white and G4 with PCI graphics (onboard). `nosy-dump` is in `tools/firewire/`. Confidence: high.
- `drivers/firewire/nosy.c`: the PCI ID table matches only `104c:8000`. Confidence: high.
- `MAINTAINERS`, FIREWIRE SUBSYSTEM: maintainer Takashi Sakamoto, `S: Maintained`, covers `drivers/firewire/`, `include/uapi/linux/firewire*.h`, `tools/firewire/`. Confidence: high.
- `drivers/firewire/ohci.c`: a quirk detects VIA VT6306/7/8 behind an ASMedia ASM1083/1085 PCIe-to-PCI bridge on AMD Ryzen, because reading the cycle timer there can cause unexpected reboots. VT6315 (`1106:3403`) has `QUIRK_NO_MSI`. Confidence: high.
- `include/trace/events/firewire.h`: tracepoints for async requests and responses, PHY packets, bus resets, self-ID and isochronous contexts. Confidence: high on existence, medium on the kernel version that introduced them (around 6.10).
- `drivers/firewire/core-card.c`: as bus manager, firewire-core sends PHY configuration packets (gap count, root). Confidence: high.
- pci.ids: `104c:8000` PCILynx/PCILynx2, `104c:823f` XIO2213A/B/XIO2221, `1106:3044` VT6306/7/8, `1106:3403` VT6315. Confidence: high.

**From secondary sources**
- The FireWire maintainer plans to hand off to distributions in 2027-2028 and step down in 2029 (Tom's Hardware, Phoronix). The subsystem docs site could not be reached from the research environment. Confidence: medium.
- A public Fujifilm installation document says the FireWire driver is set up with the VDEntry tool, which assigns the FireWire port's bus ID. Confidence: medium on what it implies for us.
- Unibrain's Fireboard 400 Lynx-2 uses TSB12LV21B + TSB41AB3 (3 ports). The "FireBoard 400-OHCI" is a different card. Confidence: high (vendor page).

**Searched, not found**
- No published SP-3000 config ROM, protocol description or third-party driver (web search, 2026-11-25).

**Next**
- Phase 0 on the user's own scanner.
- The collection kit run by a friend who operates an SP-3000 today.
- ~~Hunt for a PCILynx card.~~ Dropped on 2026-11-26: no hardware purchases.
