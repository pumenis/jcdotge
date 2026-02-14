package main

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/pumenis/jcdotge/dsl"
	"github.com/pumenis/jcdotge/parser"
)

func captureStdout(f func()) string {
	// Save the original stdout
	originalStdout := os.Stdout

	// Create a pipe
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function that prints to stdout
	f()

	// Close the writer to flush and signal EOF
	w.Close()

	// Restore original stdout
	os.Stdout = originalStdout

	// Read everything that was written
	output, _ := io.ReadAll(r)
	r.Close()

	return string(output)
}

func TestRunLang(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cst, err := parser.Parse(tt.code)
			if err != nil {
				t.Fatal(err)
			}

			for i, arg := range tt.args {
				argName := strconv.Itoa(i)
				cst.Parts["$"+argName] = parser.NewContainerNode(arg, parser.StringType, cst)
			}
			got := dsl.RunLang(cst)
			if !reflect.DeepEqual(fmt.Sprint(got.Name), tt.returnValue) {
				t.Errorf("RunLang() value = %#v, want %#v", got, tt.returnValue)
			}
			stdout := captureStdout(func() {
				dsl.RunLang(cst)
			})
			if !reflect.DeepEqual(stdout, tt.stdout) {
				t.Errorf("RunLang() stdout = %#v, want %#v", stdout, tt.stdout)
			}
		})
	}
}
