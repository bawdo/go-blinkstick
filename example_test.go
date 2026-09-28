package blinkstick_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bawdo/go-blinkstick"
)

func Example() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	if err := d.SetAll(blinkstick.RGB{R: 255}); err != nil {
		log.Fatal(err)
	}
}

func ExampleList() {
	infos, err := blinkstick.List()
	if err != nil {
		log.Fatal(err)
	}
	for _, info := range infos {
		fmt.Println(info.Serial, info.Model.Name, info.Model.LEDs)
	}
}

func ExampleOpenSerial() {
	d, err := blinkstick.OpenSerial("BS072777-3.0")
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()
	fmt.Println(d.Info().Model.Name)
}

func ExampleParseRGB() {
	c, err := blinkstick.ParseRGB("#ff8800")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(c)
	// Output: {255 136 0}
}

func ExampleDevice_SetLED() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	// Top LED red, bottom LED blue on a Nano.
	d.SetLED(0, blinkstick.RGB{R: 255})
	d.SetLED(1, blinkstick.RGB{B: 255})
}

func ExampleDevice_Pulse() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Pulse(ctx, blinkstick.RGB{G: 255}, time.Second, 3); err != nil {
		log.Fatal(err)
	}
}

func ExampleDevice_SetBrightnessLimit() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	d.SetBrightnessLimit(64) // about a quarter of full brightness
	d.SetAll(blinkstick.White)
}

func ExampleRandomVivid() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	d.SetAll(blinkstick.RandomVivid())
}

func ExampleOpenName() {
	d, err := blinkstick.OpenName("desk")
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()
	d.SetAll(blinkstick.RGB{G: 255})
}

func ExampleListNamed() {
	named, err := blinkstick.ListNamed()
	if err != nil {
		log.Fatal(err)
	}
	for _, n := range named {
		fmt.Printf("%s %s %q busy=%v\n", n.Serial, n.Model.Name, n.Name, n.Busy)
	}
}
