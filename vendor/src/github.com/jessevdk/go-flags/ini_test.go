package flags

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestWriteIni(t *testing.T) {
	var tests = []struct {
		args     []string
		options  IniOptions
		expected string
	}{
		{
			[]string{"-vv", "--intmap=a:2", "--intmap", "b:3", "command"},
			IniDefault,
			`[Application Options]
; Show verbose debug information
verbose = true
verbose = true

[Other Options]
; A map from string to int
int-map = a:2
int-map = b:3

`,
		},
		{
			[]string{"-vv", "--intmap=a:2", "--intmap", "b:3", "command"},
			IniDefault | IniIncludeDefaults,
			`[Application Options]
; Show verbose debug information
verbose = true
verbose = true

; A slice of pointers to string
; PtrSlice =

EmptyDescription = false

; Test default value
Default = Some value

; Test default array value
DefaultArray = Some value
DefaultArray = Another value

; Testdefault map value
DefaultMap = some:value
DefaultMap = another:value

; Option only available in ini
only-ini =

[Other Options]
; A slice of strings
StringSlice = some
StringSlice = value

; A map from string to int
int-map = a:2
int-map = b:3

[command.A command]
; Use for extra verbosity
; ExtraVerbose =

`,
		},
		{
			[]string{"command"},
			IniDefault | IniIncludeDefaults | IniCommentDefaults,
			`[Application Options]
; Show verbose debug information
; verbose =

; A slice of pointers to string
; PtrSlice =

; EmptyDescription = false

; Test default value
; Default = Some value

; Test default array value
; DefaultArray = Some value
; DefaultArray = Another value

; Testdefault map value
; DefaultMap = some:value
; DefaultMap = another:value

; Option only available in ini
; only-ini =

[Other Options]
; A slice of strings
; StringSlice = some
; StringSlice = value

; A map from string to int
; int-map = a:1

[command.A command]
; Use for extra verbosity
; ExtraVerbose =

`,
		},
		{
			[]string{"--default=New value", "--default-array=New value", "--default-map=new:value", "command"},
			IniDefault | IniIncludeDefaults | IniCommentDefaults,
			`[Application Options]
; Show verbose debug information
; verbose =

; A slice of pointers to string
; PtrSlice =

; EmptyDescription = false

; Test default value
Default = New value

; Test default array value
DefaultArray = New value

; Testdefault map value
DefaultMap = new:value

; Option only available in ini
; only-ini =

[Other Options]
; A slice of strings
; StringSlice = some
; StringSlice = value

; A map from string to int
; int-map = a:1

[command.A command]
; Use for extra verbosity
; ExtraVerbose =

`,
		},
	}

	for _, test := range tests {
		var opts helpOptions

		p := NewNamedParser("TestIni", Default)
		p.AddGroup("Application Options", "The application options", &opts)

		_, err := p.ParseArgs(test.args)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		inip := NewIniParser(p)

		var b bytes.Buffer
		inip.Write(&b, test.options)

		got := b.String()
		expected := test.expected

		if got != expected {
			ret, err := helpDiff(got, expected)

			if err != nil {
				t.Errorf("Unexpected ini with arguments %+v and ini options %b, expected:\n\n%s\n\nbut got\n\n%s", test.args, test.options, expected, got)
			} else {
				t.Errorf("Unexpected ini with arguments %+v and ini options %b:\n\n%s", test.args, test.options, ret)
			}
		}
	}
}

func TestReadIni(t *testing.T) {
	var opts helpOptions

	p := NewNamedParser("TestIni", Default)
	p.AddGroup("Application Options", "The application options", &opts)

	inip := NewIniParser(p)

	inic := `
; Show verbose debug information
verbose = true
verbose = true

[Application Options]
; A slice of pointers to string
; PtrSlice =

; Test default value
Default = Some value

[Other Options]
# A slice of strings
# StringSlice =

; A map from string to int
int-map = a:2
int-map = b:3

`

	b := strings.NewReader(inic)
	err := inip.Parse(b)

	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	assertBoolArray(t, opts.Verbose, []bool{true, true})

	if v, ok := opts.Other.IntMap["a"]; !ok {
		t.Errorf("Expected \"a\" in Other.IntMap")
	} else if v != 2 {
		t.Errorf("Expected Other.IntMap[\"a\"] = 2, but got %v", v)
	}

	if v, ok := opts.Other.IntMap["b"]; !ok {
		t.Errorf("Expected \"b\" in Other.IntMap")
	} else if v != 3 {
		t.Errorf("Expected Other.IntMap[\"b\"] = 3, but got %v", v)
	}
}

func TestIniCommands(t *testing.T) {
	var opts struct {
		Value string `short:"v" long:"value"`

		Add struct {
			Name int `short:"n" long:"name" ini-name:"AliasName"`

			Other struct {
				O string `short:"o" long:"other"`
			} `group:"Other Options"`
		} `command:"add"`
	}

	p := NewNamedParser("TestIni", Default)
	p.AddGroup("Application Options", "The application options", &opts)

	inip := NewIniParser(p)

	inic := `[Application Options]
value = some value

[add]
AliasName = 5

[add.Other Options]
other = subgroup
`

	b := strings.NewReader(inic)
	err := inip.Parse(b)

	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	assertString(t, opts.Value, "some value")

	if opts.Add.Name != 5 {
		t.Errorf("Expected opts.Add.Name to be 5, but got %v", opts.Add.Name)
	}

	assertString(t, opts.Add.Other.O, "subgroup")
}

func TestIniNoIni(t *testing.T) {
	var opts struct {
		Value string `short:"v" long:"value" no-ini:"yes"`
	}

	p := NewNamedParser("TestIni", Default)
	p.AddGroup("Application Options", "The application options", &opts)

	inip := NewIniParser(p)

	inic := `[Application Options]
value = some value
`

	b := strings.NewReader(inic)
	err := inip.Parse(b)

	if err == nil {
		t.Fatalf("Expected error")
	}

	assertError(t, err, ErrUnknownFlag, "unknown option: value")
}

func TestIniParse(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Cannot create temporary file: %s", err)
	}
	defer os.Remove(file.Name())

	_, err = file.WriteString("value = 123")
	if err != nil {
		t.Fatalf("Cannot write to temporary file: %s", err)
	}

	file.Close()

	var opts struct {
		Value int `long:"value"`
	}

	err = IniParse(file.Name(), &opts)
	if err != nil {
		t.Fatalf("Could not parse ini: %s", err)
	}

	if opts.Value != 123 {
		t.Fatalf("Expected Value to be \"123\" but was \"%d\"", opts.Value)
	}
}

func TestWriteFile(t *testing.T) {
	file, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatalf("Cannot create temporary file: %s", err)
	}
	defer os.Remove(file.Name())

	var opts struct {
		Value int `long:"value"`
	}

	opts.Value = 123

	p := NewParser(&opts, Default)
	ini := NewIniParser(p)

	err = ini.WriteFile(file.Name(), IniIncludeDefaults)
	if err != nil {
		t.Fatalf("Could not write ini file: %s", err)
	}

	found, err := ioutil.ReadFile(file.Name())
	if err != nil {
		t.Fatalf("Could not read written ini file: %s", err)
	}

	expected := "[Application Options]\nValue = 123\n\n"

	if string(found) != expected {
		t.Fatalf("Expected file content to be \"%s\" but was \"%s\"", expected, found)
	}
}
