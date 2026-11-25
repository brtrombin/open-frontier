# Journal

One entry per working session: date, what was done, evidence, confidence. Newest first. Only our own observations; no Fujifilm material and no private data from collaborators' machines.

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
