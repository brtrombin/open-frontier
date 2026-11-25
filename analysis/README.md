# analysis/

Notes from static analysis (Phase 2c) of the Fujifilm software installed on the project owner's own scanner PC, for interoperability.

Rules:
- **Never commit Fujifilm binaries, decompiled code or Ghidra projects.** Keep them in a private folder outside the repository (`ghidra/`, `*.gpr` and `*.rep/` are ignored as a safety net).
- Commit only our own descriptions: which IRB request types a driver uses, which addresses it touches, the order of operations, in our own words.
- Record conclusions in `docs/protocol.md` with their evidence and confidence.

Where to look first (medium confidence): the Fujifilm kernel driver(s) in `system32\drivers` that sit on top of `1394bus.sys`, and their calls that build IRBs (`REQUEST_ASYNC_READ`, `REQUEST_ASYNC_WRITE`, `REQUEST_ASYNC_LOCK`, `REQUEST_ISOCH_*`, `REQUEST_ALLOCATE_ADDRESS_RANGE`).
