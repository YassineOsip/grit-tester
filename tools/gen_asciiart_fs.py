#!/usr/bin/env python3
"""Generate suites/ascii-art-fs/cases.json for grit-tester.

Official fs-audit cases come from the project's extractor (tools/gen_fs.py,
reused via sys.path); custom cases are captured from the reference binary.
Usage-message cases are excluded — the suite compares stdout only, and
stderr behavior is covered by the project's unit tests.

Run from the grit-tester repo root:

    python tools/gen_asciiart_fs.py

"""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(r"C:/Users/Yassi/OneDrive/Desktop/crazy/great")
PROJ = ROOT / "ascii-art-fs"
sys.path.insert(0, str(PROJ / "tools"))
import gen_fs as g  # noqa: E402

OUT = ROOT / "grit-tester" / "suites" / "ascii-art-fs" / "cases.json"
REF = PROJ / "ascii-art-fs-ref.exe"

CUSTOM = [
    ("single arg defaults to standard", ["hello"]),
    ("mixed case, standard", ["aBc123 xYz!", "standard"]),
    ("spaces only, shadow", ["  spaces  ", "shadow"]),
    ("digits, thinkertoy", ["1234567890", "thinkertoy"]),
    ("newline escape, standard", ["Hello\\nThere", "standard"]),
    ("trailing newline, shadow", ["Hello\\n", "shadow"]),
]


def build_reference() -> None:
    subprocess.run(["go", "build", "-o", str(REF), "."], cwd=PROJ, check=True, capture_output=True)


def capture(args: list[str]) -> str:
    return subprocess.run(
        [str(REF), *args], capture_output=True, timeout=60, check=True, cwd=PROJ
    ).stdout.decode("utf-8")


def main() -> int:
    build_reference()
    cases = []
    n = 0
    for args, want, usage in g.extract():
        if usage:
            continue  # stderr-only; unit-tested in the project
        n += 1
        cases.append(
            {
                "id": f"A{n}",
                "description": "audit: " + " ".join(f"{a}" for a in args)[:40],
                "required": True,
                "command": "go",
                "args": ["run", ".", *args],
                "workdir": "{{TARGET}}",
                "expect_stdout": want,
            }
        )
    for i, (desc, args) in enumerate(CUSTOM, start=1):
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

    suite = {"schema": 1, "suite": "ascii-art-fs", "cases": cases}
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(suite, indent=2) + "\n", encoding="utf-8")
    REF.unlink(missing_ok=True)
    print(f"wrote {OUT}: {len(cases)} cases")
    return 0


if __name__ == "__main__":
    sys.exit(main())
