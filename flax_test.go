// Copyright (C) 2023 Michael J. Fromberger. All Rights Reserved.

package flax_test

import (
	"bytes"
	"flag"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/creachadair/flax"
)

// textFlag is a trivial implementation of encoding.TextMarshaler and
// encoding.TextUnmarshaler for testing.
type textFlag struct {
	value string
}

func (t textFlag) MarshalText() ([]byte, error) { return []byte(t.value), nil }

func (t *textFlag) UnmarshalText(data []byte) error {
	t.value = string(data)
	return nil
}

// flagValue is a trivial implementation of flag.Value for testing.
type flagValue struct {
	value string
}

func (f *flagValue) Set(s string) error { f.value = s; return nil }
func (f flagValue) String() string      { return f.value }

// bothValue is a type that implements both flag.Value and text marshaling.
type bothValue struct {
	textFlag
	flagValue
}

func TestBasic(t *testing.T) {
	// Make sure we can successfully bind all the flag types.
	var v struct {
		B   bool      `flag:"bool,Boolean"`
		F   float64   `flag:"float64,Float64"`
		Z   int       `flag:"int,Int"`
		Z64 int64     `flag:"int64,Int64"`
		S   string    `flag:"string,String"`
		T   textFlag  `flag:"text,Text"`
		U   uint      `flag:"uint,Uint"`
		U64 uint64    `flag:"uint64,Uint64"`
		FV  flagValue `flag:"flag-value,FlagValue"`
		BV  bothValue `flag:"flag-and-text,FlagAndText"`
	}
	t.Run("CheckBind", func(t *testing.T) {
		fi, err := flax.Check(&v)
		if err != nil {
			t.Fatalf("Check failed: %v", err)
		}
		fs := flag.NewFlagSet("test", flag.PanicOnError)
		fi.Bind(fs)
	})

	t.Run("CheckFind", func(t *testing.T) {
		fi, err := flax.Check(&v)
		if err != nil {
			log.Fatalf("Check failed: %v", err)
		}

		good := []string{
			"bool", "float64", "int", "int64", "string", "text",
			"uint", "uint64", "flag-value", "flag-and-text",
		}
		for _, ok := range good {
			got := fi.Flag(ok)
			if got == nil {
				t.Errorf("Flag %q missing", ok)
			} else if got.Name != ok {
				t.Errorf("Flag %q found with name %q", ok, got.Name)
			}
			bad := fi.Flag(ok + "-not")
			if bad != nil {
				t.Errorf("Flag %q unexpectedly found", bad.Name)
			}
		}
	})
}

func TestCheckError(t *testing.T) {
	tests := []struct {
		label string
		input any
	}{
		{"not a pointer", struct{}{}},
		{"not a struct", new(int)},
		{"empty struct", &struct{}{}},

		{"none flaggable", &struct {
			foo int
			Bar float64
			Baz func()
		}{}},

		{"incompatible type", &struct {
			F []byte `flag:"bad,type"`
		}{}},

		{"missing usage", &struct {
			S string `flag:"nousage"`
		}{}},

		{"empty name", &struct {
			S string `flag:",empty name"`
		}{}},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			fi, err := flax.Check(tc.input)
			if err == nil {
				t.Fatalf("Got %+v, want error", fi)
			}
		})
	}
}

func TestMustBindAll(t *testing.T) {
	var v1 struct {
		S string `flag:"s,default='ok go',string"`
	}
	var v2 struct {
		Z int `flag:"z,default=25,int"`
	}

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	flax.MustBindAll(fs, &v1, &v2)
	if want := "ok go"; v1.S != want {
		t.Errorf("S: got %q, want %q", v1.S, want)
	}
	if want := 25; v2.Z != want {
		t.Errorf("Z: got %d, want %d", v2.Z, want)
	}
}

func mustBind(t *testing.T, input any) *flag.FlagSet {
	t.Helper()

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	fi, err := flax.Check(input)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	fi.Bind(fs)
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("Parse flags: %v", err)
	}
	return fs
}

func fieldValue(v any, name string) any {
	fv := reflect.ValueOf(v).Elem()
	return fv.FieldByName(name).Interface()
}

func TestBindDefaults(t *testing.T) {
	os.Setenv("TEST_INT", "12345")
	const name = "X"
	tests := []struct {
		label string
		input any
		want  any
	}{
		{"bool", &struct {
			X bool `flag:"x,default=true,y"`
		}{}, true},
		{"bool tag", &struct {
			X bool `flag:"x,y" flag-default:"true"`
		}{}, true},

		{"float64", &struct {
			X float64 `flag:"x,default=0.25,y"`
		}{}, 0.25},
		{"float64 tag", &struct {
			X float64 `flag:"x,y" flag-default:"0.25"`
		}{}, 0.25},

		{"int", &struct {
			X int `flag:"x,default=13,y"`
		}{}, 13},
		{"int tag", &struct {
			X int `flag:"x,y" flag-default:"13"`
		}{}, 13},

		{"int from env", &struct {
			X int `flag:"x,default=$TEST_INT,y"`
		}{}, 12345},
		{"int from env tag", &struct {
			X int `flag:"x,y" flag-default:"$TEST_INT"`
		}{}, 12345},

		{"int64", &struct {
			X int64 `flag:"x,default=7,y"`
		}{}, int64(7)},
		{"int64 tag", &struct {
			X int64 `flag:"x,y" flag-default:"7"`
		}{}, int64(7)},

		{"string", &struct {
			X string `flag:"x,default=cork bat,y"`
		}{}, "cork bat"},
		{"string tag", &struct {
			X string `flag:"x,y" flag-default:"cork bat"`
		}{}, "cork bat"},

		{"complex string", &struct {
			X string `flag:"x,default='a, b, c',y"`
		}{}, "a, b, c"},
		{"complex string tag", &struct {
			X string `flag:"x,y" flag-default:"a, b, c"`
		}{}, "a, b, c"},

		{"internal quotes", &struct {
			X string `flag:"x,default='p,'',q',y"`
		}{}, "p,',q"},
		{"internal quotes tag", &struct {
			X string `flag:"x,y" flag-default:"p,',q"`
		}{}, "p,',q"},

		{"env string", &struct {
			X string `flag:"x,default=$TEST_INT,y"`
		}{}, "12345"},
		{"env string tag", &struct {
			X string `flag:"x,y" flag-default:"$TEST_INT"`
		}{}, "12345"},

		{"env esc string", &struct {
			X string `flag:"x,default=$$TEST_INT,y"` // doubled, do not expand
		}{}, "$TEST_INT"},
		{"env esc string tag", &struct {
			X string `flag:"x,y" flag-default:"$$TEST_INT"` // doubled, do not expand
		}{}, "$TEST_INT"},

		{"text tag", &struct {
			X textFlag `flag:"x,default=bleep,y"`
		}{}, textFlag{"bleep"}},
		{"text", &struct {
			X textFlag `flag:"x,y" flag-default:"bleep"`
		}{}, textFlag{"bleep"}},

		{"star text", &struct {
			X textFlag `flag:"x,default=*,y"`
		}{X: textFlag{"horsefeathers"}}, textFlag{"horsefeathers"}},
		{"star text tag", &struct {
			X textFlag `flag:"x,y" flag-default:"*"`
		}{X: textFlag{"horsefeathers"}}, textFlag{"horsefeathers"}},

		{"uint tag", &struct {
			X uint `flag:"x,default=99,y"`
		}{}, uint(99)},
		{"uint tag tag", &struct {
			X uint `flag:"x,y" flag-default:"99"`
		}{}, uint(99)},

		{"uint64", &struct {
			X uint64 `flag:"x,default=21,y"`
		}{}, uint64(21)},
		{"uint64 tag", &struct {
			X uint64 `flag:"x,y" flag-default:"21"`
		}{}, uint64(21)},

		{"flagValue", &struct {
			X flagValue `flag:"x,default=rumplestiltskin,y"`
		}{}, flagValue{"rumplestiltskin"}},
		{"flagValue tag", &struct {
			X flagValue `flag:"x,y" flag-default:"rumplestiltskin"`
		}{}, flagValue{"rumplestiltskin"}},

		{"self string", &struct {
			X string `flag:"x,default=*,y"`
		}{X: "foo"}, "foo"},
		{"self string tag", &struct {
			X string `flag:"x,y" flag-default:"*"`
		}{X: "foo"}, "foo"},

		{"self int", &struct {
			X int `flag:"x,default=*,y"`
		}{X: 25}, int(25)},
		{"self int tag", &struct {
			X int `flag:"x,y" flag-default:"*"`
		}{X: 25}, int(25)},

		{"star string", &struct {
			X string `flag:"x,default=**,y"` // doubled, use a literal "*"
		}{X: "foo"}, "*"},
		{"star string tag", &struct {
			X string `flag:"x,y" flag-default:"**"` // doubled, use a literal "*"
		}{X: "foo"}, "*"},

		{"self flagValue", &struct {
			X flagValue `flag:"x,default=*,y"`
		}{X: flagValue{"qqq"}}, flagValue{"qqq"}},
		{"self flagValue tag", &struct {
			X flagValue `flag:"x,y" flag-default:"*"`
		}{X: flagValue{"qqq"}}, flagValue{"qqq"}},

		{"star flagValue", &struct {
			X flagValue `flag:"x,default=**,y"`
		}{X: flagValue{"qqq"}}, flagValue{"*"}},
		{"star flagValue tag", &struct {
			X flagValue `flag:"x,y" flag-default:"**"`
		}{X: flagValue{"qqq"}}, flagValue{"*"}},
	}
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			mustBind(t, tc.input)
			got := fieldValue(tc.input, name)
			if got != tc.want {
				t.Fatalf("Field %q: got %v, want %v", name, got, tc.want)
			}
		})
	}
}

func TestBindDefaultConflict(t *testing.T) {
	var flags struct {
		Conf string `flag:"conf,default=x,Conflict" flag-default:"x"`
	}
	f, err := flax.Check(&flags)
	if err == nil {
		t.Errorf("Check %T: got %v, want error", flags, f)
	} else {
		t.Logf("Got expected error: %v", err)
	}
}

func TestEnvDefaultText(t *testing.T) {
	// Set up an environment default for one of the probe cases.
	// This must happen before the flags are parsed.
	t.Setenv("P", "12345")

	var flags struct {
		A int `flag:"apple,default=$A,The first flag"`
		P int `flag:"pear,The second flag" flag-default:"$P"`
		C int `flag:"cherry,default=222,The third flag"`
		U int `flag:"plum,The fourth flag"`
	}
	fs := mustBind(t, &flags)

	// Render the help text into a buffer and make sure we got the expected
	// label strings.
	var help bytes.Buffer
	fs.SetOutput(&help)
	fs.PrintDefaults()
	lines := strings.Split(help.String(), "\n")

	checks := []struct {
		probe  string // select a line
		needle string // search within the line
		want   bool   // needle should be present?
	}{
		{"first flag", "[env: A]", true},
		{"first flag", "default", false},
		{"second flag", "[env: P]", true},
		{"second flag", "(default 12345)", true},
		{"third flag", "[env:", false},
		{"third flag", "(default 222)", true},
		{"fourth flag", "[env:", false},
		{"fourth flag", "default", false},
	}
	for _, c := range checks {
		for _, line := range lines {
			if strings.Contains(line, c.probe) {
				if strings.Contains(line, c.needle) != c.want {
					t.Errorf("Line matching %q:\ngot:  %s\nwant: %s", c.probe, strings.TrimSpace(line), c.needle)
				}
				break
			}
		}
	}
}

func TestField_Env(t *testing.T) {
	fs, err := flax.Check(&struct {
		A int `flag:"a,default=$FOO,First flag"`
		B int `flag:"b,Second flag"`
	}{})
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if f := fs.Flag("a"); f == nil {
		t.Fatal("Flag a not found")
	} else if got, want := f.Env(), "FOO"; got != want {
		t.Errorf("Flag a env: got %q, want %q", got, want)
	}
	if f := fs.Flag("b"); f == nil {
		t.Fatal("Flag b not found")
	} else if got, want := f.Env(), ""; got != want {
		t.Errorf("Flag b env: got %q, want %q", got, want)
	}
}

func TestPreferValueToText(t *testing.T) {
	var tf struct {
		F bothValue `flag:"both,FlagAndText"`
	}
	fs := flag.NewFlagSet("test", flag.PanicOnError)
	flax.MustBind(fs, &tf)

	const text = "xyzzy"
	if err := fs.Parse([]string{"--both", text}); err != nil {
		t.Fatalf("Flag parse: unexpected error: %v", err)
	}

	// The flag supports both the Value interface and text marshaling.
	// Verify that we preferred Value.
	if got := tf.F.flagValue.value; got != text {
		t.Errorf("Flag value: got %q, want %q", got, text)
	}
	if got := tf.F.textFlag.value; got != "" {
		t.Errorf("Text flag: got %q, want empty", got)
	}
}
