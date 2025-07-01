package parsercombinator

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"
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

		switch pctx.OrMode {
		case OrModeFast:
			return orFast(pctx, src, parsers, allError)
		case OrModeTryFast:
			return orTryFast(pctx, src, parsers, allError)
		default: // OrModeSafe
			return orSafe(pctx, src, parsers, allError)
		}
	})
}

// orSafe implements longest match logic (default, safe behavior)
func orSafe[T any](pctx *ParseContext[T], src []Token[T], parsers []Parser[T], allError []error) (int, []Token[T], error) {
	var bestResult struct {
		consumed  int
		newTokens []Token[T]
		hasResult bool
	}

	for _, p := range parsers {
		consumed, newTokens, err := p(pctx, src)

		if err == nil { // match
			// Always choose the parser that consumes the most tokens (longest match)
			if !bestResult.hasResult || consumed > bestResult.consumed {
				bestResult.consumed = consumed
				bestResult.newTokens = newTokens
				bestResult.hasResult = true
			}
			continue
		}

		// not match - try other options for non-critical errors
		if errors.Is(err, ErrNotMatch) || errors.Is(err, ErrRepeatCount) {
			allError = append(allError, err)
			continue
		}
		// critical error (including stack overflow)
		return consumed, nil, err
	}

	if bestResult.hasResult {
		return bestResult.consumed, bestResult.newTokens, nil
	}

	return 0, nil, &ParseError{
		Parent: errors.Join(allError...),
		Pos:    getFirstPos(src),
	}
}

// orFast implements first match logic (performance optimized)
func orFast[T any](pctx *ParseContext[T], src []Token[T], parsers []Parser[T], allError []error) (int, []Token[T], error) {
	for _, p := range parsers {
		consumed, newTokens, err := p(pctx, src)

		if err == nil { // match - return immediately (first match)
			return consumed, newTokens, nil
		}

		// not match - try other options for non-critical errors
		if errors.Is(err, ErrNotMatch) || errors.Is(err, ErrRepeatCount) {
			allError = append(allError, err)
			continue
		}
		// critical error (including stack overflow)
		return consumed, nil, err
	}

	return 0, nil, &ParseError{
		Parent: errors.Join(allError...),
		Pos:    getFirstPos(src),
	}
}

// orTryFast implements first match with warnings when longest match would differ
func orTryFast[T any](pctx *ParseContext[T], src []Token[T], parsers []Parser[T], allError []error) (int, []Token[T], error) {
	var firstMatch struct {
		consumed  int
		newTokens []Token[T]
		hasResult bool
		index     int
	}
	var bestMatch struct {
		consumed  int
		newTokens []Token[T]
		hasResult bool
		index     int
	}

	for i, p := range parsers {
		consumed, newTokens, err := p(pctx, src)

		if err == nil { // match
			// Record first match
			if !firstMatch.hasResult {
				firstMatch.consumed = consumed
				firstMatch.newTokens = newTokens
				firstMatch.hasResult = true
				firstMatch.index = i
			}

			// Record best match (longest)
			if !bestMatch.hasResult || consumed > bestMatch.consumed {
				bestMatch.consumed = consumed
				bestMatch.newTokens = newTokens
				bestMatch.hasResult = true
				bestMatch.index = i
			}
			continue
		}

		// not match - try other options for non-critical errors
		if errors.Is(err, ErrNotMatch) || errors.Is(err, ErrRepeatCount) {
			allError = append(allError, err)
			continue
		}
		// critical error (including stack overflow)
		return consumed, nil, err
	}

	if firstMatch.hasResult {
		// Check if longest match would choose differently
		if bestMatch.hasResult && (firstMatch.index != bestMatch.index || firstMatch.consumed != bestMatch.consumed) {
			pos := getFirstPos(src)

			// Get caller information using runtime to find where Or was called
			var location string = "unknown location"
			for i := 1; i < 15; i++ { // Check up to 15 levels
				_, file, line, ok := runtime.Caller(i)
				if ok {
					// Skip our internal files (tools.go, parser.go) to find user code
					if !strings.Contains(file, "tools.go") &&
						!strings.Contains(file, "parser.go") &&
						!strings.Contains(file, "/go/src/") &&
						!strings.Contains(file, "/usr/") {
						// Extract just the filename from the full path
						parts := strings.Split(file, "/")
						filename := parts[len(parts)-1]
						location = fmt.Sprintf("%s:%d", filename, line)
						break
					}
				}
			}

			fmt.Fprintf(os.Stderr, "⚠️  Or parser optimization suggestion at %s (parser position %s):\n", location, pos)
			fmt.Fprintf(os.Stderr, "   Fast mode chose option %d (consumed %d tokens), but longest match would choose option %d (consumed %d tokens).\n",
				firstMatch.index+1, firstMatch.consumed, bestMatch.index+1, bestMatch.consumed)
			fmt.Fprintf(os.Stderr, "   For Fast mode compatibility, consider moving option %d before option %d in your Or(...) call.\n",
				bestMatch.index+1, firstMatch.index+1)
		}
		return firstMatch.consumed, firstMatch.newTokens, nil
	}

	return 0, nil, &ParseError{
		Parent: errors.Join(allError...),
		Pos:    getFirstPos(src),
	}
}

// Helper function to get position from source tokens
func getFirstPos[T any](src []Token[T]) *Pos {
	if len(src) > 0 {
		return src[0].Pos
	}
	return nil
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

		// Check transformation safety in Safe mode if enabled
		if pc.OrMode == OrModeSafe && pc.CheckTransformSafety {
			err := checkTransformSafety(pc, parser, newTokens, result)
			if err != nil {
				// Log warning but don't fail the parse
				fmt.Fprintf(os.Stderr, "Warning: Transformation safety check failed: %v\n", err)
			}
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

// OrWithMode creates an Or parser with specific mode for this instance
func OrWithMode[T any](mode OrMode, parsers ...Parser[T]) Parser[T] {
	return Trace("or", func(pctx *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		var allError []error

		switch mode {
		case OrModeFast:
			return orFast(pctx, src, parsers, allError)
		case OrModeTryFast:
			return orTryFast(pctx, src, parsers, allError)
		default: // OrModeSafe
			return orSafe(pctx, src, parsers, allError)
		}
	})
}

// FastOr creates an Or parser that uses first match (performance optimized)
func FastOr[T any](parsers ...Parser[T]) Parser[T] {
	return OrWithMode(OrModeFast, parsers...)
}

// SafeOr creates an Or parser that uses longest match (safe, default behavior)
func SafeOr[T any](parsers ...Parser[T]) Parser[T] {
	return OrWithMode(OrModeSafe, parsers...)
}

// TryFastOr creates an Or parser that uses first match but warns when longest match differs
func TryFastOr[T any](parsers ...Parser[T]) Parser[T] {
	return OrWithMode(OrModeTryFast, parsers...)
}

// Lazy creates a lazy parser that evaluates the parser function when called
// This prevents infinite loops during parser construction by deferring parser resolution
// until parsing time, allowing for true recursive definitions
func Lazy[T any](parserFactory func() Parser[T]) Parser[T] {
	return Trace("lazy", func(pc *ParseContext[T], src []Token[T]) (int, []Token[T], error) {
		// Get the actual parser when parsing is performed
		parser := parserFactory()
		return parser(pc, src)
	})
}

// checkTransformSafety verifies that a transformation is safe by checking if
// applying the same parser to the transformed tokens would produce the same result.
// This helps detect infinite loops in transformations.
func checkTransformSafety[T any](pc *ParseContext[T], parser Parser[T], originalTokens, transformedTokens []Token[T]) error {
	// Skip check if transformed tokens are empty (common safe case)
	if len(transformedTokens) == 0 {
		return nil
	}

	// Create a new context to avoid side effects on the original context
	testPC := &ParseContext[T]{
		MaxDepth:             pc.MaxDepth,
		OrMode:               pc.OrMode,
		CheckTransformSafety: false, // Disable recursive checks
	}

	// Try to parse the transformed tokens with the same parser
	consumed, reparsedTokens, err := parser(testPC, transformedTokens)
	if err != nil {
		// If the parser fails on transformed tokens, it's likely safe
		return nil
	}

	// Check if the parser consumed all transformed tokens
	if consumed != len(transformedTokens) {
		// Parser didn't consume all tokens, likely safe
		return nil
	}

	// Check if reparsing produces the same result as the transformed tokens
	if reflect.DeepEqual(transformedTokens, reparsedTokens) {
		// Get caller information for better error reporting
		_, file, line, ok := runtime.Caller(4) // Adjusted to find the actual transformation call
		if ok {
			return fmt.Errorf("potential infinite loop in transformation at %s:%d - parser produces same result when applied to transformed tokens", file, line)
		} else {
			return fmt.Errorf("potential infinite loop in transformation - parser produces same result when applied to transformed tokens")
		}
	}

	return nil
}

// DetectLeftRecursion analyzes traces to identify potential left recursion patterns
func DetectLeftRecursion[T any](traces []*TraceInfo) []string {
	var warnings []string

	// Track parser calls at the same position
	positionCalls := make(map[string][]string)

	for _, trace := range traces {
		if trace.TraceType == Enter {
			posKey := fmt.Sprintf("%s", trace.Pos.String())
			positionCalls[posKey] = append(positionCalls[posKey], trace.Name)

			// If same parser called multiple times at same position
			calls := positionCalls[posKey]
			if len(calls) > 3 {
				lastThree := calls[len(calls)-3:]
				if lastThree[0] == lastThree[1] && lastThree[1] == lastThree[2] {
					warnings = append(warnings, fmt.Sprintf(
						"Potential left recursion detected: '%s' called repeatedly at %s",
						lastThree[0], posKey,
					))
				}
			}
		}
	}

	return warnings
}
