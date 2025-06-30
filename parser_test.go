package parsercombinator

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("🐙: ")
}

func Digit() Parser[int] {
	return Trace("digit", func(pc *ParseContext[int], src []Token[int]) (int, []Token[int], error) {
		if src[0].Type == "raw" {
			t := src[0]
			i, err := strconv.Atoi(t.Raw)
			if err != nil {
				return 0, nil, NewErrNotMatch("integer", fmt.Sprintf("'%s'", t.Raw), src[0].Pos)
			}
			return 1, []Token[int]{{Type: "digit", Pos: t.Pos, Val: int(i)}}, nil
		} else if src[0].Type == "digit" {
			return 1, src[0:1], nil
		}
		return 0, nil, NewErrNotMatch("raw or digit type", src[0].Type, src[0].Pos)
	})
}

var ErrWrongType = errors.New("wrong type")

func String() Parser[int] {
	return Trace("digit", func(pc *ParseContext[int], src []Token[int]) (int, []Token[int], error) {
		if src[0].Type == "raw" {
			t := src[0]
			return 1, []Token[int]{{Type: "string", Pos: t.Pos, Raw: t.Raw}}, nil
		} else if src[0].Type == "string" {
			return 1, src[0:1], nil
		}
		return 0, nil, &ParseError{Parent: ErrWrongType, Pos: src[0].Pos}
	})
}

func Operator() Parser[int] {
	supportedOperators := map[string]bool{
		"+": true,
		"-": true,
		"*": true,
		"/": true,
	}
	return Trace("operator", func(pc *ParseContext[int], src []Token[int]) (int, []Token[int], error) {
		if src[0].Type == "raw" {
			t := src[0]
			if _, ok := supportedOperators[t.Raw]; ok {
				return 1, []Token[int]{{Type: "operator", Pos: t.Pos, Raw: t.Raw}}, nil
			}
			return 0, nil, NewErrNotMatch("operator", fmt.Sprintf("'%s'", t.Raw), src[0].Pos)
		} else if src[0].Type == "operator" {
			return 1, src[0:1], nil
		}
		return 0, nil, NewErrNotMatch("operator", src[0].Type, src[0].Pos)
	})
}

func Expression() Parser[int] {
	expressionTransform := func(pc *ParseContext[int], src []Token[int]) (converted []Token[int], err error) {
		var result int
		switch src[1].Raw {
		case "+":
			result = src[0].Val + src[2].Val
		case "-":
			result = src[0].Val - src[2].Val
		case "*":
			result = src[0].Val * src[2].Val
		case "/":
			result = src[0].Val / src[2].Val
		}
		return []Token[int]{{Type: "digit", Pos: src[0].Pos, Val: result}}, nil
	}
	return Trace("expression",
		Trans(
			Seq(
				Digit(), Operator(), Digit(),
			),
			expressionTransform),
	)
}

func TestSingleNode(t *testing.T) {
	pc := NewParseContext[int]()
	pc.TraceEnable = true
	result, err := EvaluateWithRawTokens(pc, []string{"100"}, Digit())
	t.Log(pc.DumpTraceAsText())
	assert.NoError(t, err)
	assert.Equal(t, []int{100}, result)
}

func TestSingleNodeError(t *testing.T) {
	pc := NewParseContext[int]()
	pc.TraceEnable = true
	result, err := EvaluateWithRawTokens(pc, []string{"text"}, Digit())
	t.Log(pc.DumpTraceAsText())
	assert.Zero(t, result)
	assert.Error(t, err)
	var pe *ParseError
	assert.True(t, errors.As(err, &pe))
}

func TestSeqNode(t *testing.T) {
	pc := NewParseContext[int]()
	pc.TraceEnable = true
	result, err := EvaluateWithRawTokens(pc, []string{"100", "+", "200"}, Expression())
	t.Log(pc.DumpTraceAsText())
	assert.NoError(t, err)
	assert.Equal(t, []int{300}, result)
}

func TestOr(t *testing.T) {
	tests := []struct {
		name    string
		src     []string
		wantErr bool
		want    []any
	}{
		{name: "match first", src: []string{"100"}, want: []any{100}},
		{name: "match second", src: []string{"test"}, want: []any{"test"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			_, err := EvaluateWithRawTokens(pc, tt.src, Or(Digit(), String()))
			t.Log(pc.DumpTraceAsText())
			if (err != nil) != tt.wantErr {
				t.Errorf("Or() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var result []any
			for _, token := range pc.Results {
				if token.Val != 0 {
					result = append(result, token.Val)
				} else {
					result = append(result, token.Raw)
				}
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestRepeat(t *testing.T) {
	tests := []struct {
		name    string
		min     uint
		max     int
		src     []string
		wantErr bool
		want    []int
	}{
		{name: "zero or more (match all)", min: 0, max: -1, src: []string{"100", "200", "300"}, want: []int{100, 200, 300}},
		{name: "zero or more (match two nodes)", min: 0, max: -1, src: []string{"100", "200", "test"}, want: []int{100, 200}},
		{name: "one or more (match all)", min: 1, max: -1, src: []string{"100", "200", "300"}, want: []int{100, 200, 300}},
		{name: "one or more (match two nodes)", min: 1, max: -1, src: []string{"100", "200", "test"}, want: []int{100, 200}},
		{name: "one or more (less match error: 1)", min: 1, max: -1, src: []string{}, wantErr: true},
		{name: "one or more (less match error: 2)", min: 1, max: -1, src: []string{"test", "test", "test"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, Repeat("digits", tt.min, tt.max, Digit()))
			t.Log(pc.DumpTraceAsText())
			if (err != nil) != tt.wantErr {
				t.Errorf("Repeat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, result)
		})
	}
}

func Sum() Parser[int] {
	addTransform := func(pc *ParseContext[int], src []Token[int]) (converted []Token[int], err error) {
		result := src[0].Val + src[1].Val
		return []Token[int]{{Type: "digit", Pos: src[0].Pos, Val: result}}, nil
	}
	return Trace("add", Trans(
		Seq(Digit(), Digit(), EOL()),
		addTransform,
	))
}

func EOL() Parser[int] {
	return Trace("eol", func(pc *ParseContext[int], src []Token[int]) (int, []Token[int], error) {
		if src[0].Type == "raw" {
			t := src[0]
			if t.Raw == ";" {
				return 1, []Token[int]{{Type: "digit", Pos: t.Pos, Raw: ";"}}, nil
			}
		} else if src[0].Type == "eol" {
			return 1, []Token[int]{}, nil
		}
		return 0, nil, NewErrNotMatch("EOL(;)", src[0].Type, src[0].Pos)
	})
}

func TestRecover(t *testing.T) {
	tests := []struct {
		name         string
		src          []string
		wantErrCount int
	}{
		{
			name:         "correct pattern",
			src:          []string{"100", "200", ";", "300", "400", ";"},
			wantErrCount: 0,
		},
		{
			name:         "correct pattern",
			src:          []string{"100", "200", ";", "300", ";"},
			wantErrCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			pattern := ZeroOrMore("sum expressions", Recover(
				Digit(),
				Sum(),
				EOL(),
			))
			EvaluateWithRawTokens(pc, tt.src, pattern)
			t.Log(pc.DumpTraceAsText())
			for _, e := range pc.Errors {
				t.Logf("  Error: %s", e.Error())
			}
			assert.Equal(t, tt.wantErrCount, len(pc.Errors))
		})
	}
}

func GenErrNotMatch() Parser[int] {
	return Trace("err-not-match", func(pc *ParseContext[int], st []Token[int]) (consumed int, newTokens []Token[int], err error) {
		return 0, []Token[int]{}, NewErrNotMatch("expected", "want", nil)
	})
}

func GenErrRepeatCount() Parser[int] {
	return Trace("err-repeat-count", func(pc *ParseContext[int], st []Token[int]) (consumed int, newTokens []Token[int], err error) {
		return 0, []Token[int]{}, NewErrRepeatCount("repeat count", 2, 1, nil)
	})
}

func GenErrCritical() Parser[int] {
	return Trace("err-critical", func(pc *ParseContext[int], st []Token[int]) (consumed int, newTokens []Token[int], err error) {
		return 0, []Token[int]{}, NewErrCritical("dummy", nil)
	})
}

func TestErrorType(t *testing.T) {
	tests := []struct {
		name         string
		parser       Parser[int]
		src          []string
		want         error
		wantCount    int
		wantErrCount int
	}{
		{
			name:         "Repeat() with not match",
			parser:       Repeat("errors", 0, -1, GenErrNotMatch()),
			src:          []string{"test"},
			want:         nil,
			wantCount:    0,
			wantErrCount: 0,
		},
		{
			name:         "Repeat() with repeat count",
			parser:       Repeat("errors", 0, -1, GenErrRepeatCount()),
			src:          []string{"test"},
			want:         ErrRepeatCount,
			wantCount:    0,
			wantErrCount: 1,
		},
		{
			name:         "Repeat() with critical error",
			parser:       Repeat("errors", 0, -1, GenErrCritical()),
			src:          []string{"test"},
			want:         ErrCritical,
			wantCount:    0,
			wantErrCount: 1,
		},
		{
			name:         "Or() with not match then match",
			parser:       Or(GenErrNotMatch(), Digit()),
			src:          []string{"10"},
			want:         nil,
			wantCount:    1,
			wantErrCount: 0,
		},
		{
			name:         "Or() with repeat count then match",
			parser:       Or(GenErrRepeatCount(), Digit()),
			src:          []string{"10"},
			want:         nil,
			wantCount:    1,
			wantErrCount: 0,
		},
		{
			name:         "Or() with critical error then match",
			parser:       Or(GenErrCritical(), Digit()),
			src:          []string{"10"},
			want:         ErrCritical,
			wantCount:    0,
			wantErrCount: 1,
		},
		{
			name:         "Or() with not match and not match all",
			parser:       Or(GenErrNotMatch(), Digit()),
			src:          []string{"test"},
			want:         ErrNotMatch,
			wantCount:    0,
			wantErrCount: 1,
		},
		{
			name:         "Or() with repeat count and not match all",
			parser:       Or(GenErrRepeatCount(), Digit()),
			src:          []string{"test"},
			want:         ErrNotMatch,
			wantCount:    0,
			wantErrCount: 1,
		},
		{
			name:         "Or() with critical error and not match all",
			parser:       Or(GenErrCritical(), Digit()),
			src:          []string{"test"},
			want:         ErrCritical,
			wantCount:    0,
			wantErrCount: 1,
		},
		{
			name: "Recover() with not match",
			parser: Seq(
				Or( // first parser absorb error then second parser match
					Recover(Digit(), Seq(Digit(), GenErrNotMatch(), Digit()), Digit()),
					Digit(),
				),
				Digit(),
			),
			src:          []string{"10", "20"},
			want:         ErrNotMatch,
			wantCount:    1,
			wantErrCount: 1,
		},
		{
			name: "Recover() with repeat count",
			parser: Seq(
				Or( // first parser absorb error then second parser match
					Recover(Digit(), Seq(Digit(), GenErrRepeatCount(), Digit()), Digit()),
					Digit(),
				),
				Digit(),
			),
			src:          []string{"10", "20"},
			want:         ErrRepeatCount,
			wantCount:    1,
			wantErrCount: 1,
		},
		{
			name: "Recover() with critical error",
			parser: Seq(
				Or( // first parser absorb error then second parser match
					Recover(Digit(), Seq(Digit(), GenErrCritical(), Digit()), Digit()),
					Digit(),
				),
				Digit(),
			),
			src:          []string{"10", "20"},
			want:         ErrCritical,
			wantCount:    1,
			wantErrCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			_, err := EvaluateWithRawTokens[int](pc, tt.src, tt.parser)
			assert.True(t, errors.Is(err, tt.want))
			assert.Equal(t, tt.wantCount, len(pc.Results))
			assert.Equal(t, tt.wantErrCount, len(pc.Errors))
		})
	}
}

func TestAlias(t *testing.T) {
	expressionBody, expression := NewAlias[int]("expression")
	parser := expressionBody(
		Or(
			Digit(),
			Trans(
				Seq(Operator(), expression, expression),
				func(pctx *ParseContext[int], src []Token[int]) (converted []Token[int], err error) {
					var result int
					switch src[0].Raw {
					case "+":
						result = src[1].Val + src[2].Val
					case "-":
						result = src[1].Val - src[2].Val
					case "*":
						result = src[1].Val * src[2].Val
					case "/":
						result = src[1].Val / src[2].Val
					}
					return []Token[int]{{Type: "digit", Pos: src[0].Pos, Val: result}}, nil
				},
			),
		),
	)

	tests := []struct {
		name    string
		src     []string
		wantErr bool
		want    int
	}{
		{
			name: "single digit",
			src:  []string{"100"},
			want: 100,
		},
		{
			name: "operator digit",
			src:  []string{"+", "100", "200"},
			want: 300,
		},
		{
			name: "operator digit",
			src:  []string{"+", "100", "-", "200", "100"},
			want: 200,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Alias error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, result[0])
		})
	}
}

func TestBefore(t *testing.T) {
	parser := Before(func(token Token[int]) bool {
		return token.Raw == ";"
	})

	tests := []struct {
		name      string
		src       []string
		wantCount int
	}{
		{
			name:      "single digit",
			src:       []string{"100", "200", ";"},
			wantCount: 2,
		},
		{
			name:      "operator digit",
			src:       []string{";", "100", "200"},
			wantCount: 0,
		},
		{
			name:      "operator digit",
			src:       []string{"200", "100", "300"},
			wantCount: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens[int](pc, tt.src, parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if err != nil {
				t.Errorf("EvaluateWithRawTokens error = %v", err)
				return
			}
			assert.Equal(t, tt.wantCount, len(result))
		})
	}
}

func TestNone(t *testing.T) {
	parser := Seq(Digit(), None[int](), Digit())

	tests := []struct {
		name      string
		src       []string
		wantCount int
	}{
		{
			name:      "single digit",
			src:       []string{"100", "200"},
			wantCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if err != nil {
				t.Errorf("EvaluateWithRawTokens error = %v", err)
				return
			}
			assert.Equal(t, tt.wantCount, len(result))
		})
	}
}

func TestLookahead(t *testing.T) {
	tests := []struct {
		name    string
		parser  Parser[int]
		src     []string
		wantErr bool
		want    []int
	}{
		{
			name:   "lookahead match - consume after check",
			parser: Seq(Lookahead(Digit()), Digit()),
			src:    []string{"100"},
			want:   []int{100},
		},
		{
			name:    "lookahead not match",
			parser:  Seq(Lookahead(Digit()), Digit()),
			src:     []string{"test"},
			wantErr: true,
		},
		{
			name: "conditional parsing with lookahead",
			parser: Or(
				Seq(Lookahead(Operator()), Operator(), Digit()),
				Digit(),
			),
			src:  []string{"100"},
			want: []int{100}, // Should match the second alternative (Digit)
		},
		{
			name: "conditional parsing with lookahead - operator case",
			parser: Or(
				Seq(Lookahead(Operator()), Operator(), Digit()),
				Digit(),
			),
			src:  []string{"+", "100"},
			want: []int{0, 100}, // Operator produces Val=0, then digit produces 100
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, tt.parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Lookahead error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestNotFollowedBy(t *testing.T) {
	tests := []struct {
		name    string
		parser  Parser[int]
		src     []string
		wantErr bool
		want    []int
	}{
		{
			name:   "not followed by - success case",
			parser: Seq(Digit(), NotFollowedBy(Operator())),
			src:    []string{"100", "200"},
			want:   []int{100},
		},
		{
			name:    "not followed by - fail case",
			parser:  Seq(Digit(), NotFollowedBy(Operator())),
			src:     []string{"100", "+"},
			wantErr: true,
		},
		{
			name:   "identifier not followed by digit",
			parser: Seq(String(), NotFollowedBy(Digit())),
			src:    []string{"var", "+"},
			want:   []int{1}, // String parser produces 1 result with Raw value, counted as 1 item
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			_, err := EvaluateWithRawTokens(pc, tt.src, tt.parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("NotFollowedBy error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Convert results for comparison
				var convertedResult []any
				for _, token := range pc.Results {
					if token.Val != 0 {
						convertedResult = append(convertedResult, token.Val)
					} else {
						convertedResult = append(convertedResult, token.Raw)
					}
				}
				assert.Equal(t, len(tt.want), len(convertedResult))
			}
		})
	}
}

func TestPeek(t *testing.T) {
	tests := []struct {
		name    string
		parser  Parser[int]
		src     []string
		wantErr bool
		want    []int
	}{
		{
			name:   "peek without consuming",
			parser: Seq(Peek(Digit()), Digit(), Digit()),
			src:    []string{"100", "200"},
			want:   []int{100, 100, 200},
		},
		{
			name: "peek for conditional logic",
			parser: Seq(
				Or(
					Seq(Peek(Operator()), Operator()),
					None[int](),
				),
				Digit(),
			),
			src:  []string{"+", "100"},
			want: []int{0, 0, 100}, // Peek produces 0, Operator produces 0, Digit produces 100
		},
		{
			name: "peek for conditional logic - no operator",
			parser: Seq(
				Or(
					Seq(Peek(Operator()), Operator()),
					None[int](),
				),
				Digit(),
			),
			src:  []string{"100"},
			want: []int{100},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, tt.parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Peek error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestLabel(t *testing.T) {
	tests := []struct {
		name       string
		parser     Parser[int]
		src        []string
		wantErr    bool
		wantErrMsg string
		want       []int
	}{
		{
			name:   "label success case",
			parser: Label("number", Digit()),
			src:    []string{"100"},
			want:   []int{100},
		},
		{
			name:       "label error case - cleaner message",
			parser:     Label("number", Digit()),
			src:        []string{"text"},
			wantErr:    true,
			wantErrMsg: "number",
		},
		{
			name:       "complex parser with label",
			parser:     Label("arithmetic expression", Seq(Digit(), Operator(), Digit())),
			src:        []string{"100", "invalid", "200"},
			wantErr:    true,
			wantErrMsg: "arithmetic expression",
		},
		{
			name: "nested labels",
			parser: Or(
				Label("number", Digit()),
				Label("text", String()),
			),
			src:  []string{"hello"},
			want: []int{0}, // String parser produces Val=0 for strings
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, tt.parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Label error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrMsg != "" {
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestExpectedAndFail(t *testing.T) {
	tests := []struct {
		name       string
		parser     Parser[int]
		src        []string
		wantErr    bool
		wantErrMsg string
		want       []int
	}{
		{
			name:       "expected always fails",
			parser:     Expected[int]("closing bracket"),
			src:        []string{"100"},
			wantErr:    true,
			wantErrMsg: "closing bracket",
		},
		{
			name:       "fail always fails",
			parser:     Fail[int]("not implemented yet"),
			src:        []string{"100"},
			wantErr:    true,
			wantErrMsg: "not implemented yet",
		},
		{
			name: "conditional error with Or",
			parser: Or(
				Digit(),
				Expected[int]("valid number"),
			),
			src:  []string{"100"},
			want: []int{100},
		},
		{
			name: "conditional error with Or - error case",
			parser: Or(
				Label("number", Digit()),
				Expected[int]("valid identifier"),
			),
			src:        []string{"invalid"},
			wantErr:    true,
			wantErrMsg: "valid identifier",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			_, err := EvaluateWithRawTokens(pc, tt.src, tt.parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("Expected/Fail error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrMsg != "" {
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			}
			if !tt.wantErr && tt.want != nil {
				// Convert results for comparison
				var convertedResult []int
				for _, token := range pc.Results {
					if token.Val != 0 {
						convertedResult = append(convertedResult, token.Val)
					}
				}
				if len(convertedResult) > 0 {
					assert.Equal(t, tt.want, convertedResult)
				}
			}
		})
	}
}

func TestAdvancedErrorPatterns(t *testing.T) {
	tests := []struct {
		name       string
		parser     Parser[int]
		src        []string
		wantErr    bool
		wantErrMsg string
		want       []int
	}{
		{
			name: "fallback to specific error message",
			parser: Or(
				Label("number", Digit()),              // 最初に数値を試す（ラベル付き）
				Label("text", String()),               // 次に文字列を試す（ラベル付き）
				Expected[int]("number or identifier"), // どちらでもない場合は特定のエラー
			),
			src:  []string{"symbol"}, // この場合StringがVal=0で成功するが、Labelでラベル化される
			want: []int{0},           // String parser produces Val=0
		},
		{
			name: "true fallback error - no valid alternatives",
			parser: Or(
				Seq(Digit(), Operator()),          // 数値+演算子のペア
				Seq(String(), Digit()),            // 文字列+数値のペア
				Expected[int]("valid expression"), // どちらでもない場合
			),
			src:        []string{"invalid"}, // 単一の無効なトークン
			wantErr:    true,
			wantErrMsg: "valid expression",
		},
		{
			name: "required closing parenthesis",
			parser: Seq(
				Label("opening parenthesis", Operator()), // "+" を開き括弧として使用
				Digit(),
				Or(
					Label("closing parenthesis", Operator()), // "-" を閉じ括弧として使用
					Expected[int]("closing parenthesis"),     // 見つからない場合の明確なエラー
				),
			),
			src:        []string{"+", "100", "invalid"},
			wantErr:    true,
			wantErrMsg: "closing parenthesis",
		},
		{
			name: "syntax error in expression",
			parser: Or(
				Seq(Digit(), Operator(), Digit()),       // 正常な式
				Seq(Digit(), Expected[int]("operator")), // 数値の後に演算子がない
			),
			src:        []string{"100", "invalid"},
			wantErr:    true,
			wantErrMsg: "operator",
		},
		{
			name: "conditional feature availability",
			parser: Or(
				Digit(), // 実装済み機能
				Fail[int]("advanced expressions not implemented in this version"), // 未実装機能
			),
			src:        []string{"function_call"},
			wantErr:    true,
			wantErrMsg: "not implemented",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, tt.parser)
			if testing.Verbose() {
				t.Log(pc.DumpTraceAsText())
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("AdvancedErrorPatterns error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.wantErrMsg != "" {
				assert.Contains(t, err.Error(), tt.wantErrMsg)
			}
			if !tt.wantErr && tt.want != nil {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}

func TestZeroOrMore(t *testing.T) {
	tests := []struct {
		name    string
		src     []string
		want    []int
		wantErr bool
	}{
		{
			name: "zero matches (empty input)",
			src:  []string{},
			want: []int{},
		},
		{
			name: "zero matches (no matching elements)",
			src:  []string{"test", "hello", "world"},
			want: []int{},
		},
		{
			name: "one match",
			src:  []string{"100"},
			want: []int{100},
		},
		{
			name: "multiple matches",
			src:  []string{"100", "200", "300"},
			want: []int{100, 200, 300},
		},
		{
			name: "partial matches (stops at non-matching)",
			src:  []string{"100", "200", "test", "400"},
			want: []int{100, 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, ZeroOrMore("digits", Digit()))
			t.Log(pc.DumpTraceAsText())

			if (err != nil) != tt.wantErr {
				t.Errorf("ZeroOrMore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, result)
		})
	}
}

func TestOneOrMore(t *testing.T) {
	tests := []struct {
		name    string
		src     []string
		want    []int
		wantErr bool
	}{
		{
			name:    "zero matches (empty input) - should error",
			src:     []string{},
			wantErr: true,
		},
		{
			name:    "zero matches (no matching elements) - should error",
			src:     []string{"test", "hello", "world"},
			wantErr: true,
		},
		{
			name: "one match",
			src:  []string{"100"},
			want: []int{100},
		},
		{
			name: "multiple matches",
			src:  []string{"100", "200", "300"},
			want: []int{100, 200, 300},
		},
		{
			name: "partial matches (stops at non-matching)",
			src:  []string{"100", "200", "test", "400"},
			want: []int{100, 200},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := NewParseContext[int]()
			pc.TraceEnable = true
			result, err := EvaluateWithRawTokens(pc, tt.src, OneOrMore("digits", Digit()))
			t.Log(pc.DumpTraceAsText())

			if (err != nil) != tt.wantErr {
				t.Errorf("OneOrMore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, tt.want, result)
			}
		})
	}
}
