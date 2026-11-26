#!/usr/bin/env python3
"""Deterministic PTY recorder that produces static/casts/demo.cast for kvx.

Launches dist/kvx in a real pseudo-terminal of fixed dimensions (100x26),
drives a 5-chapter interactive tour with reproducible virtual timestamps,
and emits an asciicast v2 recording including "m" chapter marker events.
"""

from __future__ import annotations

import fcntl
import json
import os
import pty
import re
import select
import signal
import struct
import subprocess
import sys
import tempfile
import termios
import time
from pathlib import Path

COLS = 100
ROWS = 26

# Terminal capability/color queries Bubble Tea v2 / Ultraviolet may emit on a real PTY.
QUERY_RESPONSES: list[tuple[bytes, bytes]] = [
    (b"\x1b]11;?\x07", b"\x1b]11;rgb:0505/1515/2a2a\x07"),
    (b"\x1b]11;?\x1b\\", b"\x1b]11;rgb:0505/1515/2a2a\x1b\\"),
    (b"\x1b]10;?\x07", b"\x1b]10;rgb:efef/f7f7/ffff\x07"),
    (b"\x1b]10;?\x1b\\", b"\x1b]10;rgb:efef/f7f7/ffff\x1b\\"),
    (b"\x1b[6n", b"\x1b[1;1R"),
    (b"\x1b[?u", b"\x1b[?0u"),
    (b"\x1b[c", b"\x1b[?62;22c"),
    (b"\x1b[>c", b"\x1b[>0;10;1c"),
]

# Strip terminal query sequences from the recorded cast output so viewers never see them.
QUERY_STRIP_RE = re.compile(
    rb"\x1b\]1[01];\?(?:\x07|\x1b\\)"
    rb"|\x1b\[6n"
    rb"|\x1b\[\?u"
    rb"|\x1b\[>?c"
    rb"|\x1b\[\?[0-9;]+\$p"
)

ANSI_ESCAPE_RE = re.compile(r"\x1b(?:\][^\x07\x1b]*(?:\x07|\x1b\\)|\[[0-?]*[ -/]*[@-~]|[@-_])")


def set_winsize(fd: int, rows: int, cols: int) -> None:
    winsize = struct.pack("HHHH", rows, cols, 0, 0)
    fcntl.ioctl(fd, termios.TIOCSWINSZ, winsize)


def drain_pty(master_fd: int, quiet_sec: float = 0.09, max_wait_sec: float = 1.2) -> bytes:
    """Read available PTY output until quiet for quiet_sec, answering terminal queries."""
    collected = bytearray()
    pending_scan = bytearray()
    start = time.monotonic()
    last_data = start

    while True:
        now = time.monotonic()
        remaining = (start + max_wait_sec) - now
        if remaining <= 0:
            break

        timeout = min(quiet_sec, remaining)
        rlist, _, _ = select.select([master_fd], [], [], timeout)
        if not rlist:
            if collected and (time.monotonic() - last_data) >= quiet_sec:
                break
            continue

        try:
            chunk = os.read(master_fd, 65536)
        except OSError:
            break
        if not chunk:
            break

        last_data = time.monotonic()
        pending_scan.extend(chunk)
        for query, reply in QUERY_RESPONSES:
            if query in pending_scan:
                try:
                    os.write(master_fd, reply)
                except OSError:
                    pass
                pending_scan = pending_scan.replace(query, b"")

        collected.extend(chunk)

    return QUERY_STRIP_RE.sub(b"", bytes(collected))


def build_tour() -> list[tuple[str, float, str | bytes]]:
    """Return deterministic tour steps: (kind, delay_seconds, payload)."""
    steps: list[tuple[str, float, str | bytes]] = []

    def marker(delay: float, label: str) -> None:
        steps.append(("m", delay, label))

    def key(delay: float, payload: bytes) -> None:
        steps.append(("k", delay, payload))

    def type_text(text: str, per_char: float = 0.075, initial_delay: float = 0.25) -> None:
        for idx, ch in enumerate(text):
            key(initial_delay if idx == 0 else per_char, ch.encode("utf-8"))

    # Chapter 1: Schema Card-List & Sectioned Detail View, then toggle raw table (v)
    marker(0.0, "1. Schema Views (v)")
    key(0.75, b"j")       # select auth-issuer card
    key(0.55, b"\r")      # Enter -> Sectioned Detail view (multi-line paragraph + tag badges)
    key(1.30, b"v")       # v toggles Detail view directly into Raw Table (restoring y/Y copy hints)
    key(1.05, b"v")       # v toggles back to Sectioned Detail view
    key(0.65, b"h")       # ascend back to Card-List view
    key(0.60, b"v")       # v toggles Card-List into Raw Table view

    # Chapter 2: Tree Navigation (j / k / l / h)
    marker(0.35, "2. Tree Navigation")
    key(0.30, b"gg")      # jump to top row [0] deterministically
    key(0.40, b"j")       # select auth-issuer ([1])
    key(0.50, b"l")       # drill into auth-issuer object (_[1])
    key(0.45, b"j")       # capabilities
    key(0.40, b"j")       # config (serialized JSON string)
    key(0.45, b"k")       # back to capabilities
    key(0.45, b"l")       # drill into capabilities array
    key(0.55, b"j")       # [1] jwt
    key(0.55, b"h")       # back to auth-issuer
    key(0.45, b"h")       # back to root list (_)

    # Chapter 3: Deep Search (/)
    marker(0.35, "3. Search (/)")
    key(0.40, b"/")
    type_text("eu-west-1", per_char=0.06, initial_delay=0.20)
    key(0.40, b"\r")      # execute deep search across keys & values
    key(0.70, b"\x1b[B")  # Down arrow moves across search matches while Search input is open
    key(0.70, b"\x1b")    # Esc -> restore prior view

    # Chapter 4: CEL Expression Mode + Tab Completion + Ctrl+Space Function Palette
    marker(0.35, "4. CEL + Palette")
    key(0.40, b":")       # open CEL Expression mode
    key(0.25, b"\x15")    # Ctrl+U clear prefill
    type_text("_[1].re", per_char=0.07, initial_delay=0.20)
    key(0.50, b"\t")      # Tab completes _[1].region
    key(0.40, b"\r")      # evaluate _[1].region
    key(0.65, b"\x15")    # Ctrl+U clear input
    key(0.20, b"_")       # root '_'
    key(0.40, b"\x00")    # Ctrl+Space -> open categorized Function Palette
    type_text("filter", per_char=0.07, initial_delay=0.35)
    key(0.55, b"\r")      # Enter inserts _.filter(
    type_text('s, s.status == "running")', per_char=0.048, initial_delay=0.20)
    key(0.45, b"\r")      # evaluate filtered service list (4 running services)
    key(0.95, b"\x15")    # Ctrl+U clear expression so we restore full root '_' cleanly
    key(0.20, b"_")
    key(0.35, b"\r")      # restore root '_'
    key(0.40, b"\x1b")    # exit expression mode

    # Chapter 5: Decode Bearer JWT & Nested JSON Strings (Enter)
    marker(0.35, "5. Decode JWT")
    key(0.30, b"gg")      # top of root list
    key(0.40, b"j")       # select auth-issuer ([1])
    key(0.45, b"l")       # drill into auth-issuer (row 0 is authToken Bearer JWT)
    key(0.55, b"\r")      # open/focus authToken string leaf
    key(0.55, b"\r")      # decode Bearer JWT into header / payload / signature
    key(0.65, b"j")       # move to payload
    key(0.55, b"l")       # drill into decoded JWT claims (svc:auth-issuer)

    return steps


def verify_cast_output(events: list[list[object]]) -> None:
    """Fail non-zero if recorded frames contain UI errors or miss expected tour milestones."""
    combined = "".join(str(ev[2]) for ev in events if len(ev) >= 3 and ev[1] == "o")
    plain = ANSI_ESCAPE_RE.sub("", combined)

    for bad in ("Error:", "Path error:", "(no matching functions)", "no matching functions"):
        if bad in plain:
            raise RuntimeError(f"recorded cast contains error text {bad!r}")

    for required in ("Auth Token Issuer", "Functions", "decoded", "svc:auth-issuer"):
        if required not in plain:
            raise RuntimeError(f"recorded cast is missing expected tour milestone {required!r}")


def main() -> int:
    repo_root = Path(__file__).resolve().parent.parent
    kvx_bin = repo_root / "dist" / "kvx"
    if not kvx_bin.is_file():
        print(f"error: {kvx_bin} not found; run 'task build' first", file=sys.stderr)
        return 1

    data_file = repo_root / "examples" / "data" / "demo_services.json"
    schema_file = repo_root / "examples" / "data" / "demo_services_schema.json"
    out_cast = repo_root / "static" / "casts" / "demo.cast"
    out_cast.parent.mkdir(parents=True, exist_ok=True)

    events: list[list[object]] = []
    virtual_time = 0.05

    with tempfile.TemporaryDirectory(prefix="kvx-cast-") as tmpdir:
        empty_cfg = Path(tmpdir) / "config.yaml"
        empty_cfg.write_text("{}\n", encoding="utf-8")

        master_fd, slave_fd = pty.openpty()
        master_open = True
        slave_open = True
        proc: subprocess.Popen[bytes] | None = None

        try:
            set_winsize(master_fd, ROWS, COLS)
            set_winsize(slave_fd, ROWS, COLS)

            env = os.environ.copy()
            env["TERM"] = "xterm-256color"
            env["COLORTERM"] = "truecolor"
            env["CLICOLOR_FORCE"] = "1"
            env["XDG_CONFIG_HOME"] = tmpdir
            env.pop("NO_COLOR", None)

            cmd = [
                str(kvx_bin),
                str(data_file),
                "--schema",
                str(schema_file),
                "--config-file",
                str(empty_cfg),
                "--theme",
                "midnight",
                "--keymap",
                "vim",
                "--width",
                str(COLS),
                "--height",
                str(ROWS),
                "-i",
            ]

            proc = subprocess.Popen(
                cmd,
                stdin=slave_fd,
                stdout=slave_fd,
                stderr=slave_fd,
                cwd=str(repo_root),
                env=env,
                close_fds=True,
                start_new_session=True,
            )
            os.close(slave_fd)
            slave_open = False

            initial = drain_pty(master_fd, quiet_sec=0.18, max_wait_sec=2.5)
            if initial:
                events.append([round(virtual_time, 3), "o", initial.decode("utf-8", errors="replace")])

            for kind, delay, payload in build_tour():
                virtual_time += delay
                if kind == "m":
                    events.append([round(virtual_time, 3), "m", str(payload)])
                    continue

                assert isinstance(payload, bytes)
                os.write(master_fd, payload)
                out = drain_pty(master_fd, quiet_sec=0.09, max_wait_sec=1.2)
                if out:
                    virtual_time += 0.02
                    events.append([round(virtual_time, 3), "o", out.decode("utf-8", errors="replace")])

            # Hold the final decoded JWT claims frame cleanly before ending the cast
            virtual_time += 2.2
            events.append([round(virtual_time, 3), "o", ""])
        finally:
            if slave_open:
                os.close(slave_fd)
            if proc is not None:
                try:
                    os.killpg(proc.pid, signal.SIGTERM)
                except OSError:
                    pass
                try:
                    proc.wait(timeout=2.0)
                except subprocess.TimeoutExpired:
                    try:
                        os.killpg(proc.pid, signal.SIGKILL)
                    except OSError:
                        proc.kill()
                    proc.wait(timeout=2.0)
            if master_open:
                os.close(master_fd)

    verify_cast_output(events)

    header = {
        "version": 2,
        "width": COLS,
        "height": ROWS,
        "timestamp": 1767225600,
        "duration": round(virtual_time, 3),
        "title": "kvx interactive TUI tour",
        "env": {"SHELL": "/bin/bash", "TERM": "xterm-256color"},
    }

    with out_cast.open("w", encoding="utf-8", newline="\n") as fh:
        fh.write(json.dumps(header) + "\n")
        for ev in events:
            fh.write(json.dumps(ev, ensure_ascii=False) + "\n")

    print(f"Wrote {out_cast} ({len(events)} events, {virtual_time:.1f}s)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
