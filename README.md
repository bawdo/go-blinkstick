# go-blinkstick

[![Go Reference](https://pkg.go.dev/badge/github.com/bawdo/go-blinkstick.svg)](https://pkg.go.dev/github.com/bawdo/go-blinkstick)
[![Go Report Card](https://goreportcard.com/badge/github.com/bawdo/go-blinkstick)](https://goreportcard.com/report/github.com/bawdo/go-blinkstick)

A Go package for driving BlinkStick USB LED devices.

This package is unofficial. It is not made by, or affiliated with, Agile Innovative, the makers
of BlinkStick.

## Supported devices

| Device | macOS | Linux |
|---|---|---|
| BlinkStick Nano | yes | not yet |
| BlinkStick Square | yes | not yet |

See [CAPABILITIES.md](CAPABILITIES.md) for every feature these devices have and what this
package supports so far.

## Requirements

- Go 1.26 or later.
- cgo. On macOS, install the Xcode command line tools with `xcode-select --install`. Builds
  with `CGO_ENABLED=0` will not work.
- Nothing else. hidapi is bundled.

## Install

```sh
go get github.com/bawdo/go-blinkstick
```

## Quick start

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/bawdo/go-blinkstick"
)

func main() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	red, _ := blinkstick.ParseRGB("#ff0000")
	if err := d.SetAll(red); err != nil {
		log.Fatal(err)
	}

	// Pulse green twice, one second each.
	err = d.Pulse(context.Background(), blinkstick.RGB{G: 255}, time.Second, 2)
	if err != nil {
		log.Fatal(err)
	}
}
```

## More than one stick

`List` returns every attached stick. Open a particular one by its serial:

```go
infos, err := blinkstick.List()
if err != nil {
	log.Fatal(err)
}
for _, info := range infos {
	fmt.Println(info.Serial, info.Model.Name)
}

d, err := blinkstick.OpenSerial("BS072777-3.0")
```

Each open `Device` is independent, and its methods are safe to call from several goroutines.
macOS lets only one process open a stick at a time.

## Good to know

- `Blink`, `Pulse` and `Morph` block until they finish. Cancel them with the context.
- `SetBrightnessLimit` caps how bright the LEDs get. A Square at full white draws about 500 mA.
- `SetInfoBlock` writes to EEPROM on the device, which wears out with heavy use. Do not call
  it in a loop.

## Development

```sh
make help           # list targets
make test           # vet and unit tests, no hardware needed
make build          # compile
make test-hardware  # tests against a real Nano and Square
```

## Versioning

Releases follow [semantic versioning](https://semver.org/) and are tagged `vX.Y.Z`. Until
`v1.0.0` the API may change between minor releases. See [CHANGELOG.md](CHANGELOG.md).

## Licence

MIT. See [LICENSE](LICENSE).
