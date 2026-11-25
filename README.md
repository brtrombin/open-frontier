# open-frontier

Drive a Fujifilm Frontier SP-3000 straight from a modern Linux PC: FireWire control, raw CCD capture and image processing. No Windows 2000, no Toshiba.

> **Status: Phase 0 (research and preservation). There is no scanner control code yet.**
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

Those PCs are dying, Windows 2000 speaks only SMB1, and every workaround keeps Windows 2000 alive in some form (native, or in a VM). In November 2026 Adre Lanine showed a working prototype that replaces both PCs with one modern computer and doubles the output resolution. That work is unpublished. This project aims to build an open equivalent.

## Goal

One Linux PC, one FireWire card, a web interface:

1. Speak the SP-3000 FireWire protocol from user space (`/dev/fw*`, firewire-cdev).
2. Capture raw CCD data, including the pixel-shift positions and the infrared channel.
3. Calibrate, assemble, invert and color-correct on the PC.
4. Export 16-bit TIFF and JPEG, and control the carrier (frame detection, whole-roll scans, XPan).

## Plan

| Phase | What | Status |
|---|---|---|
| 0 | Preserve the original system: disk clones, configuration, calibration data, reference scans | in progress |
| 1a | Observe the original system passively with a bus sniffer (nosy) | not started |
| 1b | Identify FireWire nodes from Linux, read-only | not started |
| 2 | Map the protocol: static analysis, bus captures, raw dump comparison | not started |
| 3 | Reproduce it, starting with harmless status reads | not started |
| 4 | Image pipeline: decode, calibrate, pixel shift, inversion, color | not started |
| 5 | Product: carrier control, queue, web UI, export | not started |

Details: [docs/plan-review.md](docs/plan-review.md), [docs/phase0-checklist.md](docs/phase0-checklist.md), [docs/phase1-runbook.md](docs/phase1-runbook.md).

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
docs/        journal, protocol notes, hardware, phase checklists, sources
collect/     read-only data collection kit for people who run an SP-3000 today
tools/       small read-only helper scripts
analysis/    notes from static analysis (never the binaries themselves)
src/         future control and image code (empty for now)
```

## How to help

- **You run an SP-3000 today.** The read-only kit in [collect/](collect/README.md) gathers the facts we need: FireWire controllers, driver names and versions, software version, boot timing. It never talks to the scanner. Send the results privately, not as a pull request.
- **You know IEEE 1394.** Read [docs/protocol.md](docs/protocol.md) and the open questions in it.
- **You have a TI PCILynx card** (Unibrain Fireboard 400 Lynx-2, IOI-1394TT): we need one for the nosy sniffer.

## Prior art and thanks

- Adre Lanine, "Frontier SP-3000 Modernisation" (Fuji Frontier Stand-alone Scanner Club, Facebook): the proof that this works.
- [fujirawconvert](https://github.com/torarin/fujirawconvert) by torarin: converts CCD dumps from menu 2241 to TIFF.
- [Sprocket](https://sprocket.boutiquefilmlab.com/) by Boutique Film Lab: reverse-engineered the MS11 RAW/INF spool formats.
- Jack Chauvel's [guide to virtualising an SP-3000](https://www.jackchauvel.com.au/guide-to-virtualising-a-fuji-frontier-sp3000/).
- The Linux FireWire subsystem and its maintainer, Takashi Sakamoto.

More in [docs/sources.md](docs/sources.md).

## License

GPL-2.0-only, the same license as the Linux FireWire stack. See [LICENSE](LICENSE).
