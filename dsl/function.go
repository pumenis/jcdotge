package dsl

import (
	"strconv"

	"github.com/pumenis/jcdotge/parser"
)

func function(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	name := args[0].String()

	code := args[1]

	methodCallFuncs[name] = func(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
		out := make(chan string)
		bareArgs := []string{}
		for _, arg := range args {
			bareArgs = append(bareArgs, arg.String())
		}
		code.Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, code)
		for i, bareArg := range bareArgs {
			code.Parts["$"+strconv.Itoa(i+1)] = parser.NewContainerNode(bareArg, parser.StringType, code)
		}
		go func() {
			var components []*parser.ContainerNode
			for i := 0; i < code.Parts["length"].Name.(int); i++ {
				components = append(components, code.Parts[strconv.Itoa(i)])
			}
			components = scopeEvalFunc(components...)
			for _, component := range components {
				if component.Type == parser.ChanStringType {
					ch, ok := component.Name.(chan string)
					if !ok {
						panic("function: this is not chan string")
					}
					for line := range ch {
						out <- line
					}
				}
			}

			close(out)
		}()
		return parser.NewContainerNode(out, parser.ChanStringType, in)
	}
	return parser.NewContainerNode(true, parser.BoolType, value)
}
