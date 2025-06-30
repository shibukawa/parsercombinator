// ZeroOrMore Example - コンマ区切りの数値リストをパースする例
package main

import (
	"fmt"
	"strconv"

	pc "github.com/shibukawa/parsercombinator"
)

// 数値パーサー
func Number() pc.Parser[int] {
	return pc.Trace("number", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if src[0].Type == "raw" {
			if i, err := strconv.Atoi(src[0].Raw); err == nil {
				return 1, []pc.Token[int]{{Type: "number", Pos: src[0].Pos, Val: i}}, nil
			}
		}
		return 0, nil, pc.NewErrNotMatch("number", src[0].Raw, src[0].Pos)
	})
}

// コンマパーサー
func Comma() pc.Parser[int] {
	return pc.Trace("comma", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
		if src[0].Type == "raw" && src[0].Raw == "," {
			return 1, []pc.Token[int]{{Type: "comma", Pos: src[0].Pos, Raw: ","}}, nil
		}
		return 0, nil, pc.NewErrNotMatch("comma", src[0].Raw, src[0].Pos)
	})
}

// コンマの後の数値パーサー
func CommaAndNumber() pc.Parser[int] {
	return pc.Trans(
		pc.Seq(Comma(), Number()),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			// コンマを無視し、数値のみを返す
			return []pc.Token[int]{tokens[1]}, nil
		},
	)
}

// 数値リストパーサー (最初の数値 + 0個以上の「,数値」の組み合わせ)
func NumberList() pc.Parser[int] {
	return pc.Trans(
		pc.Seq(
			Number(), // 最初の数値
			pc.ZeroOrMore("additional-numbers", CommaAndNumber()), // 0個以上の追加数値
		),
		func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
			// すべての数値を統合
			var result []pc.Token[int]
			for _, token := range tokens {
				if token.Type == "number" {
					result = append(result, token)
				}
			}
			return result, nil
		},
	)
}

// 空のリストも許可する数値リストパーサー
func OptionalNumberList() pc.Parser[int] {
	return pc.Or(
		NumberList(),
		pc.None[int]("empty-list"), // 空のリスト
	)
}

func main() {
	context := pc.NewParseContext[int]()
	context.TraceEnable = true

	testCases := []struct {
		name  string
		input []string
	}{
		{"単一の数値", []string{"42"}},
		{"複数の数値", []string{"1", ",", "2", ",", "3", ",", "4", ",", "5"}},
		{"空のリスト", []string{}},
		{"数値なし", []string{"abc"}},
		{"不完全なリスト", []string{"1", ",", "2", ","}},
	}

	for _, tc := range testCases {
		fmt.Printf("\n=== %s ===\n", tc.name)
		fmt.Printf("入力: %v\n", tc.input)

		// 通常の数値リスト（最低1個必要）
		result1, err1 := pc.EvaluateWithRawTokens(context, tc.input, NumberList())
		if err1 != nil {
			fmt.Printf("NumberList エラー: %v\n", err1)
		} else {
			var values []int
			for _, token := range result1 {
				values = append(values, token)
			}
			fmt.Printf("NumberList 結果: %v\n", values)
		}

		// 空のリストも許可する数値リスト
		result2, err2 := pc.EvaluateWithRawTokens(context, tc.input, OptionalNumberList())
		if err2 != nil {
			fmt.Printf("OptionalNumberList エラー: %v\n", err2)
		} else {
			var values []int
			for _, token := range result2 {
				values = append(values, token)
			}
			fmt.Printf("OptionalNumberList 結果: %v\n", values)
		}

		context = pc.NewParseContext[int]()
		context.TraceEnable = true
	}

	// ZeroOrMore 単体の動作例
	fmt.Printf("\n=== ZeroOrMore 単体の動作例 ===\n")

	// 0個以上の数値をパース
	zeroOrMoreTests := []struct {
		name  string
		input []string
	}{
		{"数値なし", []string{"abc", "def"}},
		{"数値1個", []string{"123"}},
		{"数値3個", []string{"1", "2", "3"}},
		{"数値2個の後に文字", []string{"1", "2", "abc"}},
	}

	for _, tc := range zeroOrMoreTests {
		fmt.Printf("\n%s: %v\n", tc.name, tc.input)
		result, err := pc.EvaluateWithRawTokens(context, tc.input, pc.ZeroOrMore("numbers", Number()))
		if err != nil {
			fmt.Printf("エラー: %v\n", err)
		} else {
			var values []int
			for _, token := range result {
				values = append(values, token)
			}
			fmt.Printf("結果: %v\n", values)
		}
		context = pc.NewParseContext[int]()
		context.TraceEnable = true
	}
}
