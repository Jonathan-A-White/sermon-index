package main

import (
	"encoding/json"
	"strings"
	"testing"
)

const goodLink = "https://mail.google.com/mail/vf-%5BANGjdJ_invented%5D-abcdef"

// alertsReply is a GET /api/alerts body of the given alerts.
func alertsReply(t *testing.T, alerts ...map[string]any) string {
	t.Helper()
	b, err := json.Marshal(map[string]any{"alerts": alerts, "next": 99})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func forwardAlert(id string, seq int, link string) map[string]any {
	return map[string]any{"id": id, "seq": seq, "kind": "forward-confirm", "when": "2031-03-01T10:00:00-05:00", "link": link}
}

// confirmRun licenses a fresh key on a fake Postern holding the reply and
// runs forward-confirm against it.
func confirmRun(t *testing.T, reply string) (f *fakePostern, code int, out, errOut string) {
	t.Helper()
	f, srv := newFakePostern(t, reply)
	key, pub := newKey(t)
	f.licensed[pub] = true
	code, out, errOut = cliRun(t, "forward-confirm", "--key", key, "--postern", srv.URL)
	return f, code, out, errOut
}

// Acceptance 1: two forward-confirm alerts; the link of the newest is printed.
func TestForwardConfirmPrintsTheNewestLink(t *testing.T) {
	older := "https://mail.google.com/mail/vf-%5BANGjdJ_older%5D-111111"
	newest := "https://mail.google.com/mail/vf-%5BANGjdJ_newest%5D-222222"
	// The newest is neither first nor last in the list, and a later
	// sermon-notes alert has a higher seq than both.
	reply := alertsReply(t,
		forwardAlert("00000000000000b1", 3, older),
		forwardAlert("00000000000000b2", 7, newest),
		forwardAlert("00000000000000b0", 1, "https://mail.google.com/mail/vf-%5Bancient%5D-000000"),
		map[string]any{"id": "00000000000000b3", "seq": 9, "kind": "sermon-notes", "title": "T", "reference": "Psalm 23", "when": "2031-03-02T00:00:00-05:00"},
	)
	_, code, out, errOut := confirmRun(t, reply)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	if out != newest+"\n" {
		t.Errorf("printed %q, want the newest link %q", out, newest)
	}
}

// Acceptance 2: only a link on https://mail.google.com/mail/vf- is printed;
// any other host or path is refused and not printed.
func TestForwardConfirmRefusesALinkOffGmail(t *testing.T) {
	bad := []string{
		"https://evil.example.com/mail/vf-abc",
		"https://mail.google.com.evil.example.com/mail/vf-abc",
		"https://mail.google.com@example.com/mail/vf-abc",
		"https://evil.example.com/?u=https://mail.google.com/mail/vf-abc",
		"http://mail.google.com/mail/vf-abc",
		"https://mail.google.com:8443/mail/vf-abc",
		"https://mail.google.com/mail/uf-abc",
		"https://mail.google.com/mail/u/0/vf-abc",
		"https://mail.google.com/mail/vf-abc/../../uf-abc",
		"https://mail.google.com/mail/vf-",
		"https://mail.google.com/other/vf-abc",
		"https://MAIL.google.com.evil.example/mail/vf-abc",
		"https://google.com/mail/vf-abc",
		"https://mail.google.com/mail/uf-abc?x=1",
		"https://mail-settings.google.com/mail/uf-abc",
		"https://isolated.mail.google.com/mail/uf-abc",
		"https://mail-settings.google.com.evil.example.com/mail/vf-abc",
		"https://other.mail.google.com/mail/vf-abc",
		"https://mail.google.com/mail/vf-abc?ik=abc",
		"https://mail.google.com/mail/vf-abc?",
		"https://mail.google.com/mail/vf-abc#frag",
		"https://mail.google.com/mail/vf-abc#",
		"https://mail-settings.google.com/mail/vf-abc?ik=abc",
		"https://isolated.mail.google.com/mail/vf-abc#frag",
		"https://isolated.mail.google.com@example.com/mail/vf-abc",
		"https://mail-settings.google.com:8443/mail/vf-abc",
		"https://mail-settings.google.com/mail/vf-abc/more",
		"https://mail.google.com/mail/vf-abc def",
		"https://mail.google.com/mail/vf-abc\nsecond line",
		"javascript:alert(1)",
		"mail.google.com/mail/vf-abc",
		"",
	}
	for _, link := range bad {
		_, code, out, errOut := confirmRun(t, alertsReply(t, forwardAlert("00000000000000b1", 1, link)))
		if code == 0 {
			t.Errorf("link %q: exit 0", link)
		}
		if out != "" {
			t.Errorf("link %q: printed %q", link, out)
		}
		if link != "" && link != "https://mail.google.com/mail/vf-" && strings.Contains(errOut, link) { // the bare prefix is in the rule the refusal states
			t.Errorf("link %q: the refusal repeats the link: %q", link, errOut)
		}
		if !strings.Contains(errOut, "00000000000000b1") || !strings.Contains(errOut, "refus") {
			t.Errorf("link %q: stderr does not say which alert was refused: %q", link, errOut)
		}
	}
	// The newest alert's link is the one judged: an older good link is not
	// printed in its place.
	_, code, out, _ := confirmRun(t, alertsReply(t,
		forwardAlert("00000000000000b1", 1, goodLink),
		forwardAlert("00000000000000b2", 2, "https://evil.example.com/mail/vf-abc")))
	if code == 0 || out != "" {
		t.Errorf("a bad newest link with a good older one: exit %d, printed %q", code, out)
	}
}

// Acceptance 1 (mw-qtu3zvz.49.10): each host in Postern's alerts.ConfirmHosts
// is printed, with an invented token.
func TestForwardConfirmPrintsEachConfirmHost(t *testing.T) {
	for _, host := range []string{"mail.google.com", "mail-settings.google.com", "isolated.mail.google.com"} {
		link := "https://" + host + "/mail/vf-%5BANGjdJ_invented%5D-abc123"
		_, code, out, errOut := confirmRun(t, alertsReply(t, forwardAlert("00000000000000b1", 1, link)))
		if code != 0 || out != link+"\n" {
			t.Errorf("%s: exit %d, printed %q, %s", host, code, out, errOut)
		}
	}
}

// Acceptance 3: with no forward-confirm alert it says so plainly and exits
// non-zero.
func TestForwardConfirmWithNoneSaysSo(t *testing.T) {
	for name, reply := range map[string]string{
		"no alerts":           alertsReply(t),
		"only other kinds":    fixture(t),
		"a charge and unread": alertsReply(t, map[string]any{"id": "c1", "seq": 1, "kind": "charge", "amount": "1.00"}, map[string]any{"id": "c2", "seq": 2, "kind": "unread", "subject": "x"}),
	} {
		_, code, out, errOut := confirmRun(t, reply)
		if code == 0 {
			t.Errorf("%s: exit 0", name)
		}
		if out != "" {
			t.Errorf("%s: printed %q", name, out)
		}
		if !strings.Contains(errOut, "no forward-confirm alert") {
			t.Errorf("%s: stderr does not say there is none: %q", name, errOut)
		}
	}
}

// Acceptance 4: the command makes no DELETE, nor any request but the read of
// the alerts (and the challenge and signing every read needs).
func TestForwardConfirmOnlyReadsTheAlerts(t *testing.T) {
	reply := alertsReply(t, forwardAlert("00000000000000b1", 1, goodLink), forwardAlert("00000000000000b2", 2, goodLink))
	f, code, _, errOut := confirmRun(t, reply)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errOut)
	}
	want := "GET /api/challenge,GET /api/alerts"
	if got := strings.Join(f.requests, ","); got != want {
		t.Errorf("requests %q, want %q", got, want)
	}
	if len(f.claimed) != 0 || len(f.alerts) != 2 {
		t.Errorf("claimed %v, the fake holds %d alerts", f.claimed, len(f.alerts))
	}
	// Refused and empty runs also read and nothing else.
	for _, r := range []string{alertsReply(t), alertsReply(t, forwardAlert("00000000000000b1", 1, "https://evil.example.com/"))} {
		f, _, _, _ := confirmRun(t, r)
		if got := strings.Join(f.requests, ","); got != want {
			t.Errorf("requests %q, want %q", got, want)
		}
	}
}

// Acceptance 5: fetch is unchanged: a forward-confirm alert is still not a
// sermon record, and it is not claimed.
func TestFetchLeavesAForwardConfirmAlone(t *testing.T) {
	reply := alertsReply(t, forwardAlert("00000000000000b1", 1, goodLink))
	if notes, problems, err := ReadAlerts(strings.NewReader(reply)); err != nil || len(notes) != 0 || len(problems) != 0 {
		t.Errorf("ReadAlerts: notes %v, problems %v, err %v", notes, problems, err)
	}
	f, srv := newFakePostern(t, reply)
	key, pub := newKey(t)
	f.licensed[pub] = true
	records := t.TempDir() + "/out.jsonl"
	code, out, errOut := cliRun(t, "fetch", "--key", key, "--postern", srv.URL, "--rows", rowsFile(t), "--out", records)
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	if !strings.Contains(out, "0 records") || errOut != "" || len(f.claimed) != 0 || len(f.alerts) != 1 {
		t.Errorf("fetch: %q %q claimed %v alerts %d", out, errOut, f.claimed, len(f.alerts))
	}
}

// Usage, flags and failures are the other commands': no Postern named, a
// stray argument.
func TestForwardConfirmUsage(t *testing.T) {
	t.Setenv("POSTERN_URL", "")
	key, _ := newKey(t)
	if code, _, errOut := cliRun(t, "forward-confirm", "--key", key); code == 0 || !strings.Contains(errOut, "--postern") {
		t.Errorf("no Postern: exit %d: %s", code, errOut)
	}
	if code, _, _ := cliRun(t, "forward-confirm", "extra"); code != 2 {
		t.Errorf("a stray argument: exit %d, want 2", code)
	}
}
