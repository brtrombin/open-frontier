# src/

Future scanner control and image processing code. **Empty on purpose**: no code talks to the scanner until the protocol has been observed and documented (`docs/protocol.md`) and each step has been approved by the equipment owner.

Planned layers (subject to change):
1. `bus`: the only module that touches FireWire (firewire-cdev ioctls on `/dev/fw*`), so a future backend change stays contained.
2. `scanner`: protocol, init sequence, commands, abort.
3. `image`: raw decode, dark/flat calibration, pixel-shift assembly, inversion and color, IR.
4. `web`: queue, preview and export UI.
