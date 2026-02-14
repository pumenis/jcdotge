package main

var tests = []struct {
	name        string
	code        string
	stdout      string
	args        []string
	returnValue string
	err         bool
}{
	{
		name: "function declaration with echo command",
		code: `
			function hello do 
				|: | echo hi :|
			done end
			
			|: | hello | toupper :|
			
			:(
				|: | hello :| 
			)
			`,
		args:        []string{},
		stdout:      "hi\n",
		returnValue: "HI",
		err:         false,
	},
}
