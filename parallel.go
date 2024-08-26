package main

import (
	"context"
	"fmt"
	"time"
)

type ExecFunc func(<-chan struct{})

type (
	CancelGroup struct {
		cancelers []context.CancelFunc
	}
	ExecGroup struct {
		closed    bool
		closeChan chan struct{}
	}
)

func newCancelGroup() *CancelGroup {
	return &CancelGroup{cancelers: make([]context.CancelFunc, 0)}
}

func newExecGroup() *ExecGroup {
	return &ExecGroup{closeChan: make(chan struct{})}
}

func (g *CancelGroup) Add(fn context.CancelFunc) {
	g.cancelers = append(g.cancelers, fn)
}

func (g *CancelGroup) Cancel() {
	for _, cancel := range g.cancelers {
		cancel()
	}
}

func (g *ExecGroup) Add(fn ExecFunc) {
	if g.closed {
		return
	}
	go fn(g.closeChan)
}

func (g *ExecGroup) Close() {
	g.closed = true
	close(g.closeChan)
}

func Tick(ticker *time.Ticker, done <-chan struct{}, do func()) {
	for {
		select {
		case <-done:
			fmt.Println("done...")
			ticker.Stop()
			return
		case <-ticker.C:
			do()
		}

	}
}
