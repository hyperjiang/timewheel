# Time Wheel

[![GoDoc](https://pkg.go.dev/badge/github.com/hyperjiang/timewheel)](https://pkg.go.dev/github.com/hyperjiang/timewheel)
[![CI](https://github.com/hyperjiang/timewheel/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/hyperjiang/timewheel/actions/workflows/ci.yml)
[![](https://goreportcard.com/badge/github.com/hyperjiang/timewheel)](https://goreportcard.com/report/github.com/hyperjiang/timewheel)
[![codecov](https://codecov.io/gh/hyperjiang/timewheel/branch/main/graph/badge.svg)](https://codecov.io/gh/hyperjiang/timewheel)
[![Release](https://img.shields.io/github/release/hyperjiang/timewheel.svg)](https://github.com/hyperjiang/timewheel/releases)

Simple time wheel in golang.

## Prerequisite

go version >= 1.18


## Usage

```
import "github.com/hyperjiang/timewheel"

type MyHandler struct {
}

func (h *MyHandler) Handle(param any) {
	// do your own business
}

tw := timewheel.New(
    timewheel.WithHandler(new(MyHandler)),
    timewheel.WithLogger(timewheel.Printf),
    timewheel.WithSlotNum(10),
    timewheel.WithTickDuration(time.Millisecond*100),
)

tw.Start()

tw.AddTask(time.Second, "task_unique_key_1", "1s")
tw.AddTask(time.Second*10, "task_unique_key_2", "10s")
```
