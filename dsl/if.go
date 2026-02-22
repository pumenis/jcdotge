package dsl

import (
	"strconv"

	"github.com/pumenis/jcdotge/parser"
)

func iF(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	ifCheckValue, ok := eval(value.Parts["0"]).Name.(bool)
	if ok && ifCheckValue {
		value.Parts["1"].Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, value.Parts["1"])
		var components []*parser.ContainerNode
		for i := 0; i < value.Parts["1"].Parts["length"].Name.(int); i++ {
			components = append(components, value.Parts["1"].Parts[strconv.Itoa(i)])
		}
		components = scopeEvalFunc(components...)
		go func() {
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
	} else if elseblock, ok := value.Parts["2"]; ok {
		elseblock.Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, elseblock)
		var components []*parser.ContainerNode
		for i := 0; i < elseblock.Parts["length"].Name.(int); i++ {
			components = append(components, elseblock.Parts[strconv.Itoa(i)])
		}
		components = scopeEvalFunc(components...)
		go func() {
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
