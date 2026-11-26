# Journal

One entry per working session: date, what was done, evidence, confidence. Newest first. Only our own observations; no Fujifilm material and no private data from collaborators' machines.

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

**Changes made**: `CLAUDE.md` (links, power-on behavior), `hardware.md` (bus power requirement), `plan-review.md` (new point 1b), `phase1-runbook.md` (power-on order in 1b).

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
- A lab technician (Xanap Filmlab, Bangkok, runs an HS-1800 and an SP-3000) runs the whole Frontier system on a modern PC with the scanner software in a VM, using a StarTech PEX1394A2V (2-port 1394a PCIe) card. Supports our card choice in `hardware.md` (medium confidence).
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
- Hunt for a PCILynx card.
