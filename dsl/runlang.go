// Package dsl
package dsl

import (
	"fmt"
	"maps"
	"math"
	"strconv"
	"strings"

	"github.com/pumenis/jcdotge/parser"
)

var (
	funcs           map[string]func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode
	evalFuncs       map[string]func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode
	methodCallFuncs map[string]func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode
)

func eval(value *parser.ContainerNode) *parser.ContainerNode {
	if value.Type == parser.NodeType {
		args := []*parser.ContainerNode{}
		for i := 0; i < value.Parts["length"].Name.(int); i++ {
			args = append(args, value.Parts[strconv.Itoa(i)])
		}
		return evalFuncs[value.String()](value, args...)
	}
	return value
}

func chainFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	newNode := *value
	newNode.Parts = make(map[string]*parser.ContainerNode)
	maps.Copy(newNode.Parts, value.Parts)
	newNode.Parts["0"] = eval(newNode.Parts["0"])
	newNode.Name = ":{"
	return eval(&newNode)
}

func scopeEvalFunc(args ...*parser.ContainerNode) []*parser.ContainerNode {
	chain := []*parser.ContainerNode{}
	result := []*parser.ContainerNode{}
	for i, component := range args {
		if strings.HasPrefix(component.String(), ".") || component.String() == "[" || component.Type == parser.PropertyType {
			chain = append(chain, component)
			if len(args)-1 == i || (!strings.HasPrefix(args[i+1].String(), ".") && args[i+1].String() != "[" && args[i+1].Type != parser.PropertyType) {
				variable := chain[0]

				for _, currentNode := range chain[1:] {
					if currentNode.String() == "[" {
						key := eval(currentNode.Parts["0"])
						if key.Type == parser.IntType {
							variable = variable.Parts[key.String()]
						} else {
							variable = variable.Parts["."+key.String()]
						}
					} else if currentNode.Type == parser.PropertyType {
						variable = variable.Parts[currentNode.String()]
					} else if currentNode.Type == parser.EmptyMethodType {
						name := currentNode.String()
						name = name[1 : len(name)-2]
						_, ok := methodCallFuncs[name]
						if !ok {
							fu, err := loadFunc(name)
							if err != nil {
								variable = exEc(variable, append([]*parser.ContainerNode{
									parser.NewContainerNode(name, parser.StringType, variable),
								}, args...)...)
								continue
							}
							methodCallFuncs[name] = fu
						}
						variable = variable.Call(methodCallFuncs, name)
					} else if currentNode.Type == parser.MethodType {
						argms := []*parser.ContainerNode{}
						for j := 1; j < currentNode.Parts["length"].Name.(int); j++ {
							argms = append(argms, currentNode.Parts[strconv.Itoa(j)])
						}
						argms = scopeEvalFunc(argms...)
						name := currentNode.Parts["0"].String()
						name = name[:len(name)-1]
						_, ok := methodCallFuncs[name]
						if !ok {
							fu, err := loadFunc(name)
							if err != nil {
								variable = exEc(variable, append([]*parser.ContainerNode{
									parser.NewContainerNode(name, parser.StringType, variable),
								}, argms...)...)
								continue
							}
							methodCallFuncs[name] = fu
						}
						variable = variable.Call(methodCallFuncs, name, argms...)
					}
				}
				eValue := variable
				result = append(result, eValue)
				chain = []*parser.ContainerNode{}
			}
		} else if len(args)-1 > i && (strings.HasPrefix(args[i+1].String(), ".") || args[i+1].String() == "[" || args[i+1].Type == parser.PropertyType) {
			eValue := eval(component)
			chain = append(chain, eValue)
		} else {
			eValue := eval(component)
			result = append(result, eValue)
		}
	}
	return result
}

func rawChainFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	variable := args[0]
	mapForDeletion := map[string]*parser.ContainerNode{}
	var keyForDeletion string
	isArray := false

	for _, currentNode := range args[1:] {
		if currentNode.String() == "[" {
			key := eval(currentNode.Parts["0"])
			if key.Type == parser.IntType {
				mapForDeletion = variable.Parts
				keyForDeletion = key.String()
				isArray = true
				variable = variable.Parts[key.String()]
			} else {
				mapForDeletion = variable.Parts
				keyForDeletion = "." + key.String()
				isArray = false
				variable = variable.Parts["."+key.String()]
			}
		} else if currentNode.Type == parser.PropertyType {
			mapForDeletion = variable.Parts
			keyForDeletion = currentNode.String()
			isArray = false
			variable = variable.Parts[currentNode.String()]
		} else if currentNode.Type == parser.EmptyMethodType {
			if currentNode.String() == ".unset()" {

				delete(mapForDeletion, keyForDeletion)
				if isArray {
					for i, _ := strconv.Atoi(keyForDeletion); i < mapForDeletion["length"].Name.(int)-1; i++ {
						mapForDeletion[strconv.Itoa(i)] = mapForDeletion[strconv.Itoa(i+1)]
					}
					mapForDeletion["length"].Name = mapForDeletion["length"].Name.(int) - 1
				}
			} else {
				name := currentNode.String()
				name = name[1 : len(name)-2]
				_, ok := methodCallFuncs[name]
				if !ok {
					fu, err := loadFunc(name)
					if err != nil {
						variable = exEc(variable, append([]*parser.ContainerNode{
							parser.NewContainerNode(name, parser.StringType, variable),
						}, args...)...)
						continue
					}
					methodCallFuncs[name] = fu
				}
				variable = variable.Call(methodCallFuncs, name)
			}
		} else if currentNode.Type == parser.MethodType {
			args := []*parser.ContainerNode{}
			for i := 1; i < currentNode.Parts["length"].Name.(int); i++ {
				args = append(args, eval(currentNode.Parts[strconv.Itoa(i)]))
			}
			name := currentNode.Parts["0"].String()
			name = name[:len(name)-1]
			_, ok := methodCallFuncs[name]
			if !ok {
				fu, err := loadFunc(name)
				if err != nil {
					variable = exEc(variable, append([]*parser.ContainerNode{
						parser.NewContainerNode(name, parser.StringType, variable),
					}, args...)...)
					continue
				}
				methodCallFuncs[name] = fu
			}
			variable = variable.Call(methodCallFuncs, name, args...)
		}
	}
	return variable
}

func buildArray(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	output := parser.NewContainerNode("[]{", parser.ArrayType, value)

	for _, arg := range args {
		output.Push(eval(arg))
	}
	return output
}

func buildMap(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	output := parser.NewContainerNode("{", parser.MapType, value)
	var key string
	for _, current := range args {
		switch current.Type {
		case parser.PropType:
			key = "." + current.String()
			key = key[:len(key)-1]
		default:
			output.Parts[key] = eval(current)
		}
	}
	return output
}

func addFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	firstValue := eval(args[0])
	switch firstValue.Type {
	case parser.Int8Type:
		return parser.NewContainerNode(
			firstValue.Name.(int8)+eval(args[1]).Name.(int8),
			parser.Int8Type, value)
	case parser.IntType:
		return parser.NewContainerNode(
			firstValue.Name.(int)+eval(args[1]).Name.(int),
			parser.IntType, value)
	case parser.Float64Type:
		return parser.NewContainerNode(
			firstValue.Name.(float64)+eval(args[1]).Name.(float64),
			parser.Float64Type, value)
	default:
		panic("Unknown type or Cannot Be added: " + firstValue.Type.String())
	}
}

func subtractFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	firstValue := eval(args[0])
	switch firstValue.Type {
	case parser.Int8Type:
		return parser.NewContainerNode(
			firstValue.Name.(int8)-eval(args[1]).Name.(int8),
			parser.Int8Type, value)
	case parser.IntType:
		return parser.NewContainerNode(
			firstValue.Name.(int)-eval(args[1]).Name.(int),
			parser.IntType, value)
	case parser.Float64Type:
		return parser.NewContainerNode(
			firstValue.Name.(float64)-eval(args[1]).Name.(float64),
			parser.Float64Type, value)
	default:
		panic("Unknown type or Cannot Be subtracted: " + firstValue.Type.String())
	}
}

func multiplyFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	firstValue := eval(args[0])
	switch firstValue.Type {
	case parser.Int8Type:
		return parser.NewContainerNode(
			firstValue.Name.(int8)*eval(args[1]).Name.(int8),
			parser.Int8Type, value)
	case parser.IntType:
		return parser.NewContainerNode(
			firstValue.Name.(int)*eval(args[1]).Name.(int),
			parser.IntType, value)
	case parser.Float64Type:
		return parser.NewContainerNode(
			firstValue.Name.(float64)*eval(args[1]).Name.(float64),
			parser.Float64Type, value)
	default:
		panic("Unknown type or Cannot Be multiplied: " + firstValue.Type.String())
	}
}

func divideFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	firstValue := eval(args[0])
	switch firstValue.Type {
	case parser.Int8Type:
		return parser.NewContainerNode(
			firstValue.Name.(int8)/eval(args[1]).Name.(int8),
			parser.Int8Type, value)
	case parser.IntType:
		return parser.NewContainerNode(
			firstValue.Name.(int)/eval(args[1]).Name.(int),
			parser.IntType, value)
	case parser.Float64Type:
		return parser.NewContainerNode(
			firstValue.Name.(float64)/eval(args[1]).Name.(float64),
			parser.Float64Type, value)
	default:
		panic("Unknown type or Cannot Be divided: " + firstValue.Type.String())
	}
}

func powerFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	firstValue := eval(args[0])
	first, ok := toFloat64(firstValue)
	if !ok {
		panic("first value not numeric: " + firstValue.Type.String())
	}
	secondValue := eval(args[1])
	second, ok := toFloat64(secondValue)
	if !ok {
		panic("second value not numeric: " + secondValue.Type.String())
	}

	return parser.NewContainerNode(math.Pow(first, second), parser.Float64Type, value)
}

func moduloFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	firstValue := eval(args[0])
	switch firstValue.Type {
	case parser.Int8Type:
		return parser.NewContainerNode(
			firstValue.Name.(int8)%eval(args[1]).Name.(int8),
			parser.Int8Type, value)
	case parser.IntType:
		return parser.NewContainerNode(
			firstValue.Name.(int)%eval(args[1]).Name.(int),
			parser.IntType, value)
	default:
		panic("Unknown type or Cannot get modulo: " + firstValue.Type.String())
	}
}

func RunLang(value *parser.ContainerNode, args ...string) *parser.ContainerNode {
	for i, arg := range args {
		argName := strconv.Itoa(i)
		value.Parts["$"+argName] = parser.NewContainerNode(arg, parser.StringType, value)
	}
	value.Parts["$argcount"] = parser.NewContainerNode(len(args), parser.IntType, value)

	return eval(value)
}

func runScript(code *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	go func() {
		components := scopeEvalFunc(args...)
		for _, component := range components {
			if component.Type == parser.ChanStringType {
				ch, ok := component.Name.(chan string)
				if !ok {
					panic("runscript: this is not chan string")
				}
				for line := range ch {
					out <- line
				}
			}
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, code)
}

func compareStrings(value *parser.ContainerNode, compareCallback func(string, string) bool, args ...*parser.ContainerNode) *parser.ContainerNode {
	first := eval(args[0]).String()
	second := eval(args[1]).String()

	return parser.NewContainerNode(compareCallback(first, second), parser.BoolType, value)
}

func checkIfStringGreater(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareStrings(value, func(first string, second string) bool {
		return first > second
	}, args...)
}

func checkIfStringLess(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareStrings(value, func(first string, second string) bool {
		return first < second
	}, args...)
}

func checkIfStringGreaterOrEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareStrings(value, func(first string, second string) bool {
		return first >= second
	}, args...)
}

func checkIfStringLessOrEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareStrings(value, func(first string, second string) bool {
		return first <= second
	}, args...)
}

func checkIfStringsEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareStrings(value, func(first string, second string) bool {
		return first == second
	}, args...)
}

func checkIfStringsDontEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareStrings(value, func(first string, second string) bool {
		return first != second
	}, args...)
}

func compareNumbers(value *parser.ContainerNode, compareCallback func(float64, float64) bool, args ...*parser.ContainerNode) *parser.ContainerNode {
	var first float64
	var ok bool
	first, ok = toFloat64(eval(args[0]).Name)
	if !ok {
		panic("first value not numeric: " + args[0].String())
	}

	var second float64
	second, ok = toFloat64(eval(args[1]).Name)
	if !ok {
		panic("second value not numeric: " + args[0].String())
	}

	return parser.NewContainerNode(compareCallback(first, second), parser.BoolType, value)
}

func checkIfNumbersDontEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareNumbers(value, func(first float64, second float64) bool {
		return first != second
	}, args...)
}

func checkIfNumberGreater(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareNumbers(value, func(first float64, second float64) bool {
		return first > second
	}, args...)
}

func checkIfNumberLess(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareNumbers(value, func(first float64, second float64) bool {
		return first < second
	}, args...)
}

func checkIfNumbersEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareNumbers(value, func(first float64, second float64) bool {
		return first == second
	}, args...)
}

func checkIfNumberGreaterOrEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareNumbers(value, func(first float64, second float64) bool {
		return first >= second
	}, args...)
}

func checkIfNumberLessOrEqual(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return compareNumbers(value, func(first float64, second float64) bool {
		return first <= second
	}, args...)
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	case float32:
		return float64(val), true
	case float64:
		return val, true
	default:
		return 0, false
	}
}

func logic(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	operator := value.String()
	vals := []bool{}
	for _, current := range args {

		vals = append(vals, eval(current).Name.(bool))
		continue
	}
	if len(vals) < 2 {
		panic("not enough values")
	}
	switch operator {
	case "&&(":
		for _, v := range vals {
			if !v {
				return parser.NewContainerNode(false, parser.BoolType, value)
			}
		}
		return parser.NewContainerNode(true, parser.BoolType, value)
	case "||(":
		for _, v := range vals {
			if v {
				return parser.NewContainerNode(true, parser.BoolType, value)
			}
		}
		return parser.NewContainerNode(false, parser.BoolType, value)
	default:
		panic("Unknown operator: " + operator)
	}
}

func variableGet(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	variableName := args[0].String()
	return value.FindVariableParent("$" + variableName).Parts["$"+variableName]
}

func buildString(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	quote := `"`
	if value.String() != `'` {
		quote = value.String()
	}
	unquoted, err := strconv.Unquote(quote + args[0].String() + quote)
	if err != nil {
		fmt.Println("Error: m", err)
	}
	return parser.NewContainerNode(unquoted, parser.StringType, value)
}

func functionCall(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	NameNode := args[0]
	functionName := NameNode.String()
	if NameNode.Type == parser.EmptyFuncType {
		functionName = functionName[:len(functionName)-2]
		return funcs[functionName](
			parser.NewContainerNode(functionName, parser.NodeType, value),
		)
	}
	functionName = functionName[:len(functionName)-1]
	evalValue := parser.NewContainerNode(functionName, parser.NodeType, value)
	arguments := scopeEvalFunc(args[1:]...)
	funct, ok := funcs[functionName]
	var err error
	if !ok {
		funct, err = loadFunc(functionName)
		if err != nil {
			return eXec(evalValue, append([]*parser.ContainerNode{
				parser.NewContainerNode(functionName, parser.StringType, evalValue),
			}, arguments...)...)
		}
		funcs[functionName] = funct
	}
	return funct(evalValue, arguments...)
}

func buildMapStringToAny(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	output := map[string]any{}
	var key string
	for _, current := range args {
		switch current.Type {
		case parser.PropType:
			key = current.String()
			key = key[:len(key)-1]
		default:
			output[key] = eval(current).Name
		}
	}
	return parser.NewContainerNode(output, parser.MapStringToAnyType, value)
}

func buildMapStringToString(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	output := map[string]string{}
	var key string
	for _, current := range args {
		switch current.Type {
		case parser.PropType:
			key = current.String()
			key = key[:len(key)-1]
		default:
			output[key] = eval(current).String()
		}
	}
	return parser.NewContainerNode(output, parser.MapStringToStringType, value)
}

func buildMapStringToInt(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	output := map[string]int{}
	var key string
	var ok bool
	for _, current := range args {
		switch current.Type {
		case parser.PropType:
			key = current.String()
			key = key[:len(key)-1]
		default:
			output[key], ok = eval(current).Name.(int)
			if !ok {
				panic("mapStringToInt should take only ints as an arguments")
			}
		}
	}
	return parser.NewContainerNode(output, parser.MapStringToIntType, value)
}

func buildMapStringToFloat64(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	output := map[string]float64{}
	var key string
	var ok bool
	for _, current := range args {
		switch current.Type {
		case parser.PropType:
			key = current.String()
			key = key[:len(key)-1]
		default:
			output[key], ok = eval(current).Name.(float64)
			if !ok {
				panic("mapStringToFloat64 should take only float64s as an arguments")
			}
		}
	}
	return parser.NewContainerNode(output, parser.MapStringToFloat64Type, value)
}

func flagFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	outValue := *value
	outValue.Parts = make(map[string]*parser.ContainerNode)
	maps.Copy(outValue.Parts, value.Parts)
	outValue.Parts["1"] = eval(args[1])
	return &outValue
}

func returnSelf(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return value
}

func concatFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	var argsevaled []string
	for _, component := range args {
		argsevaled = append(argsevaled, eval(component).String())
	}
	return parser.NewContainerNode(strings.Join(argsevaled, ""), parser.StringType, value)
}

func init() {
	evalFuncs = map[string]func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode{
		"":                    runScript,
		"[>":                  checkIfNumberGreater,
		"[<":                  checkIfNumberLess,
		"[>=":                 checkIfNumberGreaterOrEqual,
		"[<=":                 checkIfNumberLessOrEqual,
		"[==":                 checkIfNumbersEqual,
		"[!=":                 checkIfNumbersDontEqual,
		"[>~":                 checkIfStringGreater,
		"[<~":                 checkIfStringLess,
		"[>=~":                checkIfStringGreaterOrEqual,
		"[<=~":                checkIfStringLessOrEqual,
		"[=~":                 checkIfStringsEqual,
		"[!=~":                checkIfStringsDontEqual,
		"&&(":                 logic,
		"||(":                 logic,
		"do":                  returnSelf,
		"function":            function,
		"if":                  iF,
		":(":                  subShell,
		"(":                   chainFunc,
		":{":                  rawChainFunc,
		"$":                   variableGet,
		"!":                   functionCall,
		"-":                   flagFunc,
		"[]{":                 buildArray,
		"{":                   buildMap,
		"(.":                  concatFunc,
		"(+":                  addFunc,
		"(-":                  subtractFunc,
		"(*":                  multiplyFunc,
		"(/":                  divideFunc,
		"(%":                  moduloFunc,
		"(**":                 powerFunc,
		`"`:                   buildString,
		`'`:                   buildString,
		"`":                   buildString,
		"map[string]any{":     buildMapStringToAny,
		"map[string]string{":  buildMapStringToString,
		"map[string]int{":     buildMapStringToInt,
		"map[string]float64{": buildMapStringToFloat64,
	}
}
