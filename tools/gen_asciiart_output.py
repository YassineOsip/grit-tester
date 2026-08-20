#!/usr/bin/env python3
"""Generate suites/ascii-art-output/cases.json for grit-tester.

Official audit cases use expect_files with {{TARGET}} keys (the program
writes into the implementation dir); the engine substitutes the
placeholder, compares, then cleans the file up. Usage cases are skipped
(stderr; unit-tested in the project). Custom cases are captured from the
reference binary.

Run from the grit-tester repo root:

    python tools/gen_asciiart_output.py

"""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(r"C:/Users/Yassi/OneDrive/Desktop/crazy/great")
PROJ = ROOT / "ascii-art-output"
sys.path.insert(0, str(PROJ / "tools"))
import gen_output as g  # noqa: E402

OUT = ROOT / "grit-tester" / "suites" / "ascii-art-output" / "cases.json"
REF = PROJ / "ascii-art-output-ref.exe"

CUSTOM = [
    ("no flag: stdout render", ["hello", "standard"]),
    ("mixed case to file, shadow", ["--output=extra.txt", "aBc123 xYz!", "shadow"]),
    ("newline escape to file, standard", ["--output=extra2.txt", "Hello\\nThere", "standard"]),
]


def build_reference() -> None:
    subprocess.run(["go", "build", "-o", str(REF), "."], cwd=PROJ, check=True, capture_output=True)


def run_ref(args: list[str]) -> subprocess.CompletedProcess:
    return subprocess.run([str(REF), *args], capture_output=True, timeout=60, cwd=PROJ)


def main() -> int:
    build_reference()
    cases = []
    n = 0
    for args, out_file, want, usage in g.extract():
        if usage:
            continue  # stderr-only; unit-tested in the project
        n += 1
        cases.append(
            {
                "id": f"A{n}",
                "description": "audit: " + " ".join(args)[:40],
                "required": True,
                "command": "go",
                "args": ["run", ".", *args],
                "workdir": "{{TARGET}}",
                "expect_files": {f"{{{{TARGET}}}}/{out_file}": want},
                "expect_stdout": "",
            }
        )
    for i, (desc, args) in enumerate(CUSTOM, start=1):
        if args[0] == "--output=extra.txt" or args[0] == "--output=extra2.txt":
            # runs through the reference binary, then read the file back
            res = run_ref(args)
            want = (PROJ / args[0].split("=", 1)[1]).read_text(encoding="utf-8")
            (PROJ / args[0].split("=", 1)[1]).unlink(missing_ok=True)
            cases.append(
                {
                    "id": f"C{i}",
                    "description": desc,
                    "required": False,
                    "command": "go",
                    "args": ["run", ".", *args],
                    "workdir": "{{TARGET}}",
                    "expect_files": {f"{{{{TARGET}}}}/{args[0].split('=', 1)[1]}": want},
                    "expect_stdout": "",
                }
            )
        else:
            res = run_ref(args)
            cases.append(
                {
                    "id": f"C{i}",
                    "description": desc,
                    "required": False,
                    "command": "go",
                    "args": ["run", ".", *args],
                    "workdir": "{{TARGET}}",
                    "expect_stdout": res.stdout.decode("utf-8"),
                }
            )

    suite = {"schema": 1, "suite": "ascii-art-output", "cases": cases}
    OUT.parent.mkdir(parents=True, exist_ok=True)
    OUT.write_text(json.dumps(suite, indent=2) + "\n", encoding="utf-8")
    REF.unlink(missing_ok=True)
    print(f"wrote {OUT}: {len(cases)} cases")
    return 0


if __name__ == "__main__":
    sys.exit(main())
