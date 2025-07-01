# Go用パーサコンビネータライブラリ

事前にトークン化された入力から抽象構文木（AST）を構築するために特別に設計された、強力で柔軟なGo用パーサコンビネータライブラリです。

## 特徴

- **トークンベースのパーシング**: 生の文字列ではなく、事前にトークン化された入力で動作
- **型安全**: Goのジェネリクスを活用した型安全性
- **包括的なエラーハンドリング**: カスタムメッセージによる高度なエラー報告
- **スタックオーバーフロー保護**: 無限ループを防ぐための組み込み再帰深度制限
- **デバッグサポート**: 組み込まれたトレース機能
- **復旧メカニズム**: 堅牢なパーシングのためのエラー復旧
- **先読みサポート**: 正と負の先読み操作
- **組み合わせ可能**: 単純なパーサーを複雑なものに簡単に組み合わせ

## インストール

```bash
go get github.com/shibukawa/parsercombinator
```

## クイックスタート

```go
package main

import (
    "fmt"
    "strconv"
    pc "github.com/shibukawa/parsercombinator"
)

// 簡単な計算機パーサーを定義
func main() {
    // 基本パーサーを作成
    digit := pc.Trace("digit", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" {
            i, err := strconv.Atoi(src[0].Raw)
            if err != nil {
                return 0, nil, pc.NewErrNotMatch("integer", src[0].Raw, src[0].Pos)
            }
            return 1, []pc.Token[int]{{Type: "digit", Pos: src[0].Pos, Val: i}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("digit", src[0].Type, src[0].Pos)
    })

    operator := pc.Trace("operator", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" && (src[0].Raw == "+" || src[0].Raw == "-") {
            return 1, []pc.Token[int]{{Type: "operator", Pos: src[0].Pos, Raw: src[0].Raw}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("operator", src[0].Raw, src[0].Pos)
    })

    // パーサーを組み合わせ
    expression := pc.Seq(digit, operator, digit)

    // 入力をパース
    context := pc.NewParseContext[int]()
    result, err := pc.EvaluateWithRawTokens(context, []string{"5", "+", "3"}, expression)
    if err != nil {
        fmt.Printf("エラー: %v\n", err)
        return
    }
    
    fmt.Printf("結果: %v\n", result) // [5, 3] (演算子の値は0)
}
```

## コアコンポーネント

### パーサー関数

核となる型は `Parser[T]` です：

```go
type Parser[T any] func(*ParseContext[T], []Token[T]) (consumed int, newTokens []Token[T], err error)
```

### トークン構造

```go
type Token[T any] struct {
    Type string  // トークンタイプ識別子
    Pos  *Pos    // 位置情報
    Raw  string  // 元の生テキスト
    Val  T       // パースされた値
}
```

### パースコンテキスト

```go
type ParseContext[T any] struct {
    Tokens         []Token[T]     // 入力トークン
    Pos            int            // 現在の位置
    RemainedTokens []Token[T]     // パース後の残りトークン
    Results        []Token[T]     // パース結果トークン
    Traces         []*TraceInfo   // デバッグトレース
    Errors         []*ParseError  // 収集されたエラー
    Depth          int            // 現在の再帰深度
    TraceEnable    bool           // トレースの有効/無効
    MaxDepth       int            // 最大許可再帰深度（0 = 制限なし）
}
```

## 基本コンビネータ

### シーケンス (`Seq`)

パーサーを順番にマッチ：

```go
parser := pc.Seq(digit, operator, digit) // マッチ: digit operator digit
```

### 選択 (`Or`)

パーサーを順番に試し、最初に成功したマッチを返す：

```go
parser := pc.Or(digit, string, identifier) // 選択肢のいずれかにマッチ
```

### Or パーサーによる選択肢の解析

`Or` パーサーは複数の選択肢を試行し、**最も多くのトークンを消費した**パーサーの結果を返します（最長一致）。これにより予測可能な動作と複雑な式パターンとの互換性を確保します。

```go
// 基本的な使用法 - 各パーサーを順番に試行し、最長一致を返す
parser := pc.Or(
    longKeyword,    // 例："interface"
    shortKeyword,   // 例："if"
    identifier,     // 例："interfaceType"
)

// 入力が "interface" の場合、shortKeyword が先に現れても
// longKeyword（9トークン）が shortKeyword（2トークン）より選択される
```

#### 重要な考慮事項

1. **最長一致の動作**: 常に最も多くのトークンを消費する選択肢を返します
2. **順序の独立性**: 曖昧でない文法では、パーサーの順序は重要ではありません
3. **曖昧な文法**: 重複するパターンには注意が必要です

```go
// 良い例: 曖昧でない選択肢
parser := pc.Or(
    stringLiteral,   // "hello"
    numberLiteral,   // 42
    identifier,      // variable
)

// 問題となる可能性: 重複するパターン
parser := pc.Or(
    pc.String("for"),     // 完全一致
    identifier,           // 任意の識別子（"for"を含む）
)
// 解決策: より具体的なパターンを先に置くか、より長い一致を使用
```

#### 式解析での活用

最長一致の動作は式解析で特に有用です：

```go
// 演算子の優先順位を正しく処理する式パーサー
expr := pc.Or(
    binaryExpression,    // "a + b * c" (より長い)
    primaryExpression,   // "a" (より短い)
)
// より長い二項式を正しく選択します
```

### 繰り返し

- `ZeroOrMore`: 0回以上の出現にマッチ
- `OneOrMore`: 1回以上の出現にマッチ
- `Repeat`: 特定の最小/最大回数でマッチ
- `Optional`: 0回または1回の出現にマッチ

```go
numbers := pc.ZeroOrMore("numbers", digit)
requiredNumbers := pc.OneOrMore("required-numbers", digit)
exactlyThree := pc.Repeat("exactly-three", 3, 3, digit)
maybeDigit := pc.Optional(digit)
```

## 再帰パーサーとAliasとLazy

パーサコンビネータでは、再帰的な文法を扱うために2つのアプローチが提供されています：

### pc.Lazy - 単純な自己再帰

単純な自己参照パーサーには `pc.Lazy` を使用します：

```go
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
```

### pc.NewAlias - 複雑な相互再帰

相互再帰や複雑な文法には `NewAlias` を使用します：

```go
// 再帰的な文法を安全に定義
defineExpr, exprAlias := pc.NewAlias[int]("expression")

// 基本式（数値、括弧で囲まれた式）
primaryExpr := pc.Or(
    digit,
    pc.Trans(
        pc.Seq(pc.Literal("("), exprAlias, pc.Literal(")")), // 安全な再帰参照
        func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
            return []pc.Token[int]{tokens[1]}, nil // 内部の式を返す
        },
    ),
)

// エイリアスを定義（これで再帰が完成）
expression := defineExpr(primaryExpr)
```

### ⚠️ 重要：左再帰の回避

**左再帰は無限ループを引き起こす**ため、パーサコンビネータでは必ず回避する必要があります。安全なパーシングのためには、この理解と防止が重要です。

#### 左再帰とは何か？

左再帰は、パーサールールが最初のステップとして直接または間接的に自分自身を呼び出すときに発生します：

```go
// ❌ 危険：直接的な左再帰
expressionBody, expression := pc.NewAlias[int]("expression")
expression = expressionBody(
    pc.Or(
        pc.Seq(expression, operator, expression), // ← 'expression'が最初に自分自身を呼び出す！
        number,
    ),
)

// ❌ 危険：間接的な左再帰
// A → B C, B → A d   （AがBを通じて間接的に自分自身を呼び出す）
defineA, aliasA := pc.NewAlias[int]("A")
defineB, aliasB := pc.NewAlias[int]("B")

parserA := defineA(pc.Seq(aliasB, pc.Literal("C")))
parserB := defineB(pc.Seq(aliasA, pc.Literal("d")))  // ← 間接的な再帰！
```

#### 左再帰が危険な理由

1. **無限構築ループ**: パーサーはGoの関数/クロージャです。構築時に左再帰は無限呼び出しチェーンを作成します
2. **スタックオーバーフロー**: パーシングが始まる前にGoランタイムスタックがオーバーフローします
3. **ランタイム保護なし**: これはパーサー構築時に発生し、パース実行時ではありません
4. **サイレント失敗**: 多くの場合、謎めいたスタックオーバーフローエラーとして現れます

#### 安全な式パーシングパターン

代わりに**優先順位クライミング**と**右再帰**を使用します：

```go
// ✅ 安全：反復パターンを使った優先順位クライミング
func CreateSafeExpressionParser() pc.Parser[int] {
    defineExpr, exprAlias := pc.NewAlias[int]("expr")
    
    // 基本式（最高優先順位）
    primaryExpr := pc.Or(
        Number(),
        pc.Trans( // 括弧で囲まれた式
            pc.Seq(pc.Literal("("), exprAlias, pc.Literal(")")),
            func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
                return []pc.Token[int]{tokens[1]}, nil
            },
        ),
    )
    
    // 乗算/除算（高優先順位）
    mulExpr := pc.Trans(
        pc.Seq(
            primaryExpr,
            pc.ZeroOrMore("mul_ops", pc.Seq(
                pc.Or(pc.Literal("*"), pc.Literal("/")),
                primaryExpr,
            )),
        ),
        func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
            result := tokens[0].Val
            for i := 1; i < len(tokens); i += 2 {
                op := tokens[i].Raw
                right := tokens[i+1].Val
                switch op {
                case "*": result *= right
                case "/": result /= right
                }
            }
            return []pc.Token[int]{{Type: "expr", Val: result, Pos: tokens[0].Pos}}, nil
        },
    )
    
    // 加算/減算（低優先順位）
    addExpr := pc.Trans(
        pc.Seq(
            mulExpr,
            pc.ZeroOrMore("add_ops", pc.Seq(
                pc.Or(pc.Literal("+"), pc.Literal("-")),
                mulExpr,
            )),
        ),
        func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
            result := tokens[0].Val
            for i := 1; i < len(tokens); i += 2 {
                op := tokens[i].Raw
                right := tokens[i+1].Val
                switch op {
                case "+": result += right
                case "-": result -= right
                }
            }
            return []pc.Token[int]{{Type: "result", Val: result, Pos: tokens[0].Pos}}, nil
        },
    )
    
    // 再帰を完成
    defineExpr(addExpr)
    return addExpr
}
```

#### いつLazyを使い、いつNewAliasを使うか

**pc.Lazy を使う場合:**
- 単純な自己再帰文法
- 1つの規則内での再帰
- デバッグがシンプルでよい場合

**pc.NewAlias を使う場合:**
- 相互再帰（複数の規則が互いを参照）
- 複雑で大きな文法
- より良いトレース/デバッグサポートが必要
- チーム開発での保守性を重視

## 高度な機能

### 先読み操作

```go
// 正の先読み - 消費せずにチェック
parser := pc.Seq(
    pc.Lookahead(keyword), // キーワードをチェック
    actualParser,          // 通常通りパース
)

// 負の先読み - 何かが続かないことを確認
parser := pc.Seq(
    identifier,
    pc.NotFollowedBy(digit), // 識別子の後に数字が続かない
)

// ピーク - 消費せずに結果を取得
parser := pc.Seq(
    pc.Peek(nextToken), // 次に来るものを確認
    conditionalParser,   // ピーク結果に基づいてパース
)
```

### エラーハンドリングとユーザーフレンドリーなメッセージ

```go
// より良いエラーメッセージのためのラベル
numberParser := pc.Label("数値", digit)

// 特定のエラーメッセージのためのExpected
parser := pc.Or(
    validExpression,
    pc.Expected[int]("閉じ括弧"),
)

// 明示的な失敗のためのFail
parser := pc.Or(
    implementedFeature,
    pc.Fail[int]("このバージョンでは機能が実装されていません"),
)
```

### エラー復旧

```go
// エラーから復旧してパースを継続
parser := pc.Recover(
    pc.Digit(),        // 前提条件チェック
    parseStatement,    // メインのパースロジック
    pc.Until(";"),     // 復旧: セミコロンまでスキップ
)
```

### 変換

パース結果を変換：

```go
// トークンを変換
addParser := pc.Trans(
    pc.Seq(digit, operator, digit),
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        result := tokens[0].Val + tokens[2].Val // 数値を加算
        return []pc.Token[int]{{
            Type: "result",
            Pos:  tokens[0].Pos,
            Val:  result,
        }}, nil
    },
)
```

### エイリアスによる再帰パーサー

```go
// 再帰文法を定義
expressionBody, expression := pc.NewAlias[int]("expression")

parser := expressionBody(
    pc.Or(
        digit,
        pc.Seq(
            pc.Literal("("),
            expression, // 再帰参照
            pc.Literal(")"),
        ),
    ),
)
```

### 左再帰の問題と解決方法

**注意**: 直接的な左再帰は無限ループを引き起こします。以下のような定義は避ける必要があります：

```go
// ❌ 危険: 左再帰による無限ループ
expressionBody, expression := pc.NewAlias[Entity]("expression")
expression = expressionBody(
    pc.Or(
        pc.Seq(expression, operator(), expression), // ← 最初のexpressionで無限ループ！
        pc.Seq(leftParen(), expression, rightParen()),
        literal(),
    ),
)
```

#### 解決策1: 右再帰による書き換え

左再帰を右再帰に変換することで問題を解決できます：

### エイリアスを使った再帰パーサー

相互再帰や自己参照文法には、`NewAlias`を使用してパーサー構築時の無限ループを回避します：

```go
// 再帰文法を安全に定義
defineExpr, exprAlias := pc.NewAlias[int]("expression")

// 基本式（数値、括弧付き式）
primaryExpr := pc.Or(
    digit,
    pc.Trans(
        pc.Seq(pc.Literal("("), exprAlias, pc.Literal(")")), // 安全な再帰参照
        func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
            return []pc.Token[int]{tokens[1]}, nil // 内側の式を返す
        },
    ),
)

// エイリアスを定義（これで再帰が完成）
expression := defineExpr(primaryExpr)
```

### ⚠️ 重要: 左再帰の回避

**左再帰は無限ループを引き起こし**、パーサーコンビネータでは必ず回避する必要があります。これを理解し防ぐことは安全なパースングにとって重要です。

#### 左再帰とは何か？

左再帰は、パーサールールが直接的または間接的に、パースンの最初のステップとして自分自身を呼び出すときに発生します：

```go
// ❌ 危険: 直接左再帰
expressionBody, expression := pc.NewAlias[int]("expression")
expression = expressionBody(
    pc.Or(
        pc.Seq(expression, operator, expression), // ← 'expression'が最初に自分自身を呼び出す！
        number,
    ),
)

// ❌ 危険: 間接左再帰
// A → B C, B → A d   （AがBを通じて間接的に自分自身を呼び出す）
defineA, aliasA := pc.NewAlias[int]("A")
defineB, aliasB := pc.NewAlias[int]("B")

parserA := defineA(pc.Seq(aliasB, pc.Literal("C")))
parserB := defineB(pc.Seq(aliasA, pc.Literal("d")))  // ← 間接再帰！
```

#### 左再帰が危険な理由

1. **無限構築ループ**: パーサーはGoの関数/クロージャです。構築時に左再帰は無限呼び出しチェーンを作成します
2. **スタックオーバーフロー**: パースンが始まる前にGoランタイムスタックがオーバーフローします
3. **ランタイム保護なし**: これはパーサー構築時に発生し、パース時ではありません
4. **無音の失敗**: しばしば謎めいたスタックオーバーフローエラーとして現れます

#### 安全な式パースンパターン

代わりに**優先順位クライミング**と**右再帰**を使用します：

```go
// ✅ 安全: 反復パターンによる優先順位クライミング
func CreateSafeExpressionParser() pc.Parser[int] {
    defineExpr, exprAlias := pc.NewAlias[int]("expr")
    
    // 基本式（最高優先度）
    primaryExpr := pc.Or(
        Number(),
        pc.Trans( // 括弧付き式
            pc.Seq(pc.Literal("("), exprAlias, pc.Literal(")")),
            func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
                return []pc.Token[int]{tokens[1]}, nil
            },
        ),
    )
    
    // 乗算・除算（高優先度）
    mulExpr := pc.Trans(
        pc.Seq(
            primaryExpr,
            pc.ZeroOrMore("mul_ops", pc.Seq(
                pc.Or(pc.Literal("*"), pc.Literal("/")),
                primaryExpr,
            )),
        ),
        transformBinaryOps, // 左結合変換
    )
    
    // 加算・減算（低優先度）
    addExpr := pc.Trans(
        pc.Seq(
            mulExpr,
            pc.ZeroOrMore("add_ops", pc.Seq(
                pc.Or(pc.Literal("+"), pc.Literal("-")),
                mulExpr,
            )),
        ),
        transformBinaryOps, // 左結合変換
    )
    
    // 再帰を完成
    defineExpr(addExpr)
    return addExpr
}

// 二項演算を左結合に変換
func transformBinaryOps(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
    result := tokens[0].Val
    for i := 1; i < len(tokens); i += 2 {
        op := tokens[i].Raw
        right := tokens[i+1].Val
        switch op {
        case "+": result += right
        case "-": result -= right
        case "*": result *= right
        case "/": result /= right
        }
    }
    return []pc.Token[int]{{Type: "expr", Val: result, Pos: tokens[0].Pos}}, nil
}
```

#### 重要な安全原則

1. **ZeroOrMore/OneOrMoreを使用**: 直接再帰を繰り返し構造で置き換える
2. **優先順位クライミング**: 再帰ではなくパーサー階層で演算子優先順位を処理
3. **右再帰のみ**: 再帰が必要な場合は左再帰でないことを確認
4. **相互再帰にはNewAlias**: エイリアスを使用して循環依存を安全に切断
5. **早期テスト**: "42"のような簡単な式がスタックオーバーフローなしでパースできるかテスト

#### 完全な安全式パースンの例

動作する安全な式パーサーが`examples/safe_expression/main.go`にあります：

```bash
cd examples/safe_expression
go run main.go
```

これは以下を実証します：
- 入れ子の括弧の安全な処理: `((1+2)*3)+4`
- 正しい演算子優先順位: `1+2*3` → 7（9ではなく）
- 深い入れ子でもスタックオーバーフローなし
- ゼロ除算の適切なエラー処理
        pc.Seq(
            multiplicative,
            pc.ZeroOrMore("add_ops", pc.Seq(addOperator(), multiplicative)),
        ),
        buildLeftAssociativeExpression,
    )

    expression = additive
}

// 左結合の二項演算を構築するヘルパー関数
func buildLeftAssociativeExpression(pctx *pc.ParseContext[Entity], tokens []pc.Token[Entity]) ([]pc.Token[Entity], error) {
    result := tokens[0] // 最初の項

    // ZeroOrMoreの結果を処理（演算子と項のペアの配列）
    if len(tokens) > 1 {
        operations := tokens[1] // ZeroOrMoreの結果
        // operationsの各要素に対して左結合で処理
        for _, op := range operations.Val.([]Operation) {
            operator := op.Operator
            operand := op.Operand
            
            // 新しい二項演算ノードを作成
            result = createBinaryOpNode(result, operator, operand)
        }
    }

    return []pc.Token[Entity]{result}, nil
}
```

#### 解決策3: `pc.Lazy`を使った循環参照の回避

`pc.Lazy`を使用して、パーサーの初期化時点での循環参照を回避できます：

```go
var expression pc.Parser[Entity]

func init() {
    primary := pc.Or(
        literal(),
        identifier(),
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
                op := tokens[1].Val.(string)
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
    context := pc.NewParseContext[Entity]()
    context.TraceEnable = true
    
    input := []string{"10", "+", "5"}
    result, err := pc.EvaluateWithRawTokens(context, input, Expression())
    
    if err != nil {
        fmt.Printf("エラー: %v\n", err)
        return
    }
    
    expr := result[0].(ExprNode)
    fmt.Printf("式: %s\n", expr.String())     // (1 + (2 * 3))
    fmt.Printf("結果: %d\n", expr.Eval())     // 7
}
```

## エラータイプ

ライブラリは複数のエラータイプを定義しています：

- `ErrNotMatch`: パーサーがマッチしない（復旧可能）
- `ErrRepeatCount`: 繰り返し回数が条件を満たさない（復旧可能）
- `ErrCritical`: 致命的エラー（復旧不可能）
- `ErrStackOverflow`: 再帰深度が最大制限を超えた（無限ループを防ぐ）

```go
// カスタムエラーを作成
err := pc.NewErrNotMatch("期待値", "実際値", position)
err := pc.NewErrCritical("致命的エラー", position)
err := pc.NewErrStackOverflow(currentDepth, maxDepth, position)
```

## スタックオーバーフロー保護

ライブラリは、再帰パーサーでの無限ループを防ぐために、スタック深度制限による組み込み保護機能を提供しています。

### 設定

```go
// カスタムスタック深度制限を設定
context := pc.NewParseContext[int]()
context.MaxDepth = 50  // 最大再帰深度を50に設定

// 制限を無効化（0に設定）
context.MaxDepth = 0   // 制限なし（注意して使用）

// デフォルト制限
fmt.Println(pc.NewParseContext[int]().MaxDepth) // 1000（デフォルト）
```

### エラーハンドリング

再帰深度が制限を超えた場合、`ErrStackOverflow`エラーが返されます：

```go
result, err := pc.EvaluateWithRawTokens(context, input, parser)
if err != nil {
    if errors.Is(err, pc.ErrStackOverflow) {
        fmt.Printf("スタックオーバーフローが検出されました: %v\n", err)
        // 無限再帰の場合の処理
    }
}
```

### 使用例

この保護機能は特に以下の場合に有用です：

- **左再帰文法**: 無限左再帰の検出と防止
- **不正な入力**: 予期しない入力パターンでの暴走パーシングの停止
- **開発デバッグ**: 再帰パーサーロジックエラーの早期発見
- **本番環境の安全性**: 悪意のあるまたは不正な入力からのサーバークラッシュ防止

### 例: 保護された再帰パーサー

```go
// 特定の入力で無限ループする可能性があるパーサー
expressionBody, expression := pc.NewAlias[int]("expression")
parser := expressionBody(
    pc.Or(
        pc.Digit(),                                    // ベースケース
        pc.Seq(expression, pc.Operator(), pc.Digit()), // 再帰ケース（左再帰！）
    ),
)

context := pc.NewParseContext[int]()
context.MaxDepth = 10 // デモンストレーション用の低い制限

// これは安全に捕捉されて停止されます
result, err := pc.EvaluateWithRawTokens(context, []string{"+"}, parser)
if errors.Is(err, pc.ErrStackOverflow) {
    fmt.Println("無限再帰が検出され、防止されました！")
}
```

## 完全な例: 数式表現

```go
package main

import (
    "fmt"
    "strconv"
    pc "github.com/shibukawa/parsercombinator"
)

func Digit() pc.Parser[int] {
    return pc.Trace("digit", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" {
            i, err := strconv.Atoi(src[0].Raw)
            if err != nil {
                return 0, nil, pc.NewErrNotMatch("整数", src[0].Raw, src[0].Pos)
            }
            return 1, []pc.Token[int]{{Type: "digit", Pos: src[0].Pos, Val: i}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("数値", src[0].Type, src[0].Pos)
    })
}

func Operator() pc.Parser[int] {
    return pc.Trace("operator", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" {
            switch src[0].Raw {
            case "+", "-", "*", "/":
                return 1, []pc.Token[int]{{Type: "operator", Pos: src[0].Pos, Raw: src[0].Raw}}, nil
            }
        }
        return 0, nil, pc.NewErrNotMatch("演算子", src[0].Raw, src[0].Pos)
    })
}

func Expression() pc.Parser[int] {
    return pc.Trans(
        pc.Seq(Digit(), Operator(), Digit()),
        func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
            left, op, right := tokens[0].Val, tokens[1].Raw, tokens[2].Val
            var result int
            switch op {
            case "+": result = left + right
            case "-": result = left - right
            case "*": result = left * right
            case "/": result = left / right
            }
            return []pc.Token[int]{{
                Type: "result",
                Pos:  tokens[0].Pos,
                Val:  result,
            }}, nil
        },
    )
}

func main() {
    context := pc.NewParseContext[int]()
    context.TraceEnable = true
    
    input := []string{"10", "+", "5"}
    result, err := pc.EvaluateWithRawTokens(context, input, Expression())
    
    if err != nil {
        fmt.Printf("エラー: %v\n", err)
        return
    }
    
    expr := result[0].(ExprNode)
    fmt.Printf("式: %s\n", expr.String())     // (1 + (2 * 3))
    fmt.Printf("結果: %d\n", expr.Eval())     // 7
}
```

## ベストプラクティス

1. **ラベルを使用**: ユーザー向けパーサーには常に `Label()` を使用して明確なエラーメッセージを提供
2. **トレースを有効化**: 開発中はトレースを使用してパーサーの動作を理解
3. **エラーを適切に処理**: `Expected()` と `Fail()` を使用して意味のあるエラーメッセージを提供
4. **段階的に構成**: 単純でテスト済みのコンポーネントから複雑なパーサーを構築
5. **復旧を使用**: 不正な入力の堅牢なパーシングのためにエラー復旧を実装
6. **型安全性**: Goの型システムを活用してコンパイル時にエラーをキャッチ
7. **⚠️ 変換の安全性**: 無限ループを防ぐため、変換では常にトークンタイプを変更
   - 同じパーサーが再度パースできるトークンを出力しない
   - 安全網としてスタックオーバーフロー保護（`MaxDepth`）を使用
   - 開発中はトレースを有効にして再パースパターンを検出
   - Safeモードでの自動安全性チェックを使用: `context.CheckTransformSafety = true`

## 実践的なコンパイラ構築パターン

### トークン列からAST構築の実例

実際のコンパイラでは、トークン列から段階的にASTを構築することが多く、以下のようなパターンが有効です：

```go
// ASTノードの定義
type ASTNode interface {
    Type() string
    Position() *pc.Pos
    String() string
}

type BinaryOpNode struct {
    pos   *pc.Pos
    left  ASTNode
    op    string
    right ASTNode
}

func (n *BinaryOpNode) Type() string { return "BinaryOp" }
func (n *BinaryOpNode) Position() *pc.Pos { return n.pos }
func (n *BinaryOpNode) String() string { 
    return fmt.Sprintf("(%s %s %s)", n.left.String(), n.op, n.right.String()) 
}

type LiteralNode struct {
    pos   *pc.Pos
    value interface{}
}

func (n *LiteralNode) Type() string { return "Literal" }
func (n *LiteralNode) Position() *pc.Pos { return n.pos }
func (n *LiteralNode) String() string { return fmt.Sprintf("%v", n.value) }

// 段階的AST構築のためのパーサー
func NumberLiteral() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Label("数値リテラル", Digit()),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            // 元のトークンから値を取得し、新しいASTノードを作成
            digitToken := tokens[0]
            astNode := &LiteralNode{
                pos:   digitToken.Pos,
                value: digitToken.Val, // 元のint値を保持
            }
            
            return []pc.Token[ASTNode]{{
                Type: "ast_node",
                Pos:  digitToken.Pos,
                Val:  astNode,
            }}, nil
        },
    )
}

func BinaryExpression() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Seq(NumberLiteral(), Operator(), NumberLiteral()),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            // 既存のASTノードを参照して新しいノードを構築
            leftNode := tokens[0].Val.(ASTNode)    // 旧ノード参照
            opToken := tokens[1]                   // 演算子トークン
            rightNode := tokens[2].Val.(ASTNode)   // 旧ノード参照
            
            // 新しいASTノードを作成
            binaryNode := &BinaryOpNode{
                pos:   leftNode.Position(),
                left:  leftNode,
                op:    opToken.Raw,
                right: rightNode,
            }
            
            return []pc.Token[ASTNode]{{
                Type: "ast_node",
                Pos:  leftNode.Position(),
                Val:  binaryNode,
            }}, nil
        },
    )
}
```

### 複雑なAST構築パターン

より複雑な構造の場合、段階的に構築することで管理しやすくなります：

```go
// 関数呼び出しノード
type FunctionCallNode struct {
    pos       *pc.Pos
    name      string
    arguments []ASTNode
}

func (n *FunctionCallNode) Type() string { return "FunctionCall" }
func (n *FunctionCallNode) Position() *pc.Pos { return n.pos }
func (n *FunctionCallNode) String() string {
    args := make([]string, len(n.arguments))
    for i, arg := range n.arguments {
        args[i] = arg.String()
    }
    return fmt.Sprintf("%s(%s)", n.name, strings.Join(args, ", "))
}

// 引数リストの構築
func ArgumentList() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Seq(
            pc.Literal("("),
            pc.Optional(pc.Seq(
                Expression(),
                pc.ZeroOrMore("additional_args", pc.Seq(pc.Literal(","), Expression())),
            )),
            pc.Literal(")"),
        ),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            var arguments []ASTNode
            
            // オプショナルな引数がある場合
            if len(tokens) > 2 && tokens[1].Type == "ast_node" {
                // 最初の引数
                arguments = append(arguments, tokens[1].Val.(ASTNode))
                
                // 追加の引数（, expression の繰り返し）
                for i := 2; i < len(tokens)-1; i += 2 {
                    if tokens[i].Type == "ast_node" {
                        arguments = append(arguments, tokens[i].Val.(ASTNode))
                    }
                }
            }
            
            // 引数リストを表すメタノードを作成
            argListNode := &ArgumentListNode{
                pos:  tokens[0].Pos,
                args: arguments,
            }
            
            return []pc.Token[ASTNode]{{
                Type: "argument_list",
                Pos:  tokens[0].Pos,
                Val:  argListNode,
            }}, nil
        },
    )
}

// 関数呼び出しの構築
func FunctionCall() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Seq(Identifier(), ArgumentList()),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            nameToken := tokens[0]
            argListNode := tokens[1].Val.(*ArgumentListNode)
            
            funcCallNode := &FunctionCallNode{
                pos:       nameToken.Pos,
                name:      nameToken.Raw,
                arguments: argListNode.args,
            }
            
            return []pc.Token[ASTNode]{{
                Type: "ast_node", 
                Pos:  nameToken.Pos,
                Val:  funcCallNode,
            }}, nil
        },
    )
}
```

### 木構造後の処理パターン

一度木構造になった後の処理については、以下のようなアプローチが有効です：

```go
// Visitor パターンによるAST処理
type ASTVisitor interface {
    VisitBinaryOp(node *BinaryOpNode) error
    VisitLiteral(node *LiteralNode) error
    VisitFunctionCall(node *FunctionCallNode) error
}

// 型チェッカーの例
type TypeChecker struct {
    errors []error
    symbolTable map[string]Type
}

func (tc *TypeChecker) VisitBinaryOp(node *BinaryOpNode) error {
    // 左右の子ノードを再帰的に処理
    if err := node.left.Accept(tc); err != nil {
        return err
    }
    if err := node.right.Accept(tc); err != nil {
        return err
    }
    
    // 型チェックロジック
    leftType := tc.getNodeType(node.left)
    rightType := tc.getNodeType(node.right)
    
    if !tc.isCompatible(leftType, rightType, node.op) {
        return fmt.Errorf("型エラー: %s と %s は演算子 %s で使用できません", 
                         leftType, rightType, node.op)
    }
    
    return nil
}

// Transform パターンによるAST変換
type ASTTransformer interface {
    Transform(node ASTNode) (ASTNode, error)
}

// 最適化器の例
type Optimizer struct{}

func (o *Optimizer) Transform(node ASTNode) (ASTNode, error) {
    switch n := node.(type) {
    case *BinaryOpNode:
        // 定数畳み込み最適化
        if isConstant(n.left) && isConstant(n.right) {
            result := evaluateConstant(n)
            return &LiteralNode{pos: n.pos, value: result}, nil
        }
        
        // 子ノードを再帰的に最適化
        optimizedLeft, err := o.Transform(n.left)
        if err != nil {
            return nil, err
        }
        optimizedRight, err := o.Transform(n.right)
        if err != nil {
            return nil, err
        }
        
        return &BinaryOpNode{
            pos:   n.pos,
            left:  optimizedLeft,
            op:    n.op,
            right: optimizedRight,
        }, nil
        
    default:
        return node, nil
    }
}
```

### マルチパス処理のパターン

実際のコンパイラでは、複数のパスで処理することが一般的です：

```go
// コンパイラのメイン処理
func CompileProgram(input []string) (*Program, error) {
    // パス1: 構文解析（パーサコンビネータ使用）
    context := pc.NewParseContext[ASTNode]()
    ast, err := pc.EvaluateWithRawTokens(context, input, Program())
    if err != nil {
        return nil, fmt.Errorf("構文解析エラー: %w", err)
    }
    
    programNode := ast[0].Val.(*ProgramNode)
    
    // パス2: シンボルテーブル構築
    symbolBuilder := &SymbolTableBuilder{}
    if err := programNode.Accept(symbolBuilder); err != nil {
        return nil, fmt.Errorf("シンボル解析エラー: %w", err)
    }
    
    // パス3: 型チェック
    typeChecker := &TypeChecker{symbolTable: symbolBuilder.table}
    if err := programNode.Accept(typeChecker); err != nil {
        return nil, fmt.Errorf("型チェックエラー: %w", err)
    }
    
    // パス4: 最適化
    optimizer := &Optimizer{}
    optimizedAST, err := optimizer.Transform(programNode)
    if err != nil {
        return nil, fmt.Errorf("最適化エラー: %w", err)
    }
    
    // パス5: コード生成
    codeGenerator := &CodeGenerator{}
    code, err := codeGenerator.Generate(optimizedAST)
    if err != nil {
        return nil, fmt.Errorf("コード生成エラー: %w", err)
    }
    
    return &Program{AST: optimizedAST, Code: code}, nil
}
```

## 構造化データの検証パターン

### ツリー構造の直列化による検証

ご提案いただいたアプローチは非常に有効です。ツリー構造を疑似ノードで直列化してパーサで検証する方法：

```go
// ツリー構造を表現する型
type TreeNode struct {
    Type     string
    Value    interface{}
    Children []*TreeNode
    Pos      *pc.Pos
}

// 直列化用の疑似トークン
type SerializedToken struct {
    Type  string  // "open", "close", "leaf"
    Node  string  // ノード名
    Value interface{}
    Pos   *pc.Pos
}

// ツリーを直列化（DFS順で疑似トークン列に変換）
func SerializeTree(node *TreeNode) []SerializedToken {
    var tokens []SerializedToken
    
    if len(node.Children) == 0 {
        // 葉ノード
        tokens = append(tokens, SerializedToken{
            Type:  "leaf",
            Node:  node.Type,
            Value: node.Value,
            Pos:   node.Pos,
        })
    } else {
        // 内部ノード：開始
        tokens = append(tokens, SerializedToken{
            Type:  "open",
            Node:  node.Type,
            Value: node.Value,
            Pos:   node.Pos,
        })
        
        // 子ノードを再帰的に処理
        for _, child := range node.Children {
            tokens = append(tokens, SerializeTree(child)...)
        }
        
        // 内部ノード：終了
        tokens = append(tokens, SerializedToken{
            Type: "close",
            Node: node.Type,
            Pos:  node.Pos,
        })
    }
    
    return tokens
}

// 直列化されたトークンに対するバリデータ
func ValidateHTMLStructure() pc.Parser[bool] {
    // HTMLタグの開始
    htmlOpen := pc.Trace("html_open", func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
        token := src[0].Val.(SerializedToken)
        if token.Type == "open" && token.Node == "html" {
            return 1, []pc.Token[bool]{{Type: "validated", Val: true}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("HTML開始タグ", token.Node, src[0].Pos)
    })
    
    // HTMLタグの終了
    htmlClose := pc.Trace("html_close", func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
        token := src[0].Val.(SerializedToken)
        if token.Type == "close" && token.Node == "html" {
            return 1, []pc.Token[bool]{{Type: "validated", Val: true}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("HTML終了タグ", token.Node, src[0].Pos)
    })

    // body要素の検証
    bodyElement := pc.Seq(
        pc.Literal("body_open"),
        pc.ZeroOrMore("body_content", pc.Or(textContent, divElement)),
        pc.Literal("body_close"),
    )
    
    // 完全なHTML構造の検証
    return pc.Seq(htmlOpen, headElement, bodyElement, htmlClose)
}

// 検証の実行
func ValidateHTMLTree(tree *TreeNode) error {
    // ツリーを直列化
    tokens := SerializeTree(tree)
    
    // パーサコンビネータで検証
    context := pc.NewParseContext[bool]()
    _, err := pc.EvaluateWithTokens(context, tokens, ValidateHTMLStructure())
    
    return err
}
```

### スキーマベースの構造検証

より一般的なスキーマ検証のパターン：

```go
// スキーマ定義
type Schema struct {
    Type       string             // "object", "array", "string", etc.
    Properties map[string]*Schema // オブジェクトのプロパティ
    Items      *Schema            // 配列の要素スキーマ
    Required   []string           // 必須フィールド
    MinItems   int               // 配列の最小要素数
    MaxItems   int               // 配列の最大要素数
}

// JSON風のデータ構造
type DataNode struct {
    Type  string                 // "object", "array", "string", "number", "boolean"
    Value interface{}            // 実際の値
    Props map[string]*DataNode   // オブジェクトのプロパティ
    Items []*DataNode            // 配列の要素
    Pos   *pc.Pos
}

// スキーマ検証用のパーサ生成
func CreateSchemaValidator(schema *Schema) pc.Parser[bool] {
    return pc.Trace(fmt.Sprintf("validate_%s", schema.Type), 
        func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
            data := src[0].Val.(*DataNode)
            
            // 型チェック
            if data.Type != schema.Type {
                return 0, nil, pc.NewErrNotMatch(
                    fmt.Sprintf("型 %s", schema.Type), 
                    data.Type, 
                    data.Pos,
                )
            }
            
            switch schema.Type {
            case "object":
                return validateObject(schema, data, pctx)
            case "array":
                return validateArray(schema, data, pctx)
            default:
                return validatePrimitive(schema, data)
            }
        })
}

func validateObject(schema *Schema, data *DataNode, pctx *pc.ParseContext[bool]) (int, []pc.Token[bool], error) {
    // 必須フィールドの検証
    for _, required := range schema.Required {
        if _, exists := data.Props[required]; !exists {
            return 0, nil, pc.NewErrCritical(
                fmt.Sprintf("必須フィールド '%s' が見つかりません", required),
                data.Pos,
            )
        }
    }
    
    // 各プロパティの検証
    for propName, propData := range data.Props {
        propSchema, exists := schema.Properties[propName]
        if !exists {
            return 0, nil, pc.NewErrNotMatch(
                "有効なプロパティ",
                propName,
                propData.Pos,
            )
        }
        
        // 再帰的にプロパティを検証
        validator := CreateSchemaValidator(propSchema)
        _, _, err := validator(pctx, []pc.Token[bool]{{Val: propData, Pos: propData.Pos}})
        if err != nil {
            return 0, nil, fmt.Errorf("プロパティ '%s': %w", propName, err)
        }
    }
    
    return 1, []pc.Token[bool]{{Type: "validated_object", Val: true}}, nil
}

// 設定ファイルの検証例
func ValidateConfigFile() pc.Parser[bool] {
    // 設定ファイルのスキーマ定義
    configSchema := &Schema{
        Type: "object",
        Required: []string{"server", "database"},
        Properties: map[string]*Schema{
            "server": {
                Type: "object",
                Required: []string{"host", "port"},
                Properties: map[string]*Schema{
                    "host": {Type: "string"},
                    "port": {Type: "number"},
                    "ssl":  {Type: "boolean"},
                },
            },
            "database": {
                Type: "object",
                Required: []string{"url"},
                Properties: map[string]*Schema{
                    "url":         {Type: "string"},
                    "max_connections": {Type: "number"},
                },
            },
        },
    }
    
    return pc.Label("設定ファイル", CreateSchemaValidator(configSchema))
}
```

### フラット構造での部分検証

既存の構造化データに対して部分的な検証を行う方法：

```go
// CSVデータの行検証
type CSVRow struct {
    Fields []string
    LineNo int
}

func ValidateCSVRow(expectedColumns []string, validators map[string]pc.Parser[bool]) pc.Parser[bool] {
    return pc.Trans(
        pc.Trace("csv_row", func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
            row := src[0].Val.(*CSVRow)
            
            // カラム数チェック
            if len(row.Fields) != len(expectedColumns) {
                return 0, nil, pc.NewErrNotMatch(
                    fmt.Sprintf("%d個のフィールド", len(expectedColumns)),
                    fmt.Sprintf("%d個のフィールド", len(row.Fields)),
                    &pc.Pos{Line: row.LineNo},
                )
            }
            
            // 各フィールドの検証
            for i, field := range row.Fields {
                columnName := expectedColumns[i]
                if validator, exists := validators[columnName]; exists {
                    fieldToken := pc.Token[bool]{
                        Type: "field",
                        Raw:  field,
                        Pos:  &pc.Pos{Line: row.LineNo, Column: i + 1},
                        Val:  field,
                    }
                    
                    _, _, err := validator(pctx, []pc.Token[bool]{fieldToken})
                    if err != nil {
                        return 0, nil, fmt.Errorf("列 '%s' (行%d): %w", columnName, row.LineNo, err)
                    }
                }
            }
            
            return 1, []pc.Token[bool]{{Type: "validated_row", Val: true}}, nil
        }),
        func(pctx *pc.ParseContext[bool], tokens []pc.Token[bool]) ([]pc.Token[bool], error) {
            return tokens, nil
        },
    )
}

// 使用例：ユーザーデータCSVの検証
func CreateUserCSVValidator() pc.Parser[bool] {
    columns := []string{"name", "email", "age", "active"}
    
    validators := map[string]pc.Parser[bool]{
        "name": pc.Label("ユーザー名", validateNonEmptyString()),
        "email": pc.Label("メールアドレス", validateEmail()),
        "age": pc.Label("年齢", validatePositiveNumber()),
        "active": pc.Label("有効フラグ", validateBoolean()),
    }
    
    return pc.OneOrMore("csv_rows", ValidateCSVRow(columns, validators))
}
```

### リアルタイム検証パターン

ストリーミングデータやリアルタイムデータの検証：

```go
// イベントストリームの検証
type Event struct {
    Type      string
    Timestamp time.Time
    Data      interface{}
    Pos       *pc.Pos
}

// 状態機械による順序検証
func ValidateEventSequence() pc.Parser[bool] {
    // ユーザーログインフローの検証
    loginFlow := pc.Seq(
        pc.Label("ログイン開始", expectEvent("login_start")),
        pc.Optional(pc.Label("認証試行", expectEvent("auth_attempt"))),
        pc.Or(
            pc.Label("ログイン成功", expectEvent("login_success")),
            pc.Seq(
                pc.Label("ログイン失敗", expectEvent("login_failure")),
                pc.Optional(pc.Label("再試行", ValidateEventSequence())), // 再帰的に再試行を許可
            ),
        ),
    )
    
    return loginFlow
}

func expectEvent(eventType string) pc.Parser[bool] {
    return pc.Trace(eventType, func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
        event := src[0].Val.(*Event)
        if event.Type == eventType {
            return 1, []pc.Token[bool]{{Type: "validated_event", Val: true}}, nil
        }
        return 0, nil, pc.NewErrNotMatch(eventType, event.Type, event.Pos)
    })
}
```

これらのパターンにより、様々な構造化データの検証が可能になります：

1. **ツリー直列化**: 複雑な階層構造の検証
2. **スキーマベース**: JSON/XML風データの型安全検証  
3. **フラット構造**: CSV/TSVなどの表形式データ検証
4. **リアルタイム**: イベントストリームや状態遷移の検証

## 使用事例

このライブラリは以下のような場面で特に有用です：

- **DSL（ドメイン固有言語）パーサー**: 設定ファイル、クエリ言語、テンプレート言語
- **プログラミング言語の構文解析**: 言語処理系の構文解析フェーズ
- **データフォーマットの解析**: カスタムデータフォーマットの構造解析
- **コード生成ツール**: テンプレートや仕様からのコード生成
- **バリデーション**: 構造化データの検証とエラー報告

## 設計思想

このライブラリは以下の設計原則に基づいています：

1. **トークンベースアプローチ**: 字句解析と構文解析の分離により、より良い性能とエラー報告を実現
2. **型安全性**: Goのジェネリクスを活用してコンパイル時の安全性を保証
3. **組み合わせ可能性**: 小さく再利用可能なコンポーネントから複雑なパーサーを構築
4. **エラーファースト**: 優れたエラー報告とデバッグ機能を重視
5. **実用性**: 実際のプロダクションコードで使用できる堅牢性と性能

## ライセンス

Apache 2.0 License - 詳細はLICENSEファイルを参照してください。

## 貢献

貢献を歓迎します！プルリクエストをお気軽に提出してください。

## 関連リソース

- [英語版README](README.md)
- [APIドキュメント](https://pkg.go.dev/github.com/shibukawa/parsercombinator)
- [サンプルコード](examples/)

#### パフォーマンスモード

Or パーサーは安全性とパフォーマンスのバランスを取るために異なるモードをサポートします：

```go
pc := pc.NewParseContext[int]()

// Safeモード（デフォルト） - 一貫した動作のために最長一致を使用
pc.OrMode = pc.OrModeSafe

// Fastモード - より良いパフォーマンスのために最初のマッチを使用
pc.OrMode = pc.OrModeFast

// TryFastモード - 最初のマッチを使用するが、最長一致と異なる場合に警告
pc.OrMode = pc.OrModeTryFast
```

**モードの比較:**

- **Safeモード**（デフォルト）: 常に最長一致を選択。最も安全で予測可能。
- **Fastモード**: 最初に成功したマッチを選択。パフォーマンスが向上するが、パーサーの順序に注意が必要。
- **TryFastモード**: 最初のマッチを使用するが、最長一致と異なる選択をする場合に最適化の提案を表示。

**TryFastモードの警告例:**
```
⚠️  Or parser optimization suggestion at myparser.go:42 (parser position 0):
   Fast mode chose option 1 (consumed 2 tokens), but longest match would choose option 2 (consumed 3 tokens).
   For Fast mode compatibility, consider moving option 2 before option 1 in your Or(...) call.
```

**ヘルパー関数:**
```go
// モードを簡単に設定
pc.SetSafeMode()    // 最長一致（デフォルト）
pc.SetFastMode()    // 最初のマッチ
pc.SetTryFastMode() // 最初のマッチ（警告付き）
```

#### ⚠️ 重要: 変換での無限ループの回避

`Trans()` を使用してトークンを変換する際は、**トークンの互換性**に細心の注意を払い、無限ループを避けてください：

```go
// ❌ 危険: 無限ループを引き起こす可能性
badParser := pc.Trans(
    pc.Trace("digit", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" {
            i, err := strconv.Atoi(src[0].Raw)
            if err != nil {
                return 0, nil, pc.NewErrNotMatch("integer", src[0].Raw, src[0].Pos)
            }
            // ❌ 問題: 再パース可能な "raw" タイプを生成
            return 1, []pc.Token[int]{{Type: "raw", Pos: src[0].Pos, Raw: src[0].Raw, Val: i}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("digit", src[0].Type, src[0].Pos)
    }),
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        // 変換されたトークンがまだ "raw" タイプ - 再度パースされる可能性！
        return tokens, nil // ❌ 無限ループのリスクを作る
    },
)

// ✅ 安全: 再パースを防ぐためにトークンタイプを変更
goodParser := pc.Trans(
    pc.Trace("digit", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" {
            i, err := strconv.Atoi(src[0].Raw)
            if err != nil {
                return 0, nil, pc.NewErrNotMatch("integer", src[0].Raw, src[0].Pos)
            }
            return 1, []pc.Token[int]{{Type: "digit", Pos: src[0].Pos, Val: i}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("digit", src[0].Type, src[0].Pos)
    }),
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        // ✅ 安全: トークンタイプが "digit" で "raw" ではない - 再パースされない
        return []pc.Token[int]{{
            Type: "number",  // 異なるタイプで再パースを防止
            Pos:  tokens[0].Pos,
            Val:  tokens[0].Val,
        }}, nil
    },
)
```

**重要な安全ルール:**

1. **常にトークンタイプを変更**: 同じパーサーが再度消費できるトークンを出力しない
2. **状態変化を確認**: 変換が意味のある進歩（異なる `Type` または `Val`）を行うことを確保
3. **スタックオーバーフロー保護を使用**: 安全網として適切な `MaxDepth` 制限を設定
4. **トレースでテスト**: トレースを有効にして予期しない再パースパターンを検出

**検出戦略:**
```go
// 開発中にトレースを有効にして無限ループを検出
context := pc.NewParseContext[int]()
context.TraceEnable = true
context.MaxDepth = 50 // デバッグ用の低い制限

result, err := pc.EvaluateWithRawTokens(context, input, parser)
if errors.Is(err, pc.ErrStackOverflow) {
    fmt.Println("無限ループの可能性が検出されました！")
    context.DumpTrace() // 繰り返しパターンのトレースを調査
}
```

### 自動変換安全性チェック

ライブラリには、実行時に潜在的に危険な変換を検出できるオプションの自動安全性チェック機能が含まれています：

```go
// 自動変換安全性チェックを有効化
context := pc.NewParseContext[int]()
context.OrMode = pc.OrModeSafe          // Safeモードにする必要がある
context.CheckTransformSafety = true     // 安全性チェックを有効化

// 潜在的に安全でない変換を含むパーサー
parser := pc.Trans(
    someParser,
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        // この変換が同じパーサーで再度同じ結果を生成するトークンを返す場合、
        // 警告がログ出力されます
        return tokens, nil // 恒等変換 - 潜在的に安全でない！
    },
)
```

**動作原理:**
1. Safeモードで各変換後（`CheckTransformSafety`が有効な場合）
2. システムは変換されたトークンを同じパーサーで再パースを試行
3. 再パースが同じ結果を生成する場合、stderrに警告をログ出力
4. パースは正常に続行されますが、警告により潜在的な無限ループを特定可能

**安全性チェック出力例:**
```
Warning: Transformation safety check failed: potential infinite loop in transformation at myfile.go:123 - parser produces same result when applied to transformed tokens
```

**設定:**
- `OrMode`が`OrModeSafe`の場合のみ動作
- `CheckTransformSafety = true`で明示的に有効化する必要がある
- 警告はstderrにログ出力されるが、パースは停止しない
- ランタイム呼び出し元情報を使用して正確なファイルと行位置を表示

**制限事項:**
- 全ての安全でないパターンを検出できない（例：変換での副作用）
- 外部状態を持つ複雑な変換で偽陽性の可能性
- 有効時のパフォーマンスオーバーヘッド（主に開発時に使用）
- 即座の再パースのみチェック、多段階変換チェーンは対象外
