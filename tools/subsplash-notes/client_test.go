package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// fakeAddress is the invented alert address the fake gives the family: a
// 12-character token of a-z and 2-7 at an example domain.
const fakeAddress = "invented2345@example.com"

// fakePostern is Postern's alerts routes as docs/protocol.md section 26 and
// docs/api.md (Authentication) give them: a challenge, then every request
// signed by a licensed key with a Postern2 header over the v2 message. It
// verifies each signature itself, with the library Postern's own server uses,
// and never with the tool's code.
type fakePostern struct {
	t        *testing.T
	mu       sync.Mutex
	licensed map[string]bool
	nonces   map[string]bool // issued and not yet spent
	alerts   []map[string]any
	claimed  []string
	verified int               // signed requests whose signature verified
	refused  int               // signed requests refused for their proof
	nonceBad int               // how many of the next proofs to refuse with reason "nonce"
	wire     []byte            // every request's headers and body, and every reply's body
	requests []string          // every request's method and path, the challenges included
	notes    map[string][]byte // GET /api/alerts/{id}/notes answers these bytes; any other id is 404
	onClaim  func(id string)   // called as an alert is claimed, before it is gone
}

func newFakePostern(t *testing.T, alertsJSON string) (*fakePostern, *httptest.Server) {
	f := &fakePostern{t: t, licensed: map[string]bool{}, nonces: map[string]bool{}}
	if alertsJSON != "" {
		var reply struct {
			Alerts []map[string]any `json:"alerts"`
		}
		if err := json.Unmarshal([]byte(alertsJSON), &reply); err != nil {
			t.Fatal(err)
		}
		f.alerts = reply.Alerts
	}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return f, srv
}

func (f *fakePostern) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	defer f.mu.Unlock()
	for k, v := range r.Header {
		f.wire = append(f.wire, []byte(k+": "+strings.Join(v, ",")+"\n")...)
	}
	f.wire = append(f.wire, body...)
	f.requests = append(f.requests, r.Method+" "+r.URL.Path)
	rec := httptest.NewRecorder()
	f.serve(rec, r, body)
	f.wire = append(f.wire, rec.Body.Bytes()...)
	for k, v := range rec.Header() {
		w.Header()[k] = v
	}
	w.WriteHeader(rec.Code)
	w.Write(rec.Body.Bytes())
}

func (f *fakePostern) refuse(w http.ResponseWriter, status int, reason, words string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": words, "reason": reason})
}

func (f *fakePostern) serve(w http.ResponseWriter, r *http.Request, body []byte) {
	if r.Method == http.MethodGet && r.URL.Path == "/api/challenge" {
		b := make([]byte, 16)
		rand.Read(b)
		n := hex.EncodeToString(b)
		f.nonces[n] = true
		json.NewEncoder(w).Encode(map[string]string{"nonce": n})
		return
	}
	pub, ok := f.prove(w, r, body)
	if !ok {
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/alerts/address":
		json.NewEncoder(w).Encode(map[string]string{"address": fakeAddress})
	case r.Method == http.MethodGet && r.URL.Path == "/api/alerts":
		since := 0
		if s := r.URL.Query().Get("since"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil {
				f.refuse(w, 400, "", "since is no whole number")
				return
			}
			since = n
		}
		list := []map[string]any{}
		next := since
		for _, a := range f.alerts {
			seq := int(a["seq"].(float64))
			if seq > since {
				list = append(list, a)
				next = seq
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"alerts": list, "next": next})
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/alerts/") && strings.HasSuffix(r.URL.Path, "/notes"):
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/alerts/"), "/notes")
		pdf, ok := f.notes[id]
		if !ok {
			f.refuse(w, 404, "", "no such notes")
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(pdf)
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/alerts/"):
		id := strings.TrimPrefix(r.URL.Path, "/api/alerts/")
		if f.onClaim != nil {
			f.onClaim(id)
		}
		for i, a := range f.alerts {
			if a["id"] == id {
				f.alerts = append(f.alerts[:i], f.alerts[i+1:]...)
				f.claimed = append(f.claimed, id)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		f.refuse(w, 404, "", "no such alert")
	default:
		f.t.Errorf("the tool asked %s %s (key %s)", r.Method, r.RequestURI, pub[:12])
		f.refuse(w, 404, "", "no such route")
	}
}

// prove checks a Postern2 header: a nonce this fake issued and has not
// spent, a DER signature by the named key over sha256 of the v2 message
// (method, the raw request target, the body's sha256 and the nonce, joined by
// a line feed), and a key that holds a licence.
func (f *fakePostern) prove(w http.ResponseWriter, r *http.Request, body []byte) (string, bool) {
	const scheme = "Postern2 "
	h := r.Header.Get("Authorization")
	parts := strings.Split(strings.TrimPrefix(h, scheme), ":")
	if !strings.HasPrefix(h, scheme) || len(parts) != 3 {
		f.refused++
		f.refuse(w, 401, "malformed_authorization", "no Postern2 proof")
		return "", false
	}
	pub, nonce, sigHex := parts[0], parts[1], parts[2]
	if !f.nonces[nonce] {
		f.refused++
		f.refuse(w, 401, "nonce", "nonce is missing, expired, or already used")
		return "", false
	}
	delete(f.nonces, nonce)
	if f.nonceBad > 0 {
		f.nonceBad--
		f.refused++
		f.refuse(w, 401, "nonce", "nonce is missing, expired, or already used")
		return "", false
	}
	bodySum := sha256.Sum256(body)
	msg := "postern-v2\n" + r.Method + "\n" + r.RequestURI + "\n" + hex.EncodeToString(bodySum[:]) + "\n" + nonce
	pkb, err1 := hex.DecodeString(pub)
	pk, err2 := btcec.ParsePubKey(pkb)
	sb, err3 := hex.DecodeString(sigHex)
	sig, err4 := ecdsa.ParseDERSignature(sb)
	digest := sha256.Sum256([]byte(msg))
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || !sig.Verify(digest[:], pk) {
		f.refused++
		f.refuse(w, 401, "signature", "the signature does not verify")
		return "", false
	}
	f.verified++
	if !f.licensed[pub] {
		f.refuse(w, 401, "no_licence", "this key holds no licence")
		return "", false
	}
	return pub, true
}

// newKey makes a key in a temp dir with the tool's own keygen and returns
// its path and public key.
func newKey(t *testing.T) (path, pub string) {
	t.Helper()
	path = filepath.Join(t.TempDir(), "postern-alerts.key")
	code, out, errOut := cliRun(t, "keygen", "--key", path)
	if code != 0 {
		t.Fatalf("keygen exited %d: %s", code, errOut)
	}
	return path, strings.TrimSpace(out)
}

// rowsFile writes invented YouTube rows for the Sundays 2031-03-02 and
// 2031-03-09.
func rowsFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rows.jsonl")
	rows := `{"id":"v1","title":"One","url":"https://example.com/1","date":"2031-03-02"}` + "\n" +
		`{"id":"v2","title":"Two","url":"https://example.com/2","date":"2031-03-09"}` + "\n"
	if err := os.WriteFile(p, []byte(rows), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// Acceptance 1: against a fake Postern, the tool signs each request and the
// fake verifies it; it asks for the address, fetches the family's alerts,
// writes the records, and claims exactly the alerts it recorded.
func TestFetchSignsEachRequestAndClaimsWhatItRecorded(t *testing.T) {
	f, srv := newFakePostern(t, fixture(t))
	key, pub := newKey(t)
	f.licensed[pub] = true

	code, out, errOut := cliRun(t, "address", "--key", key, "--postern", srv.URL)
	if code != 0 {
		t.Fatalf("address exited %d: %s", code, errOut)
	}
	if out != fakeAddress+"\n" {
		t.Errorf("address printed %q", out)
	}

	records := filepath.Join(t.TempDir(), "data", "subsplash-notes.jsonl")
	code, out, errOut = cliRun(t, "fetch", "--key", key, "--postern", srv.URL, "--rows", rowsFile(t), "--out", records)
	if code != 0 {
		t.Fatalf("fetch exited %d: %s", code, errOut)
	}
	got, err := os.ReadFile(records)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"v1","date":"2031-03-02","title":"The Garden Gate","ref":"Gen.2.8-Gen.2.17"}` + "\n" +
		`{"id":"v2","date":"2031-03-09","title":"Walking the Hill","ref":"Ps.23"}` + "\n"
	if string(got) != want {
		t.Errorf("records:\n%s\nwant:\n%s", got, want)
	}
	if strings.Join(f.claimed, ",") != "00000000000000a1,00000000000000a2" {
		t.Errorf("claimed %v, want a1 and a2 only (a3 is a charge, a4 and a5 were dropped)", f.claimed)
	}
	// One address, one list, two claims: four signed requests, each verified.
	if f.verified != 4 || f.refused != 0 {
		t.Errorf("the fake verified %d signed requests and refused %d; want 4 and 0", f.verified, f.refused)
	}
	if !strings.Contains(out, "2 claimed") {
		t.Errorf("stdout does not count the claims: %q", out)
	}
	for _, id := range []string{"00000000000000a4", "00000000000000a5"} {
		if !strings.Contains(errOut, id) {
			t.Errorf("stderr does not report %s: %s", id, errOut)
		}
	}

	// A second fetch finds the dropped alerts still there, claims nothing
	// more and leaves the records byte for byte.
	f.verified = 0
	code, _, errOut = cliRun(t, "fetch", "--key", key, "--postern", srv.URL, "--rows", rowsFile(t), "--out", records)
	if code != 0 {
		t.Fatalf("a second fetch exited %d: %s", code, errOut)
	}
	again, _ := os.ReadFile(records)
	if string(again) != want || len(f.claimed) != 2 || f.verified != 1 {
		t.Errorf("a second fetch changed something: claimed %v, %d verified, records:\n%s", f.claimed, f.verified, again)
	}
}

// The fake really checks the signature: a proof signed for one request is
// refused on another, so the passing test above means something.
func TestTheFakeRefusesAProofForAnotherRequest(t *testing.T) {
	f, srv := newFakePostern(t, fixture(t))
	keyPath, pub := newKey(t)
	f.licensed[pub] = true
	key, err := LoadKey(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	c := &Client{Base: srv.URL, Key: key, HTTP: srv.Client()}
	nonce, err := c.challenge(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	header := key.authorization(http.MethodGet, "/api/alerts?since=0", nil, nonce)
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/alerts?since=1", nil)
	req.Header.Set("Authorization", header)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 401 || f.refused != 1 {
		t.Errorf("a proof for another target answered %d (%d refused)", resp.StatusCode, f.refused)
	}
}

// Acceptance 2: an alert is claimed only after its record is on disk; a
// write failure leaves every alert unclaimed. The records' directory is
// read-only, so the alerts are fetched and matched and only the write fails:
// with no file yet, and with one already there.
func TestAWriteFailureLeavesTheAlertsUnclaimed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes into a read-only directory")
	}
	f, srv := newFakePostern(t, fixture(t))
	key, pub := newKey(t)
	f.licensed[pub] = true
	for _, kept := range []string{"", `{"id":"v0","date":"2031-02-23","title":"Kept","ref":"Ps.1"}` + "\n"} {
		dir := t.TempDir()
		out := filepath.Join(dir, "subsplash-notes.jsonl")
		if kept != "" {
			if err := os.WriteFile(out, []byte(kept), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Chmod(dir, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(dir, 0o755) })
		f.verified = 0
		code, out1, errOut := cliRun(t, "fetch", "--key", key, "--postern", srv.URL, "--rows", rowsFile(t), "--out", out)
		if f.verified != 1 {
			t.Fatalf("the fetch made %d verified requests before failing, not the one list: the write was never reached", f.verified)
		}
		if code == 0 || out1 != "" {
			t.Errorf("a fetch whose write failed exited %d, printed %q", code, out1)
		}
		if !strings.Contains(errOut, "no alert claimed") {
			t.Errorf("the failure does not say no alert was claimed: %q", errOut)
		}
		if len(f.claimed) != 0 {
			t.Fatalf("alerts %v were claimed though their records were not written", f.claimed)
		}
		if got, _ := os.ReadFile(out); string(got) != kept {
			t.Errorf("the records file changed to %q", got)
		}
		want := 0
		if kept != "" {
			want = 1
		}
		if entries, _ := os.ReadDir(dir); len(entries) != want {
			t.Errorf("a failed write left files behind: %v", entries)
		}
	}
	if len(f.alerts) != 5 {
		t.Errorf("the fake holds %d alerts, not the 5 it began with", len(f.alerts))
	}
}

// Acceptance 3: a 401 no_licence is reported in one line and nothing is
// written.
func TestNoLicenceIsOneLineAndWritesNothing(t *testing.T) {
	f, srv := newFakePostern(t, fixture(t))
	key, pub := newKey(t) // not licensed
	records := filepath.Join(t.TempDir(), "data", "subsplash-notes.jsonl")
	for _, args := range [][]string{
		{"address", "--key", key, "--postern", srv.URL},
		{"fetch", "--key", key, "--postern", srv.URL, "--rows", rowsFile(t), "--out", records},
	} {
		code, out, errOut := cliRun(t, args...)
		if code == 0 {
			t.Errorf("%s exited 0 without a licence", args[0])
		}
		if out != "" {
			t.Errorf("%s printed %q", args[0], out)
		}
		if strings.Count(errOut, "\n") != 1 || !strings.HasSuffix(errOut, "\n") {
			t.Errorf("%s reported %q, not one line", args[0], errOut)
		}
		if !strings.Contains(errOut, "no_licence") || !strings.Contains(errOut, pub) {
			t.Errorf("%s's line does not name no_licence and the public key to licence: %q", args[0], errOut)
		}
	}
	if _, err := os.Stat(filepath.Dir(records)); !os.IsNotExist(err) {
		t.Errorf("a refused fetch wrote something: %v", err)
	}
	if len(f.claimed) != 0 {
		t.Errorf("a refused key claimed %v", f.claimed)
	}
}

// A nonce refusal is retried once with a fresh challenge, as Postern asks.
func TestANonceRefusalIsRetriedOnce(t *testing.T) {
	f, srv := newFakePostern(t, fixture(t))
	key, pub := newKey(t)
	f.licensed[pub] = true
	f.nonceBad = 1
	code, out, errOut := cliRun(t, "address", "--key", key, "--postern", srv.URL)
	if code != 0 || out != fakeAddress+"\n" {
		t.Errorf("address after one nonce refusal: exit %d, %q, %s", code, out, errOut)
	}
	f.nonceBad = 2
	if code, _, errOut := cliRun(t, "address", "--key", key, "--postern", srv.URL); code == 0 || !strings.Contains(errOut, "nonce") {
		t.Errorf("two nonce refusals: exit %d, %s", code, errOut)
	}
}

// With no Postern named, the tool says so and asks nothing.
func TestNoPosternNamed(t *testing.T) {
	t.Setenv("POSTERN_URL", "")
	key, _ := newKey(t)
	code, _, errOut := cliRun(t, "address", "--key", key)
	if code == 0 || !strings.Contains(errOut, "--postern") {
		t.Errorf("exit %d: %s", code, errOut)
	}
}

// Acceptance 6: no private key appears in stdout, stderr or any log line,
// nor on the wire, through keygen, address, fetch and every refusal.
func TestNoPrivateKeyInAnyOutput(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	f, srv := newFakePostern(t, fixture(t))
	var printed bytes.Buffer
	keyPath := filepath.Join(t.TempDir(), "postern-alerts.key")
	step := func(args ...string) {
		_, out, errOut := cliRun(t, args...)
		printed.WriteString(out + errOut)
	}
	step("keygen", "--key", keyPath)
	step("keygen", "--key", keyPath) // refused
	key, err := LoadKey(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	records := filepath.Join(t.TempDir(), "out.jsonl")
	step("address", "--key", keyPath, "--postern", srv.URL)                                        // no licence
	step("fetch", "--key", keyPath, "--postern", srv.URL, "--rows", rowsFile(t), "--out", records) // no licence
	f.licensed[key.PublicHex()] = true
	step("address", "--key", keyPath, "--postern", srv.URL)
	step("fetch", "--key", keyPath, "--postern", srv.URL, "--rows", rowsFile(t), "--out", records)
	step("fetch", "--key", keyPath, "--postern", srv.URL, "--rows", rowsFile(t), "--out", t.TempDir()) // write fails
	step("address", "--key", keyPath, "--postern", "http://127.0.0.1:1")                               // unreachable
	os.Chmod(keyPath, 0o644)
	step("address", "--key", keyPath, "--postern", srv.URL) // the key's mode refused
	os.Chmod(keyPath, 0o600)

	priv := privateHex(t, keyPath)
	if printed.Len() == 0 || f.verified == 0 {
		t.Fatal("the steps printed nothing or reached nothing: the test proves nothing")
	}
	for name, s := range map[string]string{"stdout and stderr": printed.String(), "the log": logged.String(), "the wire": string(f.wire)} {
		if containsSecret(s, priv) {
			t.Errorf("the private key appears in %s", name)
		}
	}
}
