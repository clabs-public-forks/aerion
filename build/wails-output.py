#!/usr/bin/env python3
"""Run Wails while redacting OAuth credentials from its live output."""

import os
import re
import subprocess
import sys


credential_names = (
    "GOOGLE_CLIENT_ID",
    "GOOGLE_CLIENT_SECRET",
    "GOOGLE_TESTING_CLIENT_ID",
    "GOOGLE_TESTING_CLIENT_SECRET",
    "MICROSOFT_CLIENT_ID",
)
secrets = [
    os.environ[name].encode()
    for name in credential_names
    if os.environ.get(name)
]
ldflags_row = re.compile(rb"^\s*LDFlags\s*\|")

if len(sys.argv) < 2:
    raise SystemExit("usage: wails-output.py COMMAND [ARG ...]")

process = subprocess.Popen(
    sys.argv[1:], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, bufsize=0
)
assert process.stdout is not None

for line in iter(process.stdout.readline, b""):
    if ldflags_row.match(line):
        sys.stdout.buffer.write(b"LDFlags | [OAuth linker flags redacted]\n")
    else:
        for secret in secrets:
            line = line.replace(secret, b"[REDACTED]")
        sys.stdout.buffer.write(line)
    sys.stdout.buffer.flush()

raise SystemExit(process.wait())
