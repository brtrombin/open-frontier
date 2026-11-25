# Journal

One entry per working session: date, what was done, evidence, confidence. Newest first. Only our own observations; no Fujifilm material and no private data from collaborators' machines.

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
