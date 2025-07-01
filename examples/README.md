# Parser Combinator Examples

このディレクトリには、Go Parser Combinator ライブラリの実用的なサンプルが含まれています。

## ディレクトリ構成

### basic/
基本的なパーサーコンビネータの使用例

- **zeroormore_example.go**: `ZeroOrMore` コンビネータの使用例
  - コンマ区切りの数値リストをパースする例
  - 空のリストと非空のリストの両方をサポート
  - パーサーコンビネータの基本的な使い方を学ぶのに最適

### interpreter/
インタプリタの実装例

- **simple_calculator.go**: 数式を直接評価する電卓
  - 数式をパースして即座に結果を計算
  - 演算子の優先度をサポート（* / > + -）
  - 左結合で演算を処理
  - 四則演算（+, -, *, /）をサポート
  - ゼロ除算エラーの適切な処理

### compiler/
コンパイラの実装例

- **simple_math_to_json.go**: 数式をJSON ASTに変換
  - 数式をパースしてJSONフォーマットのAST（抽象構文木）を生成
  - 演算子の優先度と左結合性を正しく表現
  - 構造化されたデータ形式での出力
  - さらなる処理（最適化、コード生成など）への基盤

### compare_example/
再帰パーサーの比較例

- **main.go**: `pc.Lazy` vs `pc.NewAlias` の比較デモ
  - 単純な自己再帰に適した `pc.Lazy` の使用例
  - 相互再帰や複雑な文法に必要な `pc.NewAlias` の使用例
  - 両アプローチの適切な使い分け方を実践的に学習
  - 相互再帰文法の実装例（NewAliasが必要なケース）

### lazy_basic/
基本的なLazy使用例

- **main.go**: `pc.Lazy`を使用した再帰式パーサー
  - 四則演算と括弧をサポートする式パーサー
  - 右再帰を使用した安全な実装パターン
  - ASTノードの構築と評価の実例
  - 基本的な再帰パーサーの理解に最適

### test_lazy_example/
包括的なLazy使用例

- **main.go**: より詳細な`pc.Lazy`の活用例
  - 複雑な入れ子式の解析
  - 包括的なテストケース
  - 実用的なエラーハンドリング
  - Lazyパターンの応用方法を詳しく学習

## 実行方法

各サンプルは独立したGoプログラムとして実行できます：

```bash
# 基本例の実行
cd basic
go run zeroormore_example.go

# インタプリタ例の実行
cd interpreter
go run simple_calculator.go

# コンパイラ例の実行
cd compiler
go run simple_math_to_json.go

# 再帰パーサー比較例の実行
cd compare_example
go run main.go

# 基本的なLazy使用例の実行
cd lazy_basic
go run main.go

# 包括的なLazy使用例の実行
cd test_lazy_example
go run main.go
```

## 学習の流れ

1. **basic/**: まず基本的なパーサーコンビネータの概念を理解
2. **lazy_basic/**: 単純な再帰パーサーの実装方法を学習
3. **test_lazy_example/**: より複雑なLazy使用例で理解を深める
4. **compare_example/**: LazyとAliasの使い分けを学習
5. **interpreter/**: パースした結果を直接評価する方法を学習
6. **compiler/**: パース結果をASTという中間表現に変換する方法を学習

## 特徴

- **実用的**: 実際のユースケースに基づいた例
- **段階的**: 簡単なものから複雑なものへと段階的に学習可能
- **完全**: それぞれが独立して動作する完全なプログラム
- **エラーハンドリング**: 適切なエラー処理の例を含む

## 応用

これらの例を基に、以下のような応用が可能です：

- **設定ファイルパーサー**: JSON、YAML、INIファイルなどの解析
- **ドメイン固有言語（DSL）**: 特定の目的に特化した小さな言語の実装
- **データバリデーション**: 構造化データの検証とパース
- **プロトコル解析**: 通信プロトコルやファイルフォーマットの解析

各例は拡張可能で、より複雑な機能（括弧サポート、変数、関数など）を追加する出発点として活用できます。
