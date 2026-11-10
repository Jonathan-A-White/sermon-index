package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "cache.jsonl")
	c, err := LoadCache(path) // missing: empty
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("aaa1111"); ok {
		t.Fatal("empty cache has an entry")
	}
	c.Put("bbb2222", "https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-two")
	c.Put("aaa1111", "https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-one")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(b), `{"code":"aaa1111"`) || strings.Count(string(b), "\n") != 2 {
		t.Errorf("file = %q", b)
	}
	c2, err := LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := c2.Get("bbb2222"); !ok || got != "https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-two" {
		t.Errorf("Get = %q, %v", got, ok)
	}
}

func TestCacheRefusesBadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.jsonl")
	os.WriteFile(path, []byte("not json\n"), 0o644)
	if _, err := LoadCache(path); err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Errorf("err = %v", err)
	}
}
