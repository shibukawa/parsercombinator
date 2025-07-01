package main

import (
	"fmt"
	"strconv"
	pc "github.com/shibukawa/parsercombinator"
)

// 基本型定義
type Entity interface {
	String() string
}

type LiteralNode struct {
	Value int
}

func (n *LiteralNode) String() string {
	return fmt.Sprintf("%d", n.Value)
}

// 基本パーサー
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

// アプローチ1: pc.Lazyを使用
func LazyApproach() pc.Parser[Entity] {
	var expression pc.Parser[Entity]
	
	expression = pc.Or(
		literal(),
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
	
	return expression
}

// アプローチ2: pc.NewAliasを使用
func AliasApproach() pc.Parser[Entity] {
	defineExpr, exprAlias := pc.NewAlias[Entity]("expression")
	
	primaryExpr := pc.Or(
		literal(),
		pc.Trans(
			pc.Seq(leftParen(), exprAlias, rightParen()),
			func(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
				return []pc.Token[Entity]{tokens[1]}, nil
			},
		),
	)
	
	return defineExpr(primaryExpr)
}

// 相互再帰の例（NewAliasが必要なケース）
func MutualRecursionExample() {
	fmt.Println("\n=== 相互再帰の例（NewAliasが必要）===")
	
	// A → "a" B | "a"
	// B → "b" A | "b"
	defineA, aliasA := pc.NewAlias[string]("A")
	defineB, aliasB := pc.NewAlias[string]("B")
	
	parserA := defineA(pc.Or(
		pc.Trans(
			pc.Seq(
				pc.Trace("a", func(pctx *pc.ParseContext[string], src []pc.Token[string]) (int, []pc.Token[string], error) {
					if len(src) > 0 && src[0].Type == "raw" && src[0].Raw == "a" {
						return 1, []pc.Token[string]{{Type: "a", Val: "a", Pos: src[0].Pos}}, nil
					}
					return 0, nil, pc.NewErrNotMatch("a", "other", nil)
				}),
				aliasB,
			),
			func(pctx *pc.ParseContext[string], tokens []pc.Token[string]) ([]pc.Token[string], error) {
				return []pc.Token[string]{{Type: "AB", Val: "a" + tokens[1].Val, Pos: tokens[0].Pos}}, nil
			},
		),
		pc.Trace("a", func(pctx *pc.ParseContext[string], src []pc.Token[string]) (int, []pc.Token[string], error) {
			if len(src) > 0 && src[0].Type == "raw" && src[0].Raw == "a" {
				return 1, []pc.Token[string]{{Type: "a", Val: "a", Pos: src[0].Pos}}, nil
			}
			return 0, nil, pc.NewErrNotMatch("a", "other", nil)
		}),
	))
	
	parserB := defineB(pc.Or(
		pc.Trans(
			pc.Seq(
				pc.Trace("b", func(pctx *pc.ParseContext[string], src []pc.Token[string]) (int, []pc.Token[string], error) {
					if len(src) > 0 && src[0].Type == "raw" && src[0].Raw == "b" {
						return 1, []pc.Token[string]{{Type: "b", Val: "b", Pos: src[0].Pos}}, nil
					}
					return 0, nil, pc.NewErrNotMatch("b", "other", nil)
				}),
				aliasA,
			),
			func(pctx *pc.ParseContext[string], tokens []pc.Token[string]) ([]pc.Token[string], error) {
				return []pc.Token[string]{{Type: "BA", Val: "b" + tokens[1].Val, Pos: tokens[0].Pos}}, nil
			},
		),
		pc.Trace("b", func(pctx *pc.ParseContext[string], src []pc.Token[string]) (int, []pc.Token[string], error) {
			if len(src) > 0 && src[0].Type == "raw" && src[0].Raw == "b" {
				return 1, []pc.Token[string]{{Type: "b", Val: "b", Pos: src[0].Pos}}, nil
			}
			return 0, nil, pc.NewErrNotMatch("b", "other", nil)
		}),
	))
	
	// parserBも定義されていることを示すため
	_ = parserB
	
	testCases := [][]string{
		{"a"},
		{"b"},
		{"a", "b"},
		{"b", "a"},
		{"a", "b", "a"},
		{"b", "a", "b"},
	}
	
	for _, tc := range testCases {
		context := pc.NewParseContext[string]()
		result, err := pc.EvaluateWithRawTokens(context, tc, parserA)
		
		if err != nil {
			fmt.Printf("%v: エラー - %v\n", tc, err)
		} else {
			fmt.Printf("%v: 成功 - %s\n", tc, result[0])
		}
	}
}

func main() {
	fmt.Println("=== pc.Lazy vs pc.NewAlias の比較デモ ===")

	testCases := []struct {
		name  string
		input []string
	}{
		{"単純な数値", []string{"42"}},
		{"括弧付き数値", []string{"(", "42", ")"}},
		{"入れ子の括弧", []string{"(", "(", "42", ")", ")"}},
	}

	for _, approach := range []struct {
		name   string
		parser pc.Parser[Entity]
	}{
		{"pc.Lazy", LazyApproach()},
		{"pc.NewAlias", AliasApproach()},
	} {
		fmt.Printf("\n--- %s アプローチ ---\n", approach.name)
		
		for _, tc := range testCases {
			context := pc.NewParseContext[Entity]()
			result, err := pc.EvaluateWithRawTokens(context, tc.input, approach.parser)
			
			if err != nil {
				fmt.Printf("%s: エラー - %v\n", tc.name, err)
			} else {
				fmt.Printf("%s: 成功 - %s\n", tc.name, result[0].String())
			}
		}
	}
	
	// 相互再帰の例
	MutualRecursionExample()
	
	fmt.Println("\n=== 結論 ===")
	fmt.Println("✅ pc.Lazy: 単純な自己再帰に適している（軽量、直接的）")
	fmt.Println("✅ pc.NewAlias: 相互再帰や複雑な文法定義に必要（名前付き、構造化）")
	fmt.Println("💡 両方とも有用で、使い分けが重要")
}
