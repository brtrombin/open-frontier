# src/

Future scanner control and image processing code. **Empty on purpose**: no code talks to the scanner until the protocol has been observed and documented (`docs/protocol.md`) and each step has been approved by the equipment owner.

Planned layers (subject to change):
1. `transport`: the only layer that touches hardware. One backend per link type: FireWire for the SP-3000 (firewire-cdev ioctls on `/dev/fw*`), and later USB (libusb, reported for the SP-500) or SCSI (`/dev/sg*`, reported for some SP-2000/SP-1500). Everything above it is link-agnostic.
2. `scanner`: protocol, init sequence, commands, abort. One model profile per scanner (SP-3000 first), sharing whatever the family turns out to have in common.
3. `image`: raw decode, dark/flat calibration, pixel-shift assembly, inversion and color, IR.
4. `web`: queue, preview and export UI.
