package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/alecthomas/kong"

	"github.com/kuwa72/matcher"
)

var (
	cli struct {
		QUERY string `arg:"" required:"" help:"QUERY to parse."`
	}
)

func main() {
	ctx := kong.Parse(&cli)
	m, err := matcher.NewMatcher(cli.QUERY)
	ctx.FatalIfErrorf(err)

	j, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c := matcher.Context(make(map[string]interface{}))
	json.Unmarshal([]byte(j), &c)

	b, err := m.Test(&c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Printf("QUERY: %#v\n", cli.QUERY)
	fmt.Printf("JSON structure: %#v\n", c)
	switch {
	case b:
		fmt.Println("matched")
	default:
		fmt.Println("Unmatched")
	}
}
