package types

import (
	"sync"
)

type URLGenerator func() string

var (
	urlFn URLGenerator

	m sync.RWMutex
)

// Config should be called once at the start of a program
// to configure URLGenerator
func Config(fn URLGenerator) {
	m.Lock()
	urlFn = fn
	m.Unlock()
}
