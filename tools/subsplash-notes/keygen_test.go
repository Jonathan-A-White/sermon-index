package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMain points HOME at a directory of its own, so no test can reach the
// real key at ~/.config/sermon-index/postern-alerts.key even by the default.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "subsplash-notes-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Setenv("HOME", home)
	code := m.Run()
	os.RemoveAll(home)
	os.Exit(code)
}

var pubKeyLine = regexp.MustCompile(`^0[23][0-9a-f]{64}\n$`)

// cliRun runs the tool as its command line would, reporting the exit code
// and what it printed.
func cliRun(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var so, se bytes.Buffer
	code = cli(args, strings.NewReader(""), &so, &se)
	return code, so.String(), se.String()
}

// Acceptance 5: keygen twice in a temp dir. The first writes a 0600 file and
// prints a public key; the second exits non-zero and leaves the file
// byte-identical.
func TestKeygenWritesOnceAndRefusesASecond(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "sermon-index", "postern-alerts.key")

	code, out, errOut := cliRun(t, "keygen", "--key", path)
	if code != 0 {
		t.Fatalf("keygen exited %d: %s", code, errOut)
	}
	if !pubKeyLine.MatchString(out) {
		t.Errorf("keygen printed %q, not one compressed public key", out)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("the key file's mode is %v, not 0600", info.Mode().Perm())
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	key, err := LoadKey(path)
	if err != nil {
		t.Fatal(err)
	}
	if key.PublicHex()+"\n" != out {
		t.Errorf("keygen printed %q, but the file's public key is %s", out, key.PublicHex())
	}

	code, out2, errOut2 := cliRun(t, "keygen", "--key", path)
	if code == 0 {
		t.Errorf("a second keygen exited 0 (stdout %q)", out2)
	}
	if out2 != "" {
		t.Errorf("a refused keygen printed %q", out2)
	}
	if !strings.Contains(errOut2, path) || strings.Count(strings.TrimRight(errOut2, "\n"), "\n") != 0 {
		t.Errorf("a refused keygen should say so in one line naming the file: %q", errOut2)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Error("a refused keygen changed the key file")
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("keygen left more than the key in its directory: %v", entries)
	}
}

// Every subcommand but keygen takes the key from --key, defaulting to
// ~/.config/sermon-index/postern-alerts.key (here the test's own HOME).
func TestDefaultKeyPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "sermon-index", "postern-alerts.key")
	if got := DefaultKeyPath(); got != want {
		t.Errorf("DefaultKeyPath() = %q, want %q", got, want)
	}
}

// keygen never writes a key inside a git work tree, where it could be
// committed to the public repo, and with no path it asks for one.
func TestKeygenRefusesAGitWorkTreeOrNoPath(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repo, "keys", "postern-alerts.key")
	code, out, errOut := cliRun(t, "keygen", "--key", path)
	if code == 0 || out != "" || !strings.Contains(errOut, "git work tree") {
		t.Errorf("keygen inside a work tree: exit %d, %q, %q", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Errorf("keygen made something inside the work tree: %v", err)
	}
	if code, _, errOut := cliRun(t, "keygen", "--key", ""); code == 0 || !strings.Contains(errOut, "--key") {
		t.Errorf("keygen with no path: exit %d, %q", code, errOut)
	}
}

// A key file others can read is refused, and a key file that is not a key
// is refused without its contents in the error.
func TestLoadKeyRefusesAnOpenOrBadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "k")
	if code, _, errOut := cliRun(t, "keygen", "--key", path); code != 0 {
		t.Fatal(errOut)
	}
	secret, _ := os.ReadFile(path)
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadKey(path)
	if err == nil || !strings.Contains(err.Error(), "0600") {
		t.Errorf("a 0644 key file was not refused for its mode: %v", err)
	}
	if err != nil && strings.Contains(err.Error(), strings.TrimSpace(string(secret))) {
		t.Error("the refusal holds the private key")
	}

	bad := filepath.Join(dir, "bad")
	if err := os.WriteFile(bad, []byte("not-a-key-zz-very-private-zz\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = LoadKey(bad)
	if err == nil {
		t.Fatal("a file that is no key was loaded")
	}
	if strings.Contains(err.Error(), "very-private") {
		t.Errorf("the refusal echoes the file: %v", err)
	}
	if _, err := LoadKey(filepath.Join(dir, "missing")); err == nil || !strings.Contains(err.Error(), "keygen") {
		t.Errorf("a missing key should point at keygen: %v", err)
	}
}

// A Key printed by any verb shows only its public half.
func TestKeyPrintsOnlyItsPublicHalf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "k")
	if code, _, errOut := cliRun(t, "keygen", "--key", path); code != 0 {
		t.Fatal(errOut)
	}
	key, err := LoadKey(path)
	if err != nil {
		t.Fatal(err)
	}
	priv := privateHex(t, path)
	for _, s := range []string{fmt.Sprint(key), fmt.Sprintf("%v %+v %#v %s %x", key, key, key, key, key), fmt.Sprintf("%#v", *key)} {
		if containsSecret(s, priv) {
			t.Errorf("printing a Key shows the private key: %s", s)
		}
	}
}

// privateHex is the private key in the file at path, as the tests look for
// it in what the tool printed.
func privateHex(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := strings.TrimSpace(string(b))
	if _, err := hex.DecodeString(s); err != nil || len(s) != 64 {
		t.Fatalf("the key file does not hold 64 hex characters")
	}
	return s
}

// containsSecret reports whether s holds the private key, in either case, or
// any 16-character run of it.
func containsSecret(s, privHex string) bool {
	low := strings.ToLower(s)
	for i := 0; i+16 <= len(privHex); i++ {
		if strings.Contains(low, strings.ToLower(privHex[i:i+16])) {
			return true
		}
	}
	return false
}
