package main

import (
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
)

func main() {
	fmt.Println("=== Parser Combinator Safety Features Demo ===")

	// Demo 1: Transformation Safety Check
	fmt.Println("\n1. Automatic Transformation Safety Check Demo:")
	demoTransformationSafety()

	// Demo 2: Or Parser Modes
	fmt.Println("\n2. Or Parser Modes Demo:")
	demoOrParserModes()
}

func demoTransformationSafety() {
	// Create a simple digit parser
	digitParser := func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("digit", "EOF", nil)
		}
		if src[0].Type == "raw" {
			val, err := strconv.Atoi(src[0].Raw)
			if err != nil {
				return 0, nil, pc.NewErrNotMatch("digit", src[0].Raw, src[0].Pos)
			}
			return 1, []pc.Token[int]{{Type: "digit", Val: val, Pos: src[0].Pos}}, nil
		} else if src[0].Type == "digit" {
			return 1, src[0:1], nil
		}
		// Only accept "raw" and "digit" types, not "number"
		return 0, nil, pc.NewErrNotMatch("raw or digit type", src[0].Type, src[0].Pos)
	}

	// Test 1: Safe transformation (changes token type)
	fmt.Println("  Testing safe transformation...")
	safeContext := pc.NewParseContext[int]()
	safeContext.OrMode = pc.OrModeSafe
	safeContext.CheckTransformSafety = true

	safeParser := pc.Trans(digitParser, func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
		return []pc.Token[int]{{
			Type: "number", // Different type - safe!
			Val:  tokens[0].Val,
			Pos:  tokens[0].Pos,
		}}, nil
	})

	result, err := pc.EvaluateWithRawTokens(safeContext, []string{"42"}, safeParser)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Safe transformation succeeded: %v\n", result[0])
	}

	// Test 2: Unsafe transformation (identity - same type and value)
	fmt.Println("  Testing unsafe transformation...")
	unsafeContext := pc.NewParseContext[int]()
	unsafeContext.OrMode = pc.OrModeSafe
	unsafeContext.CheckTransformSafety = true

	unsafeParser := pc.Trans(digitParser, func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
		// Return same tokens - potentially unsafe!
		return tokens, nil
	})

	result, err = pc.EvaluateWithRawTokens(unsafeContext, []string{"42"}, unsafeParser)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Unsafe transformation completed with warning: %v\n", result[0])
		fmt.Printf("    (Check stderr for safety warning)\n")
	}

	// Test 3: Safety check disabled
	fmt.Println("  Testing with safety check disabled...")
	disabledContext := pc.NewParseContext[int]()
	disabledContext.OrMode = pc.OrModeSafe
	disabledContext.CheckTransformSafety = false // Disabled

	result, err = pc.EvaluateWithRawTokens(disabledContext, []string{"42"}, unsafeParser)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    No safety check performed: %v\n", result[0])
	}
}

func demoOrParserModes() {
	// Create test parsers
	shortMatch := func(pctx *pc.ParseContext[string], src []pc.Token[string]) (int, []pc.Token[string], error) {
		if len(src) >= 2 {
			return 2, []pc.Token[string]{{Type: "short", Raw: "short", Pos: src[0].Pos}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("short pattern", "insufficient tokens", getFirstPos(src))
	}

	longMatch := func(pctx *pc.ParseContext[string], src []pc.Token[string]) (int, []pc.Token[string], error) {
		if len(src) >= 3 {
			return 3, []pc.Token[string]{{Type: "long", Raw: "long", Pos: src[0].Pos}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("long pattern", "insufficient tokens", getFirstPos(src))
	}

	input := []string{"a", "b", "c", "d"}

	// Test Safe mode (longest match)
	fmt.Println("  Safe mode (longest match):")
	safeContext := pc.NewParseContext[string]()
	safeContext.OrMode = pc.OrModeSafe
	orParser := pc.Or(shortMatch, longMatch) // short first, but longest will win

	result, err := pc.EvaluateWithRawTokens(safeContext, input, orParser)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Result: %v (Safe mode chooses longest match)\n", result[0])
	}

	// Test Fast mode (first match)
	fmt.Println("  Fast mode (first match):")
	fastContext := pc.NewParseContext[string]()
	fastContext.OrMode = pc.OrModeFast

	result, err = pc.EvaluateWithRawTokens(fastContext, input, orParser)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Result: %v (Fast mode chooses first successful match)\n", result[0])
	}

	// Test TryFast mode (first match with warning)
	fmt.Println("  TryFast mode (first match with optimization warning):")
	tryFastContext := pc.NewParseContext[string]()
	tryFastContext.OrMode = pc.OrModeTryFast

	result, err = pc.EvaluateWithRawTokens(tryFastContext, input, orParser)
	if err != nil {
		fmt.Printf("    Error: %v\n", err)
	} else {
		fmt.Printf("    Result: %v\n", result[0])
		fmt.Printf("    (Check stderr for optimization suggestion)\n")
	}
}

func getFirstPos[T any](src []pc.Token[T]) *pc.Pos {
	if len(src) > 0 {
		return src[0].Pos
	}
	return nil
}
