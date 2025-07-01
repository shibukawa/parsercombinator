package parsercombinator

import (
	"errors"
	"fmt"
	"strings"
)

type Alias[T any] struct {
	body Parser[T]
	name string
}

func NewAlias[T any](name string) (instance func(Parser[T]) Parser[T], alias Parser[T]) {
	i := &Alias[T]{}
	alias = func(pctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		return Trace(name+"-alias", i.body)(pctx, tokens)
	}
	instance = i.define
	return
}

func (a *Alias[T]) define(alias Parser[T]) Parser[T] {
	a.body = alias

	return func(pctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		return Trace(a.name+"-instance", a.body)(pctx, tokens)
	}
}

func Trace[T any](name string, p Parser[T]) Parser[T] {
	return func(pctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		var pos *Pos
		if len(tokens) > 0 {
			pos = tokens[0].Pos
		}

		// Check stack depth before proceeding
		if err := pctx.CheckDepthAndIncrement(pos); err != nil {
			return 0, nil, err
		}
		defer pctx.DecrementDepth()

		if pctx.TraceEnable {
			pctx.Traces = append(pctx.Traces, &TraceInfo{
				TraceType: Enter,
				Depth:     pctx.Depth - 1, // Use actual depth for display
				Name:      name,
				Pos:       pos,
			})
		}
		traceIndex := len(pctx.Traces)
		consumed, newTokens, err := p(pctx, tokens)
		if pctx.TraceEnable {
			tt := Match
			if err != nil {
				tt = NotMatch
			}
			var result string
			if err != nil {
				result = err.Error()
			} else {
				builder := strings.Builder{}
				builder.WriteString("[")
				for i, t := range newTokens {
					if i != 0 {
						builder.WriteString(", ")
					}
					fmt.Fprintf(&builder, "%#v", t.Val)
				}
				builder.WriteString("]")
				result = builder.String()
			}
			if len(pctx.Traces) == traceIndex {
				lastTrace := pctx.Traces[len(pctx.Traces)-1]
				if tt == NotMatch {
					lastTrace.TraceType = EnterNotMatch
				} else {
					lastTrace.TraceType = EnterMatch
				}
				lastTrace.Result = result
			} else {
				pctx.Traces = append(pctx.Traces, &TraceInfo{
					TraceType: tt,
					Depth:     pctx.Depth - 1, // Use actual depth for display
					Name:      name,
					Pos:       pos,
					Result:    result,
				})
			}
		}
		return consumed, newTokens, err
	}
}

type Transformer[T any] func(pctx *ParseContext[T], src []Token[T]) (converted []Token[T], err error)

//func Log(depth int, data string) string {
//	pc, file, line, _ := runtime.Caller(depth)
//	f := runtime.FuncForPC(pc)
//	return fmt.Sprintf("call:%s data:%s file:%s:%d", f.Name(), data, file, line)
//}

func Seq[T any](parsers ...Parser[T]) Parser[T] {
	return SeqWithLabel("seq", parsers...)
}

func SeqWithLabel[T any](label string, parsers ...Parser[T]) Parser[T] {
	//var origin = Log(3, "🐙")
	return Trace(label, func(pctx *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		converted := make([]Token[T], 0, len(parsers))
		offset := 0
		for _, p := range parsers {
			// パーサーを実行する（入力が空でもZeroOrMoreやOptionalは実行される可能性がある）
			var currentSrc []Token[T]
			if offset < len(src) {
				currentSrc = src[offset:]
			} else {
				// 入力が空でもパーサーを実行する（空のスライスを渡す）
				currentSrc = []Token[T]{}
			}

			consumed, newTokens, err := p(pctx, currentSrc)
			if err != nil {
				return 0, src, err
			}
			converted = append(converted, newTokens...)
			offset += consumed
		}
		return offset, converted, nil
	})
}

func Or[T any](parsers ...Parser[T]) Parser[T] {
	return Trace("or", func(pctx *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		var allError []error
		for _, p := range parsers {
			consumed, newTokens, err := p(pctx, src)

			if err == nil { // match
				return consumed, newTokens, nil
			}

			// not match
			// try other options, because it is not critical error
			if errors.Is(err, ErrNotMatch) || errors.Is(err, ErrRepeatCount) {
				allError = append(allError, err)
				continue
			}
			// critical error
			return consumed, nil, err
		}
		return 0, nil, &ParseError{
			Parent: errors.Join(allError...), Pos: src[0].Pos,
		}
	})
}

func Trans[T any](parser Parser[T], tf Transformer[T]) Parser[T] {
	return func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		consumed, newTokens, err := parser(pc, src)
		if err != nil {
			return 0, src, err
		}
		result, err := tf(pc, newTokens)
		if err != nil {
			return 0, src, err
		}
		return consumed, result, nil
	}
}

func Repeat[T any](label string, min uint, max int, parser Parser[T]) Parser[T] {
	return Trace(label, func(pctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		converted := make([]Token[T], 0, len(tokens))
		offset := 0
		var i int
		for i = 0; i < max || max == -1; i++ {
			if offset >= len(tokens) {
				break
			}
			consumed, newTokens, err := parser(pctx, tokens[offset:])
			if errors.Is(err, ErrNotMatch) {
				break
			} else if err != nil {
				return 0, []Token[T]{}, err
			}
			converted = append(converted, newTokens...)
			offset += consumed
		}
		if i < int(min) {
			var pos *Pos
			if len(tokens) > 0 {
				pos = tokens[0].Pos
			}
			return 0, tokens, NewErrRepeatCount(label, int(min), i, pos)
		}
		return offset, converted, nil
	})
}

func OneOrMore[T any](label string, parser Parser[T]) Parser[T] {
	return Repeat(label, 1, -1, parser)
}

func ZeroOrMore[T any](label string, parser Parser[T]) Parser[T] {
	return Repeat(label, 0, -1, parser)
}

func Optional[T any](parser Parser[T]) Parser[T] {
	return Trace("optional", func(pctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		consumed, newTokens, err := parser(pctx, tokens)
		if err == nil {
			return consumed, newTokens, nil
		}
		if errors.Is(err, ErrNotMatch) || errors.Is(err, ErrRepeatCount) {
			return 0, []Token[T]{}, nil
		}
		return 0, []Token[T]{}, err
	})
}

func Before[T any](callback func(token Token[T]) bool) Parser[T] {
	return Trace("before", func(pctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		for i, t := range tokens {
			if callback(t) {
				return i, tokens[:i], nil
			}
		}
		return len(tokens), tokens, nil
	})
}

func None[T any](label ...string) Parser[T] {
	none := func(pctx *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		return 0, nil, nil
	}
	if len(label) > 0 {
		return Trace(label[0], none)
	}
	return none
}

func Recover[T any](search, body, skipUntil Parser[T]) Parser[T] {
	return Trace("recover", func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		_, _, err := Trace("precondition-check", search)(pc, src)
		if err != nil {
			return 0, nil, err
		}
		consumed, newTokens, err := Trace("process", body)(pc, src)
		if err != nil {
			pc.AppendError(err, src[0].Pos)
			return Trace("healing", func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
				for i := range src {
					consumed, _, err = skipUntil(pc, src[i:])
					if err == nil {
						return i + consumed, nil, nil
					}
				}
				return len(src), nil, nil
			})(pc, src)
		}
		return consumed, newTokens, nil
	})
}

// Lookahead checks if the parser matches without consuming tokens
// Returns empty tokens if match, error if not match
func Lookahead[T any](parser Parser[T]) Parser[T] {
	return Trace("lookahead", func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		_, _, err := parser(pc, src)
		if err != nil {
			return 0, nil, err
		}
		return 0, []Token[T]{}, nil
	})
}

// NotFollowedBy succeeds if the parser does NOT match (negative lookahead)
// Returns empty tokens if parser fails, error if parser succeeds
func NotFollowedBy[T any](parser Parser[T]) Parser[T] {
	return Trace("not-followed-by", func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		_, _, err := parser(pc, src)
		if err == nil {
			var pos *Pos
			if len(src) > 0 {
				pos = src[0].Pos
			}
			return 0, nil, NewErrNotMatch("not followed by", "matched", pos)
		}
		return 0, []Token[T]{}, nil
	})
}

// Peek returns the result of the parser without consuming tokens
// Useful for inspection or conditional parsing
func Peek[T any](parser Parser[T]) Parser[T] {
	return Trace("peek", func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		_, newTokens, err := parser(pc, src)
		if err != nil {
			return 0, nil, err
		}
		return 0, newTokens, nil
	})
}

// FollowedBy is an alias for Lookahead for better readability
func FollowedBy[T any](parser Parser[T]) Parser[T] {
	return Lookahead(parser)
}

// Label provides a user-friendly label for error messages
// When the parser fails, it replaces technical error details with the provided label
// Unlike Trace, this is purely for error message improvement, not debugging
func Label[T any](label string, parser Parser[T]) Parser[T] {
	return func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		consumed, newTokens, err := parser(pc, src)
		if err != nil {
			var pos *Pos
			if len(src) > 0 {
				pos = src[0].Pos
			}
			return consumed, nil, NewErrNotMatch(label, "not matched", pos)
		}
		return consumed, newTokens, nil
	}
}

// Expected creates a parser that fails with a specific expected message
// Useful for creating custom error messages or placeholders
func Expected[T any](message string) Parser[T] {
	return func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		var pos *Pos
		if len(src) > 0 {
			pos = src[0].Pos
		}
		return 0, nil, NewErrNotMatch(message, "found something else", pos)
	}
}

// Fail always fails with the given message
// Useful for debugging or creating conditional failures
func Fail[T any](message string) Parser[T] {
	return func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		var pos *Pos
		if len(src) > 0 {
			pos = src[0].Pos
		}
		return 0, nil, NewErrCritical(message, pos)
	}
}
