package dsl

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pumenis/jcdotge/homedir"
	"github.com/pumenis/jcdotge/parser"
)

func printFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := []any{}
	for _, node := range args {
		if node.Type != parser.NodeType {
			out = append(out, node.String())
		}
	}
	fmt.Println(out...)
	return parser.NewContainerNode(true, parser.BoolType, value)
}

func trimPrefixFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return parser.NewContainerNode(strings.TrimPrefix(args[0].String(), args[1].String()), parser.StringType, value)
}

func trimSuffixFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return parser.NewContainerNode(strings.TrimSuffix(args[0].String(), args[1].String()), parser.StringType, value)
}

func eXec(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
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
		panic("exec func " + err.Error())
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic("exec func stderr" + err.Error())
	}

	scanner := bufio.NewScanner(stdout)
	errscanner := bufio.NewScanner(stderr)
	go func() {
		if err := cmd.Start(); err != nil {
			panic("exec func " + err.Error())
		}

		for scanner.Scan() {
			out <- scanner.Text()
		}
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

		close(out)
	}()

	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func varInitialize(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	scopeParent := value.FindScopeParent()
	variable := args[1]
	variable.Parts["parent"] = scopeParent
	scopeParent.Parts["$"+args[0].String()] = variable
	return parser.NewContainerNode(true, parser.BoolType, value)
}

func mkdir(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	go func() {
		for _, arg := range args {
			err := os.MkdirAll(arg.String(), 0o755)
			if err != nil {
				fmt.Println("Error reading file:", err)
				return
			}
		}
		close(out)
	}()

	return parser.NewContainerNode(true, parser.BoolType, value)
}

func stdin(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	scanner := bufio.NewScanner(os.Stdin)
	out := make(chan string)

	go func() {
		for scanner.Scan() {
			out <- scanner.Text()
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func echo(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	go func() {
		outPut := []string{}
		for _, arg := range args {
			if arg.Type != parser.NodeType {
				outPut = append(outPut, arg.String())
			}
		}
		for line := range strings.SplitSeq(strings.Join(outPut, " "), "\n") {
			out <- line
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func cat(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	out := make(chan string)
	go func() {
		for _, arg := range args {

			path, err := homedir.Expand(arg.String())
			if err != nil {
				panic("cat cannot expand path" + err.Error())
			}

			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Println("Error reading file:", err)
				return
			}
			for line := range strings.SplitSeq(string(data), "\n") {
				out <- line
			}
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func ls(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	pattern := "*"
	if len(args) > 0 {
		pattern = eval(args[0]).String()
	}
	out := make(chan string)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Println("error getting the glob", err)
	}
	go func() {
		for _, match := range matches {
			out <- match
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func exitFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	status, ok := args[0].Name.(int)
	if !ok {
		panic("exit status must be an integer")
	}
	os.Exit(status)
	return parser.NewContainerNode(true, parser.BoolType, value)
}

func getHTML(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	url := args[0].String()
	resp, err := http.Get(url)
	if err != nil {
		return parser.NewContainerNode("", parser.StringType, value)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		fmt.Println("Received 429 Too Many Requests")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return parser.NewContainerNode("", parser.StringType, value)
	}

	return parser.NewContainerNode(string(body), parser.StringType, value)
}

func sleepFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	val := args[0].String()

	if leftover, ok := strings.CutSuffix(val, "ms"); ok {
		amount, err := strconv.ParseFloat(leftover, 64)
		if err != nil {
			panic("sleep incorrect amount given " + err.Error())
		}
		time.Sleep(time.Duration(amount) * time.Millisecond)
	} else if leftover, ok := strings.CutSuffix(val, "s"); ok {
		amount, err := strconv.ParseFloat(leftover, 64)
		if err != nil {
			panic("sleep incorrect amount given " + err.Error())
		}
		time.Sleep(time.Duration(amount) * time.Second)
	} else if leftover, ok := strings.CutSuffix(val, "m"); ok {
		amount, err := strconv.ParseFloat(leftover, 64)
		if err != nil {
			panic("sleep incorrect amount given " + err.Error())
		}
		time.Sleep(time.Duration(amount) * time.Minute)
	} else if leftover, ok := strings.CutSuffix(val, "h"); ok {
		amount, err := strconv.ParseFloat(leftover, 64)
		if err != nil {
			panic("sleep incorrect amount given " + err.Error())
		}
		time.Sleep(time.Duration(amount) * time.Hour)
	}
	return parser.NewContainerNode(true, parser.BoolType, value)
}

func nlFunc(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	beginning := true
	start := 1
	end := 2
	separator := ""
	str := ""
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
		case "end":
			end, ok = arg.Parts["1"].Name.(int)
			if !ok {
				panic("nl methods flag end must be int type")
			}
		case "string":
			str = arg.Parts["1"].String()
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

	go func() {
		for i := start; i <= end; i++ {
			out <- callback(str, i)
		}
		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func while(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	var (
		code      *parser.ContainerNode
		condition *parser.ContainerNode
		out       = make(chan string)
	)

	for i, arg := range args {
		name := arg.String()

		if name == "do" {
			arg.Parts["scope"] = parser.NewContainerNode(true, parser.BoolType, arg)
			code = arg
		}

		if i == 0 {
			condition = arg
		}
	}

	go func() {
		for {
			cond := eval(condition)
			b, ok := cond.Name.(bool)
			if !ok {
				panic("while: expected boolean condition, got " + cond.Type.String() + " " + cond.String())
			}
			if !b {
				break
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
						panic("while this is not chan string")
					}
					for line := range ch {
						out <- line
					}
				}
			}
		}

		close(out)
	}()
	return parser.NewContainerNode(out, parser.ChanStringType, value)
}

func runscript(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	path, err := homedir.Expand(args[0].String())
	if err != nil {
		panic("runscript error expanding path" + err.Error())
	}
	src, err := os.ReadFile(path)
	if err != nil {
		panic("runscript: cannot read script " + err.Error())
	}

	script, err := parser.Parse(string(src))
	if err != nil {
		panic(err)
	}
	for i, value := range args[1:] {
		script.Parts["$"+strconv.Itoa(i+1)] = parser.NewContainerNode(value, parser.MapStringToStringType, script)
	}
	return eval(script)
}

func homeexpand(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	path, err := homedir.Expand(args[0].String())
	if err != nil {
		panic("homeexpand: " + err.Error())
	}
	return parser.NewContainerNode(path, parser.StringType, value)
}

func trimspace(value *parser.ContainerNode, args ...*parser.ContainerNode) *parser.ContainerNode {
	return parser.NewContainerNode(strings.TrimSpace(args[0].String()), parser.StringType, value)
}

func init() {
	funcs = map[string]func(*parser.ContainerNode, ...*parser.ContainerNode) *parser.ContainerNode{
		"runscript":  runscript,
		"sleep":      sleepFunc,
		"nl":         nlFunc,
		"gethtml":    getHTML,
		"print":      printFunc,
		"let":        varInitialize,
		"mkdir":      mkdir,
		"stdin":      stdin,
		"echo":       echo,
		"cat":        cat,
		"ls":         ls,
		"exit":       exitFunc,
		"while":      while,
		"trimprefix": trimPrefixFunc,
		"trimsuffix": trimSuffixFunc,
		"exec":       eXec,
		"homeexpand": homeexpand,
		"trimspace":  trimspace,
	}
}
