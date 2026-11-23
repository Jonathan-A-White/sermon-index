package passage

// SamePlace is whether two references name the same passage for a reader
// matching a mail's passage to a row's: they share a verse, so a chapter and a
// verse range inside it (either way round), or two equal references, are one
// place; another chapter or another book is not. A reference that does not
// parse is the same as nothing.
func SamePlace(a, b string) bool {
	ra, err := Parse(a)
	if err != nil {
		return false
	}
	rb, err := Parse(b)
	if err != nil {
		return false
	}
	return ra.Lo <= rb.Hi && rb.Lo <= ra.Hi
}
