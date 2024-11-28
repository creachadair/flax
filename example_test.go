// Copyright (C) 2023 Michael J. Fromberger. All Rights Reserved.

package flax_test

import (
	"flag"
	"fmt"
	"log"

	"github.com/creachadair/flax"
)

var flags struct {
	Debug   bool    `flag:"debug,default=true,Enable debugging output"`
	Text    string  `flag:"text,default=OK,Text to display"`
	Rate    float64 `flag:"rate,default=0.1,Rate of increase"`
	Ignored int
}

func init() { flax.MustBind(flag.CommandLine, &flags) }

func Example() {
	flag.Parse()

	if flags.Debug {
		fmt.Println(flags.Text, flags.Rate)
	}
	fmt.Println(flags.Ignored)
	// Output:
	// OK 0.1
	// 0
}

func ExampleCheck() {
	fields, err := flax.Check(&struct {
		A int     `flag:"apple,default=*,Apples"`       // use field value as default
		P int     `flag:"pear,Pears" flag-default:"12"` // set explicit default, tag
		U string  `flag:"plum,default=$X,Plums"`        // read default from environment
		C float64 `flag:"cherry,default=0.1,Cherries"`  // set explicit default, inline
	}{A: 3})
	if err != nil {
		log.Fatal(err)
	}

	fs := flag.NewFlagSet("test", flag.PanicOnError)
	fields.Bind(fs)

	fmt.Println("-- fields --")
	for _, f := range fields { // Fields in declaration order
		fmt.Println(f.Name, f.Usage)
	}

	fmt.Println("\n-- flags --")
	fs.VisitAll(func(f *flag.Flag) { // Flags in name order
		fmt.Printf("%s %q %v\n", f.Name, f.DefValue, f.Usage)
	})
	// Output:
	// -- fields --
	// apple Apples
	// pear Pears
	// plum Plums
	// cherry Cherries
	//
	// -- flags --
	// apple "3" Apples
	// cherry "0.1" Cherries
	// pear "12" Pears
	// plum "" Plums [env: X]
}
