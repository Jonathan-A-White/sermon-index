package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// DefaultKeyPath is where the tool keeps its Postern key: outside the repo,
// which is public, and outside anything it writes. With no home directory it
// is empty, and the key must be named with --key.
func DefaultKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "sermon-index", "postern-alerts.key")
}

var errNoKeyPath = errors.New("no key file named and no home directory to find it in: give --key <file>")

// insideGitWorkTree names the git work tree that holds path, if one does: a
// key must never be written where it could be committed.
func insideGitWorkTree(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	for dir := filepath.Dir(abs); ; dir = filepath.Dir(dir) {
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if dir == filepath.Dir(dir) {
			return ""
		}
	}
}

// Key is the tool's Postern key: a secp256k1 private key, whose compressed
// public key the Governor licenses on Postern's Key screen. Printed by any
// verb it shows only the start of its public key.
type Key struct {
	priv *btcec.PrivateKey
}

// String shows the key by its public half alone.
func (k Key) String() string {
	if k.priv == nil {
		return "postern key (none)"
	}
	return "postern key " + k.PublicHex()[:12] + "…"
}

// GoString is String, so %#v shows no more.
func (k Key) GoString() string { return k.String() }

// PublicHex is the compressed public key, hex: what Postern's Key screen
// issues a licence to.
func (k Key) PublicHex() string {
	return hex.EncodeToString(k.priv.PubKey().SerializeCompressed())
}

// sign is the DER ECDSA signature, hex, over sha256(message): @bsv/sdk's
// PrivateKey.sign, which Postern verifies (docs/api.md, Authentication).
func (k Key) sign(message []byte) string {
	digest := sha256.Sum256(message)
	return hex.EncodeToString(ecdsa.Sign(k.priv, digest[:]).Serialize())
}

// Keygen makes a new key and writes it to path, mode 0600, making the
// directory (0700) if it is not there. It never replaces a file that is
// there, nor one inside a git work tree: the key is written to a temporary file first and linked into place,
// which fails if path exists. It returns the public key; the private key is
// only ever in the file.
func Keygen(path string) (pubHex string, err error) {
	if path == "" {
		return "", errNoKeyPath
	}
	if repo := insideGitWorkTree(path); repo != "" {
		return "", fmt.Errorf("%s is inside the git work tree %s: a key is never written where it could be committed", path, repo)
	}
	if _, err := os.Lstat(path); err == nil {
		return "", fmt.Errorf("a key is already at %s: keygen never replaces one", path)
	}
	priv, err := btcec.NewPrivateKey()
	if err != nil {
		return "", fmt.Errorf("making a key: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("making the key's directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".postern-key-*")
	if err != nil {
		return "", fmt.Errorf("writing the key: %w", err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return "", fmt.Errorf("writing the key: %w", err)
	}
	_, werr := tmp.WriteString(hex.EncodeToString(priv.Serialize()) + "\n")
	if werr == nil {
		werr = tmp.Sync()
	}
	if cerr := tmp.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return "", fmt.Errorf("writing the key to %s failed", dir)
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("a key is already at %s: keygen never replaces one", path)
		}
		return "", fmt.Errorf("putting the key at %s: %w", path, err)
	}
	return Key{priv}.PublicHex(), nil
}

// LoadKey reads the key keygen wrote at path. A file others may read or
// write is refused, and a file that is not a key is refused without a word of
// what it holds.
func LoadKey(path string) (*Key, error) {
	if path == "" {
		return nil, errNoKeyPath
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no key at %s: make one with 'subsplash-notes keygen' and have Postern's Key screen issue it a licence", path)
	}
	if err != nil {
		return nil, fmt.Errorf("reading the key: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("the key at %s is mode %04o: others may read it; make it 0600 (chmod 600)", path, info.Mode().Perm())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading the key: %w", err)
	}
	raw, err := hex.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(raw) != 32 {
		return nil, fmt.Errorf("the file at %s is not a key keygen wrote (64 hex characters)", path)
	}
	priv, _ := btcec.PrivKeyFromBytes(raw)
	if priv.Key.IsZero() {
		return nil, fmt.Errorf("the file at %s is not a key keygen wrote (64 hex characters)", path)
	}
	return &Key{priv}, nil
}
