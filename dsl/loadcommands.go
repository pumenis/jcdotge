package dsl

import (
	"fmt"
	"plugin"

	"github.com/pumenis/jcdotge/homedir"
	"github.com/pumenis/jcdotge/parser"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ToCamelCase(s string) string {
	if s == "" {
		return s
	}
	caser := cases.Title(language.English)
	return caser.String(s)
}

func loadFunc(name string) (func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode, error) {
	plugPath, err := homedir.Expand(fmt.Sprintf("~/.local/lib/commands/funcs/%s.so", name))
	if err != nil {
		return nil, err
	}

	plug, err := plugin.Open(plugPath)
	if err != nil {
		return nil, err
	}

	sym, err := plug.Lookup(ToCamelCase(name))
	if err != nil {
		return nil, err
	}

	var ok bool
	fn, ok := sym.(func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode)
	if !ok {
		return nil, err
	}

	return fn, nil
}
