# go-schedule-lite

Human-friendly job scheduling for Go.

Inspired by Python's `schedule` library.

## Install

```
go get github.com/njchilds90/go-schedule-lite
```

## Example

```go
package main

import (
	"fmt"
	"github.com/njchilds90/go-schedule-lite"
)

func main() {
	s := schedule.New()

	s.Every(5).Seconds().Do(func() {
		fmt.Println("Hello AI world")
	})

	s.Run()
}
```

## Features

- Chainable API
- Simple scheduling
- Lightweight
- No cron syntax
- AI-agent friendly loops

## Why This Exists

Go has cron libraries but lacks a clean, readable, human-first scheduler like Python's schedule.

This fills that gap.

## License

MIT
