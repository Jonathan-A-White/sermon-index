package sources

import (
	"os"
	"strings"
	"testing"
)

func TestProposeFillsTheStoryFromASource(t *testing.T) {
	text, err := Propose(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"C. H. Spurgeon", // the preacher: the one the rows name
		"C. H. Spurgeon's sermons (twenty, a fixture)", // the source's name
		"https://ccel.org/about/copyright.html",        // the terms url
		"https://ccel.org",                             // the catalogue (homepage)
		"20 rows",                                      // the count
		"19 with a preaching text",                     // and of those with a text
		"Size: S",
		"These books may be used for personal, educational, or non-profit purposes", // the licence, quoted
		"sermons check", "--urls", "sermons load", // the steps
	} {
		if !strings.Contains(text, want) {
			t.Errorf("the story lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "{{") || strings.Contains(text, "<no value>") {
		t.Errorf("a field was left unfilled:\n%s", text)
	}
}

func TestProposeSizeFollowsTheRowCount(t *testing.T) {
	for _, c := range []struct {
		rows int
		want string
	}{{1, "S"}, {300, "S"}, {301, "M"}, {2000, "M"}, {2001, "L"}} {
		if got := sizeOf(c.rows); got != c.want {
			t.Errorf("sizeOf(%d) = %s, want %s", c.rows, got, c.want)
		}
	}
}

func TestProposeFallsBackToTheCreditWhenRowsNameNoPreacher(t *testing.T) {
	rows := `{"id": "1", "title": "One", "ref": "Rom.5.8", "url": "https://example.org/1"}` + "\n"
	text, err := Propose(writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": rows}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Preacher: Someone") || !strings.Contains(text, "1 rows") && !strings.Contains(text, "1 row") {
		t.Errorf("%s", text)
	}
}

func TestProposeRefusesASourceWithFaultsAndAThemeSource(t *testing.T) {
	_, err := Propose(writeSource(t, map[string]string{"source.toml": goodToml, "rows.jsonl": `{"id": "1", "title": "One", "ref": "Rom.99", "url": "https://example.org/1"}`}))
	if err == nil || !strings.Contains(err.Error(), "rows.jsonl:1: ref 'Rom.99'") {
		t.Errorf("faults: %v", err)
	}
	_, err = Propose(writeSource(t, map[string]string{"source.toml": goodThemeToml, "topics.jsonl": `{"id": "love", "label": "Love"}`, "links.jsonl": `{"topic": "love", "ref": "Rom.5.8"}`}))
	if err == nil || !strings.Contains(err.Error(), "sermon source") {
		t.Errorf("a theme source: %v", err)
	}
}

// The recipe, the template and the format doc name what a Builder needs:
// a test keeps them from losing a step or a field.
func TestRecipeAndTemplateNameTheSteps(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	recipe := read("../../docs/adding-a-source.md")
	for _, want := range []string{
		"robots.txt", "BEFORE anything else", "link-out", "tools/<source>", "by hand", "sermons check --urls",
		"sermons load", "sermons propose", "What's new", "Outside this repo", "data/sermons/sources.json",
		"Truth For Life", "Content-Signal", "the terms it could not read", "series", "expect_title", "host_rules",
		"Before any crawl or download from a web host", "on the story's bead before it is released",
		"Governor's or the owner's word", "mw-qtu3zvz.13", "A.5", "A.6", "automated means, including spiders, robots, crawlers, data mining tools",
	} {
		if !strings.Contains(recipe, want) {
			t.Errorf("docs/adding-a-source.md lacks %q", want)
		}
	}
	// Every other doc that tells a Builder how to add a web source states the
	// step or points at the recipe that does.
	for _, p := range []string{"../../README.md", "../../prompts/add-source.md", "../../sources/_template/README.md", "../../docs/candidates.md"} {
		if !strings.Contains(read(p), "adding-a-source.md") {
			t.Errorf("%s does not point at docs/adding-a-source.md", p)
		}
	}
	for _, p := range []string{"../../README.md", "../../prompts/add-source.md"} {
		if !strings.Contains(read(p), "before it is released") {
			t.Errorf("%s does not state the rule: clause quoted on the bead before release", p)
		}
	}
	tmpl := read("../../sources/_template/source.toml")
	for _, want := range []string{"host_rules", "title_in_page", "delay_ms", "series", "expect_title"} {
		if !strings.Contains(tmpl, want) {
			t.Errorf("sources/_template/source.toml does not explain %q", want)
		}
	}
	format := read("../../docs/sources.md")
	for _, want := range []string{"## Host rules", "## sermons propose", "`series`", "`expect_title`", "warning"} {
		if !strings.Contains(format, want) {
			t.Errorf("docs/sources.md lacks %q", want)
		}
	}
	if _, err := os.Stat("../../sources/_template/README.md"); err != nil {
		t.Error(err)
	}
}
