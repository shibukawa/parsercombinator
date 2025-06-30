// Simple Calculator Interpreter - 数式を直接評価する電卓
package main

import (
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
)

// 数値パーサー
func ParseNumber() pc.Parser[int] {
	return pc.Trace("number", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("number", "EOF", nil)
		}
		if src[0].Type == "raw" {
			if val, err := strconv.Atoi(src[0].Raw); err == nil {
				return 1, []pc.Token[int]{{Type: "number", Pos: src[0].Pos, Val: val}}, nil
			}
		}
		return 0, nil, pc.NewErrNotMatch("number", src[0].Raw, src[0].Pos)
	})
}

// 演算子パーサー
func ParseOperator(op string) pc.Parser[int] {
	return pc.Trace("operator_"+op, func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("operator_"+op, "EOF", nil)
		}
		if src[0].Type == "raw" && src[0].Raw == op {
			return 1, []pc.Token[int]{{Type: "operator", Pos: src[0].Pos, Raw: op}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("operator_"+op, src[0].Raw, src[0].Pos)
	})
}

// 括弧パーサー
func ParseLeftParen() pc.Parser[int] {
	return pc.Trace("lparen", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("lparen", "EOF", nil)
		}
		if src[0].Type == "raw" && src[0].Raw == "(" {
			return 1, []pc.Token[int]{{Type: "lparen", Pos: src[0].Pos, Raw: "("}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("lparen", src[0].Raw, src[0].Pos)
	})
}

func ParseRightParen() pc.Parser[int] {
	return pc.Trace("rparen", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("rparen", "EOF", nil)
		}
		if src[0].Type == "raw" && src[0].Raw == ")" {
			return 1, []pc.Token[int]{{Type: "rparen", Pos: src[0].Pos, Raw: ")"}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("rparen", src[0].Raw, src[0].Pos)
	})
}

// 因子（数値のみの簡単版）
func ParseFactor() pc.Parser[int] {
	return ParseNumber()
}

// 項（因子の乗除）
func ParseTerm() pc.Parser[int] {
	return pc.Trans(
		pc.Seq(
			ParseFactor(),
			pc.ZeroOrMore("mul_div_ops",
				pc.Seq(pc.Or(ParseOperator("*"), ParseOperator("/")), ParseFactor()),
			),
		),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			// 最初の因子の値
			result := tokens[0].Val
			
			// 続く演算子と因子のペアを処理
			for i := 1; i < len(tokens); i += 2 {
				if i+1 < len(tokens) {
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
			}
			
			return []pc.Token[int]{{Type: "result", Pos: tokens[0].Pos, Val: result}}, nil
		},
	)
}

// 式（項の加減）
func ParseExpression() pc.Parser[int] {
	return pc.Trans(
		pc.Seq(
			ParseTerm(),
			pc.ZeroOrMore("add_sub_ops",
				pc.Seq(pc.Or(ParseOperator("+"), ParseOperator("-")), ParseTerm()),
			),
		),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			// 最初の項の値
			result := tokens[0].Val
			
			// 続く演算子と項のペアを処理
			for i := 1; i < len(tokens); i += 2 {
				if i+1 < len(tokens) {
					op := tokens[i].Raw
					operand := tokens[i+1].Val
					
					switch op {
					case "+":
						result += operand
					case "-":
						result -= operand
					}
				}
			}
			
			return []pc.Token[int]{{Type: "result", Pos: tokens[0].Pos, Val: result}}, nil
		},
	)
}

func main() {
	context := pc.NewParseContext[int]()
	context.TraceEnable = false // トレースは無効にして結果を見やすくする

	testCases := []struct {
		name  string
		input []string
	}{
		{"単純な数値", []string{"42"}},
		{"足し算", []string{"1", "+", "2"}},
		{"引き算", []string{"10", "-", "3"}},
		{"掛け算", []string{"6", "*", "7"}},
		{"割り算", []string{"20", "/", "4"}},
		{"演算子の優先度", []string{"2", "+", "3", "*", "4"}},
		{"複雑な式", []string{"10", "+", "2", "*", "3", "-", "5"}},
		{"左結合テスト", []string{"1", "+", "2", "+", "3", "+", "4"}},
		{"混合演算", []string{"1", "*", "2", "+", "3", "*", "4"}},
	}

	fmt.Println("=== インタプリタ電卓 ===")
	fmt.Println("数式をパースして直接評価し、結果を返します")
	fmt.Println("演算子の優先度: * / > + -")
	fmt.Println("結合性: 左結合")
	fmt.Println("（注意: 括弧はサポートしていません）")
	fmt.Println()

	for _, tc := range testCases {
		fmt.Printf("=== %s ===\n", tc.name)
		fmt.Printf("入力: %s\n", joinTokens(tc.input))
		
		result, err := pc.EvaluateWithRawTokens(context, tc.input, ParseExpression())
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
		} else if len(result) > 0 {
			fmt.Printf("結果: %d\n", result[0])
		}
		fmt.Println()
		
		// 新しいコンテキストを作成
		context = pc.NewParseContext[int]()
		context.TraceEnable = false
	}

	// エラーハンドリングのテスト
	fmt.Println("=== エラーハンドリングのテスト ===")
	errorCases := []struct {
		name  string
		input []string
	}{
		{"ゼロ除算", []string{"10", "/", "0"}},
		{"不完全な式", []string{"10", "+"}},
		{"空の式", []string{}},
		{"演算子のみ", []string{"+"}},
	}

	for _, tc := range errorCases {
		fmt.Printf("テスト: %s\n", tc.name)
		fmt.Printf("入力: %s\n", joinTokens(tc.input))
		
		context = pc.NewParseContext[int]()
		result, err := pc.EvaluateWithRawTokens(context, tc.input, ParseExpression())
		if err != nil {
			fmt.Printf("期待通りのエラー: %v\n", err)
		} else if len(result) > 0 {
			fmt.Printf("予期しない成功: %d\n", result[0])
		}
		fmt.Println()
	}
}

// ヘルパー関数：トークンを文字列に結合
func joinTokens(tokens []string) string {
	result := ""
	for i, token := range tokens {
		if i > 0 {
			result += " "
		}
		result += token
	}
	return result
}
