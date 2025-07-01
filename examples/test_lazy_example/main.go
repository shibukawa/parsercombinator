package main

import (
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
)

// ASTノードの定義
type Entity interface {
	String() string
}

type ExprNode interface {
	Entity
	Eval() int
}

type LiteralNode struct {
	Value int
}

func (n *LiteralNode) String() string {
	return fmt.Sprintf("%d", n.Value)
}

func (n *LiteralNode) Eval() int {
	return n.Value
}

type BinaryOpNode struct {
	Left  ExprNode
	Op    string
	Right ExprNode
}

func (n *BinaryOpNode) String() string {
	return fmt.Sprintf("(%s %s %s)", n.Left.String(), n.Op, n.Right.String())
}

func (n *BinaryOpNode) Eval() int {
	left := n.Left.Eval()
	right := n.Right.Eval()
	switch n.Op {
	case "+":
		return left + right
	case "-":
		return left - right
	case "*":
		return left * right
	case "/":
		return left / right
	default:
		return 0
	}
}

type OperatorNode struct {
	Op string
}

func (n *OperatorNode) String() string {
	return n.Op
}

// パーサー関数
func literal() pc.Parser[Entity] {
	return pc.Trace("literal", func(pctx *pc.ParseContext[Entity], src []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(src) > 0 && src[0].Type == "raw" {
			if val, err := strconv.Atoi(src[0].Raw); err == nil {
				node := &LiteralNode{Value: val}
				return 1, []pc.Token[Entity]{{
					Type: "literal",
					Pos:  src[0].Pos,
					Val:  node,
				}}, nil
			}
		}
		return 0, nil, pc.NewErrNotMatch("literal", "other", src[0].Pos)
	})
}

func leftParen() pc.Parser[Entity] {
	return pc.Trace("lparen", func(pctx *pc.ParseContext[Entity], src []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(src) > 0 && src[0].Type == "raw" && src[0].Raw == "(" {
			return 1, []pc.Token[Entity]{{Type: "lparen", Pos: src[0].Pos, Raw: "("}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("(", "other", nil)
	})
}

func rightParen() pc.Parser[Entity] {
	return pc.Trace("rparen", func(pctx *pc.ParseContext[Entity], src []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(src) > 0 && src[0].Type == "raw" && src[0].Raw == ")" {
			return 1, []pc.Token[Entity]{{Type: "rparen", Pos: src[0].Pos, Raw: ")"}}, nil
		}
		return 0, nil, pc.NewErrNotMatch(")", "other", nil)
	})
}

func operator() pc.Parser[Entity] {
	return pc.Trace("operator", func(pctx *pc.ParseContext[Entity], src []pc.Token[Entity]) (int, []pc.Token[Entity], error) {
		if len(src) > 0 && src[0].Type == "raw" {
			switch src[0].Raw {
			case "+", "-", "*", "/":
				// OperatorNodeを作成してEntity interfaceを実装
				opNode := &OperatorNode{Op: src[0].Raw}
				return 1, []pc.Token[Entity]{{
					Type: "operator",
					Pos:  src[0].Pos,
					Raw:  src[0].Raw,
					Val:  opNode,
				}}, nil
			}
		}
		return 0, nil, pc.NewErrNotMatch("operator", "other", nil)
	})
}

var expression pc.Parser[Entity]

func init() {
	primary := pc.Or(
		literal(),
		// 括弧付き式: Lazyで遅延評価
		pc.Trans(
			pc.Seq(
				leftParen(),
				pc.Lazy(func() pc.Parser[Entity] { return expression }),
				rightParen(),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				return []pc.Token[Entity]{tokens[1]}, nil
			},
		),
	)

	// 右再帰で二項演算を定義
	expression = pc.Or(
		// 二項演算: primary operator expression
		pc.Trans(
			pc.Seq(
				primary,
				operator(),
				pc.Lazy(func() pc.Parser[Entity] { return expression }),
			),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				left := tokens[0].Val.(ExprNode)
				op := tokens[1].Val.(*OperatorNode).Op
				right := tokens[2].Val.(ExprNode)

				binaryNode := &BinaryOpNode{Left: left, Op: op, Right: right}
				return []pc.Token[Entity]{{
					Type: "expression",
					Pos:  tokens[0].Pos,
					Val:  binaryNode,
				}}, nil
			},
		),
		primary,
	)
}

func main() {
	fmt.Println("=== pc.Lazy を使った再帰式パーサーのデモ ===")

	context := pc.NewParseContext[Entity]()
	// context.TraceEnable = true // トレースを無効化してクリーンな出力

	// 簡単な式をテスト
	fmt.Println("\n1. 簡単な式: 10 + 5")
	input := []string{"10", "+", "5"}
	result, err := pc.EvaluateWithRawTokens(context, input, expression)

	if err != nil {
		fmt.Printf("エラー: %v\n", err)
		return
	}

	expr := result[0].(ExprNode)
	fmt.Printf("構文解析結果: %s\n", expr.String()) // (10 + 5)
	fmt.Printf("計算結果: %d\n", expr.Eval())     // 15

	// より複雑な式をテスト
	fmt.Println("\n2. 括弧付き式: (1 + 2) * 3")
	input2 := []string{"(", "1", "+", "2", ")", "*", "3"}
	context2 := pc.NewParseContext[Entity]()
	result2, err2 := pc.EvaluateWithRawTokens(context2, input2, expression)

	if err2 != nil {
		fmt.Printf("エラー: %v\n", err2)
		return
	}

	expr2 := result2[0].(ExprNode)
	fmt.Printf("構文解析結果: %s\n", expr2.String()) // ((1 + 2) * 3)
	fmt.Printf("計算結果: %d\n", expr2.Eval())     // 9

	// 入れ子の括弧をテスト
	fmt.Println("\n3. 入れ子の括弧: ((1 + 2) * (3 + 4)) / 5")
	input3 := []string{"(", "(", "1", "+", "2", ")", "*", "(", "3", "+", "4", ")", ")", "/", "5"}
	context3 := pc.NewParseContext[Entity]()
	result3, err3 := pc.EvaluateWithRawTokens(context3, input3, expression)

	if err3 != nil {
		fmt.Printf("エラー: %v\n", err3)
		return
	}

	expr3 := result3[0].(ExprNode)
	fmt.Printf("構文解析結果: %s\n", expr3.String()) // (((1 + 2) * (3 + 4)) / 5)
	fmt.Printf("計算結果: %d\n", expr3.Eval())     // 7 (21 / 5 = 4 in integer division, but (3*7=21, 21/5=4.2, but we have 3*7=21, so result should be 21/5=4)

	fmt.Println("\n✅ pc.Lazy が正常に動作して、再帰的な式解析が可能になりました！")
}
