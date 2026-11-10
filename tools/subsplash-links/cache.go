package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Cache keeps code -> full address, so that no link is ever requested
// twice. It is a JSON-lines file: {"code": "...", "address": "..."}.
type Cache struct {
	path string
	m    map[string]string
}

type cacheLine struct {
	Code    string `json:"code"`
	Address string `json:"address"`
}

// LoadCache reads the cache at path; a file that does not exist is an empty
// cache.
func LoadCache(path string) (*Cache, error) {
	c := &Cache{path: path, m: map[string]string{}}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	for n, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var l cacheLine
		if err := json.Unmarshal([]byte(line), &l); err != nil || l.Code == "" || l.Address == "" {
			return nil, fmt.Errorf("%s line %d is not a cache entry", path, n+1)
		}
		c.m[l.Code] = l.Address
	}
	return c, nil
}

// Get is the full address cached for a code.
func (c *Cache) Get(code string) (string, bool) {
	a, ok := c.m[code]
	return a, ok
}

// Put keeps an address for a code, in memory; Save writes it.
func (c *Cache) Put(code, address string) { c.m[code] = address }

// Len is how many codes are kept.
func (c *Cache) Len() int { return len(c.m) }

// Save writes the cache to its file, sorted by code, through a temporary
// file renamed into place.
func (c *Cache) Save() error {
	codes := make([]string, 0, len(c.m))
	for code := range c.m {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, code := range codes {
		if err := enc.Encode(cacheLine{code, c.m[code]}); err != nil {
			return err
		}
	}
	return writeFile(c.path, buf.Bytes())
}

// writeFile replaces path with data: a temporary file beside it, synced and
// renamed, so path is the old file or the whole new one.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".subsplash-links-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(0o644)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}
