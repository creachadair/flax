package flax_test

import (
	"flag"
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

func TestCheckOK(t *testing.T) {
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
	fi, err := flax.Check(&v)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	fs := flag.NewFlagSet("test", flag.PanicOnError)
	if err := fi.Bind(fs); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
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

		{"invalid option", &struct {
			S string `flag:"ok,what,usage"`
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

func mustBind(t *testing.T, input any) {
	t.Helper()

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	if err := flax.Bind(input, fs); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	if err := fs.Parse(nil); err != nil {
		t.Fatalf("Parse flags: %v", err)
	}
}

func fieldValue(v any, name string) any {
	fv := reflect.ValueOf(v).Elem()
	return fv.FieldByName(name).Interface()
}

func TestBindDefaults(t *testing.T) {
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

		{"int64", &struct {
			X int64 `flag:"x,default=7,y"`
		}{}, int64(7)},

		{"string", &struct {
			X string `flag:"x,default=cork bat,y"`
		}{}, "cork bat"},

		{"text", &struct {
			X textFlag `flag:"x,default=bleep,y"`
		}{}, textFlag{"bleep"}},

		{"uint", &struct {
			X uint `flag:"x,default=99,y"`
		}{}, uint(99)},

		{"uint64", &struct {
			X uint64 `flag:"x,default=21,y"`
		}{}, uint64(21)},

		{"flagValue", &struct {
			X flagValue `flag:"x,default=rumplestiltskin,y"`
		}{}, flagValue{"rumplestiltskin"}},
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
