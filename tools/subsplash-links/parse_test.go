package main

import (
	"strings"
	"testing"
)

func TestParseShort(t *testing.T) {
	good := map[string]string{
		"https://brookfieldfirst.subspla.sh/ab12cd3":      "ab12cd3",
		"  https://brookfieldfirst.subspla.sh/ab12cd3/  ": "ab12cd3",
		"https://BrookfieldFirst.subspla.sh/ab12cd3":      "ab12cd3",
		"https://brookfieldfirst.subspla.sh/ab12cd3?x=1":  "ab12cd3",
	}
	for in, want := range good {
		got, err := ParseShort(in)
		if err != nil || got != want {
			t.Errorf("ParseShort(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	// Anything not on brookfieldfirst.subspla.sh is refused, and never requested.
	for _, bad := range []string{
		"http://brookfieldfirst.subspla.sh/ab12cd3", // not https
		"https://other.subspla.sh/ab12cd3",
		"https://brookfieldfirst.subspla.sh.example.org/ab12cd3",
		"https://brookfieldfirst.subspla.sh@example.org/ab12cd3",
		"https://user@brookfieldfirst.subspla.sh/ab12cd3",
		"https://brookfieldfirst.subspla.sh:8443/ab12cd3",
		"https://brookfieldfirst.subspla.sh/",
		"https://brookfieldfirst.subspla.sh/a/b",
		"https://example.org/ab12cd3",
		"ab12cd3",
		"",
	} {
		if code, err := ParseShort(bad); err == nil {
			t.Errorf("ParseShort(%q) = %q; want an error", bad, code)
		}
	}
}

func TestParseAddress(t *testing.T) {
	a, err := ParseAddress("https://subsplash.com/u/brookfieldfirst/media/d/q59d572-the-day-of-the-lord?autoplay=true")
	if err != nil {
		t.Fatal(err)
	}
	if a.Code != "q59d572" || a.Slug != "the-day-of-the-lord" || a.URL != "https://subsplash.com/u/brookfieldfirst/media/d/q59d572-the-day-of-the-lord" {
		t.Errorf("address = %+v", a)
	}
	for _, bad := range []string{
		"https://subsplash.com/u/brookfieldfirst/media/s/z2m8kfk-a-series",        // a series
		"https://subsplash.com/u/brookfieldfirst/media/d/q59d572",                 // no slug
		"https://subsplash.com/u/otherchurch/media/d/q59d572-the-day-of-the-lord", // another church
		"http://subsplash.com/u/brookfieldfirst/media/d/q59d572-x",
		"https://example.org/u/brookfieldfirst/media/d/q59d572-x",
		"https://subsplash.com/u/brookfieldfirst/media/d/q59d572-x/extra",
	} {
		if a, err := ParseAddress(bad); err == nil {
			t.Errorf("ParseAddress(%q) = %+v; want an error", bad, a)
		}
	}
}

func TestReadLinkFile(t *testing.T) {
	in := strings.Join([]string{
		"# the links he shared",
		"",
		"https://brookfieldfirst.subspla.sh/aaa1111",
		"https://brookfieldfirst.subspla.sh/aaa1111", // twice: once
		"https://brookfieldfirst.subspla.sh/bbb2222",
		"https://example.org/ccc3333",
		"https://subsplash.com/u/brookfieldfirst/media/d/ddd4444-a-title",
		"not a link",
	}, "\n")
	got := ReadLinkFile(strings.NewReader(in))
	if strings.Join(got.Codes, ",") != "aaa1111,bbb2222" {
		t.Errorf("codes = %v", got.Codes)
	}
	if len(got.Addresses) != 1 || got.Addresses[0].Code != "ddd4444" {
		t.Errorf("addresses = %+v", got.Addresses)
	}
	if len(got.Problems) != 2 || !strings.Contains(got.Problems[0], "line 6") || !strings.Contains(got.Problems[1], "line 8") {
		t.Errorf("problems = %v", got.Problems)
	}
}

// pdftohtml -stdout -xml -i output: the address in the text, and in an href
// where the text wraps; an autoplay duplicate; a series; another site.
const fixtureXML = `<?xml version="1.0" encoding="UTF-8"?>
<pdf2xml producer="poppler" version="24.02.0">
<page number="1" position="absolute" top="0" left="0" height="1188" width="918">
<text top="100" left="50" width="600" height="14" font="0"><a href="https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-first-sermon">https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-first-sermon</a></text>
<text top="120" left="50" width="600" height="14" font="0">https://subsplash.com/u/brookfieldfirst/media/d/aaa1111-first-sermon?autoplay=true&amp;x=1</text>
<text top="140" left="50" width="300" height="14" font="0"><a href="https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-second-one?autoplay=true">https://subsplash.com/u/brookfieldfirst/media/d/bbb2222-sec</a></text>
<text top="160" left="50" width="300" height="14" font="0"><a href="https://subsplash.com/u/brookfieldfirst/media/s/zzz9999-a-series">a series</a></text>
<text top="180" left="50" width="300" height="14" font="0"><a href="https://example.org/u/brookfieldfirst/media/d/eee5555-elsewhere">elsewhere</a></text>
</page>
</pdf2xml>`

func TestAddressesInXML(t *testing.T) {
	addrs, other := AddressesInXML([]byte(fixtureXML))
	var got []string
	for _, a := range addrs {
		got = append(got, a.Code+"="+a.Slug)
	}
	// ?autoplay duplicates count once, in order of first appearance.
	if strings.Join(got, ",") != "aaa1111=first-sermon,bbb2222=second-one" {
		t.Errorf("addresses = %v", got)
	}
	if len(other) != 1 || !strings.Contains(other[0], "/media/s/zzz9999-a-series") {
		t.Errorf("other = %v", other)
	}
}
