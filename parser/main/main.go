package main

import (
	"flag"
	"fmt"

	"github.com/pumenis/jcdotge/parser"
)

func main() {
	var inlineJCdotge string
	flag.StringVar(&inlineJCdotge, "c", "", "Run inline JCdotge shell code")
	flag.Parse()
	rest := flag.Args()

	if inlineJCdotge != "" {
		runJCdotge(inlineJCdotge, rest)
	}
}

func runJCdotge(code string, args []string) {
	syntaxTree, err := parser.Parse(code)
	if err != nil {
		panic(err)
	}
	_ = args
	fmt.Println(syntaxTree.Inspect())
}
