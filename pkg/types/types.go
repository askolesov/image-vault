package types

import "fmt"

type LogFn func(string, ...any)

var PrintfLogFn LogFn = func(s string, a ...any) {
	fmt.Printf(s, a...)
}

var NilLogFn LogFn = func(string, ...any) {}

type ProgressCb func(int64)

var NilProgressCb ProgressCb = func(int64) {}
