package parsercombinator

import (
	"testing"

	"github.com/alecthomas/assert/v2"
)

// Helper: returns a parser that matches a token with given raw string
func rawLiteral(expected string) Parser[string] {
	return func(pctx *ParseContext[string], src []Token[string]) (int, []Token[string], error) {
		if len(src) == 0 {
			return 0, nil, NewErrNotMatch(expected, "EOF", nil)
		}
		if src[0].Raw == expected {
			return 1, src[:1], nil
		}
		return 0, nil, NewErrNotMatch(expected, src[0].Raw, src[0].Pos)
	}
}

func makeTokens(strings ...string) []Token[string] {
	tokens := make([]Token[string], len(strings))
	for i, s := range strings {
		tokens[i] = Token[string]{Raw: s, Type: "raw"}
	}
	return tokens
}

func TestFind(t *testing.T) {
	tokens := makeTokens("a", "b", "c", "d")
	parser := rawLiteral("c")
	pctx := NewParseContext[string]()
	before, match, consume, remain, ok := Find(pctx, parser, tokens)
	if !ok || len(match) != 1 || match[0].Raw != "c" {
		t.Errorf("Find failed: got %v, ok=%v", match, ok)
	}
	assert.Equal(t, 2, len(before))
	assert.Equal(t, 1, len(remain))
	assert.Equal(t, 1, consume)
}

func TestFind2(t *testing.T) {
	tokens := makeTokens("a", "b", "c", "d", "e")
	// match and combine tokens
	parser := Trans(
		Seq(rawLiteral("c"), rawLiteral("d")),
		func(pctx *ParseContext[string], tokens []Token[string]) ([]Token[string], error) {
			return []Token[string]{
				{Raw: "cd", Type: "raw"},
			}, nil
		})

	pctx := NewParseContext[string]()
	before, match, consume, remain, ok := Find(pctx, parser, tokens)
	assert.True(t, ok, "Find should succeed")
	assert.Equal(t, 2, len(before), "Expected 2 tokens before match")
	assert.Equal(t, 2, consume, "Expected 2 tokens consumed")
	assert.Equal(t, 1, len(match), "Expected 2 tokens match")
	assert.Equal(t, "cd", match[0].Raw, "First match token should be 'c'")
	assert.Equal(t, 1, len(remain), "Expected 2 tokens remain")
}

func TestSplit(t *testing.T) {
	tokens := makeTokens("a", ",", "b", ",", "c")
	sep := rawLiteral(",")
	pctx := NewParseContext[string]()
	parts := Split(pctx, sep, tokens)
	result := make([][]Token[string], len(parts))
	for i, part := range parts {
		result[i] = part.Skipped
	}
	expect := [][]Token[string]{
		{{Raw: "a", Type: "raw"}},
		{{Raw: "b", Type: "raw"}},
		{{Raw: "c", Type: "raw"}},
	}
	if len(parts) != len(expect) {
		t.Fatalf("Split: expected %d parts, got %d", len(expect), len(parts))
	}
	assert.Equal(t, expect, result)
}

func TestSplitN(t *testing.T) {
	tokens := makeTokens("a", ",", "b", ",", "c", ",", "d")
	sep := rawLiteral(",")
	pctx := NewParseContext[string]()
	parts := SplitN(pctx, sep, tokens, 3)
	result := make([][]Token[string], len(parts))
	for i, part := range parts {
		result[i] = part.Skipped
	}
	expect := [][]Token[string]{
		{{Raw: "a", Type: "raw"}},
		{{Raw: "b", Type: "raw"}},
		{{Raw: "c", Type: "raw"}, {Raw: ",", Type: "raw"}, {Raw: "d", Type: "raw"}},
	}
	if len(parts) != len(expect) {
		t.Fatalf("SplitN: expected %d parts, got %d", len(expect), len(parts))
	}
	assert.Equal(t, expect, result)
}

func TestFindIter(t *testing.T) {
	tokens := makeTokens("a", "b", "c", "b", "c", "d")
	parser := rawLiteral("b")

	expected := [][]Token[string]{
		makeTokens("a"),
		makeTokens("c"),
		makeTokens("c", "d"),
	}
	pctx := NewParseContext[string]()
	result := [][]Token[string]{}
	for _, consume := range FindIter(pctx, parser, tokens) {
		result = append(result, consume.Skipped)
		if !consume.Last { // not last
			assert.Equal(t, "b", consume.Match[0].Raw)
		}
	}
	assert.Equal(t, expected, result)
}

func TestFindIterLastFlag(t *testing.T) {
	tokens := makeTokens("a", ",", "b", ",", "c")
	sep := rawLiteral(",")
	pctx := NewParseContext[string]()
	var results []bool
	for _, part := range FindIter(pctx, sep, tokens) {
		results = append(results, part.Last)
	}
	// Check that the Last flag is true for the last element
	if len(results) == 0 {
		t.Fatalf("FindIter: no results")
	}
	assert.True(t, results[len(results)-1], "Last flag should be true for the last result")
	// All others should be false
	for i := 0; i < len(results)-1; i++ {
		assert.False(t, results[i], "Last flag should be false for non-last results")
	}
}

func TestFindIterLastFlag_TableDriven(t *testing.T) {
	type testCase struct {
		name   string
		input  []string
		expect []bool // Expected values for Last flag
	}
	cases := []testCase{
		{
			name:   "no trailing separator",
			input:  []string{"a", ",", "b", ",", "c"},
			expect: []bool{false, false, true},
		},
		{
			name:   "with trailing separator",
			input:  []string{"a", ",", "b", ",", "c", ","},
			expect: []bool{false, false, false, true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := makeTokens(tc.input...)
			sep := rawLiteral(",")
			pctx := NewParseContext[string]()
			var results []bool
			for _, part := range FindIter(pctx, sep, tokens) {
				results = append(results, part.Last)
			}
			if len(results) != len(tc.expect) {
				t.Fatalf("FindIter: expected %d results, got %d", len(tc.expect), len(results))
			}
			for i := range results {
				assert.Equal(t, tc.expect[i], results[i], "Last flag mismatch at index %d", i)
			}
		})
	}
}

func TestSplitLastFlag_TableDriven(t *testing.T) {
	type testCase struct {
		name       string
		input      []string
		expectLen  int
		expectLast []bool // Expected values for Last flag
	}
	cases := []testCase{
		{
			name:       "Split: no trailing separator",
			input:      []string{"a", ",", "b", ",", "c"},
			expectLen:  3,
			expectLast: []bool{false, false, true},
		},
		{
			name:       "Split: with trailing separator",
			input:      []string{"a", ",", "b", ",", "c", ","},
			expectLen:  4,
			expectLast: []bool{false, false, false, true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := makeTokens(tc.input...)
			sep := rawLiteral(",")
			pctx := NewParseContext[string]()
			parts := Split(pctx, sep, tokens)
			var results []bool
			for _, part := range parts {
				results = append(results, part.Last)
			}
			if len(results) != tc.expectLen {
				t.Fatalf("Split: expected %d results, got %d", tc.expectLen, len(results))
			}
			assert.Equal(t, tc.expectLast, results)
		})
	}
}

func TestSplitNLastFlag_TableDriven(t *testing.T) {
	type testCase struct {
		name       string
		input      []string
		n          int
		expectLen  int
		expectLast []bool // Expected values for Last flag
	}
	cases := []testCase{
		{
			name:       "SplitN: no trailing separator, n=2",
			input:      []string{"a", ",", "b", ",", "c"},
			n:          2,
			expectLen:  2,
			expectLast: []bool{false, true},
		},
		{
			name:       "SplitN: with trailing separator, n=3",
			input:      []string{"a", ",", "b", ",", "c", ","},
			n:          3,
			expectLen:  3,
			expectLast: []bool{false, false, true},
		},
		{
			name:       "SplitN: n is much larger than separator count",
			input:      []string{"a", ",", "b", ",", "c"},
			n:          10,
			expectLen:  3,
			expectLast: []bool{false, false, true},
		},
		{
			name:       "SplitN: n is exactly separator count + 1",
			input:      []string{"a", ",", "b", ",", "c"},
			n:          3,
			expectLen:  3,
			expectLast: []bool{false, false, true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tokens := makeTokens(tc.input...)
			sep := rawLiteral(",")
			pctx := NewParseContext[string]()
			parts := SplitN(pctx, sep, tokens, tc.n)
			var results []bool
			for _, part := range parts {
				results = append(results, part.Last)
			}
			if len(results) != tc.expectLen {
				t.Fatalf("SplitN: expected %d results, got %d", tc.expectLen, len(results))
			}
			assert.Equal(t, tc.expectLast, results)
		})
	}
}

func TestEOS(t *testing.T) {
	tokens := makeTokens("a", "b")
	pctx := NewParseContext[string]()

	// Error if tokens remain
	consumed, out, err := EOS[string]()(pctx, tokens)
	assert.Error(t, err)
	assert.Equal(t, 0, consumed)
	assert.Equal(t, 0, len(out))

	// Success if no tokens remain
	consumed, out, err = EOS[string]()(pctx, []Token[string]{})
	assert.NoError(t, err)
	assert.Equal(t, 0, consumed)
	assert.Equal(t, 0, len(out))
}
