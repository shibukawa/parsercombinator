# Parser Combinator Library for Go

A powerful and flexible parser combinator library for Go, specifically designed for building parsers that work with pre-tokenized input to construct Abstract Syntax Trees (ASTs).

## Features

- **Token-based parsing**: Works with pre-tokenized input rather than raw strings
- **Type-safe**: Leverages Go's generics for type safety
- **Comprehensive error handling**: Advanced error reporting with custom messages
- **Stack overflow protection**: Built-in recursion depth limiting to prevent infinite loops
- **Debugging support**: Built-in tracing capabilities
- **Recovery mechanisms**: Error recovery for robust parsing
- **Lookahead support**: Positive and negative lookahead operations
- **Composable**: Easy to combine simple parsers into complex ones

## Installation

```bash
go get github.com/shibukawa/parsercombinator
```

## Quick Start

```go
package main

import (
    "fmt"
    "strconv"
    pc "github.com/shibukawa/parsercombinator"
)

// Define a simple calculator parser
func main() {
    // Create basic parsers
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

    // Combine parsers
    expression := pc.Seq(digit, operator, digit)

    // Parse input
    context := pc.NewParseContext[int]()
    result, err := pc.EvaluateWithRawTokens(context, []string{"5", "+", "3"}, expression)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }
    
    fmt.Printf("Result: %v\n", result) // [5, 3] (operator values are 0)
}
```

## Core Components

### Parser Function

The core type is `Parser[T]`:

```go
type Parser[T any] func(*ParseContext[T], []Token[T]) (consumed int, newTokens []Token[T], err error)
```

### Token Structure

```go
type Token[T any] struct {
    Type string  // Token type identifier
    Pos  *Pos    // Position information
    Raw  string  // Original raw text
    Val  T       // Parsed value
}
```

### Parse Context

```go
type ParseContext[T any] struct {
    Tokens         []Token[T]     // Input tokens
    Pos            int            // Current position
    RemainedTokens []Token[T]     // Remaining tokens after parsing
    Results        []Token[T]     // Parsed result tokens
    Traces         []*TraceInfo   // Debug traces
    Errors         []*ParseError  // Collected errors
    Depth          int            // Current recursion depth
    TraceEnable    bool           // Enable/disable tracing
    MaxDepth       int            // Maximum allowed recursion depth (0 = no limit)
}
```

## Basic Combinators

### Sequence (`Seq`)

Matches parsers in sequence:

```go
parser := pc.Seq(digit, operator, digit) // Matches: digit operator digit
```

### Choice (`Or`)

Tries parsers in order, returns first successful match:

```go
parser := pc.Or(digit, string, identifier) // Matches any of the alternatives
```

### Choice Parsing with Or

The `Or` parser tries multiple alternatives and returns the result from the parser that **consumes the most tokens** (longest match). This ensures predictable behavior and compatibility with complex expression patterns.

```go
// Basic usage - tries each parser in order, returns longest match
parser := pc.Or(
    longKeyword,    // e.g., "interface"
    shortKeyword,   // e.g., "if"
    identifier,     // e.g., "interfaceType"
)

// For input "interface", this will match longKeyword (9 tokens)
// rather than shortKeyword (2 tokens), even if shortKeyword appears first
```

#### Important Considerations

1. **Longest Match Behavior**: Always returns the alternative that consumes the most tokens
2. **Order Independence**: For unambiguous grammars, parser order doesn't matter
3. **Ambiguous Grammars**: Be careful with overlapping patterns

```go
// Good: Unambiguous alternatives
parser := pc.Or(
    stringLiteral,   // "hello"
    numberLiteral,   // 42
    identifier,      // variable
)

// Potentially problematic: Overlapping patterns
parser := pc.Or(
    pc.String("for"),     // exact match
    identifier,           // any identifier (including "for")
)
// Solution: Put more specific patterns first or use longer matches
```

#### Working with Expression Parsing

The longest match behavior is particularly useful for expression parsing:

```go
// Expression parser that handles operator precedence correctly
expr := pc.Or(
    binaryExpression,    // "a + b * c" (longer)
    primaryExpression,   // "a" (shorter)
)
// Will correctly choose the longer binary expression
```

### Repetition

- `ZeroOrMore`: Matches zero or more occurrences
- `OneOrMore`: Matches one or more occurrences  
- `Repeat`: Matches with specific min/max counts
- `Optional`: Matches zero or one occurrence

```go
numbers := pc.ZeroOrMore("numbers", digit)
requiredNumbers := pc.OneOrMore("required-numbers", digit)
exactlyThree := pc.Repeat("exactly-three", 3, 3, digit)
maybeDigit := pc.Optional(digit)
```

## Advanced Features

### Lookahead Operations

```go
// Positive lookahead - check without consuming
parser := pc.Seq(
    pc.Lookahead(keyword), // Check for keyword
    actualParser,          // Then parse normally
)

// Negative lookahead - ensure something doesn't follow
parser := pc.Seq(
    identifier,
    pc.NotFollowedBy(digit), // Identifier not followed by digit
)

// Peek - get result without consuming
parser := pc.Seq(
    pc.Peek(nextToken), // See what's coming
    conditionalParser,   // Parse based on peek result
)
```

### Error Handling and User-Friendly Messages

```go
// Label for better error messages
numberParser := pc.Label("number", digit)

// Expected - for specific error messages
parser := pc.Or(
    validExpression,
    pc.Expected[int]("closing parenthesis"),
)

// Fail - for explicit failures
parser := pc.Or(
    implementedFeature,
    pc.Fail[int]("feature not implemented in this version"),
)
```

### Error Recovery

```go
// Recover from errors and continue parsing
parser := pc.Recover(
    pc.Digit(),        // Precondition check
    parseStatement,    // Main parsing logic
    pc.Until(";"),     // Recovery: skip until semicolon
)
```

### Transformation

Transform parsed results:

```go
// Transform tokens
addParser := pc.Trans(
    pc.Seq(digit, operator, digit),
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        result := tokens[0].Val + tokens[2].Val // Add the numbers
        return []pc.Token[int]{{
            Type: "result",
            Pos:  tokens[0].Pos,
            Val:  result,
        }}, nil
    },
)
```

#### ⚠️ Important: Avoiding Infinite Loops in Transformations

When using `Trans()` to transform tokens, **be extremely careful** about token compatibility to avoid infinite loops:

```go
// ❌ DANGEROUS: This can cause infinite loops
badParser := pc.Trans(
    pc.Trace("digit", func(pctx *pc.ParseContext[int], src []pc.Token[int]) (int, []pc.Token[int], error) {
        if src[0].Type == "raw" {
            i, err := strconv.Atoi(src[0].Raw)
            if err != nil {
                return 0, nil, pc.NewErrNotMatch("integer", src[0].Raw, src[0].Pos)
            }
            // ❌ PROBLEM: Still produces "raw" type that can be re-parsed
            return 1, []pc.Token[int]{{Type: "raw", Pos: src[0].Pos, Raw: src[0].Raw, Val: i}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("digit", src[0].Type, src[0].Pos)
    }),
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        // The transformed token is still "raw" type - can be parsed again!
        return tokens, nil // ❌ This creates an infinite loop risk
    },
)

// ✅ SAFE: Change token type to prevent re-parsing
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
        // ✅ SAFE: Token type is "digit", not "raw" - won't be re-parsed
        return []pc.Token[int]{{
            Type: "number",  // Different type prevents re-parsing
            Pos:  tokens[0].Pos,
            Val:  tokens[0].Val,
        }}, nil
    },
)
```

**Key Safety Rules:**

1. **Always change token types**: Never output tokens that could be consumed by the same parser again
2. **Check for state changes**: Ensure transformations make meaningful progress (different `Type` or `Val`)
3. **Use stack overflow protection**: Set appropriate `MaxDepth` limits as a safety net
4. **Test with tracing**: Enable tracing to detect unexpected re-parsing patterns

**Detection Strategy:**
```go
// Enable tracing to detect infinite loops during development
context := pc.NewParseContext[int]()
context.TraceEnable = true
context.MaxDepth = 50 // Lower limit for debugging

result, err := pc.EvaluateWithRawTokens(context, input, parser)
if errors.Is(err, pc.ErrStackOverflow) {
    fmt.Println("Potential infinite loop detected!")
    context.DumpTrace() // Examine the trace for repeated patterns
}
```

### Automatic Transformation Safety Checks

The library includes optional automatic safety checks that can detect potentially dangerous transformations at runtime:

```go
// Enable automatic transformation safety checks
context := pc.NewParseContext[int]()
context.OrMode = pc.OrModeSafe          // Must be in Safe mode
context.CheckTransformSafety = true     // Enable safety checks

// Parser with potentially unsafe transformation
parser := pc.Trans(
    someParser,
    func(pctx *pc.ParseContext[int], tokens []pc.Token[int]) ([]pc.Token[int], error) {
        // If this transformation returns tokens that the same parser 
        // would consume again with the same result, a warning will be logged
        return tokens, nil // Identity transformation - potentially unsafe!
    },
)
```

**How It Works:**
1. After each transformation in Safe mode (when `CheckTransformSafety` is enabled)
2. The system attempts to re-parse the transformed tokens with the same parser
3. If the re-parsing produces identical results, a warning is logged to stderr
4. The parsing continues normally, but the warning helps identify potential infinite loops

**Safety Check Output Example:**
```
Warning: Transformation safety check failed: potential infinite loop in transformation at myfile.go:123 - parser produces same result when applied to transformed tokens
```

**Configuration:**
- Only active when `OrMode` is `OrModeSafe` 
- Must explicitly enable with `CheckTransformSafety = true`
- Warnings are logged to stderr but don't stop parsing
- Uses runtime caller information to show exact file and line location

**Limitations:**
- Cannot detect all unsafe patterns (e.g., side effects in transformations)
- May produce false positives for complex transformations with external state
- Performance overhead when enabled (use primarily during development)
- Only checks immediate re-parsing, not multi-step transformation chains
### Recursive Parsers with Alias

```go
// Define recursive grammar
expressionBody, expression := pc.NewAlias[int]("expression")

parser := expressionBody(
    pc.Or(
        digit,
        pc.Seq(
            pc.Literal("("),
            expression, // Recursive reference
            pc.Literal(")"),
        ),
    ),
)
```

### Debugging and Tracing

```go
context := pc.NewParseContext[int]()
context.TraceEnable = true // Enable tracing

result, err := pc.EvaluateWithRawTokens(context, input, parser)

// Print trace information
context.DumpTrace() // Prints to stdout
traceText := context.DumpTraceAsText() // Get as string
```

## Error Types

The library defines several error types:

- `ErrNotMatch`: Parser doesn't match (recoverable)
- `ErrRepeatCount`: Repetition count not met (recoverable) 
- `ErrCritical`: Critical error (not recoverable)
- `ErrStackOverflow`: Recursion depth exceeded maximum limit (prevents infinite loops)

```go
// Create custom errors
err := pc.NewErrNotMatch("expected", "actual", position)
err := pc.NewErrCritical("fatal error", position)
err := pc.NewErrStackOverflow(currentDepth, maxDepth, position)
```

## Stack Overflow Protection

The library includes built-in protection against infinite loops in recursive parsers through stack depth limiting.

### Configuration

```go
// Set custom stack depth limit
context := pc.NewParseContext[int]()
context.MaxDepth = 50  // Maximum recursion depth of 50

// Disable limit (set to 0)
context.MaxDepth = 0   // No limit (use with caution)

// Default limit
fmt.Println(pc.NewParseContext[int]().MaxDepth) // 1000 (default)
```

### Error Handling

When the recursion depth exceeds the limit, an `ErrStackOverflow` error is returned:

```go
result, err := pc.EvaluateWithRawTokens(context, input, parser)
if err != nil {
    if errors.Is(err, pc.ErrStackOverflow) {
        fmt.Printf("Stack overflow detected: %v\n", err)
        // Handle infinite recursion case
    }
}
```

### Use Cases

This protection is particularly useful for:

- **Left-recursive grammars**: Detecting and preventing infinite left recursion
- **Malformed input**: Stopping runaway parsing on unexpected input patterns  
- **Development debugging**: Catching recursive parser logic errors early
- **Production safety**: Preventing server crashes from malicious or malformed input

### Example: Protected Recursive Parser

```go
// This parser could potentially loop infinitely on certain inputs
expressionBody, expression := pc.NewAlias[int]("expression")
parser := expressionBody(
    pc.Or(
        pc.Digit(),                                    // Base case
        pc.Seq(expression, pc.Operator(), pc.Digit()), // Recursive case (left-recursive!)
    ),
)

context := pc.NewParseContext[int]()
context.MaxDepth = 10 // Low limit for demonstration

// This will be caught and stopped safely
result, err := pc.EvaluateWithRawTokens(context, []string{"+"}, parser)
if errors.Is(err, pc.ErrStackOverflow) {
    fmt.Println("Infinite recursion detected and prevented!")
}
```

## Complete Example: Mathematical Expressions

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
                return 0, nil, pc.NewErrNotMatch("integer", src[0].Raw, src[0].Pos)
            }
            return 1, []pc.Token[int]{{Type: "digit", Pos: src[0].Pos, Val: i}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("digit", src[0].Type, src[0].Pos)
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
        return 0, nil, pc.NewErrNotMatch("operator", src[0].Raw, src[0].Pos)
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
        fmt.Printf("Error: %v\n", err)
        context.DumpTrace()
        return
    }
    
    fmt.Printf("Result: %d\n", result[0]) // Output: 15
}
```

## Best Practices

1. **Use Labels**: Always use `Label()` for user-facing parsers to provide clear error messages
2. **Enable Tracing**: Use tracing during development to understand parser behavior
3. **Handle Errors Gracefully**: Use `Expected()` and `Fail()` to provide meaningful error messages
4. **Compose Incrementally**: Build complex parsers from simple, well-tested components
5. **Use Recovery**: Implement error recovery for robust parsing of malformed input
6. **Type Safety**: Leverage Go's type system to catch errors at compile time
7. **⚠️ Transformation Safety**: Always change token types in transformations to prevent infinite loops
   - Never output tokens that could be re-parsed by the same parser
   - Use stack overflow protection (`MaxDepth`) as a safety net
   - Enable tracing during development to detect re-parsing patterns
   - Use automatic safety checks in Safe mode: `context.CheckTransformSafety = true`

## Practical Compiler Construction Patterns

### Real-world AST Construction from Token Streams

In actual compilers, AST construction from token streams often follows staged patterns like these:

```go
// AST Node definitions
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

// Staged AST construction parsers
func NumberLiteral() pc.Parser[ASTNode] {
    return pc.Trans(
        pc.Label("number literal", Digit()),
        func(pctx *pc.ParseContext[ASTNode], tokens []pc.Token[ASTNode]) ([]pc.Token[ASTNode], error) {
            // Extract value from original token and create new AST node
            digitToken := tokens[0]
            astNode := &LiteralNode{
                pos:   digitToken.Pos,
                value: digitToken.Val, // Preserve original int value
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
            // Reference existing AST nodes to build new node
            leftNode := tokens[0].Val.(ASTNode)    // Old node reference
            opToken := tokens[1]                   // Operator token
            rightNode := tokens[2].Val.(ASTNode)   // Old node reference
            
            // Create new AST node
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

### Complex AST Construction Patterns

For more complex structures, staged construction makes management easier:

```go
// Function call node
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

// Argument list construction
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
            
            // If optional arguments exist
            if len(tokens) > 2 && tokens[1].Type == "ast_node" {
                // First argument
                arguments = append(arguments, tokens[1].Val.(ASTNode))
                
                // Additional arguments (, expression repetitions)
                for i := 2; i < len(tokens)-1; i += 2 {
                    if tokens[i].Type == "ast_node" {
                        arguments = append(arguments, tokens[i].Val.(ASTNode))
                    }
                }
            }
            
            // Create meta-node representing argument list
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

// Function call construction
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

### Post-Tree Processing Patterns

For processing after tree construction, these approaches are effective:

```go
// Visitor pattern for AST processing
type ASTVisitor interface {
    VisitBinaryOp(node *BinaryOpNode) error
    VisitLiteral(node *LiteralNode) error
    VisitFunctionCall(node *FunctionCallNode) error
}

// Type checker example
type TypeChecker struct {
    errors []error
    symbolTable map[string]Type
}

func (tc *TypeChecker) VisitBinaryOp(node *BinaryOpNode) error {
    // Process left and right child nodes recursively
    if err := node.left.Accept(tc); err != nil {
        return err
    }
    if err := node.right.Accept(tc); err != nil {
        return err
    }
    
    // Type checking logic
    leftType := tc.getNodeType(node.left)
    rightType := tc.getNodeType(node.right)
    
    if !tc.isCompatible(leftType, rightType, node.op) {
        return fmt.Errorf("type error: %s and %s cannot be used with operator %s", 
                         leftType, rightType, node.op)
    }
    
    return nil
}

// Transform pattern for AST transformation
type ASTTransformer interface {
    Transform(node ASTNode) (ASTNode, error)
}

// Optimizer example
type Optimizer struct{}

func (o *Optimizer) Transform(node ASTNode) (ASTNode, error) {
    switch n := node.(type) {
    case *BinaryOpNode:
        // Constant folding optimization
        if isConstant(n.left) && isConstant(n.right) {
            result := evaluateConstant(n)
            return &LiteralNode{pos: n.pos, value: result}, nil
        }
        
        // Recursively optimize child nodes
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

### Multi-Pass Processing Patterns

Real compilers typically use multiple passes for processing:

```go
// Main compiler processing
func CompileProgram(input []string) (*Program, error) {
    // Pass 1: Syntax analysis (using parser combinators)
    context := pc.NewParseContext[ASTNode]()
    ast, err := pc.EvaluateWithRawTokens(context, input, Program())
    if err != nil {
        return nil, fmt.Errorf("syntax analysis error: %w", err)
    }
    
    programNode := ast[0].Val.(*ProgramNode)
    
    // Pass 2: Symbol table construction
    symbolBuilder := &SymbolTableBuilder{}
    if err := programNode.Accept(symbolBuilder); err != nil {
        return nil, fmt.Errorf("symbol analysis error: %w", err)
    }
    
    // Pass 3: Type checking
    typeChecker := &TypeChecker{symbolTable: symbolBuilder.table}
    if err := programNode.Accept(typeChecker); err != nil {
        return nil, fmt.Errorf("type checking error: %w", err)
    }
    
    // Pass 4: Optimization
    optimizer := &Optimizer{}
    optimizedAST, err := optimizer.Transform(programNode)
    if err != nil {
        return nil, fmt.Errorf("optimization error: %w", err)
    }
    
    // Pass 5: Code generation
    codeGenerator := &CodeGenerator{}
    code, err := codeGenerator.Generate(optimizedAST)
    if err != nil {
        return nil, fmt.Errorf("code generation error: %w", err)
    }
    
    return &Program{AST: optimizedAST, Code: code}, nil
}
```

This way, parser combinators are used for the syntax analysis stage, and subsequent processing combines traditional AST processing patterns (Visitor, Transform, Multi-pass) to build practical compilers.

## Structured Data Validation Patterns

### Tree Structure Serialization for Validation

The approach you suggested is highly effective. Here's how to serialize tree structures with pseudo-nodes for parser validation:

```go
// Tree structure representation
type TreeNode struct {
    Type     string
    Value    interface{}
    Children []*TreeNode
    Pos      *pc.Pos
}

// Serialization pseudo-tokens
type SerializedToken struct {
    Type  string  // "open", "close", "leaf"
    Node  string  // Node name
    Value interface{}
    Pos   *pc.Pos
}

// Serialize tree (DFS order to pseudo-token stream)
func SerializeTree(node *TreeNode) []SerializedToken {
    var tokens []SerializedToken
    
    if len(node.Children) == 0 {
        // Leaf node
        tokens = append(tokens, SerializedToken{
            Type:  "leaf",
            Node:  node.Type,
            Value: node.Value,
            Pos:   node.Pos,
        })
    } else {
        // Internal node: start
        tokens = append(tokens, SerializedToken{
            Type:  "open",
            Node:  node.Type,
            Value: node.Value,
            Pos:   node.Pos,
        })
        
        // Recursively process child nodes
        for _, child := range node.Children {
            tokens = append(tokens, SerializeTree(child)...)
        }
        
        // Internal node: end
        tokens = append(tokens, SerializedToken{
            Type: "close",
            Node: node.Type,
            Pos:  node.Pos,
        })
    }
    
    return tokens
}

// Validator for serialized tokens
func ValidateHTMLStructure() pc.Parser[bool] {
    // HTML tag start
    htmlOpen := pc.Trace("html_open", func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
        token := src[0].Val.(SerializedToken)
        if token.Type == "open" && token.Node == "html" {
            return 1, []pc.Token[bool]{{Type: "validated", Val: true}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("HTML start tag", token.Node, src[0].Pos)
    })
    
    // HTML tag end
    htmlClose := pc.Trace("html_close", func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
        token := src[0].Val.(SerializedToken)
        if token.Type == "close" && token.Node == "html" {
            return 1, []pc.Token[bool]{{Type: "validated", Val: true}}, nil
        }
        return 0, nil, pc.NewErrNotMatch("HTML end tag", token.Node, src[0].Pos)
    })
    
    // body element validation
    bodyElement := pc.Seq(
        pc.Literal("body_open"),
        pc.ZeroOrMore("body_content", pc.Or(textContent, divElement)),
        pc.Literal("body_close"),
    )
    
    // Complete HTML structure validation
    return pc.Seq(htmlOpen, headElement, bodyElement, htmlClose)
}

// Execute validation
func ValidateHTMLTree(tree *TreeNode) error {
    // Serialize tree
    tokens := SerializeTree(tree)
    
    // Validate with parser combinators
    context := pc.NewParseContext[bool]()
    _, err := pc.EvaluateWithTokens(context, tokens, ValidateHTMLStructure())
    
    return err
}
```

### Schema-Based Structure Validation

More general schema validation patterns:

```go
// Schema definition
type Schema struct {
    Type       string             // "object", "array", "string", etc.
    Properties map[string]*Schema // Object properties
    Items      *Schema            // Array element schema
    Required   []string           // Required fields
    MinItems   int               // Minimum array elements
    MaxItems   int               // Maximum array elements
}

// JSON-like data structure
type DataNode struct {
    Type  string                 // "object", "array", "string", "number", "boolean"
    Value interface{}            // Actual value
    Props map[string]*DataNode   // Object properties
    Items []*DataNode            // Array elements
    Pos   *pc.Pos
}

// Schema validation parser generator
func CreateSchemaValidator(schema *Schema) pc.Parser[bool] {
    return pc.Trace(fmt.Sprintf("validate_%s", schema.Type), 
        func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
            data := src[0].Val.(*DataNode)
            
            // Type check
            if data.Type != schema.Type {
                return 0, nil, pc.NewErrNotMatch(
                    fmt.Sprintf("type %s", schema.Type), 
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
    // Required field validation
    for _, required := range schema.Required {
        if _, exists := data.Props[required]; !exists {
            return 0, nil, pc.NewErrCritical(
                fmt.Sprintf("required field '%s' not found", required),
                data.Pos,
            )
        }
    }
    
    // Validate each property
    for propName, propData := range data.Props {
        propSchema, exists := schema.Properties[propName]
        if !exists {
            return 0, nil, pc.NewErrNotMatch(
                "valid property",
                propName,
                propData.Pos,
            )
        }
        
        // Recursively validate property
        validator := CreateSchemaValidator(propSchema)
        _, _, err := validator(pctx, []pc.Token[bool]{{Val: propData, Pos: propData.Pos}})
        if err != nil {
            return 0, nil, fmt.Errorf("property '%s': %w", propName, err)
        }
    }
    
    return 1, []pc.Token[bool]{{Type: "validated_object", Val: true}}, nil
}

// Configuration file validation example
func ValidateConfigFile() pc.Parser[bool] {
    // Configuration file schema definition
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
    
    return pc.Label("configuration file", CreateSchemaValidator(configSchema))
}
```

### Flat Structure Partial Validation

Methods for partial validation of existing structured data:

```go
// CSV data row validation
type CSVRow struct {
    Fields []string
    LineNo int
}

func ValidateCSVRow(expectedColumns []string, validators map[string]pc.Parser[bool]) pc.Parser[bool] {
    return pc.Trans(
        pc.Trace("csv_row", func(pctx *pc.ParseContext[bool], src []pc.Token[bool]) (int, []pc.Token[bool], error) {
            row := src[0].Val.(*CSVRow)
            
            // Column count check
            if len(row.Fields) != len(expectedColumns) {
                return 0, nil, pc.NewErrNotMatch(
                    fmt.Sprintf("%d fields", len(expectedColumns)),
                    fmt.Sprintf("%d fields", len(row.Fields)),
                    &pc.Pos{Line: row.LineNo},
                )
            }
            
            // Validate each field
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
                        return 0, nil, fmt.Errorf("column '%s' (line %d): %w", columnName, row.LineNo, err)
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

// Usage example: User data CSV validation
func CreateUserCSVValidator() pc.Parser[bool] {
    columns := []string{"name", "email", "age", "active"}
    
    validators := map[string]pc.Parser[bool]{
        "name": pc.Label("username", validateNonEmptyString()),
        "email": pc.Label("email address", validateEmail()),
        "age": pc.Label("age", validatePositiveNumber()),
        "active": pc.Label("active flag", validateBoolean()),
    }
    
    return pc.OneOrMore("csv_rows", ValidateCSVRow(columns, validators))
}
```

### Real-time Validation Patterns

Validation for streaming or real-time data:

```go
// Event stream validation
type Event struct {
    Type      string
    Timestamp time.Time
    Data      interface{}
    Pos       *pc.Pos
}

// State machine-based sequence validation
func ValidateEventSequence() pc.Parser[bool] {
    // User login flow validation
    loginFlow := pc.Seq(
        pc.Label("login start", expectEvent("login_start")),
        pc.Optional(pc.Label("auth attempt", expectEvent("auth_attempt"))),
        pc.Or(
            pc.Label("login success", expectEvent("login_success")),
            pc.Seq(
                pc.Label("login failure", expectEvent("login_failure")),
                pc.Optional(pc.Label("retry", ValidateEventSequence())), // Recursively allow retry
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

These patterns enable validation of various structured data types:

1. **Tree Serialization**: Validation of complex hierarchical structures
2. **Schema-Based**: Type-safe validation of JSON/XML-like data  
3. **Flat Structure**: Validation of tabular data like CSV/TSV
4. **Real-time**: Validation of event streams and state transitions

## License

Apache 2.0 License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.