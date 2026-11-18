package main

import (
	"io"
	"testing"
)

func TestParseArgsRootBeforeOrAfterFlags(t *testing.T) {
	for _, args := range [][]string{
		{`E:\`, "--port", "9000", "--no-browser"},
		{"--port", "9000", `E:\`, "--no-browser"},
		{"--no-browser", "--port=9000", `E:\`},
	} {
		o, err := parseArgs(args, io.Discard)
		if err != nil || o.root != `E:\` || o.port != 9000 || o.ui != "none" {
			t.Errorf("%v: %+v %v", args, o, err)
		}
	}
}

func TestParseArgsRejectsBadInput(t *testing.T) {
	for _, args := range [][]string{{"--ui", "bogus"}, {"a", "b"}, {"--nope"}} {
		if _, err := parseArgs(args, io.Discard); err == nil {
			t.Errorf("%v: expected an error", args)
		}
	}
}
