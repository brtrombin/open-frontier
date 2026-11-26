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

### 10. Contact people early (dropped)

Decision (2026-11-26): no outreach to other people. We work from our own equipment, public sources and the plain-text documentation we already have.

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

## Windows 10 as the target (added 2026-11-26, later)

Decision (2026-11-26): the program must run natively and standalone on Windows 10.

### 14. What runs natively on Windows 10 today (high)

The spool side. Reading `.RAW` and `.INF` files and writing JPEG/TIFF needs only file access, and Windows is where the documented safe read exists (opening with `FILE_SHARE_DELETE`). `src/` builds one `.exe` that does this with nothing to install.

### 15. FireWire from Windows 10 user space needs a driver (medium-high)

As far as we know, Windows 10's 1394 stack (`1394ohci.sys`) gives user-mode programs no way to send asynchronous or isochronous transactions: the user-mode interface we know of (`IOCTL_IEEE1394_API_REQUEST`) only adds or removes virtual devices for a kernel driver to bind to. Talking to the scanner from Windows therefore needs a kernel-mode driver of our own, along the lines of Microsoft's 1394 driver samples. A driver we build ourselves only loads in test-signing mode, with Secure Boot off (a persistent change on that PC); production signing needs a paid certificate, which the no-purchase rule excludes.

Linux gives the same access from user space with no driver (`firewire-cdev`). So the FireWire research (Phases 1b to 3) stays on Linux, for example from a Linux live USB on the same PC, which changes nothing on the Windows installation. The Windows driver question comes back once the protocol is known.

Hypothesis (low-medium): VDEntry, the Fujifilm tool that sets up the scanner PC's FireWire driver, may use that same virtual-device interface. The Fuji driver would then be a virtual-device driver that addresses the scanner node directly, rather than binding to a unit directory in the scanner's configuration ROM. The collection kit's export of `HKLM\SYSTEM\CurrentControlSet\Enum` would show it as a `V1394\...` entry.

### 16. No VM shortcut on a Windows 10 host (medium-high)

Jack Chauvel runs the scanner PC's software in a VM on ESXi, with the FireWire card passed through. Windows 10 client hypervisors cannot pass a PCI device to a guest: Hyper-V Discrete Device Assignment is a Windows Server feature, and VMware Workstation and VirtualBox on Windows have no PCI passthrough. The VM route remains possible on a separate machine running ESXi or Linux with KVM, not on the Windows 10 PC itself.

## Server-side safety rules (added 2026-11-26)

Any future tool that touches InSpool or FDIA_DB must follow these, confirmed from real failures documented in Sprocket's user documentation:

1. Never open InSpool files before `OrderTable.Status == 2`. A sharing violation aborts `ExecLSeqObj.exe` and wedges the scanner.
2. Never delete spool folders. A missing folder produces Fuji error W-5302 ("insufficient disk space", which really means "folder is gone").
3. Never write to `FDIA_DB`. The scanner manages its own state.
4. Never run the collection kit during a scan.
5. On Windows, open InSpool files with `FILE_SHARE_DELETE` (`FILE_SHARE_ALL`).

## Revised order (2026-11-26, later)

Supersedes the order below. The spool side comes first, because its formats are documented and it can be built and checked without touching the scanner; the target is a Windows 10 PC; there is no outreach.

1. **Phase 0**: preserve and document. Still first for anything that touches the original system.
2. **S1**, now: decode finished spool orders on Windows 10, offline, and compare with the Fuji output (`docs/windows-test-plan.md`). This confirms the RAW and INF formats on real files.
3. **S2**: a watcher that reads only finished orders (`OrderTable.Status == 2`), opening files with `FILE_SHARE_DELETE`. Needs read access to FDIA_DB and the SMB1 client on the Windows 10 PC, with the equipment owner's go-ahead.
4. **S3**: stand in for FDIA_DB over TDS, then retire the front-end server. Needs captures of the scanner PC's own SQL traffic first.
5. **Phase 2c**, in parallel: static analysis of the user's own disk image.
6. **Phase 1a**, then **1b** from a Linux live USB on the Windows 10 PC, then **Phase 2**, the abort and recovery procedure, and **Phase 3**, as below.
7. The Windows FireWire driver (point 15), once the protocol is known.

## Earlier proposed order (2026-11-25, superseded)

1. **Phase 0**: preserve and document, including the FireWire topology and calibration folders.
2. **Phase 2c**, static analysis of the user's own disk image.
3. **Phase 1a**, passive host-side logging of the original system (Windows 2000 in a VM snapshot or on a cloned disk).
4. **Phase 1b**, Linux as host, read-only, with transmitting drivers blacklisted and tracepoints on.
5. **Phase 2d**, more captures, matching them with menu 2241 dumps.
6. Write the abort and recovery procedure.
7. **Phase 3**, reproduction, one approved step at a time.
