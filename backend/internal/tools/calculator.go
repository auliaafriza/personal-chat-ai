package tools

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/expr-lang/expr"
)

// Calculator — math expression evaluator pakai expr-lang/expr.
// Sandboxed (nggak ada akses ke env apa pun selain math funcs yang kita inject).
type Calculator struct{}

func NewCalculator() *Calculator { return &Calculator{} }

func (c *Calculator) Name() string { return "calculator" }

func (c *Calculator) Schema() Schema {
	return Schema{
		Type: "function",
		Function: SchemaFunction{
			Name:        "calculator",
			Description: "Evaluate a math expression. Supports +, -, *, /, %, **, parens, dan fungsi: sqrt, pow, abs, log, ln, exp, sin, cos, tan, floor, ceil, round, min, max. Constants: pi, e. Contoh: 'sqrt(2) * pi', '(1.05 ** 12 - 1) * 1000', 'sin(pi/2)'. Use INSTEAD OF mental math — LLM sering salah kalau hitung manual.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"expression": map[string]any{
						"type":        "string",
						"description": "Math expression to evaluate.",
					},
				},
				"required": []string{"expression"},
			},
		},
	}
}

// mathEnv: subset functions yang aman di-expose. Semua dari `math` package stdlib.
var mathEnv = map[string]any{
	"sqrt":  math.Sqrt,
	"pow":   math.Pow,
	"abs":   math.Abs,
	"log":   math.Log10, // log base 10 (matches common calculator convention)
	"ln":    math.Log,   // natural log
	"exp":   math.Exp,
	"sin":   math.Sin,
	"cos":   math.Cos,
	"tan":   math.Tan,
	"asin":  math.Asin,
	"acos":  math.Acos,
	"atan":  math.Atan,
	"floor": math.Floor,
	"ceil":  math.Ceil,
	"round": math.Round,
	"min":   math.Min,
	"max":   math.Max,
	"pi":    math.Pi,
	"e":     math.E,
}

func (c *Calculator) Run(ctx context.Context, args map[string]any) (any, error) {
	expression, _ := args["expression"].(string)
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, fmt.Errorf("expression is required")
	}

	program, err := expr.Compile(expression, expr.Env(mathEnv), expr.AsFloat64())
	if err != nil {
		return nil, fmt.Errorf("invalid expression: %w", err)
	}

	result, err := expr.Run(program, mathEnv)
	if err != nil {
		return nil, fmt.Errorf("eval failed: %w", err)
	}

	// expr.AsFloat64() guarantees float64 result.
	val, _ := result.(float64)
	return map[string]any{
		"expression": expression,
		"result":     val,
	}, nil
}
