package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultNotesDir is where the notes PDFs are kept: on the Laptop, under the
// state directory beside the Brookfield refresh's, outside the checkout. With
// no home directory it is empty, and the directory must be named with --notes.
func DefaultNotesDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "state", "sermon-index", "notes")
}

// NotesStore keeps the notes PDFs of the alerts it records in Dir, one file
// <row id>.pdf each. The PDF holds the church's notes and the Governor's own
// fill-ins: it is never committed, so Dir is refused inside a git work tree.
type NotesStore struct {
	Dir    string
	Client *Client
	Asked  int // downloads made
}

// checkDir refuses an empty directory and one inside a git work tree.
func (s *NotesStore) checkDir() error {
	if s.Dir == "" {
		return errors.New("no notes directory named and no home directory to find it in: give --notes <dir>")
	}
	if tree := insideGitWorkTree(filepath.Join(s.Dir, "x")); tree != "" {
		return fmt.Errorf("the notes directory %s is inside the git work tree %s: the notes are never committed, name a directory outside it with --notes", s.Dir, tree)
	}
	return nil
}

// notReplaced is a refusal to write over a file or a record that is there: the
// tool leaves it as it is, and the message says why and what a person can do.
// merge counts these for the summary line.
type notReplaced struct{ msg string }

func (e notReplaced) Error() string { return e.msg }

// prevName is the file beside <row id>.pdf that holds the notes the newer
// mail replaced.
func prevName(rowID string) string { return rowID + ".prev.pdf" }

// Keep makes sure the PDF of a note is on disk as <rowID>.pdf and has the
// size and sha256 the alert names. A file already there with that sha256 is
// left alone and nothing is downloaded.
//
// A file with another sha256 is the same sermon's notes made again (a second
// mail for it) when same is true, which is when the row's record is the very
// record this note gives: then the newer mail's notes are kept as <rowID>.pdf
// and the older beside them as <rowID>.prev.pdf (replaced says so). A note
// whose sha256 is the one already kept as .prev.pdf is the older mail again
// and changes nothing. When same is false nothing says the file is this
// sermon's, so it is not replaced and the error says what to do. The newer
// PDF is downloaded and checked before any file is moved, then written to a
// temporary file in Dir (0600), synced and renamed into place.
func (s *NotesStore) Keep(ctx context.Context, n Note, rowID string, same bool) (replaced string, err error) {
	if rowID == "" || filepath.Base(rowID) != rowID || rowID == "." || rowID == ".." {
		return "", fmt.Errorf("row id '%s' is no file name", rowID)
	}
	path := filepath.Join(s.Dir, rowID+".pdf")
	prev := filepath.Join(s.Dir, prevName(rowID))
	replacing := false
	if have, err := os.ReadFile(path); err == nil {
		if sha(have) == n.NotesSHA256 {
			return "", nil
		}
		if !same {
			return "", notReplaced{fmt.Sprintf("%s already holds notes with another sha256 and no record kept for this row says this alert is the same sermon; not replaced (if this alert's notes are the right ones, delete that file and the next run keeps them)", path)}
		}
		if old, err := os.ReadFile(prev); err == nil && sha(old) == n.NotesSHA256 {
			return "", nil // the older mail again
		}
		replacing = true
	} else if !os.IsNotExist(err) {
		return "", err
	}
	s.Asked++
	b, err := s.Client.Notes(ctx, n.AlertID)
	if err != nil {
		return "", fmt.Errorf("notes not downloaded: %w", err)
	}
	if len(b) != n.NotesSize {
		return "", fmt.Errorf("notes are %d bytes, the alert says notes_size %d (size)", len(b), n.NotesSize)
	}
	if got := sha(b); got != n.NotesSHA256 {
		return "", fmt.Errorf("notes have sha256 %s, the alert says %s (sha256)", got, n.NotesSHA256)
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(s.Dir, ".notes-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	_, err = tmp.Write(b)
	if err == nil {
		err = tmp.Chmod(0o600)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil && replacing {
		err = os.Rename(path, prev)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return "", fmt.Errorf("writing the notes to %s: %w", path, err)
	}
	if d, err := os.Open(s.Dir); err == nil {
		d.Sync()
		d.Close()
	}
	if replacing {
		return fmt.Sprintf("a second mail for this sermon: its notes are kept in %s and the older beside them in %s", path, prev), nil
	}
	return "", nil
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
