package main

import (
	"fmt"
	"github.com/meetai/echo-lang/internal/modules/frontend/domain/analysis"
)

func main() {
	source := analysis.NewSourceCode("42 => \"hello\"", "test.eo")
	tokenizer := analysis.NewSimpleTokenizer()
	tokens, err := tokenizer.Tokenize(source)
	if err != nil {
		fmt.Printf("Tokenize error: %v\n", err)
		return
	}

	fmt.Printf("Tokens:\n")
	for i, token := range tokens {
		fmt.Printf("  %d: %s '%s'\n", i, token.Kind(), token.Value())
	}
}
