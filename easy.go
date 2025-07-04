package parsercombinator

import "iter"

// Find: Returns the first match of the parser in the input token slice, along with tokens before and after the match.
// Returns: (skipped, match, remained, found)
func Find[T any](ctx *ParseContext[T], parser Parser[T], tokens []Token[T]) (skipped, match, remained []Token[T], found bool) {
	for i := 0; i < len(tokens); i++ {
		_, match, err := parser(ctx, tokens[i:])
		if err == nil {
			skipped := tokens[:i]
			remained := tokens[i+len(match):]
			return skipped, match, remained, true
		}
	}
	return nil, nil, nil, false
}

type Pair[T any] struct {
	Skipped []Token[T]
	Match   []Token[T]
}

// Split: Splits the input tokens by the separator parser, returning slices of tokens between separators.
func Split[T any](ctx *ParseContext[T], sep Parser[T], tokens []Token[T]) []Pair[T] {
	var result []Pair[T]
	rest := tokens
	for len(rest) > 0 {
		before, match, after, found := Find(ctx, sep, rest)
		if found {
			result = append(result, Pair[T]{Skipped: before, Match: match})
			rest = after
		} else {
			result = append(result, Pair[T]{Skipped: rest, Match: nil})
			break
		}
	}
	return result
}

// SplitN: Splits the input tokens by the separator parser, up to N pieces (N <= 0 means unlimited).
func SplitN[T any](ctx *ParseContext[T], sep Parser[T], tokens []Token[T], n int) []Pair[T] {
	var result []Pair[T]
	rest := tokens
	count := 1
	for (n <= 0 || count < n) && len(rest) > 0 {
		before, match, after, found := Find(ctx, sep, rest)
		if found {
			result = append(result, Pair[T]{Skipped: before, Match: match})
			rest = after
			count++
		} else {
			break
		}
	}
	if len(rest) > 0 {
		result = append(result, Pair[T]{Skipped: rest, Match: nil})
	}
	return result
}

// FindIter: Calls yield for each non-overlapping match of the parser in the input token slice.
// Usage: FindIter(ctx, parser, tokens, func(match []Token[T]) bool { ... return true to continue, false to stop })
func FindIter[T any](ctx *ParseContext[T], sep Parser[T], tokens []Token[T]) iter.Seq2[[]Token[T], []Token[T]] {
	return func(yield func(skipped, match []Token[T]) bool) {
		rest := tokens
		for len(rest) > 0 {
			skipped, match, remained, found := Find(ctx, sep, rest)
			if found {
				if !yield(skipped, match) {
					return
				}
				rest = remained
			} else {
				yield(rest, nil)
				return
			}
		}
	}
}
