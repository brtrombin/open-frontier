# Collection kit for people who run an SP-3000 today

If you run a working Frontier SP-3000, an hour of your time gives this project facts that are not published anywhere: which FireWire controller the scanner PC uses, the Fujifilm driver names and versions, the software version, boot timing and cabling.

**Nothing here talks to the scanner or changes your machines.** The two scripts only run built-in Windows listing commands and write their output to the USB stick. They never delete anything, never write to the registry and never copy programs or drivers.

## What you need

- One USB stick, FAT32, 8 GB or more.
- A phone for photos and video.
- About an hour, at a time when no scan is running (end of the day is best).

Rule for every step: **do not change any setting.** When a window offers OK/Apply/Save, close it with **Cancel**.

**IMPORTANT: run the scripts only when no scan is running.** Reading InSpool files during a scan can cause a sharing violation that aborts the scanner software and wedges the scanner. Wait until the scanner is idle.

## Steps

### 1. Photos (everything powered off)

- The back of the scanner PC (Toshiba) and of the server, with all cables plugged in.
- Each FireWire cable, both ends, so we can see where it starts and where it ends.
- The FireWire connector on the scanner side (close-up).
- The cable that goes from the scanner's MCU to its power supply unit, if you can reach it.
- Model and serial labels of the scanner, the Toshiba and the server.

### 2. Timed boot video

Put a stopwatch (another phone works) in the frame. Start with everything off and power on in your usual order, saying each action aloud ("scanner on", "Toshiba on"...). Keep filming until the scanner says it is ready.

### 3. Software version screens

In the Frontier software, open Setup and Maintenance and look for a version or system information screen. Photograph it. Leave without saving.

### 4. Device Manager on the scanner PC

Right-click My Computer > Properties > Hardware > Device Manager.
- Open **"IEEE 1394 Bus host controllers"**. For each item inside it, open Properties and photograph the **General**, **Driver** (and its **Driver Details** button) and **Resources** tabs.
- Do the same for any item with "Fuji" or "Frontier" in its name, in any category.
- Close everything with **Cancel**.

### 5. Run `scanner-pc-win2000.bat` on the scanner PC

1. Copy `scanner-pc-win2000.bat` to the USB stick.
2. Plug the stick into the Toshiba.
3. Open the stick in My Computer and double-click the file. It must run **from the stick**: it refuses to run from C: or D:.
4. Wait until it says **DONE** (several minutes; the file listing and the hardware report are slow).
5. If a Windows message box appears (for example a registry key that cannot be read), click **OK** and let it continue.
6. If the hardware report step opens a "System Information" window instead of finishing by itself, use its menu to save as a text file onto the stick (name it `msinfo32.txt`), then close it.

It creates a folder named `collect-<computer name>-1` on the stick.

### 6. Run `server-xp.bat` on the server

Same as step 5, on the front-end server. If Windows asks you to insert a disk into a CD drive, click **Cancel** or **Continue**.

### 7. Optional, only if the software is MS11 and you are comfortable with it

This step changes one setting, which you must turn back off at the end.
1. Setup and Maintenance > Self-diagnostics > Image data diagnostics (menu 2241): set **Image Log Storage** = **On**.
2. Scan 3 frames of the same film at each resolution, and keep the final JPEG/TIFF of those frames.
3. Copy the `.bin` and `.tag` files from `D:\Fujifilm\Frontier\Log` to the stick.
4. Set **Image Log Storage** back to **Off** and check that it shows Off.

### 8. Send it back

Zip the `collect-...` folders, the photos, the video and (if done) the step 7 files. Send them **privately** (cloud drive link or file transfer) to the person who asked you. Please do not upload them to GitHub or post them publicly.

## What is in the results, and who it belongs to

| Item | Contains | Belongs to | Where it goes |
|---|---|---|---|
| `files-*.txt`, `drivers-list.txt` | File names, dates and sizes | You (listing of your disk) | Private |
| `registry/*.reg`, `fujifilm-keys.txt` | Hardware enumeration, driver and Fujifilm settings | You and Fujifilm | Private |
| `OPTIONAL-may-contain-licenses/` | All installed software settings, possibly license keys | You and software vendors | Send only if you agree |
| `config-files/` | Fujifilm `.ini`/`.cfg` settings | Fujifilm format, your settings | Private |
| `network.txt` | IP addresses, computer names, shares | You | Private |
| `msinfo32.txt` | Hardware and driver report | You | Private |
| Photos of Fujifilm screens | Fujifilm user interface | Fujifilm | Private |
| `.bin`/`.tag` from menu 2241 | Raw scans of your film in a Fujifilm format | You (images), Fujifilm (format) | Private |

None of this is ever committed to the open-frontier repository. The project only publishes facts we derive ourselves, in our own words (for example "the scanner PC uses a controller with PCI ID xxxx:yyyy").

## What the scripts run

Scanner PC (Windows 2000): `ver`, `set`, `ipconfig /all`, `net share`, `net use`, `net config`, `net start`, `dir` (including `D:\InSpool`, `D:\OutSpool`, `D:\ImgExchange` by name), `regedit /e` (export, read-only, including `HKLM\SOFTWARE\FUJIFILM` explicitly), `xcopy` of `*.ini`/`*.cfg` from Fujifilm/Frontier folders only, `msinfo32 /report`.

Server (Windows XP and later): the same, plus `systeminfo`, `sc query`, `driverquery /v`, `reg query` (including `HKLM\SOFTWARE\FUJIFILM` and related keys) and `reg export` (read-only), and `dir` of InSpool/OutSpool/ImgExchange.

Every output path is inside the new folder on the USB stick. Read the `.bat` files before running them: they are short plain text.
