package types

import (
	"fmt"
	"sync"
	"time"
)

type URLGenerator func() string

var (
	urlFn     URLGenerator
	rotateURL bool

	m sync.RWMutex
)

// Config should be called once at the start of a program to configure
// URLGenerator and secret rotation policy
func Config(fn URLGenerator, autoRotate bool) {
	m.Lock()
	urlFn = fn
	rotateURL = autoRotate
	m.Unlock()
}

// ref: https://play.golang.org/p/vMssfd6ZY8e

func RotateDaily() string {
	return time.Now().UTC().Format("2006-01-02")
}

func RotateMonthly() string {
	return time.Now().UTC().Format("2006-01")
}

func RotateQuarterly() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d-Q%d", t.Year(), (t.Month()-1)/3+1)
}
