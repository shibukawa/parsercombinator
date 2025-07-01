package parsercombinator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type TraceType int

const (
	Enter TraceType = iota
	Match
	EnterMatch
	NotMatch
	EnterNotMatch
)

func (tp TraceType) String() string {
	switch tp {
	case Enter:
		return ">"
	case Match:
		return "<"
	case EnterMatch:
		return "="
	case NotMatch:
		return "!"
	case EnterNotMatch:
		return "!"
	}
	return ""
}

type Pos struct {
	Line   int
	Col    int
	Index  int
	Length int
}

func (p *Pos) String() string {
	if p == nil {
		return "1:1"
	}
	if p.Line == 0 && p.Col == 0 {
		return strconv.Itoa(p.Index)
	}
	var result []byte
	result = strconv.AppendInt(result, int64(p.Line), 10)
	result = append(result, ':')
	result = strconv.AppendInt(result, int64(p.Col), 10)
	return string(result)
}

func (p Pos) Copy() *Pos {
	return &Pos{Line: p.Line, Col: p.Col, Index: p.Index, Length: p.Length}
}

type Token[T any] struct {
	Type string
	Pos  *Pos
	Raw  string
	Val  T
}

func (t Token[T]) GoString() string {
	var raw string
	if t.Raw != "" {
		raw = " raw='" + t.Raw + "'"
	}
	return fmt.Sprintf("{%s at %s%s val: %#v}", t.Type, t.Pos, raw, t.Val)
}

type Tokens[T any] []*Token[T]

type TraceInfo struct {
	TraceType TraceType
	Depth     int
	Name      string
	Pos       *Pos
	Result    string
}

type ParseContext[T any] struct {
	Tokens         []Token[T]
	Pos            int
	RemainedTokens []Token[T]
	Results        []Token[T]
	Traces         []*TraceInfo
	Errors         []*ParseError
	Depth          int
	TraceEnable    bool
	MaxDepth       int // Maximum allowed recursion depth (0 means no limit)
	OrMode         OrMode // Or parser behavior mode (default: OrModeSafe)
}

func (pc *ParseContext[T]) AppendError(err error, pos *Pos) error {
	if pe, ok := err.(*ParseError); !ok {
		err := &ParseError{Parent: err, Pos: pos}
		pc.Errors = append(pc.Errors, err)
		return err
	} else {
		pc.Errors = append(pc.Errors, pe)
		return pe
	}
}

func (pc ParseContext[T]) GetError() error {
	if len(pc.Errors) == 0 {
		return nil
	}
	var errorList []error
	for _, e := range pc.Errors {
		errorList = append(errorList, e)
	}
	return errors.Join(errorList...)
}

func (pc *ParseContext[T]) DumpTrace() {
	pc.DumpTraceTo(os.Stdout)
}

func (pc *ParseContext[T]) DumpTraceAsText() string {
	buffer := strings.Builder{}
	pc.DumpTraceTo(&buffer)
	return buffer.String()
}

func (pc *ParseContext[T]) DumpTraceTo(w io.Writer) {
	for _, t := range pc.Traces {
		switch t.TraceType {
		case Enter:
			fmt.Fprintf(w, "%s%s %s at %s\n", strings.Repeat("  ", t.Depth), Enter, t.Name, t.Pos.String())
		case EnterMatch:
			fmt.Fprintf(w, "%s%s %s at %s -> %s\n", strings.Repeat("  ", t.Depth), Enter, t.Name, t.Pos.String(), t.Result)
		case Match:
			fmt.Fprintf(w, "%s%s %s => %#v\n", strings.Repeat("  ", t.Depth), Match, t.Name, t.Result)
		case NotMatch:
			fmt.Fprintf(w, "%s%s %s => %#v\n", strings.Repeat("  ", t.Depth), NotMatch, t.Name, t.Result)
		}
	}
}

func NewParseContext[T any]() *ParseContext[T] {
	return &ParseContext[T]{
		MaxDepth: 1000, // Default maximum depth limit
		OrMode:   OrModeSafe, // Default Or parser behavior mode
	}
}

func (pc *ParseContext[T]) CheckDepthAndIncrement(pos *Pos) error {
	if pc.MaxDepth > 0 && pc.Depth >= pc.MaxDepth {
		return NewErrStackOverflow(pc.Depth, pc.MaxDepth, pos)
	}
	pc.Depth++
	return nil
}

func (pc *ParseContext[T]) DecrementDepth() {
	if pc.Depth > 0 {
		pc.Depth--
	}
}

// OrMode defines the behavior of Or parser
type OrMode int

const (
	// OrModeSafe (default) - Uses longest match logic for consistent and safe behavior
	OrModeSafe OrMode = iota
	// OrModeFast - Uses first match logic for better performance
	OrModeFast
	// OrModeTryFast - Uses first match but warns if longest match would choose differently
	OrModeTryFast
)

func (om OrMode) String() string {
	switch om {
	case OrModeSafe:
		return "Safe"
	case OrModeFast:
		return "Fast"
	case OrModeTryFast:
		return "TryFast"
	default:
		return "Unknown"
	}
}
