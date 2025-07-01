package main

import (
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
)

func main() {
	fmt.Println("=== Safe Expression Parser Demo ===")
	fmt.Println("Demonstrates parsing complex nested expressions without recursion issues")

	// Test cases
	testCases := []string{
		"42",
		"1+2",
		"1+2*3",
		"(1+2)*3",
		"((1+2)*3)+4",
		"(1+2)+(3*4)",
		"((1+2)*3)+((4-5)*6)",
	}

	for _, input := range testCases {
		fmt.Printf("\nParsing: %s\n", input)

		context := pc.NewParseContext[int]()
		context.TraceEnable = false // Disable for cleaner output

		// Parse the input
		result, err := pc.EvaluateWithRawTokens(context, splitTokens(input), CreateExpressionParser())
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  Result: %d\n", result[0])
		}
	}
}

// Create parsers using aliases to avoid infinite recursion
func CreateExpressionParser() pc.Parser[int] {
	// Create aliases for recursive parsers
	defineExpr, exprAlias := pc.NewAlias[int]("expr")

	// Define the expression parser hierarchy
	primaryExpr := createPrimaryExpr(exprAlias)
	multiplicationExpr := createMultiplicationExpr(primaryExpr)
	additionExpr := createAdditionExpr(multiplicationExpr)

	// Define the main expression as addition (lowest precedence)
	defineExpr(additionExpr)

	return additionExpr
}

// Addition and subtraction (lowest precedence)
func createAdditionExpr(multiplicationExpr pc.Parser[int]) pc.Parser[int] {
	return pc.Trans(
		pc.Seq(
			multiplicationExpr,
			pc.ZeroOrMore("add_tail",
				pc.Seq(
					pc.Or(Literal("+"), Literal("-")),
					multiplicationExpr,
				),
			),
		),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			result := tokens[0].Val

			// Process remaining tokens in pairs: operator, operand
			for i := 1; i < len(tokens); i += 2 {
				op := tokens[i].Raw
				operand := tokens[i+1].Val
				switch op {
				case "+":
					result += operand
				case "-":
					result -= operand
				}
			}

			return []pc.Token[int]{{Type: "expr", Val: result, Pos: tokens[0].Pos}}, nil
		},
	)
}

// Multiplication and division (higher precedence)
func createMultiplicationExpr(primaryExpr pc.Parser[int]) pc.Parser[int] {
	return pc.Trans(
		pc.Seq(
			primaryExpr,
			pc.ZeroOrMore("mul_tail",
				pc.Seq(
					pc.Or(Literal("*"), Literal("/")),
					primaryExpr,
				),
			),
		),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			result := tokens[0].Val

			// Process remaining tokens in pairs: operator, operand
			for i := 1; i < len(tokens); i += 2 {
				op := tokens[i].Raw
				operand := tokens[i+1].Val
				switch op {
				case "*":
					result *= operand
				case "/":
					if operand == 0 {
						return nil, fmt.Errorf("division by zero")
					}
					result /= operand
				}
			}

			return []pc.Token[int]{{Type: "expr", Val: result, Pos: tokens[0].Pos}}, nil
		},
	)
}

// Primary expressions (highest precedence: numbers and parenthesized expressions)
func createPrimaryExpr(exprAlias pc.Parser[int]) pc.Parser[int] {
	return pc.Or(
		Number(),
		createParenthesizedExpr(exprAlias),
	)
}

// Parenthesized expressions - uses the alias to safely refer back to the full expression
func createParenthesizedExpr(exprAlias pc.Parser[int]) pc.Parser[int] {
	return pc.Trans(
		pc.Seq(
			Literal("("),
			exprAlias, // Use the alias instead of direct recursion
			Literal(")"),
		),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			// Return the middle token (the expression result)
			return []pc.Token[int]{tokens[1]}, nil
		},
	)
}

func Number() pc.Parser[int] {
	return pc.Trace("number", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("number", "EOF", nil)
		}
		if src[0].Type == "raw" {
			val, err := strconv.Atoi(src[0].Raw)
			if err != nil {
				return 0, nil, pc.NewErrNotMatch("number", src[0].Raw, src[0].Pos)
			}
			return 1, []pc.Token[int]{{Type: "number", Val: val, Pos: src[0].Pos}}, nil
		} else if src[0].Type == "number" {
			return 1, src[0:1], nil
		}
		return 0, nil, pc.NewErrNotMatch("number", src[0].Type, src[0].Pos)
	})
}

func Literal(expected string) pc.Parser[int] {
	return pc.Trace("literal_"+expected, func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch(expected, "EOF", nil)
		}
		if src[0].Type == "raw" && src[0].Raw == expected {
			return 1, []pc.Token[int]{{Type: "literal", Raw: expected, Pos: src[0].Pos}}, nil
		}
		return 0, nil, pc.NewErrNotMatch(expected, src[0].Raw, src[0].Pos)
	})
}

// Simple tokenizer for demonstration
func splitTokens(input string) []string {
	var tokens []string
	var current string

	for _, r := range input {
		switch r {
		case '+', '-', '*', '/', '(', ')':
			if current != "" {
				tokens = append(tokens, current)
				current = ""
			}
			tokens = append(tokens, string(r))
		case ' ', '\t':
			if current != "" {
				tokens = append(tokens, current)
				current = ""
			}
		default:
			current += string(r)
		}
	}

	if current != "" {
		tokens = append(tokens, current)
	}

	return tokens
}
