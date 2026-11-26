# src/

The `openfrontier` program, in Go. It builds to one Windows `.exe` with no runtime and no installation, and also runs on Linux and macOS. Nothing here talks to the scanner: the program reads finished spool files and lists the FireWire hardware Windows has enumerated.

```text
cmd/openfrontier/    command line: inspect, convert, compare, firewire
internal/spool/      .RAW header, size check and pixel layout; .INF (DIPP) records;
                     pairing frames with their .INF; live-spool detection and
                     shared reads (FILE_SHARE_DELETE on Windows)
internal/imaging/    orientation, JPEG and TIFF output, EXIF orientation of
                     reference images, comparison with the Fuji output
internal/firewire/   read-only Plug and Play inventory of IEEE 1394 devices (Windows)
internal/synth/      synthetic spool files for the tests
```

Format facts come from `docs/protocol.md`, section 9, in our own words; the code is our own implementation. Points still unconfirmed on real files (header word positions, interleaved or planar pixels, channel order, the meaning of the `.INF` rotation value) are reported by `inspect` and `compare` instead of being assumed silently.

## Build and test

Go 1.24 or later. Only one dependency: `golang.org/x/image`, to read TIFF and BMP reference images.

```bash
go test ./...
```

```bash
GOOS=windows GOARCH=amd64 go build -trimpath -o ../dist/openfrontier.exe ./cmd/openfrontier
```

`GOARCH=386` gives a 32-bit Windows build. The tests use synthetic files built from our own reading of the formats, so they check the code against that reading, not against the scanner.

## Planned layers

1. `transport`: the only layer that will touch hardware. One backend per link type, FireWire for the SP-3000 first.
2. `scanner`: protocol, init sequence, commands, abort, one profile per scanner model.
3. `spool` and `imaging` (started): files the scanner PC writes, and image output.
4. `web`: queue, preview and export UI.
