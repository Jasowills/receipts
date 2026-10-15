package demo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Fixed demo values keep output deterministic for golden tests.
var (
	demoTime = time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC)
	demoEnv  = []string{"GIT_AUTHOR_NAME=Receipts Demo", "GIT_AUTHOR_EMAIL=demo@example.com", "GIT_COMMITTER_NAME=Receipts Demo", "GIT_COMMITTER_EMAIL=demo@example.com"}
)

// fakeRepo creates a temp Git repository with base files and returns the
// directory plus a cleanup function honoring keep.
func fakeRepo(keep bool) (string, func(), error) {
	dir, err := os.MkdirTemp("", "receipts-demo-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		if !keep {
			os.RemoveAll(dir)
		}
	}
	run := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), demoEnv...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git %v: %v\n%s", args, err, out)
		}
		return nil
	}
	if err := run("init"); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := run("config", "user.email", "demo@example.com"); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := run("config", "user.name", "Receipts Demo"); err != nil {
		cleanup()
		return "", nil, err
	}
	base := map[string]string{
		"src/auth.ts":              "export function auth() { return true; }\n",
		"src/auth.test.ts":         "expect(auth()).toBe(true)\nexpect(auth()).not.toBe(false)\n",
		"src/auth.go":              "package auth\nfunc Auth() bool { return true }\n",
		".github/workflows/ci.yml": "name: ci\non: [push]\n",
		"package.json":             "{\"dependencies\":{\"express\":\"4.17.1\"}}\n",
	}
	for p, content := range base {
		full := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			cleanup()
			return "", nil, err
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			cleanup()
			return "", nil, err
		}
	}
	if err := run("add", "."); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := run("commit", "-m", "base"); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}

func writeFile(dir, p, content string) error {
	full := filepath.Join(dir, p)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0644)
}

// mutate applies the fake agent file changes for a scenario.
func mutate(scenario, dir string) error {
	switch scenario {
	case "default":
		if err := writeFile(dir, "src/auth.ts", "export function auth() { return false; }\n"); err != nil {
			return err
		}
		if err := writeFile(dir, "src/auth.test.ts", "expect(auth()).toBe(true)\nit.skip('old',()=>{})\n"); err != nil {
			return err
		}
		return writeFile(dir, ".github/workflows/ci.yml", "name: ci\non: [push, pull_request]\n")
	case "duplicate-claims":
		return writeFile(dir, "src/auth.ts", "export function auth() { return false; }\n")
	case "test-weakening":
		return writeFile(dir, "src/auth.test.ts", "expect(auth()).toBe(true)\nit.skip('old',()=>{})\n")
	case "scope-drift":
		if err := writeFile(dir, "src/auth.go", "package auth\nfunc Auth() bool { return false }\n"); err != nil {
			return err
		}
		return writeFile(dir, ".github/workflows/ci.yml", "name: ci\non: [push, pull_request]\n")
	default:
		return fmt.Errorf("demo: unknown scenario %q", scenario)
	}
}

// messages returns the fake agent final messages per scenario.
func messages(scenario string) []struct {
	ID, Text string
} {
	switch scenario {
	case "duplicate-claims":
		return []struct{ ID, Text string }{{"m-1", "All tests pass"}, {"m-2", "All tests pass!"}}
	case "test-weakening":
		return []struct{ ID, Text string }{{"m-1", "All tests pass"}}
	case "scope-drift":
		return []struct{ ID, Text string }{{"m-1", "I only changed src/auth.go"}}
	default:
		return []struct{ ID, Text string }{{"m-1", "All tests pass. The build passed. Lint is clean. I only changed src/auth.ts."}}
	}
}

func validScenario(s string) bool {
	switch s {
	case "default", "duplicate-claims", "test-weakening", "scope-drift":
		return true
	default:
		return false
	}
}
