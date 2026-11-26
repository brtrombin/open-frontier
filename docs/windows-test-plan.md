# Test plan on Windows 10

Goal: run open-frontier natively on a Windows 10 PC, with nothing to install, and get to hands-off delivery: each order the scanner finishes lands as a named folder of JPEG or TIFF files on an SSD or NAS, the way Sprocket does it.

```
Scanner ── FireWire ── scanner PC (as it is today)
                            │  network (SMB1)
                   PC with openfrontier watch
                            │
                   SSD or NAS: one folder per order
```

The program never talks to the scanner, and never writes to, renames in or deletes from the folders it reads. It refuses folders that may be the scanner's live spool (a network path, or a folder named InSpool, OutSpool or ImgExchange) unless the conditions below are met.

## Get the program

- **Prebuilt**: `openfrontier.exe` for 64-bit Windows 10 and 11, `openfrontier-32bit.exe` for 32-bit Windows 10. One file, no installation. Shared privately in the project chat. Windows SmartScreen may warn that it is unsigned: "More info", then "Run anyway".
- **From source**, with Go 1.24 or later: in `src`, `go build -trimpath -o ..\dist\openfrontier.exe .\cmd\openfrontier`. From Linux or macOS, add `GOOS=windows GOARCH=amd64` in front.

Put it in a folder such as `C:\openfrontier`, open a Command Prompt there, and run `openfrontier help` for every option.

## T1. The whole chain with a simulated scanner (today, no scanner)

Risk: none. Everything happens in two test folders.

Open two Command Prompt windows. In the first, start the watcher:

```bat
openfrontier watch C:\of-test\spool --out C:\of-test\ssd --quiet 10
```

In the second, let the simulator write two orders the way a scanner fills its spool:

```bat
openfrontier simulate C:\of-test\spool --orders 2 --frames 4 --size normal
```

About ten seconds after each order stops growing, the watcher prints `delivered 4 frames` and the folder `C:\of-test\ssd\00000001` appears with four JPEGs (frames 02A and 04A rotated). Stop the watcher with Ctrl+C. Run it again: it delivers nothing twice. Its log and memory of delivered orders are in `C:\of-test\ssd\.openfrontier`.

Without the order database, `watch` decides an order is finished when it stops changing (`--quiet`, in seconds). It only allows that on test folders and copies; a live spool needs the database (T3).

## T2. Real files from the scanner, offline

Risk: none to the scanner, if the copy is made as described.

1. On the scanner PC (or wherever InSpool is), with no scan running and the order finished, copy one order folder (8 digits, such as `00006360`) to a USB stick, plus the matching `.INF` files from OutSpool if they are not beside the `.RAW` files, plus the Fuji output (JPEG, BMP or TIFF) of the same order. Copy only: never move or delete anything in the spool. The collection kit's `spool-*.txt` listings show what is there.
2. On the Windows 10 PC, copy them to a local folder, for example `C:\of-test\order\00006360` and `C:\of-test\fuji\`, then:
   ```bat
   openfrontier inspect C:\of-test\order\00006360 > inspect.txt
   ```
   ```bat
   openfrontier compare C:\of-test\order\00006360\00000001.RAW C:\of-test\fuji\FRAME.jpg > compare.txt
   ```
   ```bat
   openfrontier watch C:\of-test\order --out C:\of-test\ssd --once
   ```
   Add `--inf-dir FOLDER` if the `.INF` files came from OutSpool.
3. Open our JPEG next to the Fuji one: same picture, orientation and colors? Send `inspect.txt` and `compare.txt` privately; they contain the scanner's serial number. The files themselves never go to GitHub.

They tell us where the header keeps width and height, whether the pixels are interleaved, the channel order, what the `.INF` rotation value means, and whether the RAW already carries the final Fuji rendering (a `compare` difference under about 6 of 255).

## T3. Reading the order database (on a visit, with the owner)

Risk: very low. The program only runs SELECT queries with NOLOCK; it cannot change the database.

1. Connect the Windows 10 PC to the scanner PC's network. Both Windows 2000 and XP speak only SMB1: on Windows 10, turn on "SMB 1.0/CIFS Client" (Control Panel, Programs, Turn Windows features on or off). It is a setting on the Windows 10 PC, and can be turned off again afterwards.
2. Log in to the scanner PC's shares once, with its Windows account (the factory default is FRONTIER / FRONTIER):
   ```bat
   net use \\EZPC00\IPC$ /user:FRONTIER FRONTIER
   ```
3. Read the order database through the named pipe, which MSDE accepts without any change on the scanner PC:
   ```bat
   openfrontier db EZPC00 --db-pipe
   ```
   It prints the server version and the latest orders with their status. If it fails, it says why; send that line. Replace `EZPC00` with the scanner PC's name or IP address (the collection kit's `network.txt` has both).

## T4. Hands-off delivery from the live spool (with the owner's go-ahead)

Only after T2 and T3 work.

```bat
openfrontier watch \\EZPC00\InSpool --out D:\Scans --db-host EZPC00 --db-pipe
```

Each order is read only when the order database says the scan finished (`OrderTable.Status = 2`). It is copied to the PC in one pass, through reads that let the Fuji software keep writing, renaming and deleting, then converted and moved into `D:\Scans\<order number>\` in one step, so nothing downstream ever sees half a roll. Scan one test roll, then compare with what the Fuji software delivers.

What to watch for during the first live rolls: the Fuji software must finish every roll normally. If it ever stops with an error while `watch` runs, stop `watch` and report it.

## Also available

- `openfrontier firewire > firewire.txt`: the FireWire controllers and devices Windows sees, read-only. Run it with the scanner not connected.
- `openfrontier convert ORDER --out DIR`: convert one copied order by hand.
