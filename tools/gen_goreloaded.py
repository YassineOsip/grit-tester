#!/usr/bin/env python3
"""Generate suites/go-reloaded/cases.json from the go-reloaded auditor
battery (auditor_test.go, sections A/B/C).

Convention: setup files and expected files end with a trailing newline,
matching how auditors create sample.txt with `echo`. Cases marked
exact=True keep their bytes verbatim (newline-specific or empty cases).

Run from the repo root:  python tools/gen_goreloaded.py
"""

import json
import pathlib

# (id, description, required, input, expected, exact)
CASES = [
    # ---- Section A: official audit cases ----
    ("A1", "audit case 1: (low,3) (cap) (up,2) + punctuation", True,
     "If I make you BREAKFAST IN BED (low, 3) just say thank you instead of: how (cap) did you get in my house (up, 2) ?",
     "If I make you breakfast in bed just say thank you instead of: How did you get in MY HOUSE?", False),
    ("A2", "audit case 2: (bin) and (hex) conversion", True,
     "I have to pack 101 (bin) outfits. Packed 1a (hex) just to be sure",
     "I have to pack 5 outfits. Packed 26 just to be sure", False),
    ("A3", "audit case 3: punctuation only", True,
     "Don not be sad ,because sad backwards is das . And das not good",
     "Don not be sad, because sad backwards is das. And das not good", False),
    ("A4", "audit case 4: (cap,2) colon quotes a->an", True,
     "harold wilson (cap, 2) : ' I am a optimist ,but a optimist who carries a raincoat . '",
     "Harold Wilson: 'I am an optimist, but an optimist who carries a raincoat.'", False),

    # ---- Section B: subject README examples ----
    ("B1", "README main example", True,
     "it (cap) was the best of times, it was the worst of times (up) , it was the age of wisdom, it was the age of foolishness (cap, 6) , it was the epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of darkness, it was the spring of hope, IT WAS THE (low, 3) winter of despair.",
     "It was the best of times, it was the worst of TIMES, it was the age of wisdom, It Was The Age Of Foolishness, it was the epoch of belief, it was the epoch of incredulity, it was the season of Light, it was the season of darkness, it was the spring of hope, it was the winter of despair.", False),
    ("B2", "hex and bin together", True,
     "Simply add 42 (hex) and 10 (bin) and you will see the result is 68.",
     "Simply add 66 and 2 and you will see the result is 68.", False),
    ("B3", "a->an before vowel", True,
     "There is no greater agony than bearing a untold story inside you.",
     "There is no greater agony than bearing an untold story inside you.", False),
    ("B4", "punctuation group ellipsis", True,
     "Punctuation tests are ... kinda boring ,what do you think ?",
     "Punctuation tests are... kinda boring, what do you think?", False),
    ("B5", "single up with punctuation", True,
     "Ready, set, go (up) !", "Ready, set, GO!", False),
    ("B6", "single low", True,
     "I should stop SHOUTING (low)", "I should stop shouting", False),
    ("B7", "single cap", True,
     "Welcome to the Brooklyn bridge (cap)", "Welcome to the Brooklyn Bridge", False),
    ("B8", "counted up", True,
     "This is so exciting (up, 2)", "This is SO EXCITING", False),
    ("B9", "uppercase hex digits", True,
     "1E (hex) files were added", "30 files were added", False),
    ("B10", "binary two", True,
     "It has been 10 (bin) years", "It has been 2 years", False),
    ("B11", "max hex value", True,
     "ff (hex) is two hundred fifty five", "255 is two hundred fifty five", False),
    ("B12", "binary zero", True,
     "there are 0 (bin) problems", "there are 0 problems", False),
    ("B13", "punctuation with double marks", True,
     "I was sitting over there ,and then BAMM !!",
     "I was sitting over there, and then BAMM!!", False),
    ("B14", "mixed punctuation group", True,
     "What !? Really", "What!? Really", False),
    ("B15", "single word quotes", True,
     "I am exactly how they describe me: ' awesome '",
     "I am exactly how they describe me: 'awesome'", False),
    ("B16", "multi word quotes", True,
     "As Elton John said: ' I am the most well-known homosexual in the world '",
     "As Elton John said: 'I am the most well-known homosexual in the world'", False),
    ("B17", "capital A to An", True,
     "There it was. A amazing rock!", "There it was. An amazing rock!", False),
    ("B18", "a to an before h", True,
     "it was a hour long wait", "it was an hour long wait", False),
    ("B19", "a stays a before consonant", True,
     "a raincoat is useful", "a raincoat is useful", False),
    ("B20", "flag with nothing before it", True,
     "(up, 2) hello world", "hello world", False),
    ("B21", "zero count changes nothing", True,
     "(cap, 0) keep this", "keep this", False),
    ("B22", "count larger than available words", True,
     "one (up, 5)", "ONE", False),
    ("B23", "unknown flag word stays untouched", True,
     "keep (wat) here", "keep (wat) here", False),
    ("B24", "trailing newline preserved", True,
     "hello (up) world\n", "HELLO world\n", True),
    ("B25", "no trailing newline stays absent", True,
     "hello (up) world", "HELLO world", True),
    ("B26", "contraction stays one word", True,
     "don't stop 'now'", "don't stop 'now'", False),
    ("B27", "possessive plus quoted phrase", True,
     "the world's end ' is here '", "the world's end 'is here'", False),
    ("B28", "multiple quote pairs in one line", True,
     " ' a ' and ' b ' ", "'a' and 'b'", False),
    ("B29", "internal newline collapses to space", True,
     "hello\nworld (up)", "hello WORLD", False),
    ("B30", "tabs collapse to space", True,
     "tab\tseparated (low, 2)", "tab separated", False),

    # ---- Section C: custom battery (flags) ----
    ("C1", "count with no space (up,2)", False,
     "This is so exciting (up,2)", "This is SO EXCITING", False),
    ("C2", "count bigger than available words", False,
     "one (up, 5)", "ONE", False),
    ("C3", "marker before any word", False,
     "(up, 2) hello world", "hello world", False),
    ("C4", "zero count", False,
     "(cap, 0) keep this", "keep this", False),
    ("C5", "negative count", False,
     "keep (up, -2) this", "keep this", False),
    ("C6", "chained markers feed each other", True,
     "hello (up) world (low, 2) end", "hello world end", False),
    ("C7", "markers back to back", False,
     "a (cap) (low) b", "a b", False),
    ("C8", "marker at end of text", True,
     "hello (up)", "HELLO", False),
    ("C9", "only a marker, nothing before", False,
     "(hex)", "", True),
    ("C10", "uppercase hex digits", True,
     "1E (hex) files", "30 files", False),
    ("C11", "lowercase hex digits", True,
     "1a (hex) files", "26 files", False),
    ("C12", "hex max value", True,
     "ff (hex)", "255", False),
    ("C13", "binary zero", True,
     "0 (bin) x", "0 x", False),
    ("C14", "binary fifteen", True,
     "1111 (bin) x", "15 x", False),
    ("C15", "counted flag across punctuation", True,
     "one, two (up, 2)", "ONE, TWO", False),
    ("C16", "flag after a contraction", False,
     "don't (up)", "DON'T", False),
    ("C17", "unicode word uppercased", False,
     "héllo (up)", "HÉLLO", False),
    ("C18", "marker result feeds a-an", True,
     "a (cap) amazing rock", "An amazing rock", False),
    ("C19", "unknown marker passes through", False,
     "keep (wat) here", "keep (wat) here", False),
    ("C20", "spaced marker syntax (up , 2)", False,
     "go (up , 2) !", "GO!", False),

    # ---- Section C: custom battery (punctuation) ----
    ("C21", "double marks separated by space", True,
     "BAMM ! !", "BAMM!!", False),
    ("C22", "mixed group !?", True,
     "What !? Really", "What!? Really", False),
    ("C23", "punctuation at start of text", False,
     ". hi", ". hi", False),
    ("C24", "punctuation at end of text", False,
     "hi .", "hi.", False),
    ("C25", "punctuation-only input", False,
     "! ?", "!?", False),
    ("C26", "comma glued with word after", True,
     "hello,world", "hello, world", False),
    ("C27", "comma spaced before word", True,
     "there ,and", "there, and", False),
    ("C28", "ellipsis with spaces", True,
     "wait ... what", "wait... what", False),
    ("C29", "colon before word", True,
     "instead of: how (cap)", "instead of: How", False),
    ("C30", "semicolon handling", True,
     "say ; now", "say; now", False),

    # ---- Section C: custom battery (quotes) ----
    ("C31", "contraction is not a quote", False,
     "don't stop 'now'", "don't stop 'now'", False),
    ("C32", "possessive apostrophe", False,
     "the world's end", "the world's end", False),
    ("C33", "several quote pairs one line", True,
     " ' a ' and ' b ' ", "'a' and 'b'", False),
    ("C34", "punctuation inside quotes", True,
     " ' hello ! ' ", "'hello!'", False),
    ("C35", "multi word quotes", True,
     " ' I am here ' ", "'I am here'", False),
    ("C36", "unmatched opening quote", False,
     "say ' hello", "say 'hello", False),
    ("C37", "quote directly after colon", True,
     "said: ' hi '", "said: 'hi'", False),

    # ---- Section C: custom battery (a->an) ----
    ("C38", "capital A before vowel", True,
     "A amazing rock", "An amazing rock", False),
    ("C39", "a before h", True,
     "a hour", "an hour", False),
    ("C40", "a before consonant stays", True,
     "a raincoat", "a raincoat", False),
    ("C41", "capital A before h", True,
     "A historic day", "An historic day", False),
    ("C42", "a at end of text untouched", False,
     "this is a", "this is a", False),

    # ---- Section C: custom battery (whitespace) ----
    ("C43", "multiple spaces collapse", False,
     "hello    world (up)", "hello WORLD", False),
    ("C44", "tabs collapse", False,
     "tab\tseparated (low, 2)", "tab separated", False),
    ("C45", "internal newline collapses", False,
     "hello\nworld (up)", "hello WORLD", False),
    ("C46", "trailing newline preserved", False,
     "hello (up)\n", "HELLO\n", True),
    ("C47", "empty input", False, "", "", True),
    ("C48", "spaces only", False, "   ", "", True),
]


def main():
    cases = []
    for cid, desc, required, inp, out, exact in CASES:
        if not exact:
            inp += "\n"
            out += "\n"
        cases.append({
            "id": cid,
            "description": desc,
            "required": required,
            "setup": {"sample.txt": inp},
            "command": "go",
            "args": ["run", ".", "{{CASE_DIR}}/sample.txt", "{{CASE_DIR}}/result.txt"],
            "workdir": "{{TARGET}}",
            "timeout_sec": 60,
            "expect_files": {"result.txt": out},
        })

    suite = {"schema": 1, "suite": "go-reloaded", "cases": cases}
    out_path = pathlib.Path(__file__).resolve().parent.parent / "suites" / "go-reloaded" / "cases.json"
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(suite, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"wrote {len(cases)} cases to {out_path}")


if __name__ == "__main__":
    main()
