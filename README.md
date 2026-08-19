# grit-tester

Fast, zero-dependency test runner for 01-edu projects. One binary, runs
declarative JSON test suites against any implementation — Linux, macOS and
Windows.

```console
$ go install github.com/yassineosip/grit-tester/cmd/tester@latest

$ tester run --suite go-reloaded --target ../my-go-reloaded
  ✓  PASS  A1 — audit case 1: (low,3) (cap) (up,2) + punctuation
  ✓  PASS  A2 — audit case 2: (bin) and (hex) conversion
  ...
== 82 passed, 0 failed ==
```

- **Suites are JSON** — add a project by writing `cases.json`, not Go code.
- **Byte-exact comparisons** — files and stdout are diffed character for
  character, exactly like an auditor does.
- **Bonus cases** — `"required": false` failures show in amber as
  `BONUS FAIL` and don't fail the run; `--strict` counts them.
- **No packages to install** — Go stdlib only; or download a prebuilt
  binary from Releases.

## Usage

```console
tester validate <cases.json>          # check a suite file
tester run --suite <name> --target <path-to-implementation>
tester run --cases <file> --target <path> [-j N] [--strict] [--no-color]
```

## Suites

| Suite | Project | Cases |
|---|---|---|
| `go-reloaded` | text editing tool | 82 |

## Adding a suite

Create `suites/<project>/cases.json`:

```json
{
  "schema": 1,
  "suite": "<project>",
  "cases": [
    {
      "id": "A1",
      "description": "what this checks",
      "required": true,
      "setup": {"sample.txt": "input text\n"},
      "command": "go",
      "args": ["run", ".", "{{CASE_DIR}}/sample.txt", "{{CASE_DIR}}/result.txt"],
      "workdir": "{{TARGET}}",
      "expect_files": {"result.txt": "expected output\n"}
    }
  ]
}
```

Placeholders: `{{TARGET}}` = the implementation path passed with `--target`;
`{{CASE_DIR}}` = the isolated per-case temp dir where `setup` files live
(use it to pass files to programs that must run in their own directory,
like `go run .`).

## What failures look like

Run against a deliberately naive implementation
(`fixtures/broken-go-reloaded`):

```console
$ tester run --suite go-reloaded --target fixtures/broken-go-reloaded
  ✗  FAIL  A1 — audit case 1: (low,3) (cap) (up,2) + punctuation
          result.txt: got "If I make you BREAKFAST IN BED (low, 3) just...",
                      want "If I make you breakfast in bed just..."
  !  BONUS FAIL  C44 — tabs collapse
          result.txt: got "tab separated (low, 2)\n", want "tab separated\n"
== 18 passed, 50 failed, 14 bonus failed ==
```

Every mismatch shows the exact difference — required failures are red,
bonus failures are amber and don't fail the run unless `--strict` is set.

## License

MIT
