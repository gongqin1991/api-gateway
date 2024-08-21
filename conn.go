package main

import (
	"context"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"net"
	"strings"
	"sync/atomic"
)

const (
	connStateClosed int32 = iota
	connStateNew
	connStateOpened
)

type HandleFunc func(context.Context, handshakeBody, handshakeBody)

type (
	connServer struct {
		net.Listener
		handler HandleFunc
	}
	connClient struct {
		net.Conn
		ctx   context.Context
		err   error
		state int32
	}
)

var (
	errDialConnect   = errors.New("already connect or connecting")
	errDialNoConnect = errors.New("not connect")
)

func (srv *connServer) Listen(port string) error {
	addr, err := net.ResolveTCPAddr("tcp", port)
	if err != nil {
		return errors.Wrap(err, "can't resolve tcp address")
	}
	conn, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "unavailable tcp address")
	}
	srv.Listener = conn
	return nil
}

func (srv *connServer) Recv(ctx context.Context) *connClient {
	client := &connClient{ctx: ctx}
	cconn, err := srv.Accept()
	if ctx.Err() != nil {
		client.err = ctx.Err()
	} else if err != nil {
		client.err = err
	} else {
		client.Conn = cconn
	}
	return client
}

func (cli *connClient) Serve(handler HandleFunc) {
	ctx := cli.ctx
	request := new(handshake)
	response := &handshake{Body: make(handshakeBody)}
	request.reset()
	{
		ipAddr := cli.Conn.RemoteAddr().String()
		nodeIp := strings.Split(ipAddr, ":")[0]
		ctx = context.WithValue(ctx, "ip", nodeIp)
		ctx, _ = context.WithCancel(ctx)
		cli.ctx = ctx
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			cli.serve(request, response, handler)
			if err := cli.err; err != nil {
				//该连接异常关闭
				logger.Fatal(err)
				return
			}
		}
	}
}

func (cli *connClient) serve(request, response *handshake, handler HandleFunc) {
	ctx := cli.ctx
	//read request body
	ok, err := request.Read(cli)
	if err == nil && ctx.Err() != nil {
		err = ctx.Err()
	}
	if err != nil || !ok {
		cli.err = err
		return
	}
	//handle request and ack to client
	maps.Clear(response.Body)
	body := response.Body
	handler(ctx, request.Body, body)
	if !body.hasStatus() {
		body.WriteStatus(AckOk)
	}
	if err = response.WriteTo(cli); err != nil {
		cli.err = err
	}
}

func (srv *connServer) Run(ctx context.Context) {
	ctx, _ = context.WithCancel(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			cconn := srv.Recv(ctx)
			if err := cconn.err; err != nil {
				logger.Fatalf("accept client error,%v", err)
				continue
			}
			//处理tcp握手请求
			go cconn.Serve(srv.handler)
		}
	}
}

func (cli *connClient) Dial(address string) {
	var (
		conn net.Conn
		err  error
	)
	defer func() {
		cli.err = err
		cli.Conn = conn
	}()
	if err = cli.ctx.Err(); err != nil {
		return
	}
	if atomic.LoadInt32(&cli.state) > connStateClosed {
		err = errDialConnect
		return
	}
	atomic.StoreInt32(&cli.state, connStateNew)
	conn, err = net.Dial("tcp", address)
	if err != nil {
		atomic.StoreInt32(&cli.state, connStateClosed)
		return
	}
	atomic.StoreInt32(&cli.state, connStateOpened)
}

func (cli *connClient) State() int32 {
	return atomic.LoadInt32(&cli.state)
}

func (cli *connClient) Close() error {
	err := cli.Conn.Close()
	atomic.StoreInt32(&cli.state, connStateClosed)
	return err
}

func (cli *connClient) Send(request, response *handshake) error {
	state := cli.State()
	if state == connStateClosed {
		return errDialNoConnect
	}
	if state == connStateNew {
		//wait pipe connected
		return nil
	}
	if request.Magic == "" {
		request.Magic = magic
	}
	if request.Body == nil {
		request.Body = make(handshakeBody)
	}
	if err := request.WriteTo(cli); err != nil {
		_ = cli.Close()
		return err
	}
	//read response once
	for {
		ok, err := response.Read(cli)
		if err != nil {
			_ = cli.Close()
			return err
		}
		if ok {
			break
		}
	}
	return nil
}
