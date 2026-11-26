# open-frontier

Run a Fujifilm Frontier SP-3000 from one modern PC, natively on Windows 10: spool decoding and image export first, then the scanner's FireWire link. No Windows 2000, no Toshiba.

> **Status: Phase 0 (research and preservation), plus a first Windows 10 program that converts finished spool orders (`.RAW` + `.INF`) to JPEG/TIFF.**
> Nothing in this repository talks to a scanner today.

## Why

The SP-3000 is one of the best film scanners ever built, and it is chained to obsolete computers:

```
Scanner head + image box (area CCD with pixel shift, 135/120 carriers, GPA23 DSP boards)
   │ FireWire (IEEE 1394a, S400)
Scanner PC "EZPC00" (Toshiba, Windows 2000 Pro, A1 or MS01+MS11 or S1)
   │ Ethernet 100 Mbit (SMB1 spool folders InSpool/OutSpool + MSDE order database)
Front-end server "FRONTEND" (Windows XP/7/10: C4/C5, S2, MS01, or Sprocket)
```

Those PCs are dying, Windows 2000 speaks only SMB1, and every workaround keeps Windows 2000 alive in some form (native, or in a VM). In November 2026 Adre Lanine showed a working prototype that replaces both PCs with one modern computer, is much faster, and reaches about twice the linear resolution of the usual medium export. That work is unpublished. This project aims to build an open equivalent.

## Scope: the SP-3000 first, the whole SP family by design

Fujifilm's MS11 scanner software drives the SP-500, SP-1500, SP-2000, SP-2500 and SP-3000, which suggests a shared software layer across the family (medium confidence). The physical link differs per model: FireWire on the SP-3000 (high confidence), USB reported for the SP-500 and SCSI reported for some SP-2000/SP-1500 boards (low confidence, from sale listings). The code therefore keeps the transport (FireWire, USB, SCSI) separate from the scanner protocol and the image pipeline, so other models can be added. The SP-3000 is the first target because it is the one we can test.

No open-source project for any Frontier SP scanner was found (GitHub and web search, 2026-11-25).

## Goal

One computer we already have, running Windows 10 natively and standalone, with a FireWire port and a web interface. In the order we expect to get there:

1. Decode the spool files the scanner PC writes (`.RAW` + `.INF`) and export JPEG/TIFF. **Started**, see below.
2. Replace the front-end server: watch the spool safely and stand in for the order database (FDIA_DB).
3. Speak the SP-3000 FireWire protocol, after observing the Fuji software's own traffic, and retire the Toshiba.
4. Capture raw CCD data with the pixel-shift positions and the infrared channel; calibrate, assemble, invert and color-correct on the PC.
5. Carrier control (frame detection, whole-roll scans, XPan), queue, web UI, 16-bit TIFF and JPEG export.

## The Windows 10 program

`openfrontier.exe` is one file with nothing to install. It reads finished spool orders and never talks to the scanner.

```text
openfrontier inspect <order folder>           report the frames, the RAW header and every .INF field
openfrontier convert <order folder> --out DIR write JPEG, or TIFF with --format tiff
openfrontier compare <frame.RAW> <fuji.jpg>   check our decode against the Fuji software's output
openfrontier firewire                         list the FireWire hardware Windows sees (read-only)
```

It refuses to read a folder that may be the scanner's live spool (a network path, or a folder named InSpool, OutSpool or ImgExchange) unless told the order is finished. On Windows it opens every spool file with `FILE_SHARE_DELETE`, the documented way to read without blocking the Fuji software. Build it with Go 1.24 or later from `src/`. How to test it: [docs/windows-test-plan.md](docs/windows-test-plan.md).

## Plan

| Phase | What | Status |
|---|---|---|
| 0 | Preserve the original system: disk clones, configuration, calibration data, reference scans | in progress |
| S1 | Decode finished spool orders (RAW/INF) to JPEG/TIFF, natively on Windows 10 | written, to be checked on real files |
| S2 | Watch the spool safely, reading order status from FDIA_DB | not started |
| S3 | Stand in for FDIA_DB and retire the front-end server | not started |
| 1a | Observe the original system passively, logging from inside Windows 2000 | not started |
| 1b | Identify FireWire nodes from the new PC, read-only | not started |
| 2 | Map the protocol: static analysis, bus captures, raw dump comparison | not started |
| 3 | Reproduce it, starting with harmless status reads | not started |
| 4 | Image pipeline from raw CCD data: calibration, pixel shift, inversion, color | not started |
| 5 | Product: carrier control, queue, web UI, export | not started |

Details: [docs/plan-review.md](docs/plan-review.md) (revised order), [docs/windows-test-plan.md](docs/windows-test-plan.md), [docs/phase0-checklist.md](docs/phase0-checklist.md), [docs/phase1-runbook.md](docs/phase1-runbook.md).

## Safety rules (for the hardware)

- The original system (scanner PC and server) must stay restorable at all times. Nothing happens before Phase 0 is complete.
- Never send the scanner a write, motion or calibration command that has not been observed in original traffic and understood. Start with read-only access (config ROM, status registers).
- Never write to scanner flash or EEPROM.
- Never hot-plug FireWire cables. 6-pin FireWire carries power. Put the scanner and the PCs on one grounded power strip.
- Any step that could move a motor, turn on the light source or change persistent settings needs an explicit, written go-ahead from the equipment owner.

## Legal note and what this repository will never contain

Reverse engineering here is done only for interoperability with equipment the contributors own. This repository holds only our own notes and our own code. It never contains Fujifilm software, drivers, firmware, manuals, registry or configuration exports, raw scan dumps, or screenshots of Fujifilm software. The `.gitignore` blocks those file types. Facts we derive ourselves (IDs, register addresses, packet layouts), described in our own words, are what gets documented.

Fujifilm and Frontier are trademarks of FUJIFILM Corporation. This project is not affiliated with or endorsed by Fujifilm.

## Repository layout

```
docs/        journal, protocol notes, plan review, phase checklists, test plan, sources
collect/     read-only data collection kit for people who run an SP-3000 today
tools/       small read-only helper scripts
analysis/    notes from static analysis (never the binaries themselves)
src/         the openfrontier program (Go): spool decoding, image export, FireWire inventory
```

## How to help

- **You run an SP-3000 today.** The read-only kit in [collect/](collect/README.md) gathers the facts we need: FireWire controllers, driver names and versions, software version, boot timing. It never talks to the scanner. Send the results privately, not as a pull request.
- **You know IEEE 1394.** Read [docs/protocol.md](docs/protocol.md) and the open questions in it.

## Prior art and thanks

- Adre Lanine, "Frontier SP-3000 Modernisation" (Fuji Frontier Stand-alone Scanner Club, Facebook): the proof that this works.
- [fujirawconvert](https://github.com/torarin/fujirawconvert) by torarin: converts CCD dumps from menu 2241 to TIFF.
- [Sprocket](https://sprocket.boutiquefilmlab.com/) by Boutique Film Lab: reverse-engineered the MS11 RAW/INF spool formats.
- Jack Chauvel's [guide to virtualising an SP-3000](https://www.jackchauvel.com.au/guide-to-virtualising-a-fuji-frontier-sp3000/).
- The Linux FireWire subsystem and its maintainer, Takashi Sakamoto.

More in [docs/sources.md](docs/sources.md).

## License

GPL-2.0-only, the same license as the Linux FireWire stack. See [LICENSE](LICENSE).
