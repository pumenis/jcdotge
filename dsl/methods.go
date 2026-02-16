package dsl

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pumenis/jcdotge/homedir"
	"github.com/pumenis/jcdotge/parser"
)

func toupper(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	inchan, ok := in.Name.(chan string)
	if !ok {
		panic("toupper input should be chan string but it is " + in.Type.String())
	}
	go func() {
		for inputLine := range inchan {
			out <- strings.ToUpper(inputLine)
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, in)
}

func tolower(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	inchan, ok := in.Name.(chan string)
	if !ok {
		panic("tolower input should be chan string but it is " + in.Type.String())
	}
	go func() {
		for inputLine := range inchan {
			out <- strings.ToLower(inputLine)
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, in)
}

func whileRead(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	var (
		ifs      = " "
		fields   = false
		code     *parser.ContainerNode
		argnames []string
		out      = make(chan string)
	)

	for _, arg := range args {
		if arg.Type == parser.NodeType {

			name := arg.String()

			switch name {
			case "-":
				if flagname, ok := arg.Parts["0"]; ok {
					switch flagname.String() {
					case "ifs":
						ifs = arg.Parts["1"].String()
					case "fields":
						fields = true
					}
				}
			case "do":
				arg.Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, arg)
				code = arg
			}
		} else {
			argnames = append(argnames, arg.String())
		}
	}
	inchan, ok := in.Name.(chan string)
	if !ok {
		panic("while_reads input should be the type chan string but it is " + in.Type.String())
	}
	go func() {
		for inputLine := range inchan {
			var partsAfterSplit []string
			if fields {
				partsAfterSplit = strings.FieldsFunc(inputLine, func(r rune) bool {
					return strings.ContainsRune(ifs, r)
				})
			} else {
				partsAfterSplit = strings.Split(inputLine, ifs)
			}

			for i, name := range argnames {
				if i < len(partsAfterSplit) {
					code.Parts["$"+name] = parser.NewContainerNode(partsAfterSplit[i], parser.StringType, code)
				} else {
					code.Parts["$"+name] = parser.NewContainerNode("", parser.StringType, code)
				}
			}
			var components []*parser.ContainerNode
			for i := 0; i < code.Parts["length"].Name.(int); i++ {
				components = append(components, code.Parts[strconv.Itoa(i)])
			}
			components = scopeEvalFunc(components...)

			for _, component := range components {
				if component.Type == parser.ChanStringType {
					ch, ok := component.Name.(chan string)
					if !ok {
						panic("while-read this is not chan string")
					}
					for line := range ch {
						out <- line
					}
				}
			}
		}

		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, in)
}

func exEc(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	arguments := []string{}

	for _, arg := range args[1:] {
		if arg.String() == "-" {
			if arg.Parts["1"].Type == parser.BoolType && arg.Parts["1"].Name.(bool) {
				arguments = append(arguments, "-"+arg.Parts["0"].String())
			} else if arg.Parts["1"].Type != parser.BoolType {
				arguments = append(arguments, "-"+arg.Parts["0"].String())
				arguments = append(arguments, arg.Parts["1"].String())
			}
		} else {
			arguments = append(arguments, arg.String())
		}
	}

	cmd := exec.Command(args[0].String(), arguments...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic("exec func stderr" + err.Error())
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	inchan, ok := in.Name.(chan string)
	if !ok {
		panic("stdin method inchan is not chan string")
	}

	go func() {
		defer stdin.Close()
		for line := range inchan {
			if _, err := stdin.Write([]byte(line + "\n")); err != nil {
				return
			}
		}
	}()

	// Handle stdout
	go func() {
		defer close(out)

		if err := cmd.Start(); err != nil {
			log.Fatal(err)
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			out <- scanner.Text()
		}

		errscanner := bufio.NewScanner(stderr)
		accumulator := []string{}
		for errscanner.Scan() {
			accumulator = append(accumulator, errscanner.Text())
		}
		if err := scanner.Err(); err != nil {
			fmt.Println(strings.Join(accumulator, "\n"))
		}

		if err := cmd.Wait(); err != nil {
			fmt.Println(strings.Join(accumulator, "\n"))
		}
	}()

	return parser.NewContainerNode(out, parser.ChanStringType, in)
}

func lengthFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return value.Parts["length"]
}

func setFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	newValue := eval(args[0])
	value.Set(newValue)
	return newValue
}

func printMethodCall(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	number := 1
	var ok bool
	if len(args) > 0 {
		number, ok = eval(args[0]).Name.(int)
		if !ok {
			panic("print method expects int as an argument")
		}
	}

	for range number {
		fmt.Println(value.String())
	}

	return value
}

func evalFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return evalFuncs[value.String()](value)
}

// parsEvalFunc first parses string as ast then runs it as code and returns evaluated value
func parsEvalFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	cst, err := parser.Parse(value.String())
	if err != nil {
		panic("cannot parse string in parseval")
	}
	cst.Parts["parent"] = value

	return RunLang(cst)
}

// outFunc method takes string channel as input and outputs to scopes out channel
func outFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	outParent := value.Parts["parent"]
	var out (chan string)
	var ok bool
	for outParent.Parts["out"] == nil {
		outParent = outParent.Parts["parent"]
	}

	out, ok = outParent.Parts["out"].Name.(chan string)
	if !ok {
		panic("run method expects chan string as an output channel")
	}
	for line := range value.Name.(chan string) {
		out <- line
	}
	return parser.NewContainerNode(true, parser.BoolType, value)
}

func stdoutFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	in, ok := value.Name.(chan string)
	if !ok {
		panic("stdoutMethod expects chan string as input type")
	}
	for line := range in {
		fmt.Println(line)
	}
	return parser.NewContainerNode(true, parser.BoolType, value)
}

func toStringFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	in, ok := value.Name.(chan string)
	if !ok {
		panic("stdoutMethod expects chan string as input type")
	}
	str := []string{}
	for line := range in {
		str = append(str, line)
	}
	return parser.NewContainerNode(strings.Join(str, "\n"), parser.StringType, value)
}

func tmplParse(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	tmplStr := value.String()
	data := args[0].Name
	tmpl := template.Must(template.New("").Parse(tmplStr))
	var buf bytes.Buffer

	err := tmpl.Execute(&buf, data)
	if err != nil {
		panic(err)
	}

	result := buf.String()
	return parser.NewContainerNode(result, parser.StringType, value)
}

func padMethod(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	beginning := true
	separator := ""
	str := ""
	var callback func(string) string
	out := make(chan string)

	var ok bool
	for _, arg := range args {
		switch arg.Parts["0"].String() {
		case "beginning":
			beginning, ok = arg.Parts["1"].Name.(bool)
			if !ok {
				panic("nl methods flag beginning must be bool type")
			}
		case "separator":
			separator = arg.Parts["1"].String()
		case "string":
			str = arg.Parts["1"].String()
		}
	}

	if beginning {
		callback = func(s string) string {
			return str + separator + s
		}
	} else {
		callback = func(s string) string {
			return s + separator + str
		}
	}

	in, ok := value.Name.(chan string)
	if !ok {
		panic("pad methods input should be chan string")
	}

	go func() {
		for line := range in {
			out <- callback(line)
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func nlMethod(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	beginning := true
	start := 1
	separator := ""
	var callback func(string, int) string
	out := make(chan string)

	var ok bool
	for _, arg := range args {
		switch arg.Parts["0"].String() {
		case "beginning":
			beginning, ok = arg.Parts["1"].Name.(bool)
			if !ok {
				panic("nl methods flag beginning must be bool type")
			}
		case "separator":
			separator = arg.Parts["1"].String()
		case "start":
			start, ok = arg.Parts["1"].Name.(int)
			if !ok {
				panic("nl methods flag start must be int type")
			}
		}
	}

	if beginning {
		callback = func(s string, i int) string {
			return strconv.Itoa(i) + separator + s
		}
	} else {
		callback = func(s string, i int) string {
			return s + separator + strconv.Itoa(i)
		}
	}

	in, ok := value.Name.(chan string)
	if !ok {
		panic("nl methods input should be chan string")
	}

	go func() {
		i := start
		for line := range in {
			out <- callback(line, i)
			i++
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func writeToFile(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	filename, err := homedir.Expand(args[0].String())
	if err != nil {
		panic("writefile: " + err.Error())
	}
	file, err := os.Create(filename)
	if err != nil {
		panic("writefile: " + err.Error())
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	ch, ok := in.Name.(chan string)
	if !ok {
		panic("writefile: this is not chan string")
	}
	for line := range ch {
		_, err = writer.WriteString(line + "\n")
		if err != nil {
			panic("writefile: " + err.Error())
		}
	}

	err = writer.Flush()
	if err != nil {
		panic("writefile: " + err.Error())
	}
	return parser.NewContainerNode(true, parser.BoolType, in)
}

func appendToFile(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	filename, err := homedir.Expand(args[0].String())
	if err != nil {
		panic("writefile: " + err.Error())
	}
	file, err := os.OpenFile(
		filename,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE, // flags
		0o644,                               // permissions (rw-r--r--)
	)
	if err != nil {
		panic("writefile: " + err.Error())
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	ch, ok := in.Name.(chan string)
	if !ok {
		panic("writefile: this is not chan string")
	}
	for line := range ch {
		_, err = writer.WriteString(line + "\n")
		if err != nil {
			panic("writefile: " + err.Error())
		}
	}

	err = writer.Flush()
	if err != nil {
		panic("writefile: " + err.Error())
	}
	return parser.NewContainerNode(true, parser.BoolType, in)
}

func read(in *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	count := 1
	if len(args) > 0 {
		var ok bool
		count, ok = args[0].Name.(int)
		if !ok {
			panic("read: this is not int")
		}
	}
	ch, ok := in.Name.(chan string)
	if !ok {
		panic("read: this is not chan string")
	}
	go func() {
		i := 0
		for line := range ch {
			out <- line
			i++
			if count == i {
				break
			}
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, in)
}

func init() {
	methodCallFuncs = map[string]func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode{
		"pad":        padMethod,
		"nl":         nlMethod,
		"length":     lengthFunc,
		"eval":       evalFunc,
		"parseval":   parsEvalFunc,
		"out":        outFunc,
		"stdout":     stdoutFunc,
		"tostring":   toStringFunc,
		"print":      printMethodCall,
		"set":        setFunc,
		"toupper":    toupper,
		"tolower":    tolower,
		"while-read": whileRead,
		"tmplparse":  tmplParse,
		"exec":       exEc,
		"writefile":  writeToFile,
		"appendfile": appendToFile,
		"read":       read,
	}
}
