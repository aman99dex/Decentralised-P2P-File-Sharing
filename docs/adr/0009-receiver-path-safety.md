# ADR-0009: The receiver treats every name from the sender as hostile

- **Status:** Accepted, 2026-10-16. Added after the adversarial review, which rated this
  CRITICAL.
- **Evidence:** the croc 2023 security audit (issues 3.1–3.3, path traversal), via
  oss-security

## Context
Anyone who has the code, or who guessed it (a 1-in-16.7M chance), controls the manifest. If
the receiver writes whatever paths the sender names, a malicious sender can:
- write outside the target directory (`../../.ssh/authorized_keys`, `C:\Windows\…`);
- abuse Windows reserved names (`CON`, `NUL`) and alternate data streams (`a.txt:ads`);
- plant symlinks that redirect later writes;
- inject ANSI escape codes into the `Receive …? [y/N]` prompt, to change what the user sees.

croc's audit found exactly this class of bug.

## Decision
A single function, `safePath(dest, name)`, is the **only** way a manifest name becomes a
filesystem path. Its job is to reject anything dangerous.

**It rejects:**
- empty names;
- absolute paths and drive letters;
- `..` path segments;
- backslashes in names (the wire format uses `/` only);
- NUL and other control characters;
- `:` anywhere in the name;
- trailing dots or spaces;
- Windows reserved names (CON, PRN, AUX, NUL, COM1–9, LPT1–9), with or without an
  extension;
- names longer than 255 bytes per segment or 4096 bytes in total;
- duplicate names after case-folding, since Windows and macOS are case-insensitive.

**When writing:**
- the cleaned path must stay inside `dest` (checked with `filepath.Rel` after `Clean`);
- files are created with `O_EXCL` (never overwritten without asking);
- no symlink is ever created or followed, and every created directory is checked with
  `Lstat`.

**Display:** every sender-controlled string (names, counts, sizes) has control characters
stripped before it's printed.

**Testing:** fuzz `safePath` with the property *"output is inside `dest`, or an error"*,
plus a table of known-bad inputs, on both Windows and Linux runners in CI.

## Consequences
Some legitimate but exotic filenames are rejected. That's the right trade-off, and the error
message names the rule that rejected the file.

## What an interviewer might ask
- *Where's the trust boundary?* The PAKE proves who you're talking to. It doesn't prove
  that person is benign.
- *Why `filepath.Rel` after `Clean`, rather than checking for the string `..`?* String
  checks miss encodings and platform quirks. Asking "is the resolved path inside `dest`?"
  checks the property you actually care about.
