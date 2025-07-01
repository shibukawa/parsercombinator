package main

import (
	"fmt"
	"log"

	pc "github.com/shibukawa/parsercombinator"
)

func main() {
	// Example 1: Set custom stack depth limit
	parseContext := pc.NewParseContext[int]()
	parseContext.MaxDepth = 50 // Set maximum recursion depth to 50
	parseContext.TraceEnable = true

	// Create a simple parser
	parser := pc.Or(
		pc.Trace("digit", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
			if len(src) == 0 {
				return 0, nil, pc.NewErrNotMatch("digit", "EOF", nil)
			}
			return 1, src[0:1], nil
		}),
		pc.Trace("string", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
			if len(src) == 0 {
				return 0, nil, pc.NewErrNotMatch("string", "EOF", nil)
			}
			return 1, src[0:1], nil
		}),
	)

	result, err := pc.EvaluateWithRawTokens(parseContext, []string{"test"}, parser)
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Result: %v\n", result)
	}

	// Example 2: Demonstrate stack overflow protection
	parseContext2 := pc.NewParseContext[int]()
	parseContext2.MaxDepth = 5 // Very low limit
	parseContext2.TraceEnable = false

	// Create a problematic recursive parser
	var recursiveParser pc.Parser[int]
	recursiveParser = pc.Trace("recursive", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		// This parser tries to call itself infinitely
		return recursiveParser(pctx, src)
	})

	_, err = pc.EvaluateWithRawTokens(parseContext2, []string{"test"}, recursiveParser)
	if err != nil {
		fmt.Printf("Expected stack overflow error: %v\n", err)
	}

	// Example 3: Disable stack depth limit (set to 0)
	parseContext3 := pc.NewParseContext[int]()
	parseContext3.MaxDepth = 0 // No limit
	fmt.Printf("Default max depth: %d\n", pc.NewParseContext[int]().MaxDepth)
	fmt.Printf("No limit max depth: %d\n", parseContext3.MaxDepth)
}
