package main

import (
	"os/exec"
	"strings"
	"testing"
)

// The private files subsplash-links and subsplash-notes write under data/
// must stay git-ignored (the cache says never to commit it), while
// data/morning-rephil.jsonl stays a tracked file.
func TestPrivateDataFilesAreGitIgnored(t *testing.T) {
	root := "../.."
	for _, p := range []string{defaultCache, "data/subsplash-notes.jsonl"} {
		cmd := exec.Command("git", "check-ignore", "-q", p)
		cmd.Dir = root
		if err := cmd.Run(); err != nil {
			t.Errorf("git check-ignore %s: not ignored (%v); add it to .gitignore", p, err)
		}
	}

	cmd := exec.Command("git", "check-ignore", "-q", "data/morning-rephil.jsonl")
	cmd.Dir = root
	if err := cmd.Run(); err == nil {
		t.Errorf("data/morning-rephil.jsonl is ignored, but it is a tracked source file")
	}
	cmd = exec.Command("git", "ls-files", "data/morning-rephil.jsonl")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil || strings.TrimSpace(string(out)) != "data/morning-rephil.jsonl" {
		t.Errorf("data/morning-rephil.jsonl is not tracked: %q, %v", out, err)
	}
}
