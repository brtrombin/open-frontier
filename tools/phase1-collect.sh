#!/usr/bin/env bash
# SPDX-License-Identifier: GPL-2.0-only
#
# phase1-collect.sh: gather read-only information about the local FireWire
# controllers and the nodes on the bus, and pack it into one .tar.gz.
#
# This script NEVER sends anything on the FireWire bus. It only reads files
# the kernel already exposes (sysfs, kernel log, trace buffer) and runs
# listing commands (lspci, lsfirewire). It does not load or unload modules.
#
# Usage:
#   ./tools/phase1-collect.sh [output-directory]
#   sudo ./tools/phase1-collect.sh [output-directory]   # also saves the trace buffer and full dmesg
#
# Output: phase1-<hostname>-<UTC timestamp>.tar.gz in the output directory
# (default: current directory). Send that file back; it contains no
# Fujifilm software.

set -u

out_parent="${1:-.}"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"
host="$(hostname -s 2>/dev/null || echo unknown)"
name="phase1-${host}-${stamp}"
work="$(mktemp -d)"
dir="${work}/${name}"
mkdir -p "${dir}"

log() { printf '%s\n' "$*" >&2; }

# run <outfile> <command...>: run a read-only command, keep stdout+stderr.
run() {
    local outfile="$1"; shift
    {
        printf '$ %s\n' "$*"
        "$@" 2>&1
        printf '[exit %s]\n' "$?"
    } >>"${dir}/${outfile}"
}

log "Collecting into ${dir}"

# --- System ---
run system.txt uname -a
run system.txt cat /etc/os-release
run system.txt date -u
run system.txt cat /proc/cmdline
run modules.txt lsmod
for m in firewire_core firewire_ohci firewire_net firewire_sbp2; do
    run modules.txt modinfo "$m"
done
run modules.txt cat /etc/modprobe.d/open-frontier.conf

# --- PCI controllers ---
if command -v lspci >/dev/null 2>&1; then
    run pci.txt lspci -nn
    run pci.txt lspci -nnvk
else
    echo "lspci not installed" >"${dir}/pci.txt"
fi

# --- FireWire devices in sysfs (read-only attributes) ---
fwroot=/sys/bus/firewire/devices
fwout="${dir}/firewire-sysfs.txt"
if [ -d "${fwroot}" ]; then
    {
        echo "# ls -l ${fwroot}"
        ls -l "${fwroot}"
        echo
    } >"${fwout}" 2>&1
    for dev in "${fwroot}"/*; do
        [ -e "${dev}" ] || continue
        base="$(basename "${dev}")"
        {
            echo "=== ${base} ==="
            for attr in guid vendor vendor_name model model_name hardware_version \
                        units is_local specifier_id version modalias rom_index; do
                if [ -r "${dev}/${attr}" ]; then
                    printf '%-17s %s\n' "${attr}:" "$(tr -d '\0' <"${dev}/${attr}" 2>/dev/null)"
                fi
            done
            echo
        } >>"${fwout}" 2>&1
        if [ -r "${dev}/config_rom" ]; then
            {
                echo "=== ${base}/config_rom ==="
                od -A x -t x1z -v "${dev}/config_rom"
                echo
            } >>"${dir}/firewire-config-rom.txt" 2>&1
            cp "${dev}/config_rom" "${dir}/${base}.config_rom" 2>/dev/null || true
        fi
    done
else
    echo "${fwroot} does not exist (is firewire_ohci loaded?)" >"${fwout}"
fi

# Character devices and their sysfs links.
run firewire-cdev.txt ls -l /dev/fw*
for d in /dev/fw*; do
    [ -e "$d" ] || continue
    if command -v udevadm >/dev/null 2>&1; then
        run firewire-cdev.txt udevadm info --query=all --name="$d"
    fi
done

# lsfirewire (from linux-firewire-utils), if present.
if command -v lsfirewire >/dev/null 2>&1; then
    run lsfirewire.txt lsfirewire -v
else
    echo "lsfirewire not installed (apt install linux-firewire-utils)" >"${dir}/lsfirewire.txt"
fi

# --- Kernel log ---
if command -v journalctl >/dev/null 2>&1; then
    run kernel-log.txt journalctl -k -b --no-pager
fi
run dmesg.txt dmesg

# --- Trace buffer (only as root; read-only, does not change tracing state) ---
trace_file=""
for t in /sys/kernel/tracing/trace /sys/kernel/debug/tracing/trace; do
    if [ -r "$t" ]; then trace_file="$t"; break; fi
done
if [ -n "${trace_file}" ]; then
    cp "${trace_file}" "${dir}/trace.txt" 2>/dev/null || echo "could not read ${trace_file}" >"${dir}/trace.txt"
else
    echo "trace buffer not readable (run with sudo after 'trace-cmd start -e firewire')" >"${dir}/trace.txt"
fi

# --- Pack ---
mkdir -p "${out_parent}"
tarball="$(cd "${out_parent}" && pwd)/${name}.tar.gz"
tar -C "${work}" -czf "${tarball}" "${name}"
rm -rf "${work}"
log "Done: ${tarball}"
