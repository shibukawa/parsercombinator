// Simple Math Expression to JSON AST Compiler - シンプルな数式→JSON AST コンパイラ
package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
)

// AST ノードの型定義
type ASTNode struct {
	Type     string     `json:"type"`
	Value    *int       `json:"value,omitempty"`    // 数値ノードの場合
	Operator *string    `json:"operator,omitempty"` // 演算子ノードの場合
	Left     *ASTNode   `json:"left,omitempty"`     // 左の子ノード
	Right    *ASTNode   `json:"right,omitempty"`    // 右の子ノード
}

// 数値リテラルのASTノード
func NewNumberNode(value int) *ASTNode {
	return &ASTNode{
		Type:  "number",
		Value: &value,
	}
}

// 二項演算子のASTノード
func NewBinaryOpNode(operator string, left, right *ASTNode) *ASTNode {
	return &ASTNode{
		Type:     "binary_op",
		Operator: &operator,
		Left:     left,
		Right:    right,
	}
}

// 数値パーサー（AST構築版）
func ParseNumber() pc.Parser[*ASTNode] {
	return pc.Trace("number", func(pctx *pc.ParseContext[*ASTNode], src []pc.Token[*ASTNode]) (int, []pc.Token[*ASTNode], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("number", "EOF", nil)
		}
		if src[0].Type == "raw" {
			if val, err := strconv.Atoi(src[0].Raw); err == nil {
				node := NewNumberNode(val)
				return 1, []pc.Token[*ASTNode]{{Type: "number", Pos: src[0].Pos, Val: node}}, nil
			}
		}
		return 0, nil, pc.NewErrNotMatch("number", src[0].Raw, src[0].Pos)
	})
}

// 演算子パーサー（AST構築版）
func ParseOperator(op string) pc.Parser[*ASTNode] {
	return pc.Trace("operator_"+op, func(pctx *pc.ParseContext[*ASTNode], src []pc.Token[*ASTNode]) (int, []pc.Token[*ASTNode], error) {
		if len(src) == 0 {
			return 0, nil, pc.NewErrNotMatch("operator_"+op, "EOF", nil)
		}
		if src[0].Type == "raw" && src[0].Raw == op {
			return 1, []pc.Token[*ASTNode]{{Type: "operator", Pos: src[0].Pos, Raw: op}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("operator_"+op, src[0].Raw, src[0].Pos)
	})
}

// 因子（数値のみ）
func ParseFactor() pc.Parser[*ASTNode] {
	return ParseNumber()
}

// 項（因子の乗除）- 左結合でAST構築
func ParseTerm() pc.Parser[*ASTNode] {
	return pc.Trans(
		pc.Seq(
			ParseFactor(),
			pc.ZeroOrMore("mul_div_ops",
				pc.Seq(pc.Or(ParseOperator("*"), ParseOperator("/")), ParseFactor()),
			),
		),
		func(pctx *pc.ParseContext[*ASTNode], tokens []pc.Token[*ASTNode]) ([]pc.Token[*ASTNode], error) {
			// 最初の因子から開始
			result := tokens[0].Val
			
			// 続く演算子と因子のペアを左結合で処理
			for i := 1; i < len(tokens); i += 2 {
				if i+1 < len(tokens) {
					operator := tokens[i].Raw
					rightOperand := tokens[i+1].Val
					
					// 新しい二項演算ノードを作成
					result = NewBinaryOpNode(operator, result, rightOperand)
				}
			}
			
			return []pc.Token[*ASTNode]{{Type: "ast", Pos: tokens[0].Pos, Val: result}}, nil
		},
	)
}

// 式（項の加減）- 左結合でAST構築
func ParseExpression() pc.Parser[*ASTNode] {
	return pc.Trans(
		pc.Seq(
			ParseTerm(),
			pc.ZeroOrMore("add_sub_ops",
				pc.Seq(pc.Or(ParseOperator("+"), ParseOperator("-")), ParseTerm()),
			),
		),
		func(pctx *pc.ParseContext[*ASTNode], tokens []pc.Token[*ASTNode]) ([]pc.Token[*ASTNode], error) {
			// 最初の項から開始
			result := tokens[0].Val
			
			// 続く演算子と項のペアを左結合で処理
			for i := 1; i < len(tokens); i += 2 {
				if i+1 < len(tokens) {
					operator := tokens[i].Raw
					rightOperand := tokens[i+1].Val
					
					// 新しい二項演算ノードを作成
					result = NewBinaryOpNode(operator, result, rightOperand)
				}
			}
			
			return []pc.Token[*ASTNode]{{Type: "ast", Pos: tokens[0].Pos, Val: result}}, nil
		},
	)
}

func main() {
	context := pc.NewParseContext[*ASTNode]()
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
		{"連続掛け算", []string{"2", "*", "3", "*", "4"}},
	}

	fmt.Println("=== 数式→JSON AST コンパイラ ===")
	fmt.Println("数式をパースしてJSON形式のAST（抽象構文木）を生成します")
	fmt.Println("演算子の優先度: * / > + -")
	fmt.Println("結合性: 左結合")
	fmt.Println("（注意: 括弧はサポートしていません）")
	fmt.Println()

	for _, tc := range testCases {
		fmt.Printf("=== %s ===\n", tc.name)
		fmt.Printf("入力: %s\n", joinTokens(tc.input))
		
		result, err := pc.EvaluateWithRawTokens(context, tc.input, ParseExpression())
		if err != nil {
			fmt.Printf("パースエラー: %v\n", err)
		} else if len(result) > 0 && result[0] != nil {
			// ASTをJSONに変換
			jsonData, jsonErr := json.MarshalIndent(result[0], "", "  ")
			if jsonErr != nil {
				fmt.Printf("JSON変換エラー: %v\n", jsonErr)
			} else {
				fmt.Printf("JSON AST:\n%s\n", string(jsonData))
			}
		}
		fmt.Println()
		
		// 新しいコンテキストを作成
		context = pc.NewParseContext[*ASTNode]()
		context.TraceEnable = false
	}

	// エラーハンドリングのテスト
	fmt.Println("=== エラーハンドリングのテスト ===")
	errorCases := []struct {
		name  string
		input []string
	}{
		{"不完全な式", []string{"10", "+"}},
		{"空の式", []string{}},
		{"演算子のみ", []string{"+"}},
		{"無効な文字", []string{"abc"}},
	}

	for _, tc := range errorCases {
		fmt.Printf("テスト: %s\n", tc.name)
		fmt.Printf("入力: %s\n", joinTokens(tc.input))
		
		context = pc.NewParseContext[*ASTNode]()
		result, err := pc.EvaluateWithRawTokens(context, tc.input, ParseExpression())
		if err != nil {
			fmt.Printf("期待通りのエラー: %v\n", err)
		} else if len(result) > 0 {
			fmt.Printf("予期しない成功\n")
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
