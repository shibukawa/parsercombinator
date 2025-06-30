# AST Building Guide for LLMs

A concise guide for using the Go parser combinator library to build Abstract Syntax Trees from token streams.

## Core Concepts

### Token Structure
```go
type Token[T any] struct {
    Type string  // Token type ("raw", "ast_node", etc.)
    Pos  *Pos    // Position info
    Raw  string  // Original text
    Val  T       // Parsed value (int, ASTNode, etc.)
}
```

### Parser Function
```go
type Parser[T any] func(*ParseContext[T], []Token[T]) (consumed int, newTokens []Token[T], err error)
```

## Basic AST Node Pattern

### Node Interface
```go
type ASTNode interface {
    Type() string
    Position() *pc.Pos
}

type LiteralNode struct {
    pos   *pc.Pos
    value interface{}
}

type BinaryOpNode struct {
    pos   *pc.Pos
    left  ASTNode
    op    string
    right ASTNode
}
```

## Token-to-AST Conversion

### Basic Literal Parser
```go
func NumberLiteral() pc.Parser[ASTNode] {
    return pc.Trans(
        Digit(), // Assumes Digit() returns Parser[int]
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            return []pc.Token[ASTNode]{{
                Type: "ast_node",
                Pos:  tokens[0].Pos,
                Val:  &LiteralNode{pos: tokens[0].Pos, value: tokens[0].Val},
            }}, nil
        },
    )
}
```

### Binary Expression Parser
```go
func BinaryExpression() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Seq(NumberLiteral(), Operator(), NumberLiteral()),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            leftNode := tokens[0].Val.(ASTNode)
            opToken := tokens[1]
            rightNode := tokens[2].Val.(ASTNode)
            
            return []pc.Token[ASTNode]{{
                Type: "ast_node",
                Pos:  leftNode.Position(),
                Val: &BinaryOpNode{
                    pos:   leftNode.Position(),
                    left:  leftNode,
                    op:    opToken.Raw,
                    right: rightNode,
                },
            }}, nil
        },
    )
}
```

## Essential Combinators

### Sequence
```go
pc.Seq(parser1, parser2, parser3) // Match in order
```

### Choice
```go
pc.Or(parser1, parser2, parser3) // Try alternatives
```

### Repetition
```go
pc.ZeroOrMore("name", parser)  // 0 or more
pc.OneOrMore("name", parser)   // 1 or more
pc.Optional(parser)            // 0 or 1
```

### Transformation
```go
pc.Trans(parser, func(pctx, tokens) ([]Token, error) {
    // Convert tokens to new format
    return newTokens, nil
})
```

## Error Handling

### Error Types
- `ErrNotMatch`: Parser doesn't match (recoverable)
- `ErrCritical`: Fatal error (stops parsing)

### Error Creation
```go
pc.NewErrNotMatch("expected", "actual", position)
pc.NewErrCritical("error message", position)
```

### Error Labels
```go
pc.Label("expression", parser) // Better error messages
```

## Complex Structure Example

### Function Call AST
```go
type FunctionCallNode struct {
    pos  *pc.Pos
    name string
    args []ASTNode
}

func FunctionCall() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Seq(
            Identifier(),
            pc.Literal("("),
            pc.Optional(ArgumentList()),
            pc.Literal(")"),
        ),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            var args []ASTNode
            if len(tokens) > 2 && tokens[2].Type == "ast_node" {
                args = tokens[2].Val.([]ASTNode)
            }
            
            return []pc.Token[ASTNode]{{
                Type: "ast_node",
                Pos:  tokens[0].Pos,
                Val: &FunctionCallNode{
                    pos:  tokens[0].Pos,
                    name: tokens[0].Raw,
                    args: args,
                },
            }}, nil
        },
    )
}
```

## Recursive Grammar

### Using Alias for Recursion
```go
expressionBody, expression := pc.NewAlias[ASTNode]("expression")

parser := expressionBody(
    pc.Or(
        NumberLiteral(),
        pc.Seq(pc.Literal("("), expression, pc.Literal(")")),
    ),
)
```

## Usage Pattern

### Parse Context Setup
```go
context := pc.NewParseContext[ASTNode]()
context.TraceEnable = true // For debugging
```

### Execute Parser
```go
result, err := pc.EvaluateWithRawTokens(context, inputStrings, parser)
if err != nil {
    return err
}
ast := result[0].Val.(ASTNode)
```

## Key Points for LLMs

1. **Token Flow**: Raw strings → Tokens → AST nodes via `Trans()`
2. **Node References**: Use old nodes (`tokens[i].Val`) to build new nodes
3. **Error Handling**: Use `Label()` for user-friendly errors
4. **Type Safety**: Cast `tokens[i].Val.(ASTNode)` when accessing AST nodes
5. **Position Tracking**: Always preserve `Pos` information for error reporting

## Minimal Working Example

```go
// 1. Define AST nodes
type Expr interface { Type() string }
type NumExpr struct { value int }

// 2. Create token-to-AST parser
func Number() pc.Parser[Expr] {
    return pc.Trans(Digit(), func(pctx, tokens) ([]Token[Expr], error) {
        return []Token[Expr]{{
            Type: "ast", 
            Val: &NumExpr{value: tokens[0].Val.(int)},
        }}, nil
    })
}

// 3. Parse
context := pc.NewParseContext[Expr]()
result, _ := pc.EvaluateWithRawTokens(context, []string{"42"}, Number())
ast := result[0].Val.(Expr)
```

This guide provides the essential patterns for building ASTs using parser combinators efficiently.
