package dsl

import (
	"github.com/pumenis/jcdotge/parser"
)

func subShell(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	go func() {
		components := scopeEvalFunc(args...)
		for _, component := range components {
			if component.Type == parser.ChanStringType {
				ch, ok := component.Name.(chan string)
				if !ok {
					panic("subshell this is not chan string")
				}
				for line := range ch {
					out <- line
				}
			}
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}
