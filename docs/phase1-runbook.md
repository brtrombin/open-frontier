# Phase 1 runbook: identify the FireWire nodes, read-only

Prerequisite: Phase 0 complete (backups proven by a restore test).

Phase 1 has three stages. **1.0 never touches the scanner.** 1a and 1b put our hardware on the scanner's FireWire bus, and each needs an explicit go-ahead from the equipment owner before it starts.

Applies to Debian 13 or Ubuntu 24.04 LTS. Commands are run as a normal user with `sudo`.

---

## 1.0 Prepare the Linux PCs (no scanner connected)

### 1.0.1 Install tools

```bash
sudo apt update
```

```bash
sudo apt install linux-firewire-utils pciutils usbutils trace-cmd build-essential linux-source
```

`linux-firewire-utils` provides `lsfirewire`. `linux-source` gives you `tools/firewire/` to build `nosy-dump` (sniffer PC only). Package names can differ slightly between Debian and Ubuntu.

### 1.0.2 Stop every FireWire driver from loading by itself

The goal is that nothing is ever sent on the bus unless we load the controller driver by hand. Create the file `/etc/modprobe.d/open-frontier.conf`:

```bash
sudo tee /etc/modprobe.d/open-frontier.conf >/dev/null <<'EOF'
# open-frontier: keep the FireWire bus silent until loaded by hand.
# Controller drivers (load manually with modprobe).
blacklist firewire_ohci
blacklist nosy
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

`install ... /bin/false` makes the module impossible to load even as a dependency. `firewire_ohci` and `nosy` are only blacklisted (not forbidden), so `modprobe` still loads them when we ask explicitly.

```bash
sudo update-initramfs -u
```

```bash
sudo reboot
```

### 1.0.3 Check the PC before connecting anything

```bash
lsmod | grep -Ei 'firewire|nosy|snd_(bebob|dice|oxfw|fire)' || echo "OK: no FireWire module loaded"
```

```bash
lspci -nnk | grep -A3 -Ei '1394|firewire'
```

Expected: your card with its ID (for example `[1106:3403]` VT6315, `[104c:823f]` XIO2213B, or `[104c:8000]` PCILynx on the sniffer), and no "Kernel driver in use" line.

```bash
modinfo firewire_ohci | head -5
```

```bash
modinfo nosy | head -5
```

If `modinfo nosy` fails on the sniffer PC, your distribution does not build it. Tell us the kernel version (`uname -r`), and we will build it from `linux-source`.

Bench test without the scanner: `sudo modprobe firewire_ohci`, then `ls /sys/bus/firewire/devices/`. You should see only `fw0` (the PC itself). Then `sudo modprobe -r firewire_ohci`.

### 1.0.4 Build nosy-dump (sniffer PC only)

```bash
cd /usr/src && tar xf linux-source-*.tar.* && cd linux-source-*/tools/firewire && make
```

Result: a `nosy-dump` binary in that folder. Check with `./nosy-dump --help`.

### What to send back from 1.0

Run `tools/phase1-collect.sh` (from this repository) with no scanner connected. It only reads, and produces `phase1-<host>-<date>.tar.gz`. Send that file.

---

## 1a. Passive capture of the original system (nosy)

> **STOP.** This stage adds a node to the scanner's FireWire bus. Do not start it before we have reviewed the 1.0 results together and the equipment owner has agreed in writing.

Outline (the detailed steps will be written after 1.0 is reviewed, because they depend on the topology found in Phase 0):
1. Everything off, everything on the same grounded power strip.
2. Insert the sniffer PC's PCILynx card between the Toshiba and the scanner, using two of its ports (its PHY repeats traffic between them).
3. Boot the sniffer PC first, then `sudo modprobe nosy`, then start `nosy-dump` writing to a file.
4. Power on the scanner and the Toshiba in the usual order. Capture the whole boot, one preview scan, and the shutdown.
5. Stop the capture, `sudo modprobe -r nosy`, power everything off, and restore the original cabling.

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
