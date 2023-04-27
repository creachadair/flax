// Copyright (C) 2023 Michael J. Fromberger. All Rights Reserved.

// Package flax implements a helper for attaching flags to the fields of
// struct values.
package flax

import (
	"encoding"
	"errors"
	"flag"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// MustBind binds the flaggable fields of v to fs, or panics. The concrete type
// of v must be a pointer to a value of struct type.  This function is intended
// for use in program initialization; callers who need to check errors should
// call Bind or Check.
func MustBind(fs *flag.FlagSet, v any) {
	fi, err := Check(v)
	if err != nil {
		panic("check flags: " + err.Error())
	} else if err := fi.Bind(fs); err != nil {
		panic("bind flags: " + err.Error())
	}
}

// MustCheck constructs a Fields value from the flaggable fields of v, or
// panics.  This function is intended for use in program initialization;
// callers who need to check errors should call Check directly.
func MustCheck(v any) Fields {
	fields, err := Check(v)
	if err != nil {
		panic("check flags: " + err.Error())
	}
	return fields
}

// Bind is shorthand for calling Check and then immediately binding the flags
// to the specified flag set on success.
func Bind(fs *flag.FlagSet, v any) error {
	fields, err := Check(v)
	if err != nil {
		return err
	}
	return fields.Bind(fs)
}

// Check constructs information about the flaggable fields of v, whose concrete
// type must be a pointer to a value of struct type.
//
// Check reports an error if v has the wrong type, or if it does not define any
// flaggable fields.  An exported field of v is flaggable if it is of a
// compatible type and has a struct tag with the following form:
//
//	flag:"name[,options],Usage string"
//
// The name and usage string are required. Unexported fields and fields without
// a flag tag are ignored. The supported options include:
//
//	default=V   -- specify a default value for the field
//
// Compatible types include bool, float64, int, int64, string, time.Duration,
// uint, and uint64, as well as any type implementing the flag.Value interface
// or the encoding.TextMarshaler and encoding.TextUnmarshaler interfaces.
func Check(v any) (Fields, error) {
	if v == nil {
		return nil, errors.New("value is nil")
	}
	rp := reflect.ValueOf(v)
	if rp.Kind() != reflect.Ptr {
		return nil, errors.New("value is not a pointer")
	}
	rv := rp.Elem()
	if rv.Kind() != reflect.Struct {
		return nil, errors.New("value is not a struct")
	}
	rt := rv.Type()

	var fields Fields
	for i := 0; i < rt.NumField(); i++ {
		fi, err := parseFieldValue(rt.Field(i), rv.Field(i))
		if err == errSkipField {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("field %q: %w", rt.Field(i).Name, err)
		}
		fields = append(fields, fi)
	}
	if len(fields) == 0 {
		return nil, errors.New("no flaggable fields")
	}
	return fields, nil
}

// Fields records information about the flaggable fields of a struct type.  Use
// the Bind method to attach flags to the corresponding fields.
type Fields []*Field

// Bind attaches the flags defined by f to the given flag set.
func (f Fields) Bind(fs *flag.FlagSet) error {
	for _, fi := range f {
		if err := fi.Bind(fs); err != nil {
			return err
		}
	}
	return nil
}

// Flag returns the first entry in f whose flag name matches s, or nil if no
// such entry exists.
func (f Fields) Flag(s string) *Field {
	for _, fi := range f {
		if fi.Name == s {
			return fi
		}
	}
	return nil
}

// A Field records information about a single flaggable field in a struct type.
// The caller can modify the Name and Usage fields if desired before binding
// the flag to a FlagSet.
type Field struct {
	Name, Usage string // name and usage text (required)

	dvalue string        // string representation of default (or "")
	target reflect.Value // target field value
}

// Bind binds the flag in the given flag set.
func (fi *Field) Bind(fs *flag.FlagSet) error {
	vptr := fi.target.Addr().Interface()

	// If the field already implements flag.Value, register that directly.
	if flagVal, ok := vptr.(flag.Value); ok {
		if fi.dvalue != "" {
			err := flagVal.Set(fi.dvalue)
			if err != nil {
				return fmt.Errorf("set %q default: %w", fi.Name, err)
			}
		}
		fs.Var(flagVal, fi.Name, fi.Usage)
		return nil
	}

	// Otherwise, check for built-in types.
	switch t := vptr.(type) {
	case *bool:
		d, err := parseDefault(fi.Name, fi.dvalue, strconv.ParseBool)
		if err != nil {
			return err
		}
		fs.BoolVar(t, fi.Name, d, fi.Usage)

	case *float64:
		d, err := parseDefault(fi.Name, fi.dvalue, func(s string) (float64, error) {
			return strconv.ParseFloat(s, 64)
		})
		if err != nil {
			return err
		}
		fs.Float64Var(t, fi.Name, d, fi.Usage)

	case *int:
		d, err := parseDefault(fi.Name, fi.dvalue, strconv.Atoi)
		if err != nil {
			return err
		}
		fs.IntVar(t, fi.Name, d, fi.Usage)

	case *int64:
		d, err := parseDefault(fi.Name, fi.dvalue, func(s string) (int64, error) {
			return strconv.ParseInt(s, 10, 64)
		})
		if err != nil {
			return err
		}
		fs.Int64Var(t, fi.Name, d, fi.Usage)

	case *string:
		fs.StringVar(t, fi.Name, fi.dvalue, fi.Usage)

	case encoding.TextUnmarshaler:
		_, err := parseDefault(fi.Name, fi.dvalue, func(s string) (any, error) {
			return nil, t.UnmarshalText([]byte(s))
		})
		if err != nil {
			return err
		}
		// The base value was checked previously for satisfaction.
		m := fi.target.Interface().(encoding.TextMarshaler)
		fs.TextVar(t, fi.Name, m, fi.Usage)

	case *time.Duration:
		d, err := parseDefault(fi.Name, fi.dvalue, time.ParseDuration)
		if err != nil {
			return err
		}
		fs.DurationVar(vptr.(*time.Duration), fi.Name, d, fi.Usage)

	case *uint:
		d, err := parseDefault(fi.Name, fi.dvalue, func(s string) (uint, error) {
			u, err := strconv.ParseUint(s, 10, 64)
			return uint(u), err
		})
		if err != nil {
			return err
		}
		fs.UintVar(t, fi.Name, d, fi.Usage)

	case *uint64:
		d, err := parseDefault(fi.Name, fi.dvalue, func(s string) (uint64, error) {
			return strconv.ParseUint(s, 10, 64)
		})
		if err != nil {
			return err
		}
		fs.Uint64Var(t, fi.Name, d, fi.Usage)

	default:
		return fmt.Errorf("cannot flag type %T", t)
	}
	return nil
}

var errSkipField = errors.New("skip this field")

func parseFieldValue(ft reflect.StructField, fv reflect.Value) (*Field, error) {
	if !ft.IsExported() {
		return nil, errSkipField // unexported fields are not considered
	}
	tag, ok := ft.Tag.Lookup("flag")
	if !ok {
		return nil, errSkipField // un-flagged fields are not considered
	}
	parts := strings.Split(tag, ",")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid flag tag format: %q", tag)
	} else if parts[0] == "" {
		return nil, fmt.Errorf("empty flag name: %q", tag)
	}

	info := &Field{
		Name:   parts[0],
		Usage:  parts[len(parts)-1],
		target: fv,
	}

	// Parse options.
	for _, p := range parts[1 : len(parts)-1] {
		opt, val, _ := strings.Cut(p, "=")
		switch opt {
		case "default":
			info.dvalue = val
		default:
			return nil, fmt.Errorf("unknown option %q", p)
		}
	}

	// Check for compatible type.
	switch t := fv.Interface().(type) {
	case bool, float64, int, int64, string, time.Duration, uint, uint64:
		// OK
	case encoding.TextMarshaler:
		// OK if its pointer also implements TextUnmarshaler.
		tp := fv.Addr().Interface()
		if _, ok := tp.(encoding.TextUnmarshaler); !ok {
			return nil, fmt.Errorf("type %T is not text compatible", tp)
		}
	default:
		// OK if its pointer implements flag.Value.
		tp := fv.Addr().Interface()
		if _, ok := tp.(flag.Value); !ok {
			return nil, fmt.Errorf("type %T is not flag compatible", t)
		}
	}

	return info, nil
}

func parseDefault[T any](name, s string, parse func(string) (T, error)) (T, error) {
	var zero T
	if s == "" {
		return zero, nil
	}
	v, err := parse(s)
	if err != nil {
		return zero, fmt.Errorf("invalid default for %q: %w", name, err)
	}
	return v, nil
}
