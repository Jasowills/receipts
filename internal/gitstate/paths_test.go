package gitstate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePathBasic(t *testing.T) {
	got, err := NormalizePath("/repo", "src/auth.ts")
	if err != nil || got != "src/auth.ts" {
		t.Errorf("got %q err %v, want src/auth.ts", got, err)
	}
	for in, want := range map[string]string{
		"./src/auth.ts":  "src/auth.ts",
		"src//auth.ts":   "src/auth.ts",
		"src/./auth.ts":  "src/auth.ts",
		"/repo/src/a.ts": "src/a.ts",
	} {
		got, err := NormalizePath("/repo", in)
		if err != nil || got != want {
			t.Errorf("NormalizePath(%q) = %q err %v, want %q", in, got, err, want)
		}
	}
}

func TestNormalizePathRejectsTraversal(t *testing.T) {
	for _, in := range []string{"", "..", "../x", "a/../../x", "/other/x", "/repo/../x"} {
		if got, err := NormalizePath("/repo", in); err == nil {
			t.Errorf("NormalizePath(%q) = %q, want error", in, got)
		}
	}
}

func TestNormalizePathKeepsCaseAndSpaces(t *testing.T) {
	got, err := NormalizePath("/repo", "Src/My File.TS")
	if err != nil || got != "Src/My File.TS" {
		t.Errorf("got %q err %v", got, err)
	}
	uni, err := NormalizePath("/repo", "src/caf\u00e9.ts")
	if err != nil || uni != "src/caf\u00e9.ts" {
		t.Errorf("unicode got %q err %v", uni, err)
	}
}

func TestRenameDetected(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@e.c")
	run("config", "user.name", "T")
	os.WriteFile(filepath.Join(dir, "old.txt"), []byte("x\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init")
	run("mv", "old.txt", "new.txt")
	snap, err := Collect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	found := false
	for _, f := range snap.Files {
		if f.Path == "new.txt" && f.RenamedFrom == "old.txt" {
			found = true
		}
	}
	if !found {
		t.Errorf("rename not captured, files %+v", snap.Files)
	}
}

func TestUntrackedDirExpandsToFiles(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@e.c")
	run("config", "user.name", "T")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0644)
	run("add", ".")
	run("commit", "-m", "init")
	os.MkdirAll(filepath.Join(dir, "newdir"), 0755)
	os.WriteFile(filepath.Join(dir, "newdir", "one.txt"), []byte("1\n"), 0644)
	os.WriteFile(filepath.Join(dir, "newdir", "two.txt"), []byte("2\n"), 0644)
	snap, err := Collect(context.Background(), dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	paths := map[string]bool{}
	for _, f := range snap.Files {
		paths[f.Path] = true
	}
	if !paths["newdir/one.txt"] || !paths["newdir/two.txt"] {
		t.Errorf("untracked dir should expand to files, got %v", paths)
	}
	for p := range paths {
		if strings.HasSuffix(p, "/") {
			t.Errorf("bare dir entry %q should not appear with untracked all", p)
		}
	}
}

func TestChangedPathsAndHashes(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@e.c")
	run("config", "user.name", "T")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0644)
	os.WriteFile(filepath.Join(dir, "bin.dat"), []byte{0, 1, 2, 3, 0, 0}, 0644)
	run("add", ".")
	run("commit", "-m", "init")
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello world\n"), 0644)
	os.WriteFile(filepath.Join(dir, "sp ace.txt"), []byte("new\n"), 0644)

	ctx := context.Background()
	changes, err := Changed(ctx, dir)
	if err != nil {
		t.Fatalf("Changed: %v", err)
	}
	byPath := map[string]ChangedPath{}
	for _, c := range changes {
		byPath[c.Path] = c
	}
	m, ok := byPath["a.txt"]
	if !ok {
		t.Fatalf("missing a.txt in %v", byPath)
	}
	if m.PatchHash == "" {
		t.Errorf("a.txt missing patch hash")
	}
	if m.Binary {
		t.Errorf("a.txt should not be binary")
	}
	again, err := Changed(ctx, dir)
	if err != nil {
		t.Fatalf("Changed again: %v", err)
	}
	againBy := map[string]ChangedPath{}
	for _, c := range again {
		againBy[c.Path] = c
	}
	if againBy["a.txt"].PatchHash != m.PatchHash {
		t.Errorf("patch hash not deterministic")
	}
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("different content\n"), 0644)
	third, err := Changed(ctx, dir)
	if err != nil {
		t.Fatalf("Changed third: %v", err)
	}
	for _, c := range third {
		if c.Path == "a.txt" && c.PatchHash == m.PatchHash {
			t.Errorf("patch hash unchanged after content change")
		}
	}
	if _, ok := byPath["sp ace.txt"]; !ok {
		t.Errorf("missing spaced path in changed, got %v", byPath)
	}
}

func TestChangedMarksBinary(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@e.c", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@e.c")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "t@e.c")
	run("config", "user.name", "T")
	os.WriteFile(filepath.Join(dir, "img.bin"), []byte{0, 1, 2, 3}, 0644)
	run("add", ".")
	run("commit", "-m", "init")
	os.WriteFile(filepath.Join(dir, "img.bin"), []byte{0, 1, 2, 3, 4, 5, 0}, 0644)
	changes, err := Changed(context.Background(), dir)
	if err != nil {
		t.Fatalf("Changed: %v", err)
	}
	found := false
	for _, c := range changes {
		if c.Path == "img.bin" {
			found = true
			if !c.Binary {
				t.Errorf("img.bin should be marked binary")
			}
		}
	}
	if !found {
		t.Errorf("img.bin missing from changed")
	}
}

func TestParseNameStatusZRenameOrder(t *testing.T) {
	// diff -z order is status, old, new (verified against git 2.50.1).
	got := parseNameStatusZ("R100\x00old.txt\x00new.txt\x00")
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].Path != "new.txt" || got[0].RenamedFrom != "old.txt" || got[0].Status != 'R' {
		t.Errorf("got %+v, want Path new.txt from old.txt status R", got[0])
	}
	simple := parseNameStatusZ("M\x00a.txt\x00")
	if len(simple) != 1 || simple[0].Path != "a.txt" || simple[0].Status != 'M' {
		t.Errorf("simple parse got %+v", simple)
	}
}
