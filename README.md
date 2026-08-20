# grit-tester

Fast, zero-dependency test runner for 01-edu projects. One binary, runs
declarative JSON test suites against any implementation — Linux, macOS and
Windows.

```console
$ go install github.com/yassineosip/grit-tester/cmd/tester@latest

$ tester run --suite go-reloaded --target ../my-go-reloaded
ID   STATUS      DESCRIPTION
A1   PASS        audit case 1: (low,3) (cap) (up,2) + punctuation
A2   PASS        audit case 2: (bin) and (hex) conversion
...
== 82 passed, 0 failed ==
```

- **Suites are embedded** — the built-in suites ship inside the binary, so it
  runs from any directory. Add a project by writing `cases.json`, not Go code.
- **Live progress** — while cases run, a spinner shows the current case and
  counts (terminals only; piped output stays clean).
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

> **Install trouble?** If `go install` fails with
> `lookup proxy.golang.org: no such host` (some networks block Google's
> module proxy), point Go at a mirror:
>
> ```console
> go env -w GOPROXY=https://goproxy.io,direct GOSUMDB=off
> ```
>
> Then re-run `go install github.com/yassineosip/grit-tester/cmd/tester@latest`.
> The binary embeds the built-in suites, so it works from any directory —
> no repo clone needed. If a `suites/` folder exists next to where you run
> it, local suite files take precedence (useful when writing new suites).

## Suites

| Suite | Project | Cases |
|---|---|---|
| `go-reloaded` | text editing tool | 82 |
| `ascii-art` | ASCII-art banners | 33 |
| `ascii-art-fs` | banners via the fs API | 15 |
| `ascii-art-output` | `--output` file flag | 11 |
| `ascii-art-justify` | `--align` to terminal width | 28 |
| `ascii-art-color` | `--color` ANSI spans | 27 |

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
      "env": {"COLUMNS": "100"},
      "workdir": "{{TARGET}}",
      "expect_files": {"result.txt": "expected output\n"}
    }
  ]
}
```

Placeholders: `{{TARGET}}` = the implementation path passed with `--target`;
`{{CASE_DIR}}` = the isolated per-case temp dir where `setup` files live
(use it to pass files to programs that must run in their own directory,
like `go run .`). `env` adds environment variables for the case's process
(e.g. `COLUMNS` pins the terminal width alignment programs measure).

`expect_files` keys may contain `{{TARGET}}` / `{{CASE_DIR}}` to check
files a case writes outside the case dir (e.g. an `--output` flag writing
into the implementation). Such files are removed after the check, so
suites never leave files behind in the target.

New suites are compiled into the next release automatically (the embed
glob picks up `*/cases.json`); to iterate without rebuilding, keep a
`suites/` folder next to the binary — local files win.

## What failures look like

Run against a deliberately naive implementation
(`fixtures/broken-go-reloaded`):

```console
$ tester run --suite go-reloaded --target fixtures/broken-go-reloaded
ID   STATUS       DESCRIPTION
A1   FAIL         audit case 1: (low,3) (cap) (up,2) + punctuation
     │ input (sample.txt)
     │   If I make you BREAKFAST IN BED (low, 3) just say thank you ...
     │   \n
     │ got (result.txt)
     │   If I make you BREAKFAST IN BED (low, 3) just say thank you ...
     │   \n
     │ want
     │   If I make you breakfast in bed just say thank you instead of: How ...
     │   \n
C44  BONUS FAIL   tabs collapse
     │ got (result.txt)
     │   tab separated (low, 2)\n
     │ want
     │   tab separated\n
== 18 passed, 50 failed, 14 bonus failed ==
```

Every failure shows the full story — the input that was fed in, what the
program produced, and what was expected — each in its own labeled block.
Required failures are red, bonus failures are amber and don't fail the run
unless `--strict` is set.

## License

MIT
