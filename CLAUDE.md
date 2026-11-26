# open-frontier: instructions for Claude

## Goal

Do what Adre Lanine's "Frontier SP-3000 Modernisation" project did: remove the Toshiba scanner PC running Windows 2000 (EZPC00) and the XP front-end server, and drive the scanner directly from a single modern computer. That computer speaks FireWire to the SP-3000 image box, captures the CCD data and produces JPEG/TIFF itself.

Target platform: Linux (Debian/Ubuntu LTS) on hardware we already have, desktop or laptop, with a FireWire port. **No hardware purchases** (decision of 2026-11-26). Adre Lanine runs his version on a laptop (in his video it looks like a MacBook Pro). Reasons for Linux:
- Linux exposes FireWire to user space (firewire-core, `/dev/fw*` devices, firewire-cdev ioctls).
- macOS 26 Tahoe removed all FireWire support, including through Thunderbolt adapters (macOS 15 still has it).
- Windows 10/11 have a 1394 OHCI driver, but raw user-space access is much harder.

Final interface: a web application served by the Linux PC, usable from Windows, Mac or a phone.

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
- Reverse engineering only for interoperability with the user's own equipment.
- **Proprietary and private data never enters the repository.** Fujifilm binaries, firmware, manuals, registry/ini exports, raw dumps (.bin/.tag/.RAW/.INF), screenshots of Fujifilm software, and anything collected from third-party machines (such as the user's friend Marcelo) stay outside git. The user shares them privately in the chat. Only facts we derived ourselves, in our own words, are committed. When proprietary material shows up, point it out to the user explicitly.

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
- Inference (medium confidence): his own software speaks the Fuji FireWire protocol, captures raw CCD data, assembles pixel shift and applies color on the PC. Worth contacting him through the group.

## Phases

See `docs/plan-review.md` for the reviewed order. Summary:

0. Preserve the original (disk clones, configuration and calibration data, reference scans, documented cabling and FireWire topology).
1. Identify without touching: 1a passive host-side logging of the original system (Windows 2000 in a VM snapshot or on a cloned disk); 1b Linux as host, read-only, with drivers that transmit blacklisted.
2. Observe the protocol: static analysis of the user's own disk image (Ghidra), bus captures, comparison with menu 2241 dumps.
3. Reproduce, starting with harmless reads, then init and a single low-resolution frame, each with explicit approval.
4. Image: decode, dark/flat calibration, pixel shift, inversion and color (CMYD), IR, export.
5. Product: carrier control, queue, web UI, export.

## Open questions

- Does the second FireWire link (MCU to power supply) carry power/light commands?
- Does image data arrive as asynchronous or isochronous transactions?
- Do the GPA23 DSPs deliver processed or raw data? Can raw be requested?
- What does the menu 2241 .tag file contain?
- Are the commands high-level ("scan frame N") or low-level (motors, CCD shift)?
- Where on the scanner PC disk does per-machine calibration data live?
