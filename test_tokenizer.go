package main

import (
	"fmt"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/analysis"
)

func main() {
	source := analysis.NewSourceCode("match 42 { 42 => \"answer\" }", "test.eo")
	tokenizer := analysis.NewSimpleTokenizer()
	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		fmt.Printf("Tokenize error: %v\n", err)
		return
	}

	fmt.Printf("Tokens:\n")
	for _, token := range tokens {
		fmt.Printf("  %s: '%s'\n", token.Kind(), token.Value())
	}
}
