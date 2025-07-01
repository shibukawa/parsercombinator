package parsercombinator

import (
	"fmt"
)

type ParseError struct {
	Parent error
	Pos    *Pos
}

func (e ParseError) Error() string {
	if e.Pos != nil {
		return fmt.Sprintf("%s at %s", e.Parent.Error(), e.Pos.String())
	} else {
		return e.Parent.Error()
	}
}

func (e ParseError) Unwrap() error {
	return e.Parent
}

var (
	// ErrNotMatch means parser doesn't match structure
	// Repeat, Or ignore this error
	ErrNotMatch = fmt.Errorf("not match")

	// ErrRepeatCount means repeat count miss matching.
	// Repeat, Or don't ignore this error
	ErrRepeatCount = fmt.Errorf("repeat count")

	// ErrCritical means critical error
	//
	// It is not just structure error that doesn't have extra options.
	// Repeat, Or don't ignore this error
	ErrCritical = fmt.Errorf("critical error")

	// ErrStackOverflow means the parser recursion depth exceeded the maximum limit
	// This prevents infinite loops in recursive parsers
	ErrStackOverflow = fmt.Errorf("stack overflow")
)

func NewErrNotMatch(expected, actual string, pos *Pos) error {
	var err error
	if actual != "" {
		err = fmt.Errorf("%w expected: %s, actual: %s", ErrNotMatch, expected, actual)
	} else {
		err = fmt.Errorf("%w expected: %s, but not", ErrNotMatch, expected)
	}
	return &ParseError{
		Parent: err,
		Pos:    pos,
	}
}

func NewErrRepeatCount(label string, expected, actual int, pos *Pos) error {
	return &ParseError{
		Parent: fmt.Errorf("%w expected count: %d, actual count: %d", ErrRepeatCount, expected, actual),
		Pos:    pos,
	}
}

func NewErrCritical(message string, pos *Pos) error {
	return &ParseError{
		Parent: fmt.Errorf("%w: %s", ErrCritical, message),
		Pos:    pos,
	}
}

func NewErrStackOverflow(currentDepth, maxDepth int, pos *Pos) error {
	return &ParseError{
		Parent: fmt.Errorf("%w: recursion depth %d exceeded maximum %d", ErrStackOverflow, currentDepth, maxDepth),
		Pos:    pos,
	}
}
