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

// Keep makes sure the PDF of a note is on disk as <rowID>.pdf and has the
// size and sha256 the alert names. A file already there with that sha256 is
// left alone and nothing is downloaded; one with another is an error and is
// not replaced. Otherwise the PDF is downloaded, checked, and written to a
// temporary file in Dir (0600), synced and renamed into place.
func (s *NotesStore) Keep(ctx context.Context, n Note, rowID string) error {
	if rowID == "" || filepath.Base(rowID) != rowID || rowID == "." || rowID == ".." {
		return fmt.Errorf("row id '%s' is no file name", rowID)
	}
	path := filepath.Join(s.Dir, rowID+".pdf")
	if have, err := os.ReadFile(path); err == nil {
		if sha(have) == n.NotesSHA256 {
			return nil
		}
		return fmt.Errorf("%s already holds notes with another sha256; not replaced", path)
	} else if !os.IsNotExist(err) {
		return err
	}
	s.Asked++
	b, err := s.Client.Notes(ctx, n.AlertID)
	if err != nil {
		return fmt.Errorf("notes not downloaded: %w", err)
	}
	if len(b) != n.NotesSize {
		return fmt.Errorf("notes are %d bytes, the alert says notes_size %d (size)", len(b), n.NotesSize)
	}
	if got := sha(b); got != n.NotesSHA256 {
		return fmt.Errorf("notes have sha256 %s, the alert says %s (sha256)", got, n.NotesSHA256)
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, ".notes-*")
	if err != nil {
		return err
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
	if err == nil {
		err = os.Rename(tmp.Name(), path)
	}
	if err != nil {
		return fmt.Errorf("writing the notes to %s: %w", path, err)
	}
	if d, err := os.Open(s.Dir); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
