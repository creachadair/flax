// Copyright (C) 2023 Michael J. Fromberger. All Rights Reserved.

package flax_test

import (
	"flag"
	"fmt"

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
