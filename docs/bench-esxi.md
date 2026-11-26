# Working on an SP-3000 virtualised on ESXi

> Correction (2026-11-27): the owner's scanner PC is not virtualised; only a server runs as a VM, at a lab. The scanner PC parts of this guide do not apply to his system. Kept for systems set up along Chauvel's guide.

For the system we work on: an SP-3000 whose scanner PC and server run as two virtual machines on VMware ESXi, set up along Jack Chauvel's guide. The owner authorized tests, passive captures and analysis of his own disk images (2026-11-26). Everything collected from his system is private: it is shared in the project chat, never on GitHub.

Names used here:
- **DI**: the scanner PC VM (Windows 2000, Fuji scanner software, the FireWire card passed through to it);
- **PIC**: the server VM (Windows XP, front-end software);
- **host**: the PC that runs ESXi; **Host Client**: ESXi's web page, opened from another PC's browser.

## Ground rules

- Work at the end of a day, with no scan running and no order in progress.
- Change no setting in the Fuji software. Never connect this DI VM to another scanner: its calibration belongs to this scanner.
- Every step says what it changes and how to undo it. If anything unexpected shows up, stop and ask in the project chat.
- ESXi takes no snapshots of a VM that has a PCI device passed through, such as the FireWire card (medium-high confidence). So for DI, "undo" means restoring a copy made beforehand (B0).
- Windows 2000 and XP have no USB 3 driver. When a step adds a USB controller to a VM, choose USB 2.0.

## B0. Backups

Changes nothing. Undo: nothing to undo.

1. Shut down Windows in both VMs (Start, Shut Down) and wait until the Host Client shows them powered off.
2. In the Host Client, for each VM: Virtual Machines, select the VM, Actions, Export. Save the files to an external drive. If Export fails, download every file of the VM's folder instead (Storage, Datastore browser).
3. Write down the SHA256 of each file. In a Windows command prompt: `certutil -hashfile FILE SHA256`.
4. Copy everything to a second drive and keep it in another place.
5. Keep the Clonezilla image of the Toshiba disk in two places too. If possible, check it with Clonezilla's own image check.
6. Leave the physical Toshiba and its disk as they are: they are the fallback if everything else fails.
7. Start the VMs again the usual way.

## B1. One finished order through openfrontier

Changes: a USB device added to a VM's settings for a while. Undo: remove it.

1. Find where InSpool is. On a Chauvel-style system it is probably on PIC, as `D:\InSpool` (medium confidence; B2 tells for sure).
2. Attach a USB stick to that VM: plug it into the host, then in the Host Client, the VM, Edit settings, Add other device, USB device. If the VM has no USB controller, add one first (USB 2.0).
3. With the order finished and no scan running, copy to the stick:
   - one order folder from InSpool (an 8-digit name);
   - the matching `.INF` files, if they are in OutSpool rather than beside the `.RAW` files;
   - the front-end's JPG or BMP files of the same order.

   Copy only. Never move, rename or delete anything in InSpool or OutSpool.
4. Remove the USB device from the VM's settings.
5. On any Windows 10 PC, follow T1 in `docs/windows-test-plan.md` (`inspect`, `convert`, `compare`).

## B2. Collection kit in both VMs

Changes: a USB device added to each VM's settings for a while. Undo: remove it.

1. Put `collect/scanner-pc-win2000.bat` and `collect/server-xp.bat` on the stick.
2. Attach the stick to DI and run `scanner-pc-win2000.bat` from it, as `collect/README.md` describes. Then attach it to PIC and run `server-xp.bat`.
3. From the Host Client, also collect:
   - the `.vmx` file of each VM (Storage, Datastore browser, the VM's folder, Download). It shows how the FireWire card is passed through;
   - the ESXi version (Help, About);
   - the name of the FireWire card (Manage, Hardware, PCI devices).
4. Send the `collect-...` folders, the `.vmx` files and these notes privately.

B2 also tells where the order database (FDIA_DB) runs: on PIC or on DI. That decides what B3 can record.

## B3. Recording the network between DI and PIC

What it gives: the spool writes and the database queries between the two VMs, which we need to replace the server (S2 and S3). If the database runs on DI itself, its queries do not cross the network, and B3 records only the spool traffic. It does not touch the FireWire link, so the scanner is not affected (medium-high confidence).

Changes: the host's SSH service is switched on for the recording. Undo: switch it off. This step needs someone comfortable with a command line.

1. Host Client: Manage, Services, `TSM-SSH`, Start.
2. From a PC: `ssh root@HOST-ADDRESS`.
3. `net-stats -l` lists the virtual switch ports. Note the `PortNum` of DI's network card (its `ClientName` starts with the VM's name).
4. Check the options of your ESXi version with `pktcap-uw -h`, then start the recording on a datastore, for example:
   ```
   pktcap-uw --switchport PORTNUM --dir 2 -o /vmfs/volumes/DATASTORE/di.pcap
   ```
5. Restart DI the usual way, so the recording covers its startup. Then scan a short roll (two or three frames) as usual, and wait until the order is finished.
6. Stop the recording with Ctrl+C. Download `di.pcap` with the Datastore browser, then delete it from the datastore.
7. Host Client: Manage, Services, `TSM-SSH`, Stop.
8. Send the file privately. It contains the owner's orders and images.

## B4. Later: FireWire logging inside DI (Phase 1a)

Not yet. It needs a reviewed tool (an existing 1394 bus monitor that runs on Windows 2000, or our own logger), a fresh B0 copy of DI, and the owner's go-ahead on the day. The undo is restoring that copy.
