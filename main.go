package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pumenis/jcdotge/dsl"
	"github.com/pumenis/jcdotge/parser"
)

func main() {
	var inlineJCdotge string
	flag.StringVar(&inlineJCdotge, "c", "", "Run inline JCdotge shell code")
	flag.Parse()
	rest := flag.Args()

	if inlineJCdotge != "" {
		runJCdotge(inlineJCdotge, rest)
	} else {
		data, err := os.ReadFile(rest[0])
		if err != nil {
			fmt.Println("Error reading file:", err)
			return
		}
		runJCdotge(string(data), rest)
	}
}

func runJCdotge(code string, args []string) {
	script, err := parser.Parse(code)
	if err != nil {
		panic(err)
	}
	outNode := dsl.RunLang(script, args...)
	out, ok := outNode.Name.(chan string)
	if !ok {
		panic("runJCdotge: it is not chan string")
	}
	for line := range out {
		fmt.Println(line)
	}
}
