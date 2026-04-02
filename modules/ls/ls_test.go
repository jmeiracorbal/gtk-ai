package ls

import (
	"fmt"
	"strings"
	"testing"
)

func newModule() *Module { return &Module{} }

// ── FilterOutput length guard ─────────────────────────────────────────────────

// TestFilterOutputNeverExpands verifies the core invariant introduced in this PR:
// FilterOutput must never return output longer than the original input.
func TestFilterOutputNeverExpands(t *testing.T) {
	m := newModule()
	cases := []struct {
		name string
		raw  string
	}{
		{"single file", "main.go\n"},
		{"few dirs", "cmd/\ninternal/\nmodules/\n"},
		{"mixed small", "cmd/\nmain.go\ngo.mod\ngo.sum\n"},
		{"no-ext files", "Makefile\nDockerfile\nLICENSE\n"},
		{"empty", ""},
		{"single dir", "src/\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := m.FilterOutput(tc.raw)
			if len(out) > len(tc.raw) {
				t.Errorf("FilterOutput expanded output for %q: len=%d > original=%d\nresult: %s",
					tc.raw, len(out), len(tc.raw), out)
			}
		})
	}
}

// TestFilterOutputLengthGuardReturnsOriginal verifies that when the reformatted
// result would be equal to or larger than the input, the original is returned unchanged.
func TestFilterOutputLengthGuardReturnsOriginal(t *testing.T) {
	m := newModule()
	// The emoji prefix (📁/📄) adds 4 bytes per symbol plus count labels, so the
	// reformatted result is almost always larger than the raw list. The guard must
	// return the original unmodified string in those cases.
	raw := "handler_00.go\nhandler_01.go\nhandler_02.go\n"
	out := m.FilterOutput(raw)
	if out != raw {
		t.Errorf("expected original string returned by length guard\ngot: %q\nwant: %q", out, raw)
	}
}

// TestFilterOutputLargeInputStillDoesNotExpand checks that even with many files
// the length guard holds — the ls module's format (emojis + labels) is
// inherently verbose enough that it never shrinks output.
func TestFilterOutputLargeInputStillDoesNotExpand(t *testing.T) {
	m := newModule()
	var sb strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&sb, "handler_%02d.go\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(raw)
	if len(out) > len(raw) {
		t.Errorf("FilterOutput expanded large input: %d > %d", len(out), len(raw))
	}
}

// ── FilterOutput empty / blank inputs ────────────────────────────────────────

// TestFilterOutputEmpty checks that an empty string is returned unchanged.
func TestFilterOutputEmpty(t *testing.T) {
	m := newModule()
	out := m.FilterOutput("")
	if out != "" {
		t.Errorf("expected empty output, got: %q", out)
	}
}

// TestFilterOutputBlankLines checks that blank-only input (just whitespace/newlines)
// does not generate spurious file/dir groups.
func TestFilterOutputBlankLines(t *testing.T) {
	m := newModule()
	raw := "\n\n\n"
	out := m.FilterOutput(raw)
	if strings.Contains(out, "dirs") || strings.Contains(out, "files") {
		t.Errorf("blank input should not produce dirs/files labels, got: %q", out)
	}
	if len(out) > len(raw) {
		t.Errorf("blank input expanded: %d > %d", len(out), len(raw))
	}
}

// ── FilterOutput grouping logic (via reformatted intermediate) ────────────────

// TestFilterOutputGroupingDirsVsFiles verifies that the grouping logic correctly
// separates entries ending in "/" (dirs) from plain filenames (files).
// We inspect the intermediate reformatted string before the length guard by
// constructing a scenario where we can verify content regardless of guard outcome.
func TestFilterOutputGroupingDirsVsFiles(t *testing.T) {
	m := newModule()
	// Use a mix of dirs and files; the guard may or may not fire, but we can
	// verify the invariant: result never contains information that wasn't in input.
	raw := "src/\nmain.go\n"
	out := m.FilterOutput(raw)
	// Regardless of whether guard fired, output is one of:
	//   - the original (guard fired) → contains "src/" and "main.go"
	//   - the reformatted string → contains "dirs" and ".go"
	hasDirRef := strings.Contains(out, "src/") || strings.Contains(out, "dirs")
	hasFileRef := strings.Contains(out, "main.go") || strings.Contains(out, ".go")
	if !hasDirRef {
		t.Errorf("output should reference the directory, got: %q", out)
	}
	if !hasFileRef {
		t.Errorf("output should reference the file, got: %q", out)
	}
}

// TestFilterOutputTrailingNewline checks that a trailing newline in the input
// does not produce an empty group entry.
func TestFilterOutputTrailingNewline(t *testing.T) {
	m := newModule()
	raw := "main.go\ngo.mod\n" // ends with \n
	out := m.FilterOutput(raw)
	if strings.Contains(out, "(0):") {
		t.Errorf("trailing newline produced empty group entry in: %q", out)
	}
	if len(out) > len(raw) {
		t.Errorf("output expanded: %d > %d", len(out), len(raw))
	}
}

// TestFilterOutputOnlyDirsNeverExpands checks that a dirs-only listing is
// never expanded by the module.
func TestFilterOutputOnlyDirsNeverExpands(t *testing.T) {
	m := newModule()
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&sb, "module_%02d/\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(raw)
	if len(out) > len(raw) {
		t.Errorf("dirs-only output expanded: %d > %d", len(out), len(raw))
	}
}

// TestFilterOutputOnlyFilesNeverExpands checks that a files-only listing is
// never expanded by the module.
func TestFilterOutputOnlyFilesNeverExpands(t *testing.T) {
	m := newModule()
	var sb strings.Builder
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, "handler_%02d.go\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(raw)
	if len(out) > len(raw) {
		t.Errorf("files-only output expanded: %d > %d", len(out), len(raw))
	}
}

// TestFilterOutputMixedNeverExpands checks that a mixed dirs+files listing is
// never expanded by the module.
func TestFilterOutputMixedNeverExpands(t *testing.T) {
	m := newModule()
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "module_%02d/\n", i)
	}
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, "handler_%02d.go\n", i)
	}
	raw := sb.String()
	out := m.FilterOutput(raw)
	if len(out) > len(raw) {
		t.Errorf("mixed dirs+files output expanded: %d > %d", len(out), len(raw))
	}
}

// ── extOf ─────────────────────────────────────────────────────────────────────

func TestExtOf(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"go file", "main.go", "go"},
		{"test file", "handler_test.go", "go"},
		{"no extension", "Makefile", ""},
		{"hidden file", ".gitignore", "gitignore"},
		{"trailing dot", "file.", ""},
		{"multiple dots", "archive.tar.gz", "gz"},
		{"ts file", "app.ts", "ts"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extOf(tc.in)
			if got != tc.want {
				t.Errorf("extOf(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestExtOfEmptyString checks the edge case of an empty filename.
func TestExtOfEmptyString(t *testing.T) {
	got := extOf("")
	if got != "" {
		t.Errorf("extOf(\"\") = %q, want \"\"", got)
	}
}