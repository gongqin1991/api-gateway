package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"golang.org/x/exp/maps"
	"io"
	"strings"
)

const (
	tokenBufN = 1024 * 512
	ioBufN    = 1024
	magic     = "magic:v1-20231024"
)

const (
	handshakeStateEOF uint8 = iota
	handshakeStateMagic
)

const (
	responseStatus = "status"
	responseBody   = "body"
)

const (
	AckOk = "ok"
)

type handshakeBody map[string]string

type handshake struct {
	Magic string
	Body  handshakeBody

	buffer []byte

	readToken []byte
	readState uint8
	tokenOff  int
	tokenCap  int
}

var (
	tokenTooLarge = errors.New("token too large")
	invalidMagic  = errors.New("invalid magic")
	errKeyValue   = errors.New("not key value")
)

func (h *handshake) reset() {
	h.buffer = make([]byte, ioBufN)
	h.readToken = make([]byte, tokenBufN)
	h.tokenCap = tokenBufN
	h.Body = make(handshakeBody)
	h.readState = handshakeStateEOF
}

func (h *handshake) Read(reader io.Reader) (bool, error) {
	buf := h.buffer
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		err = errors.Wrap(err, "read error")
	} else if n <= 0 {
		err = errors.New("connect reset")
	}
	if err != nil {
		return false, err
	}
	h.readBuffer(n)
	return h.Magic != "" && h.readState == handshakeStateEOF, err
}

func (h *handshake) WriteTo(writer io.Writer) error {
	mgc := h.Magic
	if mgc == "" {
		mgc = magic
	}
	buf := bytes.NewBuffer([]byte(mgc))
	for k, v := range h.Body {
		buf.WriteString("\r\t")
		buf.WriteString(h.writeField(k, v))
	}
	buf.WriteString("\r\n")
	_, err := writer.Write(buf.Bytes())
	return err
}

func (h *handshake) readBuffer(N int) {
	read := 0
	for read < N {
		custom, token, end := h.nextToken(read, N)
		read += custom

		if token == 0 {
			continue
		}

		tokenStr := string(h.readToken[:token])
		state := h.readState
		if end {
			h.readState = handshakeStateEOF
		}
		if state == handshakeStateEOF {
			if tokenStr != magic {
				panic(invalidMagic)
			}
			h.Magic = tokenStr
			if !end {
				h.readState = handshakeStateMagic
			}
			maps.Clear(h.Body)
		} else {
			k, v := h.readField(tokenStr)
			h.Body[k] = v
		}

	}
}

func (h *handshake) readField(token string) (k, v string) {
	kv := strings.Split(token, ":")
	if len(kv) != 2 {
		panic(errKeyValue)
	}
	kk, err := base64.StdEncoding.DecodeString(kv[0])
	if err != nil {
		panic(errors.Wrap(err, "base64 decode"))
	}
	vv, err := base64.StdEncoding.DecodeString(kv[1])
	if err != nil {
		panic(errors.Wrap(err, "base64 decode"))
	}
	k = string(kk)
	v = string(vv)
	return
}

func (h *handshake) writeField(key, value string) string {
	kk := base64.StdEncoding.EncodeToString([]byte(key))
	vv := base64.StdEncoding.EncodeToString([]byte(value))
	return fmt.Sprintf("%s:%s", kk, vv)
}

func (h *handshake) nextToken(offset, N int) (read, token int, end bool) {
	buf := h.buffer
	pivot := h.tokenOff
	next := func(index int) bool {
		return index < N
	}
	for i := offset; i < N; i++ {
		nex := i + 1
		if buf[i] == '\r' && next(nex) && (buf[nex] == '\t' || buf[nex] == '\n') {
			token = read + pivot
			end = buf[nex] == '\n'
			read += 2
			h.tokenOff = 0
			return
		}
		h.readToken[pivot+read] = buf[i]
		if pivot+read+1 >= h.tokenCap {
			panic(tokenTooLarge)
		}
		read += 1
	}
	h.tokenOff += read
	return
}

func (b handshakeBody) DecodeStruct(output interface{}) error {
	return mapstructure.Decode(b, output)
}

func (b handshakeBody) WriteStatus(status string) {
	b[responseStatus] = status
}

func (b handshakeBody) Write(data []byte) {
	b[responseBody] = string(data)
}

func (b handshakeBody) Copy(src map[string]string) {
	maps.Clear(b)
	for k, v := range src {
		b[k] = v
	}
}

func (b handshakeBody) hasStatus() bool {
	_, ok := b[responseStatus]
	return ok
}
