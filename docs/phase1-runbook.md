# Phase 1 runbook: identify the FireWire nodes, read-only

Prerequisite: Phase 0 complete (backups proven by a restore test).

Phase 1 has three stages. **1.0 never touches the scanner.** 1a observes the original Fujifilm software from inside Windows 2000. 1b puts our Linux machine on the scanner's FireWire bus. 1a and 1b each need an explicit go-ahead from the equipment owner before they start.

We use the hardware we already have (a desktop or a laptop with a FireWire port). No purchases.

Applies to Debian 13 or Ubuntu 24.04 LTS. Commands are run as a normal user with `sudo`.

---

> **On the Windows 10 PC** (the project's target since 2026-11-26): the machine check is `openfrontier firewire` with no scanner connected (`docs/windows-test-plan.md`, T0). The Linux steps below can run from a Linux live USB on the same PC, so nothing is installed on the Windows disk. A live system forgets `/etc/modprobe.d` at every boot and may load `firewire_ohci` before you can stop it: boot it with `modprobe.blacklist=firewire_ohci` added to the kernel command line, then create the file of step 1.0.2 in the running session (skip `update-initramfs` and the reboot) before loading the driver by hand.

## 1.0 Prepare the Linux PCs (no scanner connected)

### 1.0.1 Install tools

```bash
sudo apt update
```

```bash
sudo apt install linux-firewire-utils pciutils usbutils trace-cmd
```

`linux-firewire-utils` provides `lsfirewire`. Package names can differ slightly between Debian and Ubuntu.

### 1.0.2 Stop every FireWire driver from loading by itself

The goal is that nothing is ever sent on the bus unless we load the controller driver by hand. Create the file `/etc/modprobe.d/open-frontier.conf`:

```bash
sudo tee /etc/modprobe.d/open-frontier.conf >/dev/null <<'EOF'
# open-frontier: keep the FireWire bus silent until loaded by hand.
# Controller drivers (load manually with modprobe).
blacklist firewire_ohci
# Protocol drivers that would bind to devices and transmit.
blacklist firewire_net
blacklist firewire_sbp2
blacklist snd_firewire_lib
blacklist snd_bebob
blacklist snd_dice
blacklist snd_oxfw
blacklist snd_fireworks
blacklist snd_firewire_digi00x
blacklist snd_firewire_tascam
blacklist snd_firewire_motu
blacklist snd_fireface
blacklist snd_isight
install firewire_net /bin/false
install firewire_sbp2 /bin/false
EOF
```

`install ... /bin/false` makes the module impossible to load even as a dependency. `firewire_ohci` is only blacklisted (not forbidden), so `modprobe` still loads it when we ask explicitly.

```bash
sudo update-initramfs -u
```

```bash
sudo reboot
```

### 1.0.3 Check the PC before connecting anything

```bash
lsmod | grep -Ei 'firewire|snd_(bebob|dice|oxfw|fire)' || echo "OK: no FireWire module loaded"
```

```bash
lspci -nnk | grep -A3 -Ei '1394|firewire'
```

Expected: your FireWire controller with its vendor:device ID (for example `[1106:3403]` VIA VT6315, `[104c:823f]` TI XIO2213, `[11c1:5901]` LSI/Agere FW643 on many Macs), and no "Kernel driver in use" line. If nothing shows up, the machine has no FireWire controller Linux can see: tell us the model before going further.

```bash
modinfo firewire_ohci | head -5
```

Bench test without the scanner: `sudo modprobe firewire_ohci`, then `ls /sys/bus/firewire/devices/`. You should see only `fw0` (the PC itself). Then `sudo modprobe -r firewire_ohci`.

### What to send back from 1.0

Run `tools/phase1-collect.sh` (from this repository) with no scanner connected. It only reads, and produces `phase1-<host>-<date>.tar.gz`. Send that file.

---

## 1a. Passive logging of the original system (host side)

> **STOP.** This stage installs a logging tool inside Windows 2000. It never sends anything itself, but it sits in the path between the Fujifilm software and the FireWire bus. Do not start it before the tool exists, has been reviewed, and the equipment owner has agreed in writing.

Outline (the tool is not written yet):
1. Run it only where it can be undone in one step: a **VM snapshot** of the scanner PC (as in Jack Chauvel's guide) or a **cloned disk**, never the original disk.
2. Install the logger, reboot Windows 2000, then power up and scan as usual: full boot, one preview, one scan, shutdown.
3. Copy the log file out.
4. Revert the snapshot, or put the original disk back.

---

## 1b. Linux as host, read-only

> **STOP.** In this stage Linux replaces the Toshiba on the bus and transmits standard 1394 traffic (bus reset, self-ID, config ROM reads, possibly PHY config packets as bus manager). Do not start it before 1a has been reviewed and the equipment owner has agreed in writing.

Outline:
1. Everything off. Disconnect the Toshiba's FireWire cable(s) and connect the Linux PC instead, following the Phase 0 photos.
2. Power-on order. In Jack Chauvel's setup, powering the host PC is what switches the scanner on, through the FireWire link to the scanner's power unit; he then waits about 3 minutes before starting the scanner software (medium confidence). So expect **the scanner to power up as soon as the Linux PC boots**, even with `firewire_ohci` not loaded (the card's PHY and bus power are active without a driver; medium confidence). Boot the Linux PC, wait about 3 minutes, and only then load the driver in step 5. Confirm the exact order seen in the Phase 0 boot video before doing this.
3. Terminal 1: `sudo journalctl -kf`
4. Terminal 2: enable the FireWire tracepoints, so that every packet Linux sends or receives is logged:
   ```bash
   sudo trace-cmd start -e firewire
   ```
5. Load the controller by hand:
   ```bash
   sudo modprobe firewire_ohci
   ```
6. Wait 30 seconds, then:
   ```bash
   ./tools/phase1-collect.sh
   ```
   The script reads `/sys/bus/firewire/devices/*` (GUID, vendor, model, units, config ROM in hex, unit directories), `lsfirewire -v`, the trace buffer, the kernel log and `lspci -nnvk`.
7. Unload and stop:
   ```bash
   sudo modprobe -r firewire_ohci
   ```
   ```bash
   sudo trace-cmd stop && sudo trace-cmd reset
   ```
8. Power everything off, restore the original cabling, and confirm the original system boots and scans.

### What to send back from 1b

- The `.tar.gz` from `tools/phase1-collect.sh`.
- Anything seen on the scanner panel or heard from the machine (messages, fans, clicks), with the time.
- The exact power-on order you used.
