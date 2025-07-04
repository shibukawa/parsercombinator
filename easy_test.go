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

func makeTokens(strs ...string) []Token[string] {
	tokens := make([]Token[string], len(strs))
	for i, s := range strs {
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
