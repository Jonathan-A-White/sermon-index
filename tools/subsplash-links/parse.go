package main

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"net/url"
	"regexp"
	"strings"
)

const (
	// shortHost is the one host the tool ever requests.
	shortHost = "brookfieldfirst.subspla.sh"
	// addressPrefix starts every address the tool writes: the church's
	// sermon (media/d) pages on subsplash.com.
	addressPrefix = "https://subsplash.com/u/brookfieldfirst/media/d/"
)

var (
	codeRE = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	slugRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	// churchRE finds any address of the church's on subsplash.com in a page of text.
	churchRE = regexp.MustCompile(`https://subsplash\.com/u/brookfieldfirst/[^\s"'<>)\]]*`)
)

// Address is a sermon's page on subsplash.com: the full address the tool
// writes to a row's alt_url, with the short code that is the media's code
// and the slug of its title.
type Address struct {
	Code string
	Slug string
	URL  string // addressPrefix + Code + "-" + Slug, with no query
}

// ParseShort is the code of a short link on brookfieldfirst.subspla.sh, or
// an error: nothing but that host, over https, with a single path segment
// of letters and digits, is ever requested.
func ParseShort(s string) (string, error) {
	s = strings.TrimSpace(s)
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("'%s' is not an https link", s)
	}
	if u.User != nil || !strings.EqualFold(u.Host, shortHost) {
		return "", fmt.Errorf("'%s' is not on %s", s, shortHost)
	}
	code := strings.Trim(u.Path, "/")
	if !codeRE.MatchString(code) {
		return "", fmt.Errorf("'%s' has no single code in its path", s)
	}
	return code, nil
}

// ParseAddress reads a full address of a sermon on subsplash.com,
// https://subsplash.com/u/brookfieldfirst/media/d/<code>-<slug>. Its query
// (?autoplay=true) and fragment are dropped. Anything else, a series
// (media/s) among it, is an error.
func ParseAddress(s string) (Address, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Host != "subsplash.com" {
		return Address{}, fmt.Errorf("'%s' is not an https address on subsplash.com", s)
	}
	rest, ok := strings.CutPrefix(u.Path, "/u/brookfieldfirst/media/d/")
	if !ok {
		return Address{}, fmt.Errorf("'%s' is not a sermon of the church (/u/brookfieldfirst/media/d/<code>-<slug>)", s)
	}
	code, slug, ok := strings.Cut(rest, "-")
	if !ok || !codeRE.MatchString(code) || !slugRE.MatchString(slug) {
		return Address{}, fmt.Errorf("'%s' has no <code>-<slug> after /media/d/", s)
	}
	return Address{Code: code, Slug: slug, URL: addressPrefix + code + "-" + slug}, nil
}

// Inputs are the links found in the inputs, in order, each once: the codes
// of short links still to expand, the full addresses already given, and what
// could not be used.
type Inputs struct {
	Codes     []string
	Addresses []Address
	Problems  []string
}

// ReadLinkFile reads a file of links, one a line: a short link on
// brookfieldfirst.subspla.sh, or (given in full already) an address of a
// sermon on subsplash.com. A blank line and a line beginning # are skipped.
// Any other line is a problem, and is never requested.
func ReadLinkFile(r io.Reader) Inputs {
	var in Inputs
	seenCode := map[string]bool{}
	seenAddr := map[string]bool{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if code, err := ParseShort(line); err == nil {
			if !seenCode[code] {
				seenCode[code] = true
				in.Codes = append(in.Codes, code)
			}
			continue
		}
		if a, err := ParseAddress(line); err == nil {
			if !seenAddr[a.Code] {
				seenAddr[a.Code] = true
				in.Addresses = append(in.Addresses, a)
			}
			continue
		}
		in.Problems = append(in.Problems, fmt.Sprintf("line %d: '%s' is not a link on %s nor a sermon address on subsplash.com: not requested", n, line, shortHost))
	}
	if err := sc.Err(); err != nil {
		in.Problems = append(in.Problems, "reading the links: "+err.Error())
	}
	return in
}

// AddressesInXML finds the church's sermon addresses in the XML that
// `pdftohtml -stdout -xml -i <file>` makes of a search-result PDF: in the
// text and in the href of a link (where a long address wraps). Entities are
// read first. An address met again, with or without ?autoplay, counts once.
// other lists the church's addresses that are not a sermon (a series), once
// each, without their query.
func AddressesInXML(xml []byte) (addrs []Address, other []string) {
	text := html.UnescapeString(string(xml))
	seen := map[string]bool{}
	seenOther := map[string]bool{}
	for _, m := range churchRE.FindAllString(text, -1) {
		m = strings.TrimRight(m, ".,;")
		if a, err := ParseAddress(m); err == nil {
			if !seen[a.Code] {
				seen[a.Code] = true
				addrs = append(addrs, a)
			}
			continue
		}
		bare, _, _ := strings.Cut(m, "?")
		if !seenOther[bare] {
			seenOther[bare] = true
			other = append(other, bare)
		}
	}
	return addrs, other
}
