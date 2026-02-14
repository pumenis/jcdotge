package dsl

import (
	"strconv"

	"github.com/pumenis/jcdotge/parser"
)

func iF(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	if ifCheckValue, ok := eval(value.Parts["0"]).Name.(bool); ok && ifCheckValue {
		value.Parts["1"].Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, value.Parts["1"])

		go func() {
			var components []*parser.ContainerNode
			for i := 0; i < value.Parts["1"].Parts["length"].Name.(int); i++ {
				components = append(components, value.Parts["1"].Parts[strconv.Itoa(i)])
			}

			components = scopeEvalFunc(components...)
			for _, component := range components {
				if component.Type == parser.ChanStringType {
					ch, ok := component.Name.(chan string)
					if !ok {
						panic("ifs main this is not chan string")
					}
					for line := range ch {
						out <- line
					}
				}
			}

			close(out)
		}()
		if value.Parts["length"].Name.(int) <= 2 {
			return parser.NewContainerNode(out, parser.ChanStringType, value)
		}
	} else {
		value.Parts["2"].Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, value.Parts["2"])
		go func() {
			var components []*parser.ContainerNode
			for i := 0; i < value.Parts["2"].Parts["length"].Name.(int); i++ {
				components = append(components, value.Parts["2"].Parts[strconv.Itoa(i)])
			}
			components = scopeEvalFunc(components...)
			for _, component := range components {
				if component.Type == parser.ChanStringType {
					ch, ok := component.Name.(chan string)
					if !ok {
						panic("ifs else this is not chan string")
					}
					for line := range ch {
						out <- line
					}
				}
			}

			close(out)
		}()
	}
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}
