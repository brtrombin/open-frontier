# open-frontier: instructions for Claude

## Goal

Do what Adre Lanine's "Frontier SP-3000 Modernisation" project did: remove the Toshiba scanner PC running Windows 2000 (EZPC00) and the XP front-end server, and drive the scanner directly from a single modern computer. That computer speaks FireWire to the SP-3000 image box, captures the CCD data and produces JPEG/TIFF itself.

Target platform: **Windows 10, natively and standalone** (decision of 2026-11-26), on hardware we already have, with a FireWire port. **No hardware purchases** (decision of 2026-11-26). Adre Lanine runs his version on a laptop (in his video it looks like a MacBook Pro). The spool side already runs natively on Windows 10 (`src/`, one `.exe`, nothing to install). For the FireWire link, Linux stays the research platform:
- Linux exposes FireWire to user space (firewire-core, `/dev/fw*` devices, firewire-cdev ioctls), with no driver to write.
- Windows 10/11 have a 1394 OHCI driver, but user-mode programs cannot send 1394 transactions: that needs a kernel driver of our own, which only loads in test-signing mode (`docs/plan-review.md`, point 15).
- macOS 26 Tahoe removed all FireWire support, including through Thunderbolt adapters (macOS 15 still has it).

So FireWire research runs on Linux (a live USB on the Windows 10 PC is enough), and the Windows driver follows once the protocol is known.

Final interface: a web application served by that PC, usable from Windows, Mac or a phone.

## Working rules

- **Talk to the user in Portuguese.** Everything committed to this repository is in English, because the repository is public.
- Use commas instead of em dashes, in both languages.
- Always state a confidence level (high, medium, low) for anything that is an inference or a hypothesis.
- Record every finding in `docs/journal.md` (date, what was done, evidence, confidence) and protocol facts in `docs/protocol.md`.
- Equipment safety:
  - The original system (Toshiba + server) must stay restorable at all times. Nothing happens before Phase 0.
  - Never send the scanner write, motion or calibration commands that have not been observed in original traffic and understood. Start with reads only (config ROM, status registers).
  - Never write to scanner flash/EEPROM.
  - Stop and ask before any step that could move motors, turn on the light source or change persistent configuration. Stop and ask before any step that sends anything to the scanner.
  - **Server-side safety** (confirmed hazards, not hypothetical):
    - Never open InSpool files before `OrderTable.Status == 2`. Reading files while the scanner is still writing causes a sharing violation that aborts `ExecLSeqObj.exe` and wedges the scanner.
    - Never delete spool folders. A missing folder produces Fuji error W-5302 on Continue.
    - Never write to `FDIA_DB`. The scanner sets its own status; we only read.
    - Never run the collection kit or any InSpool tool during a scan.
    - On Windows, open InSpool files with `FILE_SHARE_DELETE` (`FILE_SHARE_ALL`).
- Reverse engineering only for interoperability with the equipment we work on (next section), with its owner's authorization.
- **Proprietary and private data never enters the repository.** Fujifilm binaries, firmware, manuals, registry/ini exports, raw dumps (.bin/.tag/.RAW/.INF), screenshots of Fujifilm software, and anything collected from third-party machines (such as the user's friend Marcelo) stay outside git. The user shares them privately in the chat. Only facts we derived ourselves, in our own words, are committed. When proprietary material shows up, point it out to the user explicitly.
- Write every procedure so that a non-technical scanner owner can follow it: what to do, what not to touch, how to undo it. The long-term aim is steps anyone with an SP-3000 can follow.

## The system we work on (2026-11-26, corrected 2026-11-27)

- The only SP-3000 in the project belongs to the user's friend. The user has no scanner.
- Correction (owner's own words, 2026-11-27): he does **not** run his scanner PC in a VM. The only VM is the server, at a lab, and he finds VMs slow and unpleasant. The earlier note that both PCs run on ESXi was our misunderstanding. How his scanner PC runs today (the physical Toshiba or not) is to be seen on a visit.
- He says that each time a scanner PC system is brought up on the scanner, it redoes the parameters for the next system, which is why he cannot simply connect another system (for example an old disk image) to his scanner. So the scanner PC and the scanner exchange and rewrite setup data when they are paired (medium confidence on the mechanism). Never connect any system to his scanner without his presence and approval.
- He offered to show the setup in person. The owner authorized tests, passive captures and analysis of his own disk images (2026-11-26). Everything from his system is private and never enters git.
- The near-term product follows the same concept as Sprocket's "Setup A": the scanner PC stays as it is, and our program on a modern PC reads finished orders across the network and delivers them, named and converted, to an SSD or NAS folder (`openfrontier watch`). The ESXi route is dropped.

## What is known about the hardware (research of 2026-11-25)

Original architecture:

```
Scanner head + image box (area CCD with pixel shift, 135/120 carriers, GPA23 DSP boards, 2 RAM modules)
   │ FireWire (IEEE 1394a / S400)
Scanner PC "EZPC00" (Toshiba, Windows 2000 Pro, A1 or MS01+MS11 or S1)
   │ Ethernet 100 Mbit (SMB1 spool folders InSpool/OutSpool + FDIA_DB order database on MSDE)
Server "FRONTEND" (XP/7/10: C4/C5, S2, MS01, PTS SSI or Sprocket)
```

- The Toshiba has two FireWire ports. An owner reports they connect to "MCU1" and "MCU2" (medium confidence). Jack Chauvel's guide describes one link going to the power unit, which **triggers the scanner's power-up when the PC is on**, and the other as the main data link between the image controller (scan head) and the scanner PC (medium confidence). A replacement must cover both links.
- **Plugging a powered host into that FireWire link can switch the scanner on** (Chauvel: "Power on my Lenovo box, in doing so this powers up the scanner through the attached firewire"). Treat connecting the Linux PC as a power-on event, not a passive step.
- In Chauvel's original setup the two links used two controllers (onboard port + add-in card). In his VM, one card with two ports worked. A small VIA card "without enough juice" failed: the host card probably must supply 6-pin bus power (medium confidence).
- Internal scanner links use RS422 and similar (not reachable from the PC).
- The CCD makes micro-movements: the same image is scanned once or up to 8 times while shifting the CCD. Scanning 4 positions and assembling on the PC gives about 2x linear resolution.
- The image box has GPA23 boards with DSPs. Scanners with two GPA23 boards are faster. Probable bottleneck of about 3.4 s/frame (low-medium confidence).
- Light: RGB LEDs (blue 440-460 nm, green 535-545 nm, plus red), per a measurement published on Medium (Alexi Maschas). There is an infrared channel for dust removal (Digital ICE).
- Factory defaults: names EZPC00 and FRONTEND, user FRONTIER/FRONTIER, maintenance menu password 7777, standalone mode "S-Alone" or "Printer OFF".
- Power: single phase 100-240 V, 50/60 Hz, up to 10 A at 100 V. 118 kg.
- Original software boot is timing-sensitive: the scanner takes about 3 min to initialize, and too-fast hardware makes the software miss checks.
- The Fujifilm FireWire driver is set up with a tool called VDEntry, which assigns the FireWire bus ID (from a public installation document; medium confidence on what that means for us).

## Reference data

- CCD dump from menu 2241: Setup and Maintenance > Self-diagnostics > Image data diagnostics, "Image Log Storage" = On, .bin + .tag files in `D:\Fujifilm\Frontier\Log` (on the scanner PC).
- fujirawconvert (github.com/torarin/fujirawconvert, C++): converts .bin + .tag to TIFF. Default inverted log, `-lin` linear, `-ir` puts infrared in the alpha channel. Read the code to understand the format. Unknown whether the data is already calibrated (dark/flat) by MS11.
- Sprocket (sprocket.boutiquefilmlab.com) claims to have decoded the .RAW and .INF formats MS11 writes to InSpool, and emulates the scanner database. Useful for comparing output.
- Original reference output: medium size exports 2740 x 1830 px.

## Adre Lanine's project (what is possible)

Post "Frontier SP-3000 Modernisation", 2026-11-15, Facebook group "Fuji Frontier Stand-alone Scanner Club" (still beta, not sold):
- No Toshiba, no server, no XP/2000, no VM. One modern computer; plans a Linux station with a web interface.
- Medium size: 36 frames in about 2 min, 4 passes, output 5573 x 3722 px (2.03x the original per side).
- Full roll at maximum definition in under 5 min (versus 14); high in 2 min; live preview.
- Says FireWire is not the bottleneck; time was lost in waits between data blocks and in the original software's processing/export.
- Features: CMYD adjustment after the scan, auto rotation, XPan 10,277 x 3,635 px via stitched linear scan, whole-roll TIFF, exposure report.
- In the video: a laptop resembling a MacBook Pro showing a thumbnail grid in a browser; the Fuji monitor is off.
- Inference (low-medium confidence): his software either speaks the Fuji FireWire protocol itself or drives the Fuji scanner software in some form; his posts do not say which. His medium output (5573 x 3722) is about the size of the scanner's own Large scan (135L, 5444 x 3650), so his doubling is relative to the usual medium export, and his speed and workflow are the parts we cannot explain yet (medium confidence).
- No outreach to him or to anyone else (decision of 2026-11-26).

## Phases

See `docs/plan-review.md` for the reviewed order (revised 2026-11-26). Summary:

- **0.** Preserve the original (disk clones, configuration and calibration data, reference scans, documented cabling and FireWire topology). For the friend's system: to be done on a visit.
- **S1.** Decode finished spool orders (`.RAW` + `.INF`) to JPEG/TIFF on Windows 10, offline, and compare with the Fuji output. Code in `src/`, to be checked on real files (`docs/windows-test-plan.md`).
- **S2.** Hands-off delivery (`openfrontier watch`): read an order only when `Status == 2`, deliver it to an SSD or NAS. Written and tested with a simulated scanner (`docs/windows-test-plan.md`, T1); next is the owner's system (T2 to T4). **S3.** Stand in for FDIA_DB and retire the front-end server.
- **1.** Identify without touching: 1a passive host-side logging of the original system (on a cloned disk or a VM copy); 1b a Linux host (live USB), read-only, with drivers that transmit blacklisted.
- **2.** Observe the protocol: static analysis of the owner's disk image (Ghidra), bus captures, comparison with menu 2241 dumps.
- **3.** Reproduce, starting with harmless reads, then init and a single low-resolution frame, each with explicit approval.
- **4.** Image from raw CCD data: dark/flat calibration, pixel shift, inversion and color (CMYD), IR, export.
- **5.** Product: carrier control, queue, web UI, export, and the Windows FireWire driver.

## Open questions

- Does the second FireWire link (MCU to power supply) carry power/light commands?
- Does image data arrive as asynchronous or isochronous transactions?
- Do the GPA23 DSPs deliver processed or raw data? Can raw be requested?
- What does the menu 2241 .tag file contain?
- Are the commands high-level ("scan frame N") or low-level (motors, CCD shift)?
- Where on the scanner PC disk does per-machine calibration data live?
- In the spool `.RAW`, which header words hold width and height, are the pixels interleaved, in which channel order, and what does the `.INF` rotation value mean? (`openfrontier inspect` and `compare` on one real frame answer these.)
