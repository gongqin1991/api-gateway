package main

import (
	"awesomeProject/pkg/cache"
	"context"
	"github.com/spf13/viper"
	"time"
)

type replicaSetPin struct {
	servers  []string
	conns    []*connClient
	send     *handshake
	recv     *handshake
	interval int64
	Message  func() map[string]string
	Result   func(handshakeBody)
}

func (pin *replicaSetPin) Setup() {
	addrs := viper.GetStringSlice("cluster.replicaSet.address")
	interval := viper.GetInt64("cluster.replicaSet.handshake.interval")
	pin.servers = addrs
	pin.interval = interval
}

func (pin *replicaSetPin) ready(ctx context.Context) {
	N := len(pin.servers)
	conns := make([]*connClient, N)
	for i, addr := range pin.servers {
		conn := &connClient{ctx: ctx}
		conn.Dial(addr)
		conns[i] = conn
	}
	send := &handshake{Body: make(handshakeBody)}
	recv := new(handshake)
	recv.reset()
	pin.conns = conns
	pin.send = send
	pin.recv = recv
}

func (pin *replicaSetPin) Go(ctx context.Context) {
	tick := time.NewTicker(cache.ToDuration(pin.interval, time.Second))
	ctx, _ = context.WithCancel(ctx)
	pin.ready(ctx)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case <-tick.C:
			for i := range pin.Conn() {
				if ctx.Err() != nil {
					break
				}
				if err := pin.pin(i); err != nil {
					logger.Fatalf("address:%s,error:%v", pin.servers[i], err)
				} else if handler := pin.Result; handler != nil {
					handler(pin.recv.Body)
				}
			}
		}
	}
}

func (pin *replicaSetPin) Conn() []*connClient {
	return pin.conns[:]
}

func (pin *replicaSetPin) pin(offset int) error {
	addr := pin.servers[offset]
	conn := pin.conns[offset]
	if err := conn.err; err != nil {
		logger.Fatal(err)
	}
	switch conn.State() {
	case connStateClosed:
		go conn.Dial(addr)
	case connStateNew:
		return errDialConnect
	}
	send := pin.send
	send.Body.Copy(pin.Message())
	return conn.Send(send, pin.recv)
}
