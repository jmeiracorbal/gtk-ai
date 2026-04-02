package git

import (
	"fmt"
	"strings"
	"testing"
)

func TestFilterStatusStagedVsModified(t *testing.T) {
	// M at col 0 = staged modified → Staged
	// space at col 0 + M at col 1 = unstaged modified → Modified
	// A at col 0 = staged added → Staged
	// Input is large enough for the length guard to allow compression.
	var sb strings.Builder
	sb.WriteString("M  staged_file.go\n")
	sb.WriteString(" M unstaged_file.go\n")
	sb.WriteString("A  added_file.go\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, " M internal/pkg/module_%02d/handler_with_long_name.go\n", i)
	}
	raw := sb.String()

	out := filterStatus(raw)

	if out == raw {
		t.Fatal("filterStatus returned original — input not large enough to trigger compression")
	}
	if !strings.Contains(out, "Staged") {
		t.Errorf("expected Staged section, got: %s", out)
	}
	if !strings.Contains(out, "staged_file.go") {
		t.Errorf("M at col 0 should be in Staged, got: %s", out)
	}
	if !strings.Contains(out, "added_file.go") {
		t.Errorf("A  should be in Staged, got: %s", out)
	}
	if !strings.Contains(out, "Modified") {
		t.Errorf("expected Modified section, got: %s", out)
	}
	if !strings.Contains(out, "unstaged_file.go") {
		t.Errorf("M at col 1 should be in Modified, got: %s", out)
	}
}

func TestFilterStatusClean(t *testing.T) {
	out := filterStatus("")
	if out != "nothing to commit, working tree clean\n" {
		t.Errorf("unexpected output for clean status: %q", out)
	}
}

func TestFilterStatusUntrackedTruncated(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		sb.WriteString("?? file_\n")
	}
	out := filterStatus(sb.String())
	if !strings.Contains(out, "+5 more") {
		t.Errorf("expected truncation at 10 untracked, got: %s", out)
	}
}

func TestFilterStatusLengthGuard(t *testing.T) {
	// small input: reformatted result may be longer → guard returns original
	raw := "M  a.go\n M b.go\nA  c.go\n"
	out := filterStatus(raw)
	// either the guard kicked in (returns raw) or result is shorter — never longer
	if len(out) > len(raw) {
		t.Errorf("filterStatus should never return output longer than input: %d > %d", len(out), len(raw))
	}
}

func TestFilterStatusOnlyUntracked(t *testing.T) {
	// Only untracked files — no staged/modified sections should appear.
	// Use enough files with long-enough names so the reformatted
	// "Untracked (N): f1, f2, ..." line is shorter than the raw "?? ...\n" list,
	// allowing the length guard to pass through the compressed form.
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&sb, "?? internal/pkg/cache/temporary_artifact_file_%02d.json\n", i)
	}
	out := filterStatus(sb.String())
	if strings.Contains(out, "Staged") {
		t.Errorf("expected no Staged section for untracked-only input, got: %s", out)
	}
	if strings.Contains(out, "Modified") {
		t.Errorf("expected no Modified section for untracked-only input, got: %s", out)
	}
	if !strings.Contains(out, "Untracked") {
		t.Errorf("expected Untracked section, got: %s", out)
	}
}

func TestFilterStatusExactly10Untracked(t *testing.T) {
	// Exactly 10 untracked files — boundary: no truncation notice expected.
	var sb strings.Builder
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "?? untracked_%02d_with_a_longer_path.go\n", i)
	}
	// Pad input with enough modified files so the length guard doesn't fire.
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&sb, " M internal/module_%02d/handler_with_long_name.go\n", i)
	}
	out := filterStatus(sb.String())
	if strings.Contains(out, "more") {
		t.Errorf("exactly 10 untracked should not be truncated, got: %s", out)
	}
}

func TestFilterStatusOnlyStaged(t *testing.T) {
	// Only staged files (both "A " and "M " prefixes) — Modified and Untracked must be absent.
	var sb strings.Builder
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&sb, "A  new_feature_%02d_implementation_file.go\n", i)
	}
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&sb, "M  existing_module_%02d_with_long_name.go\n", i)
	}
	raw := sb.String()
	out := filterStatus(raw)
	if out == raw {
		t.Fatal("filterStatus returned original — input not large enough to trigger compression")
	}
	if !strings.Contains(out, "Staged") {
		t.Errorf("expected Staged section, got: %s", out)
	}
	if strings.Contains(out, "Modified") {
		t.Errorf("staged-only input should not produce Modified section, got: %s", out)
	}
	if strings.Contains(out, "Untracked") {
		t.Errorf("staged-only input should not produce Untracked section, got: %s", out)
	}
}

func TestFilterStatusNeverExpandsOutput(t *testing.T) {
	// The length guard must ensure filterStatus never returns output longer than input
	// for non-empty inputs. (Empty input is a special case that returns a clean-state
	// message and is intentionally larger than the empty input.)
	cases := []string{
		"M  a.go\n",
		" M b.go\n",
		"A  c.go\n",
		"?? d.go\n",
		"M  x.go\n M y.go\nA  z.go\n?? w.go\n",
	}
	for _, raw := range cases {
		out := filterStatus(raw)
		if len(out) > len(raw) {
			t.Errorf("filterStatus expanded input %q: len(out)=%d > len(raw)=%d\nout=%q",
				raw, len(out), len(raw), out)
		}
	}
}