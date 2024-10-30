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
		if pctx.TraceEnable {
			if len(tokens) > 0 {
				pos = tokens[0].Pos
			}
			pctx.Traces = append(pctx.Traces, &TraceInfo{
				TraceType: Enter,
				Depth:     pctx.Depth,
				Name:      name,
				Pos:       pos,
			})
			pctx.Depth++
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
			pctx.Depth--
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
					Depth:     pctx.Depth,
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
			if offset >= len(src) {
				if len(src) > 0 {
					return 0, nil, NewErrNotMatch(label, "end of tokens", src[0].Pos)
				}
				return 0, nil, NewErrNotMatch(label, "end of tokens", nil)
			}
			consumed, newTokens, err := p(pctx, src[offset:])
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
		offset := 0
		for _, p := range parsers {
			consumed, newTokens, err := p(pctx, src[offset:])

			// Process recover
			offset += consumed

			if err == nil { // match
				return offset, newTokens, nil
			}

			// not match
			// try other options, because it is not critical error
			if errors.Is(err, ErrNotMatch) || errors.Is(err, ErrRepeatCount) {
				allError = append(allError, err)
				continue
			}
			// critical error
			return offset, nil, err
		}
		return offset, nil, &ParseError{
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
