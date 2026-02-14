// Package parser
package parser

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type (
	Node        string
	Property    string
	Prop        string
	EmptyMethod string
	Method      string
	Array       string
	Map         string
	Func        string
	EmptyFunc   string
)

var (
	IntType                = reflect.TypeFor[int]()
	Int8Type               = reflect.TypeFor[int8]()
	Float64Type            = reflect.TypeFor[float64]()
	StringType             = reflect.TypeFor[string]()
	MapStringToAnyType     = reflect.TypeFor[map[string]any]()
	MapStringToStringType  = reflect.TypeFor[map[string]string]()
	MapStringToIntType     = reflect.TypeFor[map[string]int]()
	MapStringToFloat64Type = reflect.TypeFor[map[string]float64]()
	NodeType               = reflect.TypeFor[Node]()
	ContainerType          = reflect.TypeFor[*ContainerNode]()
	PropertyType           = reflect.TypeFor[Property]()
	PropType               = reflect.TypeFor[Prop]()
	EmptyMethodType        = reflect.TypeFor[EmptyMethod]()
	MethodType             = reflect.TypeFor[Method]()
	ArrayType              = reflect.TypeFor[Array]()
	MapType                = reflect.TypeFor[Map]()
	FuncType               = reflect.TypeFor[Func]()
	EmptyFuncType          = reflect.TypeFor[EmptyFunc]()
	BoolType               = reflect.TypeFor[bool]()
	ChanStringType         = reflect.TypeFor[chan string]()
)

type LangSpec struct {
	Starters   map[string]string
	Flatteners map[string]string
}

var DsshitSpec = LangSpec{
	Starters: map[string]string{
		"":         "^^^",
		"function": "end",
		"then":     "done",
		"else":     "done",
		"do":       "done",
		"if":       "fi",
		":(":       "):",
		":{":       "}:",
		"[":        "]",
		"[]{":      "}",
		"{":        "}",
		":":        ":",
		"[>":       "]",
		"[<":       "]",
		"[==":      "]",
		"[!=":      "]",
		"[=~":      "]",
		"[!~":      "]",
		"[>~":      "]",
		"[<~":      "]",
		"[>=~":     "]",
		"[<=~":     "]",
		"[!=~":     "]",
		"[>=":      "]",
		"[<=":      "]",
		"[-f":      "]",
		"[-d":      "]",
		"[-e":      "]",
		"[-x":      "]",
		"[-r":      "]",
		"[-w":      "]",
		"[-s":      "]",
		"[-S":      "]",
		"[-h":      "]",
		"(":        ")",
		"((":       "))",
		"{{":       "}}",
		"[[":       "]]",
		"<<":       ">>",
		"&&(":      ")",
		"||(":      ")",
		"(.":       ")",
		"(+":       ")",
		"(-":       ")",
		"(/":       ")",
		"(*":       ")",
		"(**":      ")",
		"(%":       ")",
		".":        ")",
		"!":        ")",
		`"`:        `"`,
		`'`:        `'`,
		"`":        "`",
		// From here come the maps
		"map[string]any{":     "}",
		"map[string]string{":  "}",
		"map[string]int{":     "}",
		"map[string]float64{": "}",
		// From here comes the DSML stuff
		`="`: `"`,
		`='`: `'`,
		">":  "</",
		"<":  "/>",
	},
	Flatteners: map[string]string{
		"`": "`",
		"'": "'",
		`"`: `"`,
	},
}

func NewContainerNode(name any, targetType reflect.Type, parent *ContainerNode) *ContainerNode {
	return &ContainerNode{
		Name: name,
		Type: targetType,
		Parts: map[string]*ContainerNode{
			"parent": parent,
			"length": {Name: 0, Type: IntType, Parts: nil},
		},
	}
}

type ContainerNode struct {
	Name  any
	Type  reflect.Type
	Parts map[string]*ContainerNode
}

func (node *ContainerNode) Call(funcsMap map[string]func(*ContainerNode, ...*ContainerNode) *ContainerNode, methodName string, args ...*ContainerNode) *ContainerNode {
	return funcsMap[methodName](node, args...)
}

func (node *ContainerNode) FindVariableParent(variable string) *ContainerNode {
	parentNode := node.Parts["parent"]
	for parentNode != nil {
		if _, ok := parentNode.Parts[variable]; ok {
			return parentNode
		}
		parentNode = parentNode.Parts["parent"]
	}
	return nil
}

func (node *ContainerNode) FindScopeParent() *ContainerNode {
	node = node.Parts["parent"]
	for node != nil {
		if scopeParentNode, ok := node.Parts["scope"]; ok {
			return scopeParentNode.Parts["parent"]
		}
		node = node.Parts["parent"]
	}
	return nil
}

func (node *ContainerNode) Set(newNode *ContainerNode) {
	node.Name = newNode.Name
}

func (node *ContainerNode) ChangeType(targetType reflect.Type) {
	v := reflect.ValueOf(node.Name)
	srcType := v.Type()

	if srcType.ConvertibleTo(targetType) {
		node.Name = v.Convert(targetType).Interface()
	}
	node.Type = targetType
}

func (node *ContainerNode) Length() int {
	length, ok := node.Parts["length"].Name.(int)
	if !ok {
		fmt.Println("invalid length")
	}
	return length
}

func (node *ContainerNode) Push(newNode *ContainerNode) {
	length, ok := node.Parts["length"].Name.(int)
	if !ok {
		fmt.Println("invalid length")
	}
	newNode.Parts["parent"] = node
	lengthString := strconv.Itoa(length)
	node.Parts[lengthString] = newNode
	length = length + 1
	node.Parts["length"].Name = length
}

func (node *ContainerNode) Pop() *ContainerNode {
	length, ok := node.Parts["length"].Name.(int)
	if !ok {
		fmt.Println("invalid length")
	}
	length = length - 1
	node.Parts["length"].Name = length
	lastNode := node.Parts[strconv.Itoa(length)]
	delete(node.Parts, strconv.Itoa(length))
	return lastNode.Parts["parent"]
}

func (node *ContainerNode) HalfPush(newNode *ContainerNode) {
	lastItem := strconv.Itoa(node.Parts["length"].Name.(int) - 1)
	index := lastItem + ".5"
	if lastItem == "-1" {
		index = "-0.5"
	}
	node.Parts[index] = newNode
}

func (node *ContainerNode) AppendToHalfPush(part string) {
	currentsLength := node.Parts["length"].Name.(int)
	lastSpace := ""
	if currentsLength == 0 {
		if space, ok := node.Parts["-0.5"]; ok {
			lastSpace = space.String()
		}
	} else {
		lastSpace = node.Parts[strconv.Itoa(currentsLength-1)+".5"].String()
	}
	node.HalfPush(
		NewContainerNode(lastSpace+part, StringType, node),
	)
}

func (node *ContainerNode) HalfPop() *ContainerNode {
	lastItem := strconv.Itoa(node.Parts["length"].Name.(int) - 1)
	index := lastItem + ".5"
	if lastItem == "-1" {
		index = "-0.5"
	}
	lastNode := node.Parts[index]
	delete(node.Parts, index)
	return lastNode
}

func (node *ContainerNode) String() string {
	if node == nil {
		return ""
	}

	return node.printTree()
}

func (node *ContainerNode) printTree() string {
	if node == nil {
		return ""
	}

	returnValue := ""

	switch node.Type {
	case StringType, ArrayType, MethodType, FuncType, MapType, EmptyMethodType, EmptyFuncType, NodeType, PropType, PropertyType:
		value, ok := node.Name.(string)
		if !ok {
			fmt.Println(node.Name)
			panic(".String() error evaluating value as string")
		}
		returnValue += value
	case IntType:
		value, ok := node.Name.(int)
		if !ok {
			panic(".String() error evaluating value as int")
		}
		returnValue += strconv.Itoa(value)
	case Int8Type:
		value, ok := node.Name.(int8)
		if !ok {
			panic(".String() error evaluating value as int")
		}
		returnValue += strconv.Itoa(int(value))
	case Float64Type:
		value, ok := node.Name.(float64)
		if !ok {
			panic(".String() error evaluating value as int")
		}
		returnValue += strconv.FormatFloat(value, 'f', 2, 64)
	case BoolType:
		value, ok := node.Name.(bool)
		if !ok {
			panic(".String() error evaluating value as int")
		}
		returnValue += strconv.FormatBool(value)
	case ContainerType:
		returnValue += "<nil>"
	default:
		returnValue += fmt.Sprint(node.Name)
	}

	return returnValue
}

func (node *ContainerNode) Inspect() string {
	if node == nil {
		return ""
	}

	return node.printTreeHelper(0, "0")
}

func (node *ContainerNode) printTreeHelper(level int, index string) string {
	if node == nil {
		return ""
	}

	returnValue := ""
	lengthNode, lengthExists := node.Parts["length"]
	returnValue = printSpaces(level)

	returnValue += fmt.Sprintf("%s (%s): '%v'\n", node.Type, index, node.Name)

	if lengthExists {
		if _, ok := node.Parts["-0.5"]; ok {
			returnValue += node.Parts["-0.5"].printTreeHelper(level+1, "-0.5")
		}
		for i := 0; i < lengthNode.Name.(int); i++ {
			returnValue += node.Parts[strconv.Itoa(i)].printTreeHelper(level+1, strconv.Itoa(i))
			if _, ok := node.Parts[strconv.Itoa(i)+".5"]; ok {
				returnValue += node.Parts[strconv.Itoa(i)+".5"].printTreeHelper(level+1, strconv.Itoa(i)+".5")
			}
		}
	}
	return returnValue
}

func printSpaces(level int) string {
	return strings.Repeat("  ", level)
}

func splitIntoChunks(input string) []string {
	var chunks []string
	if input == "" {
		return chunks
	}

	var currentChunk string
	var isCurrentWhitespace bool

	currentChunk = string(input[0])
	isCurrentWhitespace = isWhitespaceRune(rune(input[0]))

	for i := 1; i < len(input); i++ {
		r := rune(input[i])
		isWhitespace := isWhitespaceRune(r)

		if r == '\n' && len(currentChunk) > 0 && currentChunk[len(currentChunk)-1] == '\r' {
			currentChunk += string(r)
			continue
		}

		if isWhitespace == isCurrentWhitespace {
			currentChunk += string(r)
		} else {
			chunks = append(chunks, currentChunk)
			currentChunk = string(r)
			isCurrentWhitespace = isWhitespace
		}
	}

	chunks = append(chunks, currentChunk)
	return chunks
}

func isWhitespaceRune(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' || r == '\v'
}

func Parse(code string) (*ContainerNode, error) {
	tokens := splitIntoChunks(code)
	root := NewContainerNode("", NodeType, nil)
	root.Parts["scope"] = NewContainerNode(true, BoolType, root)
	current := root

	stringBuilder := []string{}

	for _, token := range tokens {
		ender := DsshitSpec.Starters[current.String()]
		flattenerEnd, insideFlattener := DsshitSpec.Flatteners[current.String()]

		if len(stringBuilder) > 0 {
			if stringBuilder[0] == "/*" {
				if stringBuilder[len(stringBuilder)-1] == "*/" {
					stringBuilder = append(stringBuilder, token)
					current.AppendToHalfPush(strings.Join(stringBuilder, ""))
					stringBuilder = []string{}
					continue
				} else {
					if token == "*/" {
						stringBuilder = append(stringBuilder, token)
					} else if lastPart, ok := strings.CutSuffix(token, "*/"); ok {
						stringBuilder = append(stringBuilder, lastPart, "*/")
					} else {
						stringBuilder = append(stringBuilder, token)
					}
					continue
				}
			} else if stringBuilder[0] == "//" {
				if strings.Contains(token, "\n") {
					stringBuilder = append(stringBuilder, token)
					current.AppendToHalfPush(strings.Join(stringBuilder, ""))
					stringBuilder = []string{}
					continue
				} else {
					stringBuilder = append(stringBuilder, token)
					continue
				}
			}
		}

		if ender == token {
			if current.Parts["parent"] == nil {
				fmt.Println(current.Inspect())
				return root, fmt.Errorf("unexpected end token")
			}

			if insideFlattener {
				current.Push(NewContainerNode(strings.Join(stringBuilder, ""), StringType, current))
				stringBuilder = []string{}
			}

			current = current.Parts["parent"]
			continue
		}

		if insideFlattener {
			if beforeEnd, found := strings.CutSuffix(token, flattenerEnd); found {
				current.Push(NewContainerNode(strings.Join(append(stringBuilder, beforeEnd), ""), StringType, current))
				current = current.Parts["parent"]
				stringBuilder = []string{}
				continue
			}
			stringBuilder = append(stringBuilder, token)
			continue
		}

		if strings.HasPrefix(token, ".") {
			if strings.HasSuffix(token, "(") {
				newNode := NewContainerNode(".", MethodType, current)
				newNode.Push(NewContainerNode(token[1:], StringType, newNode))
				current.Push(newNode)
				current = newNode
			} else if strings.HasSuffix(token, "()") {
				newNode := NewContainerNode(token, EmptyMethodType, current)
				current.Push(newNode)
			} else {
				newNode := NewContainerNode(token, PropertyType, current)
				current.Push(newNode)
			}
			continue
		}

		if current.String() == "-" && current.Parts["1"] != nil {
			current = current.Parts["parent"]
			continue
		}

		if strings.HasPrefix(token, "-") {
			if current.Name == "-" {
				current = current.Parts["parent"]
			}
			flag := token[1:]
			newNode := NewContainerNode("-", NodeType, current)

			current.Push(newNode)
			if strings.Contains(flag, "=") {
				parts := strings.Split(flag, "=")
				newNode.Push(NewContainerNode(parts[0], StringType, newNode))
				newNode.Push(NewContainerNode(parts[1], StringType, newNode))
				continue
			}
			newNode.Push(NewContainerNode(flag, StringType, newNode))
			current = newNode
			continue
		}

		_, isStarter := DsshitSpec.Starters[token]
		if isStarter {
			newNode := NewContainerNode(token, NodeType, current)

			current.Push(newNode)
			current = newNode
			continue
		}

		if (current.String() == "{" || strings.HasPrefix(current.String(), "map[string]")) &&
			strings.HasSuffix(token, ":") {
			newNode := NewContainerNode(token, PropType, current)
			current.Push(newNode)
			continue
		}

		if strings.HasPrefix(token, "!") {
			switch token[1:] {
			case "true":
				current.Push(NewContainerNode(true, BoolType, current))
			case "false":
				current.Push(NewContainerNode(false, BoolType, current))
			case "nil":
				current.Push(NewContainerNode(nil, ContainerType, current))
			case "*":
				current.Push(NewContainerNode(make(chan string), ChanStringType, current))
			default:
				if strings.HasSuffix(token, "()") {
					newNode := NewContainerNode("!", NodeType, current)
					newNode.Push(NewContainerNode(token[1:], EmptyFuncType, newNode))
					current.Push(newNode)
				} else if strings.HasSuffix(token, "(") {
					newNode := NewContainerNode("!", NodeType, current)
					newNode.Push(NewContainerNode(token[1:], FuncType, newNode))
					current.Push(newNode)
					current = newNode
				}
			}
			continue
		}

		if strings.HasPrefix(token, "$") {
			name := token[1:]
			newNode := NewContainerNode("$", NodeType, current)
			newNode.Push(NewContainerNode(name, StringType, newNode))
			current.Push(newNode)
			continue
		}

		if strings.HasPrefix(token, "#") {
			name, err := strconv.Atoi(token[1:])
			if err != nil {
				return root, err
			}
			current.Push(NewContainerNode(name, IntType, current))
			continue
		}

		if strings.HasPrefix(token, "@") {
			name, err := strconv.ParseFloat(token[1:], 64)
			if err != nil {
				return root, err
			}
			current.Push(NewContainerNode(name, Float64Type, current))
			continue
		}

		if strings.HasPrefix(token, "~") {
			unquoted, err := strconv.Unquote(token[1:])
			if err != nil {
				fmt.Println("Error:", err)
			}
			current.Push(NewContainerNode(unquoted, StringType, current))
			continue
		}

		if strings.HasPrefix(token, "'") ||
			strings.HasPrefix(token, `"`) ||
			strings.HasPrefix(token, "`") {
			quote := token[:1]
			if strings.HasSuffix(token, quote) {
				unquoted, err := strconv.Unquote(`"` + token[1:len(token)-1] + `"`)
				if err != nil {
					fmt.Println("Error: f", err)
				}
				if quote == "`" {
					unquoted = token[1 : len(token)-1]
				}
				current.Push(NewContainerNode(unquoted, StringType, current))
				continue
			}
			newNode := NewContainerNode(quote, NodeType, current)
			stringBuilder = []string{token[1:]}
			current.Push(newNode)
			current = newNode
			continue
		}

		if comment, ok := strings.CutPrefix(token, "/*"); ok {
			if middle, ok := strings.CutSuffix(comment, "*/"); ok {
				stringBuilder = []string{"/*", middle, "*/"}
				continue
			}
			stringBuilder = []string{"/*", comment}
			continue
		}

		if comment, ok := strings.CutPrefix(token, "//"); ok {
			if strings.Contains(token, "\n") {
				stringBuilder = append(stringBuilder, token)
				current.AppendToHalfPush(strings.Join(stringBuilder, ""))
				stringBuilder = []string{}
				continue
			}
			stringBuilder = []string{"//", comment}
			continue
		}

		newNode := NewContainerNode(token, StringType, current)
		isWhitespace := isWhitespaceRune(rune(token[0]))

		if isWhitespace {
			current.HalfPush(newNode)
			continue
		}

		unquoted, err := strconv.Unquote(`"` + token + `"`)
		if err != nil {
			fmt.Println("Error:", err)
		}

		current.Push(NewContainerNode(unquoted, StringType, current))
	}
	if current != root {
		fmt.Println(root.Inspect())
		return root, fmt.Errorf("parser did not return the root node")
	}
	if len(stringBuilder) > 0 {
		switch stringBuilder[0] {
		case "/*":
			if stringBuilder[len(stringBuilder)-1] == "*/" {
				current.AppendToHalfPush(strings.Join(stringBuilder, ""))
			} else {
				fmt.Println(root.Inspect())
				return root, fmt.Errorf("mutliline comment is not closed")
			}
		case "//":
			current.AppendToHalfPush(strings.Join(stringBuilder, ""))
		}
	}

	return root, nil
}
