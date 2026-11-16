package main

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeChromium is the DevTools side of a tab: /json lists one page and the
// WebSocket answers the few commands CDP sends. status is the HTTP status it
// reports for the document it "loads".
type fakeChromium struct {
	srv      *httptest.Server
	status   int
	page     string // the snapshot JSON the page evaluates to
	commands []string
}

func (f *fakeChromium) start(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `[{"id":"T1","type":"page","url":"https://subsplash.com/brookfieldfirst/media","webSocketDebuggerUrl":"ws://%s/devtools/page/T1"}]`, r.Host)
	})
	mux.HandleFunc("/devtools/page/T1", func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("Sec-WebSocket-Key")
		sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		conn, rw, _ := w.(http.Hijacker).Hijack()
		defer conn.Close()
		fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", base64.StdEncoding.EncodeToString(sum[:]))
		rw.Flush()
		for {
			msg, err := readClientFrame(rw.Reader)
			if err != nil {
				return
			}
			var req struct {
				ID     int            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			json.Unmarshal(msg, &req)
			f.commands = append(f.commands, req.Method)
			reply := fmt.Sprintf(`{"id":%d,"result":{}}`, req.ID)
			switch req.Method {
			case "Page.navigate":
				// An event first, as Chromium does, then the reply.
				writeServerFrame(conn, fmt.Sprintf(`{"method":"Network.responseReceived","params":{"type":"Document","response":{"url":"https://subsplash.com/brookfieldfirst/media","status":%d}}}`, f.status))
			case "Runtime.evaluate":
				v, _ := json.Marshal(f.page)
				reply = fmt.Sprintf(`{"id":%d,"result":{"result":{"type":"string","value":%s}}}`, req.ID, v)
			}
			writeServerFrame(conn, reply)
		}
	})
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
}

func readClientFrame(r *bufio.Reader) ([]byte, error) {
	var h [2]byte
	if _, err := io.ReadFull(r, h[:]); err != nil {
		return nil, err
	}
	n := int(h[1] & 0x7f)
	switch n {
	case 126:
		var b [2]byte
		io.ReadFull(r, b[:])
		n = int(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		io.ReadFull(r, b[:])
		n = int(binary.BigEndian.Uint64(b[:]))
	}
	var mask [4]byte
	io.ReadFull(r, mask[:])
	p := make([]byte, n)
	if _, err := io.ReadFull(r, p); err != nil {
		return nil, err
	}
	for i := range p {
		p[i] ^= mask[i%4]
	}
	return p, nil
}

// writeServerFrame writes an unmasked text frame, split in two when long to
// exercise continuation frames.
func writeServerFrame(c net.Conn, s string) {
	b := []byte(s)
	frame := func(fin bool, op byte, p []byte) []byte {
		first := op
		if fin {
			first |= 0x80
		}
		out := []byte{first}
		switch n := len(p); {
		case n < 126:
			out = append(out, byte(n))
		case n < 1<<16:
			out = append(out, 126, byte(n>>8), byte(n))
		default:
			out = append(out, 127)
			out = binary.BigEndian.AppendUint64(out, uint64(n))
		}
		return append(out, p...)
	}
	if len(b) > 200 {
		c.Write(frame(false, 1, b[:100]))
		c.Write(frame(true, 0, b[100:]))
		return
	}
	c.Write(frame(true, 1, b))
}

func TestCDPOpenReadsThePageAndItsStatus(t *testing.T) {
	long := strings.Repeat("a long line of the page\n", 4000) // over 64 KB: the 8-byte length form
	f := &fakeChromium{status: 200, page: `{"url":"https://subsplash.com/brookfieldfirst/media","title":"Media","ready":"complete","text":` + mustJSON(long) + `,"links":[{"t":"A Sermon","h":"https://subsplash.com/u/brookfieldfirst/media/d/aaaaaaa-a-sermon"}],"notes":true,"pages":10}`}
	f.start(t)
	c, err := NewCDP(f.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Poll, c.Settle = time.Millisecond, 5*time.Second
	p, err := c.Open(context.Background(), listURL)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status != 200 || p.Text != long || p.Pages != 10 || !p.HasNotes || len(p.Links) != 1 || p.Links[0].Text != "A Sermon" {
		t.Errorf("page: status %d, %d bytes of text, pages %d, links %v", p.Status, len(p.Text), p.Pages, p.Links)
	}
	if f.commands[0] != "Network.enable" || f.commands[1] != "Page.navigate" {
		t.Errorf("commands: %v", f.commands)
	}

	f.status = 429
	p, err = c.Open(context.Background(), listURL)
	if err != nil || p.Status != 429 {
		t.Errorf("status %d, %v: a 429 was not seen", p.Status, err)
	}
	if err := Check(p, KindList); err == nil {
		t.Error("a 429 did not stop")
	}
}

func mustJSON(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestNewCDPRefusesAnotherMachine(t *testing.T) {
	if _, err := NewCDP("http://192.168.1.5:9222"); err == nil || !strings.Contains(err.Error(), "not on this machine") {
		t.Errorf("got %v", err)
	}
}
