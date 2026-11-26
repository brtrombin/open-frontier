# Critical review of the phased plan

Review of the original plan (see `CLAUDE.md`), 2026-11-25. Each point has a confidence level. "High" means checked against a primary source (usually the Linux kernel tree, master at 7.4-rc4). "Medium" means a reasoned inference or a secondary source. "Low" means a hypothesis to test.

## Equipment risks

### 1. "Plug Linux in place of the Toshiba and only read" is not read-only (high)

As soon as an OHCI controller is loaded, Linux joins the bus as a full node and transmits on its own:
- It triggers bus resets and takes part in self-ID.
- firewire-core may become bus manager, then sends PHY configuration packets (gap count, forced root) and lock transactions to the IRM (`drivers/firewire/core-card.c`).
- It reads the config ROM of every node with asynchronous read requests.
- If **firewire-net** loads, it adds an IPv4-over-1394 unit directory to the local config ROM. Once the interface is up (NetworkManager may do this by itself), it sends broadcast packets.
- **firewire-sbp2** and the **snd-firewire** family (snd_bebob, snd_dice, snd_oxfw, snd_fireworks...) bind automatically if a unit directory happens to match, and then talk to the device.

All of this is standard 1394 behavior that the scanner should tolerate, because the Toshiba does similar things. But it is not "nothing". Mitigation: blacklist every protocol driver and the OHCI driver itself, then load `firewire_ohci` by hand with tracepoints on, so that we log every packet Linux sends. See `phase1-runbook.md`.

### 1b. Connecting a host is a power-on event (medium)

Jack Chauvel's guide says one FireWire link goes to the scanner's power unit and that powering the host PC switches the scanner on through it. So "connect the Linux PC and only read" also means "power up the scanner, its lamp and its init". Any stage that connects our hardware (1a or 1b) must be planned as a full scanner power-up, with the operator present and the usual warm-up wait. The power link probably relies on 6-pin bus power, so the port we use must supply it (see point 12).

### 2. Electrical safety on 6-pin FireWire (medium-high)

6-pin FireWire carries bus power (8 to 30 V). Two machines on different outlets can have a ground offset, and hot-plugging is a known way to kill 1394 PHYs, on either side. A dead PHY in the image box would be a serious repair. Rule: one grounded power strip for scanner and PCs, plug and unplug FireWire only with everything off.

### 3. No abort and recovery procedure yet (high, as a gap)

Before Phase 3 sends anything beyond reads, we need a written answer to "how do I stop this safely and get back to a known state?" Is it a power-off sequence? Does the scanner need the original software to park the CCD and carrier? This can be learned in Phase 0 by watching how the original system handles an interrupted scan and a power cut, and from the service manual.

### 4. The init sequence and boot calibration are where this gets hard (medium)

Phase 3.3 ("scan one low-resolution frame") hides the biggest jump. The original software has a timing-sensitive boot of about 3 minutes, and too-fast hardware makes it miss checks. During boot it probably runs a calibration that moves the CCD and turns on the LEDs. We will have to reproduce that sequence before any frame, and it is the first step that moves things. Expect to get stuck here.

## Missing pieces

### 5. Phase 0 must map the FireWire topology (high)

We do not know yet:
- whether the Toshiba's two FireWire ports are one controller (one bus) or two controllers (two buses);
- what the cable from the MCU to the power supply carries, and whether it is on the same bus;
- which PCI controller the Toshiba uses (vendor:device ID) and its driver;
- which bus ID VDEntry assigned (a public Fujifilm installation document says the FireWire driver is set up with VDEntry, which sets the port's bus ID).

These answers decide how many FireWire ports the Linux machine needs and whether Linux must emulate a specific controller identity. The collection kit in `collect/` gathers most of it without touching the scanner.

### 6. Calibration and setup data live on the PC disk, probably (medium)

Per-scanner data (shading tables, lens and carrier adjustments, color setup) is probably stored on the scanner PC rather than in the scanner. Without it, Phase 4 cannot reach the original image quality, and the init sequence may even need to upload part of it. Phase 0 must back up those folders, and Phase 2 must find out what they are. They are proprietary data and stay out of git.

### 7. Start static analysis now, in parallel (medium)

Option 2c (Ghidra on the Fujifilm drivers and DLLs) needs no hardware, only a disk image of the user's own machine. It is probably the fastest way to the command map. Windows 2000 WDM drivers talk to `1394bus.sys` through IRBs (`REQUEST_ASYNC_READ`, `REQUEST_ASYNC_WRITE`, `REQUEST_ASYNC_LOCK`, `REQUEST_ISOCH_*`), and those call sites are easy to find and cross-reference. It can start the day the disk image exists.

### 8. A technique the plan lacks: an IRB logging filter on Windows 2000 (medium)

A small lower filter driver on the Fujifilm device stack, logging every IRB with its address, length and payload, gives the protocol from the host side. It is simpler than instrumenting QEMU and needs no extra hardware. It must only ever run in a **VM snapshot or on a cloned disk**, never on the original.

### 9. Isolate bus access behind one layer, and pin a kernel (medium)

The FireWire subsystem is maintained today (`MAINTAINERS`: `S: Maintained`, high confidence). Secondary sources say the maintainer plans to hand off to distributions in 2027-2028 and step down in 2029 (medium confidence). The code will likely stay in LTS kernels for years after that, but we should pin an LTS kernel (Debian 13 ships 6.12) and keep all bus access in one module (firewire-cdev), so that a future backend change does not touch the rest of the code.

### 10. Contact people early (no confidence needed)

Adre Lanine (Facebook group) and the fujirawconvert author (GitHub issue #2 already asks about MS11 to scanner traffic) may share findings or collaborate, which could save months.

## Wrong order

### 11. Observe the original system before Linux becomes the host (medium)

Decision (2026-11-26): **no hardware purchases.** We build with the machines we already have; Adre Lanine runs his version on a laptop. That rules out the nosy bus sniffer, which needs a rare TI PCILynx PCI card. Observation instead comes from the host side:
- static analysis of the Fujifilm drivers on our own disk image (point 7);
- a passive logger in the Windows 2000 system (point 8), run in a VM snapshot or on a cloned disk, recording every 1394 request and response the Fujifilm software makes during a normal boot and scan.

This still comes **before** Linux becomes the host: we first see what Fujifilm's own software sends, then repeat only that.

### 12. FireWire on the hardware we have (medium)

Whatever machine we use must have a FireWire controller that Linux drives with `firewire_ohci` (check with `lspci -nnk` or, for Thunderbolt adapters, `lspci` after connecting). Older MacBook Pros have a built-in FireWire 800 port, and Thunderbolt-to-FireWire adapters exist; whether one of those can power up the scanner through its link is unknown (low confidence). Kernel facts worth knowing: VIA VT6306/7/8 behind an ASMedia ASM1083/1085 bridge on AMD Ryzen has an unexpected-reboot quirk (`ohci.c`, high confidence), and the scanner's power link probably needs a port that supplies 6-pin bus power (medium confidence; FireWire 800 ports on Macs do supply power, adapters may not).

## Open unknown that sets the project size

### 13. High-level or low-level commands? (low-medium)

If the scanner MCU accepts high-level commands ("scan frame N at resolution R with 4 shift positions") and handles motors and pixel shift itself, the project is mostly protocol plus image processing. If the PC drives each step (CCD shift, LED timing, carrier moves), the project is much bigger and riskier. The config ROM unit directory (specifier ID, version) and the static analysis should answer this early.

## Proposed order

1. **Phase 0**: preserve and document, including the FireWire topology and calibration folders.
2. **Phase 2c**, static analysis of the user's own disk image.
3. **Phase 1a**, passive host-side logging of the original system (Windows 2000 in a VM snapshot or on a cloned disk).
4. **Phase 1b**, Linux as host, read-only, with transmitting drivers blacklisted and tracepoints on.
5. **Phase 2d**, more captures, matching them with menu 2241 dumps.
6. Write the abort and recovery procedure.
7. **Phase 3**, reproduction, one approved step at a time.
