# Test plan on Windows 10

Goal: run the first open-frontier program natively on a Windows 10 PC, with nothing to install, and check our reading of the Fuji spool formats against the scanner system's own output.

What the program does today:
- reads the `.RAW` and `.INF` files of a **finished** order and writes JPEG or TIFF;
- compares one of our frames with the Fuji software's export of the same frame;
- lists the FireWire hardware that Windows sees.

It never talks to the scanner, and it never writes to, renames in or deletes from the folders it reads.

What it does not do yet: control the scanner over FireWire (the protocol has not been observed), watch the spool live, or stand in for the order database.

## Get the program

- **Prebuilt**: `openfrontier.exe` for 64-bit Windows 10 and 11, `openfrontier-32bit.exe` for 32-bit Windows 10. One file, no installation, no runtime. Shared privately in the project chat. Windows SmartScreen may warn that the program is unsigned: "More info", then "Run anyway".
- **From source**, with Go 1.24 or later installed:
  ```bat
  cd src
  go build -trimpath -o ..\dist\openfrontier.exe .\cmd\openfrontier
  ```
  From Linux or macOS, add `GOOS=windows GOARCH=amd64` in front of `go build`.

Put the `.exe` in a folder such as `C:\openfrontier` and open a Command Prompt there. Run `openfrontier help` for all options. Dropping an order folder onto `openfrontier.exe` in Explorer runs `inspect` on it and keeps the window open.

## T0. FireWire check, scanner not connected

Risk: none. Nothing is connected to the PC's FireWire port.

```bat
openfrontier firewire > firewire.txt
```

It asks Windows (through PowerShell, read-only) for its IEEE 1394 controllers, their drivers and anything on the bus. Send `firewire.txt`. It tells us whether this PC has a FireWire controller that works, which chip, and which driver.

## T1. One finished order, offline

Risk: none to the scanner, if the copy is made as described.

1. **Make a copy, on the machine that holds the spool** (the scanner PC or the server). Only when **no scan is running** and the order is **finished**:
   - copy one order folder from InSpool (an 8-digit folder such as `00006360`) to a USB stick;
   - if the `.INF` files are not beside the `.RAW` files (Fuji's own software puts them in OutSpool), also copy the matching folder or files from OutSpool.

   Do not move or delete anything in InSpool or OutSpool, and leave the copy's folder name as the order number. The collection kit listing (`inspool-listing.txt`) shows which order folders are there. If the scanner PC and the server run as VMs on ESXi, `docs/bench-esxi.md` (B1) says how to get the files out.
2. Copy the Fuji software's own output (JPEG or TIFF) of the **same** order to the stick.
3. On the Windows 10 PC, copy both from the stick to a local folder, for example `C:\of-test\order\00006360` and `C:\of-test\fuji\`.
4. Run, one at a time:
   ```bat
   openfrontier inspect C:\of-test\order\00006360 > inspect.txt
   ```
   ```bat
   openfrontier convert C:\of-test\order\00006360 --out C:\of-test\out
   ```
   ```bat
   openfrontier compare C:\of-test\order\00006360\00000001.RAW C:\of-test\fuji\FRAME.jpg > compare.txt
   ```
   Replace `FRAME.jpg` with the Fuji file of the same frame. If the `.INF` files came from OutSpool, add `--inf-dir C:\of-test\outspool` to `inspect` and `convert`.
5. Open one of our files next to the Fuji file of the same frame: same picture, same orientation, same colors?
6. Send `inspect.txt`, `compare.txt` and what you saw. The RAW, INF and image files stay private and never go to GitHub. The reports contain the scanner's serial number (`00_SERIAL_NO`): share them privately too.

What the reports tell us:
- `inspect`: where the header keeps width and height, whether the pixels are interleaved or planar, which keys the `.INF` really holds, and their types;
- `compare`: the orientation and channel order that make our decode match Fuji's, the meaning of the `.INF` rotation value, and whether the RAW already carries the final Fuji rendering (a difference under about 6 of 255 means the same rendering).

If a folder on the path is named InSpool, OutSpool or ImgExchange, or the path is on a network drive, the program refuses to read it. That is on purpose: reading the scanner's live spool during a scan can abort the Fuji software and wedge the scanner. Rename the copy's parent folder (for example `order` instead of `InSpool`), or add `--live-spool-ok` only when the order is finished and no scan is running.

## Later steps, each one needs the equipment owner's go-ahead

- **Reading the spool over the network.** Windows 10 has no SMB1 client by default, and the Windows 2000 and XP machines only speak SMB1. Reading their shares needs the "SMB 1.0/CIFS Client" Windows feature, a setting change on the Windows 10 PC. Even then: finished orders only, no scan running. The program opens every spool file with `FILE_SHARE_DELETE`, the documented way to read without blocking the Fuji software.
- **Connecting the PC to the scanner's FireWire** (Phase 1b). A powered PC on the scanner's FireWire link can switch the scanner on. Windows will reset the bus, read the scanner's configuration ROM and may attach its own drivers. Read `docs/phase1-runbook.md` and `docs/plan-review.md` first. After connecting, `openfrontier firewire` shows what Windows found, including the hardware IDs built from the scanner's configuration ROM.
