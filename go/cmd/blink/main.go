// Command blink turns a BlinkStick's LEDs on (white) or off.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/bawdo/blinkstick/go/blinkstick"
)

const usage = "usage: blink on|off"

type stick interface {
	SetAll(blinkstick.RGB) error
	Close() error
}

func main() {
	open := func() (stick, error) { return blinkstick.OpenFirst() }
	os.Exit(run(os.Args[1:], open, os.Stderr))
}

func run(args []string, open func() (stick, error), stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, usage)
		return 2
	}
	var colour blinkstick.RGB
	switch args[0] {
	case "on":
		colour = blinkstick.White
	case "off":
		colour = blinkstick.Off
	default:
		fmt.Fprintln(stderr, usage)
		return 2
	}

	s, err := open()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer s.Close()

	if err := s.SetAll(colour); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
