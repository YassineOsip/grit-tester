#!/usr/bin/env python3
"""Generate suites/ascii-art/cases.json for grit-tester.

Reuses the official-case extractor from the ascii-art project
(tools/gen_asciiart.py) so audit cases stay transcription-free, and
captures custom cases (banner variants, random strings, error paths)
from a freshly built reference binary.

Run from the grit-tester repo root:

    python tools/gen_asciiart.py

"""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(r"C:/Users/Yassi/OneDrive/Desktop/crazy/great")
PROJ = ROOT / "ascii-art"
sys.path.insert(0, str(PROJ / "tools"))
import gen_asciiart as g  # noqa: E402

OUT = ROOT / "grit-tester" / "suites" / "ascii-art" / "cases.json"
REF = PROJ / "ascii-art-ref.exe"

# Custom inputs exercised against the reference binary (bonus cases).
CUSTOM_INPUTS = [
    ("shadow banner", ["hello", "shadow"]),
    ("thinkertoy banner", ["hello", "thinkertoy"]),
    ("repeated words", ["hello hello hello"]),
    ("spaces only", ["  spaces  "]),
    ("mixed case and digits", ["aBc123 xYz!"]),
    ("digits only", ["1234567890"]),
    ("tab is zero-width", ["\t"]),
    ("two trailing newlines", ["Hello\\n\\n"]),
]


def build_reference() -> None:
    subprocess.run(
        ["go", "build", "-o", str(REF), "."],
        cwd=PROJ,
        check=True,
        capture_output=True,
    )


def capture(args: list[str]) -> str:
    # Run from the project dir — banner files resolve relative to the
    # process working directory, like an auditor's shell.
    return subprocess.run(
        [str(REF), *args], capture_output=True, timeout=60, check=True, cwd=PROJ
    ).stdout.decode("utf-8")


def main() -> int:
    build_reference()
    cases = []

    audit = g.extract_audit()
    for i, (arg, want) in enumerate(audit, start=1):
        cases.append(
            {
                "id": f"A{i}",
                "description": f"audit: {arg[:30]!r}",
                "required": True,
                "command": "go",
                "args": ["run", ".", arg],
                "workdir": "{{TARGET}}",
                "expect_stdout": want,
            }
        )

    subject = g.extract_subject()
    seen = {(arg, want) for arg, want in audit}
    n = 0
    for arg, want in subject:
        if (arg, want) in seen:
            continue
        seen.add((arg, want))
        n += 1
        cases.append(
            {
                "id": f"S{n}",
                "description": f"subject: {arg[:30]!r}",
                "required": True,
                "command": "go",
                "args": ["run", ".", arg],
                "workdir": "{{TARGET}}",
                "expect_stdout": want,
            }
        )

    for i, (desc, args) in enumerate(CUSTOM_INPUTS, start=1):
        cases.append(
            {
                "id": f"C{i}",
                "description": desc,
                "required": False,
                "command": "go",
                "args": ["run", ".", *args],
                "workdir": "{{TARGET}}",
                "expect_stdout": capture(args),
            }
        )

    # Exit-code behavior is covered by main_test.go unit tests instead of
    # suite cases: `go run` flattens non-zero child exits to 1 on Windows
    # but propagates them on Linux, so expect_exit through `go run` is not
    # portable across platforms.

    suite = {"schema": 1, "suite": "ascii-art", "cases": cases}
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(suite, indent=2) + "\n", encoding="utf-8")
    REF.unlink(missing_ok=True)
    print(f"wrote {OUT}: {len(cases)} cases")
    return 0


if __name__ == "__main__":
    sys.exit(main())
