package parsercombinator

import (
	"fmt"
	"iter"
)

// EOS: Parser that matches only if there are no remaining input tokens.
func EOS[T any]() Parser[T] {
	return func(ctx *ParseContext[T], tokens []Token[T]) (int, []Token[T], error) {
		if len(tokens) == 0 {
			return 0, nil, nil
		}
		return 0, nil, fmt.Errorf("expected end of sequence, but got: %v", tokens[0])
	}
}

// Find: Returns the first match of the parser in the input token slice, along with tokens before and after the match.
// Returns: (skipped, match, remained, found)
func Find[T any](ctx *ParseContext[T], parser Parser[T], tokens []Token[T]) (skipped, match []Token[T], consume int, remained []Token[T], found bool) {
	for i := 0; i < len(tokens); i++ {
		consume, match, err := parser(ctx, tokens[i:])
		if err == nil {
			skipped := tokens[:i]
			remained := tokens[i+consume:]
			return skipped, match, consume, remained, true
		}
	}
	return nil, nil, 0, nil, false
}

type Consume[T any] struct {
	Skipped []Token[T]
	Match   []Token[T]
	Consume int
	Last    bool
}

// Split: Splits the input tokens by the separator parser, returning slices of tokens between separators.
func Split[T any](ctx *ParseContext[T], sep Parser[T], tokens []Token[T]) []Consume[T] {
	var result []Consume[T]
	rest := tokens
	foundLastSeparator := false
	for len(rest) > 0 {
		before, match, consume, after, found := Find(ctx, sep, rest)
		if found {
			result = append(result, Consume[T]{
				Skipped: before,
				Match:   match,
				Consume: consume,
			})
			rest = after
			foundLastSeparator = true
		} else {
			result = append(result, Consume[T]{
				Skipped: rest,
				Match:   nil,
				Consume: 0,
				Last:    true,
			})
			foundLastSeparator = false
			break
		}
	}
	// If the last token was a separator, add an empty element (like strings.Split)
	if foundLastSeparator {
		result = append(result, Consume[T]{
			Skipped: nil,
			Match:   nil,
			Consume: 0,
			Last:    true,
		})
	}
	return result
}

// SplitN: Splits the input tokens by the separator parser, up to N pieces (N <= 0 means unlimited).
func SplitN[T any](ctx *ParseContext[T], sep Parser[T], tokens []Token[T], n int) []Consume[T] {
	var result []Consume[T]
	rest := tokens
	count := 1
	for (n <= 0 || count < n) && len(rest) > 0 {
		before, match, consume, after, found := Find(ctx, sep, rest)
		if found {
			result = append(result, Consume[T]{
				Skipped: before,
				Match:   match,
				Consume: consume,
			})
			rest = after
			count++
		} else {
			break
		}
	}
	if len(rest) > 0 {
		result = append(result, Consume[T]{Skipped: rest, Last: true})
	}
	return result
}

// FindIter: Calls yield for each non-overlapping match of the parser in the input token slice.
// Usage: FindIter(ctx, parser, tokens, func(match []Token[T]) bool { ... return true to continue, false to stop })
func FindIter[T any](ctx *ParseContext[T], sep Parser[T], tokens []Token[T]) iter.Seq2[int, Consume[T]] {
	return func(yield func(index int, consume Consume[T]) bool) {
		rest := tokens
		index := 0
		foundLastSeparator := false
		for len(rest) > 0 {
			skipped, match, consume, remained, found := Find(ctx, sep, rest)
			if found {
				if !yield(index, Consume[T]{
					Skipped: skipped,
					Match:   match,
					Consume: consume,
				}) {
					return
				}
				rest = remained
				foundLastSeparator = true
			} else {
				yield(index, Consume[T]{Skipped: rest, Last: true})
				foundLastSeparator = false
				return
			}
			index++
		}
		if foundLastSeparator {
			yield(index, Consume[T]{Last: true})
		}
	}
}
