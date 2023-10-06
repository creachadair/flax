// Copyright (C) 2023 Michael J. Fromberger. All Rights Reserved.

package flax_test

import (
	"flag"
	"log"
	"os"
	"reflect"
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
			"uint", "uint64", "flag-value",
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

func mustBind(t *testing.T, input any) {
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

		{"float64", &struct {
			X float64 `flag:"x,default=0.25,y"`
		}{}, 0.25},

		{"int", &struct {
			X int `flag:"x,default=13,y"`
		}{}, 13},

		{"int from env", &struct {
			X int `flag:"x,default=$TEST_INT,y"`
		}{}, 12345},

		{"int64", &struct {
			X int64 `flag:"x,default=7,y"`
		}{}, int64(7)},

		{"string", &struct {
			X string `flag:"x,default=cork bat,y"`
		}{}, "cork bat"},

		{"complex string", &struct {
			X string `flag:"x,default='a, b, c',y"`
		}{}, "a, b, c"},

		{"internal quotes", &struct {
			X string `flag:"x,default='p,'',q',y"`
		}{}, "p,',q"},

		{"env string", &struct {
			X string `flag:"x,default=$TEST_INT,y"`
		}{}, "12345"},

		{"env esc string", &struct {
			X string `flag:"x,default=$$TEST_INT,y"`
		}{}, "$TEST_INT"},

		{"text", &struct {
			X textFlag `flag:"x,default=bleep,y"`
		}{}, textFlag{"bleep"}},

		{"star text", &struct {
			X textFlag `flag:"x,default=*,y"`
		}{X: textFlag{"horsefeathers"}}, textFlag{"horsefeathers"}},

		{"uint", &struct {
			X uint `flag:"x,default=99,y"`
		}{}, uint(99)},

		{"uint64", &struct {
			X uint64 `flag:"x,default=21,y"`
		}{}, uint64(21)},

		{"flagValue", &struct {
			X flagValue `flag:"x,default=rumplestiltskin,y"`
		}{}, flagValue{"rumplestiltskin"}},

		{"self string", &struct {
			X string `flag:"x,default=*,y"`
		}{X: "foo"}, "foo"},

		{"self int", &struct {
			X int `flag:"x,default=*,y"`
		}{X: 25}, int(25)},

		{"star string", &struct {
			X string `flag:"x,default=**,y"` // doubled, use a literal "*"
		}{X: "foo"}, "*"},

		{"self flagValue", &struct {
			X flagValue `flag:"x,default=*,y"`
		}{X: flagValue{"qqq"}}, flagValue{"qqq"}},

		{"star flagValue", &struct {
			X flagValue `flag:"x,default=**,y"`
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
