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

// The recipe's two tool tables keep up with the tools: every directory under
// tools/ has a line in the table of what the tools read, and each of these
// sources has a line, naming its tool, in the table of the shapes here.
func TestRecipeTablesNameTheTools(t *testing.T) {
	b, err := os.ReadFile("../../docs/adding-a-source.md")
	if err != nil {
		t.Fatal(err)
	}
	// line returns the row that begins with cell in the table whose header
	// begins with header, or "".
	line := func(header, cell string) string {
		_, table, found := strings.Cut(string(b), "\n"+header)
		if !found {
			t.Fatalf("docs/adding-a-source.md has no table %q", header)
		}
		table, _, _ = strings.Cut(table, "\n\n")
		for _, l := range strings.Split(table, "\n") {
			if strings.HasPrefix(l, "| "+cell+" |") {
				return l
			}
		}
		return ""
	}
	tools, err := os.ReadDir("../../tools")
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range tools {
		if d.IsDir() && line("| Tool | Reads |", "`tools/"+d.Name()+"`") == "" {
			t.Errorf("the table of what the tools read has no line for tools/%s", d.Name())
		}
	}
	for _, source := range []string{"clarkson-works", "lifebible-ttb"} {
		if l := line("| Source | Catalogue |", "`"+source+"`"); !strings.Contains(l, "`tools/"+source+"`") {
			t.Errorf("the table of the shapes here has no line for %s naming its tool", source)
		}
	}
}

// A notes mail with no passage is read from its notes PDF (pdfPassage); the
// recipe and the hourly script's header say so and not that it is left as it is.
func TestRecipeSaysARecordWithNoPassageIsReadFromItsNotesPDF(t *testing.T) {
	read := func(p string) string {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	recipe := read("../../docs/adding-a-source.md")
	for _, want := range []string{"pdfPassage", "--notes-pdfs", "kept dash", "named in the report"} {
		if !strings.Contains(recipe, want) {
			t.Errorf("docs/adding-a-source.md lacks %q on the notes mails' passage", want)
		}
	}
	if strings.Contains(recipe, "a record with no passage and a record no single row holds are left as they are") {
		t.Errorf("docs/adding-a-source.md still says a record with no passage is left as it is")
	}
	hourly := read("../../scripts/subsplash-hourly.sh")
	for _, want := range []string{"pdfPassage", "notes PDF header"} {
		if !strings.Contains(hourly, want) {
			t.Errorf("scripts/subsplash-hourly.sh step 2c lacks %q", want)
		}
	}
}
