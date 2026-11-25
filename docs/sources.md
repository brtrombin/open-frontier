# Sources

Public references only. Links and our own one-line summaries, never copies. Documents marked *Fujifilm* are copyrighted: read them, do not mirror them here.

## The SP-3000 and its software

- Jack Chauvel, [Guide to virtualising a Fuji Frontier SP-3000](https://www.jackchauvel.com.au/guide-to-virtualising-a-fuji-frontier-sp3000/): architecture, ESXi setup, VIA FireWire card in passthrough, boot timing.
- [Sprocket](https://sprocket.boutiquefilmlab.com/) (Boutique Film Lab): replacement front-end; decodes MS11 RAW/INF spool files.
- [Fotronix SP-3000 page](https://www.fotronix.uk/sp3000) and [MS11 software](https://www.fotronix.uk/shop/ms11-software/).
- [MS01 Basic software v10](https://minilablaser.com/ms01-basic-software-v-10/).
- *Fujifilm*, MS11 SP-3000 manual: https://fotobrell.com/wp-content/uploads/pp3-b1271e2-1.pdf
- *Fujifilm*, SP-3000 Frontier installation procedure (Scribd): https://www.scribd.com/document/382515868/5-SP3000-Frontier-Installation-Procedure
- *Fujifilm*, Installing the FireWire drivers on the FMPC (Scribd): https://www.scribd.com/document/100026080/Installing-the-Firewire-Drivers-on-the-Fmpc. The driver is set up with the VDEntry tool, which assigns the FireWire port's bus ID.
- *Fujifilm*, Service manual PP3-C1056E, SP-3000/LP5500/LP5700 (Scribd): https://www.scribd.com/document/859232430/Pp3-c1056e-Sp3000-Lp5500-Lp5700-Service-Manual
- Forum: minilabhelp.com, Fuji subforum (blocks automated reading).
- Adre Lanine, "Frontier SP-3000 Modernisation": https://www.facebook.com/groups/417276349694136/permalink/1855588375862919/

## Raw data

- [fujirawconvert](https://github.com/torarin/fujirawconvert): menu 2241 .bin + .tag to TIFF. Issue #2 (2026-11-10) asks whether anyone has looked at the MS11 to scanner communication.

## Linux FireWire

- Kernel `drivers/firewire/Kconfig`, option `FIREWIRE_NOSY`: https://github.com/torvalds/linux/blob/master/drivers/firewire/Kconfig
- Kernel `MAINTAINERS`, entry "FIREWIRE SUBSYSTEM": https://github.com/torvalds/linux/blob/master/MAINTAINERS
- Kernel `drivers/firewire/ohci.c` (controller quirks): https://github.com/torvalds/linux/blob/master/drivers/firewire/ohci.c
- Kernel FireWire driver interface guide (firewire-cdev): https://www.kernel.org/doc/html/latest/driver-api/firewire.html
- Subsystem documentation and roadmap: https://ieee1394.docs.kernel.org/
- Original nosy submission (2010): https://lkml.iu.edu/hypermail/linux/kernel/1007.2/02194.html
- Tom's Hardware, [Linux to support FireWire until 2029](https://www.tomshardware.com/news/linux-to-support-firewire-until-2029).
- Phoronix, [FireWire maintainer looking for testers](https://www.phoronix.com/news/Firewire-Linux-6.12).
- TidBITS, [FireWire removed from macOS 26 Tahoe](https://tidbits.com/2025/09/19/support-for-firewire-removed-from-macos-26-tahoe/).

## Hardware

- Unibrain, [FireBoard 400 1394a Lynx-2 PCI adapter](https://www.unibrain.com/products/fireboard-400-1394a-lynx-2-pci-adapter/) (TSB12LV21B + TSB41AB3) and [adapter FAQ](https://www.unibrain.com/faq/adapters-fa/) (Lynx vs OHCI variants).
- IOI, [IOI-1394TT spec](https://ioi.com.tw/spec/cat_106/1060006.htm).

## Method models

- RME Fireface protocol, reverse-engineered by sniffing the 1394 bus on Linux: https://docs.rs/firewire-fireface-protocols/latest/firewire_fireface_protocols/
