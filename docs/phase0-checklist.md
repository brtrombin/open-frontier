# Phase 0 checklist: preserve the original system

Goal: at the end of Phase 0, the original Toshiba + server can be rebuilt from backups in an afternoon, and we know everything about the setup that we can learn without touching the FireWire bus.

Scanner PC and server running as VMs on ESXi (the system we work on): most of this phase is done differently, see `docs/bench-esxi.md` (B0 for the copies, B2 for the collection kit). Sections A, B, D and E still apply as written.

Rules for the whole phase:
- Change **nothing** persistent unless a step says so, and undo it afterwards.
- Everything collected here (disk images, photos of Fujifilm screens, registry and ini files, scans, dumps) is private or proprietary. It goes to the external drives, **never to git**.
- Log each completed block in `docs/journal.md`, with our own observations only.

Tick each box and write the result next to it.

## A. Identify the setup

- [ ] **A1.** Write down the scanner model, serial number, and whether it has one or two GPA23 boards (service manual or label). Result: ______
- [ ] **A2.** Identify the scanner PC software: **A1**, **MS01+MS11**, or **S1**. Look in Setup and Maintenance for a version/system information screen, and note the version numbers. Result: ______
- [ ] **A3.** Identify the server software (C4/C5, S2, MS01, PTS SSI, Sprocket) and its version. Result: ______
- [ ] **A4.** Note the operating mode (connected to a printer, or standalone "S-Alone"/"Printer OFF"). Result: ______

## B. Photographs (everything off)

- [ ] **B1.** Back of the Toshiba: every port, with every cable plugged in.
- [ ] **B2.** Every FireWire cable, both ends: where it starts, where it ends. Label each cable (FW-A, FW-B, ...).
- [ ] **B3.** The FireWire connector on the scanner side: 6-pin, 4-pin or 9-pin?
- [ ] **B4.** The cable from the MCU to the power supply unit, both ends.
- [ ] **B5.** Model and serial labels of the scanner, the Toshiba and the server.
- [ ] **B6.** Toshiba BIOS setup: every page (boot order, IDE/SATA mode, integrated devices, power management).
- [ ] **B7.** Draw a cabling diagram from the photos and store it with them.

## C. Software and topology, read-only

- [ ] **C1.** Run the collection kit (`collect/scanner-pc-win2000.bat`) on the Toshiba and `collect/server-xp.bat` on the server. Follow `collect/README.md`.
- [ ] **C2.** Device Manager on the Toshiba: under "IEEE 1394 Bus host controllers", count the controllers. Expected result: 1 controller (both ports on one bus) or 2 controllers (two buses). Result: ______
- [ ] **C3.** For each 1394 controller, photograph the General, Driver (and "Driver Details"), and Resources tabs. Close with **Cancel**.
- [ ] **C4.** Same for every device with "Fuji" or "Frontier" in its name, in any Device Manager category.
- [ ] **C5.** If the VDEntry tool exists on the Toshiba, photograph its screen showing the configured bus ID. **Do not change anything, close with Cancel.** If you are not sure it can be opened without saving, skip this step.

## D. Timing and behavior

- [ ] **D1.** Timed boot video: everything off, then power on in the usual order, with a stopwatch in frame. Say each action aloud. Stop when the scanner reports ready. Result (total time): ______
- [ ] **D2.** Note any messages on the scanner panel or monitor during boot.
- [ ] **D3.** Abort behavior: during a normal **preview** scan, press the software's cancel/stop button once and film what the scanner does (carrier, sounds, messages). Skip if the operator is not comfortable with it.
- [ ] **D4.** Shutdown: film the normal shutdown order.

## E. Reference scans

Use one roll of film that you keep as the reference for the whole project.

- [ ] **E1.** Scan the reference roll at **every** resolution and size the software offers. Save all output files (JPEG/TIFF) with the settings in the file name.
- [ ] **E2.** If the software is **MS11**: Setup and Maintenance > Self-diagnostics > Image data diagnostics (menu 2241), set "Image Log Storage" = **On**.
- [ ] **E3.** Scan 3 frames at each resolution. Copy the .bin + .tag files from `D:\Fujifilm\Frontier\Log` to the external drive, together with the final output of the **same** frames.
- [ ] **E4.** Set "Image Log Storage" back to **Off**. Confirm on screen. (This step undoes the only persistent change in Phase 0.)

## F. Configuration and calibration data

- [ ] **F1.** Use the directory listing from the collection kit to find the Fujifilm folders on C:\ and D:\ (configuration, setup, calibration, correction, lens, shading or similar names).
- [ ] **F2.** Copy those folders entirely to the external drive (this is in addition to the full disk clone, so they can be studied without mounting an image).
- [ ] **F3.** If the software has a "backup setup data" or similar function, use it and keep the result. Note the menu path. Result: ______
- [ ] **F4.** Keep every recovery CD and installation CD, with photos of their labels and version numbers.

## G. Disk clones

- [ ] **G1.** Shut down normally. Power off the scanner.
- [ ] **G2.** Clone the Toshiba disk, either:
  - with Clonezilla Live **i686** (32-bit) from USB, to the external drive (may be slow if the Toshiba has only USB 1.1), or
  - by removing the disk and imaging it on another PC through the IDE/SATA-USB adapter.
  Use a full disk image (all partitions, including hidden ones).
- [ ] **G3.** Write down the SHA256 of the image files. Result: ______
- [ ] **G4.** Clone the server disk the same way, with its SHA256.
- [ ] **G5.** **Restore test**: restore the Toshiba image to the spare disk, put it in the Toshiba, boot, and run one preview scan. Then put the original disk back. A backup is only proven when a restore works. Result: ______
- [ ] **G6.** Copy all images and collected data to the **second** external drive, and store it in a different place.

## H. Close

- [ ] **H1.** Original disks back in, everything cabled as in the photos.
- [ ] **H2.** Normal boot and one normal scan of the reference roll. It must work exactly as before. Result: ______
- [ ] **H3.** Journal entry in `docs/journal.md`: date, what was done, where the backups are (generic description, not private paths), open findings.
