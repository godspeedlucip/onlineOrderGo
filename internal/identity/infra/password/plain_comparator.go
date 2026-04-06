package password

import "errors"

type PlainComparator struct{}

func NewPlainComparator() *PlainComparator { return &PlainComparator{} }

func (p *PlainComparator) Compare(hashed string, plain string) error {
	// TODO: replace with Java-compatible password hashing strategy.
	if hashed != plain {
		return errors.New("password mismatch")
	}
	return nil
}