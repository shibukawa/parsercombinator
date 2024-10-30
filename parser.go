package parsercombinator

import (
	"errors"
)

type Parser[T any] func(*ParseContext[T], []Token[T]) (consumed int, newTokens []Token[T], err error)

func Evaluate[T any](pctx *ParseContext[T], src []Token[T], parser Parser[T]) (result []T, err error) {
	pctx.Tokens = src
	pctx.Pos = 0
	pctx.Traces = make([]*TraceInfo, 0)
	pctx.Errors = make([]*ParseError, 0)
	pctx.Depth = 0
	consumed, newTokens, err := parser(pctx, src)
	if err != nil {
		var pos *Pos
		if len(src) > 0 {
			pos = src[0].Pos
		}
		pctx.AppendError(err, pos)
	}
	pctx.Pos = consumed
	pctx.Results = newTokens
	pctx.RemainedTokens = pctx.Tokens[consumed:]

	if len(pctx.Errors) > 0 {
		var errs []error
		for _, e := range pctx.Errors {
			errs = append(errs, e)
		}
		return nil, errors.Join(errs...)
	}

	result = make([]T, len(newTokens))
	for i, t := range newTokens {
		result[i] = t.Val
	}

	return result, nil
}

func EvaluateWithRawTokens[T any](pc *ParseContext[T], src []string, parser Parser[T]) (result []T, err error) {
	tokens := make([]Token[T], len(src))
	for i, rt := range src {
		tokens[i] = Token[T]{Type: "raw", Pos: &Pos{Index: i}, Raw: rt}
	}
	return Evaluate(pc, tokens, parser)
}
