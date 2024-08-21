package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/spf13/cast"
	"strings"
	"time"
)

const (
	fieldRequestID  = "proxy-request-id"
	headerRequestID = "requestId"
)

func createRID(req *DirectorRequest) string {
	stringer := strings.Builder{}
	stringer.WriteString(req.RequestURI)
	if ip := req.Header.Get("X-Forwarded-For"); ip != "" {
		fmt.Println("X-Forwarded-For:", ip)
		stringer.WriteString(ip)
	} else if ip = req.Header.Get("X-Real-Ip"); ip != "" {
		fmt.Println("X-Real-Ip:", ip)
		stringer.WriteString(ip)
	}
	{
		ts := time.Now().UnixNano()
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, ts)
		stringer.Write(buf.Bytes())
	}
	hash := md5.New()
	// 更新哈希对象的值
	hash.Write([]byte(stringer.String()))
	// 获取哈希值
	hashValue := hash.Sum(nil)
	// 将哈希值转换为十六进制字符串
	return hex.EncodeToString(hashValue[:])
}

func RequestID(request *DirectorRequest) {
	ctx := request.Context()
	rid := ""
	if rid = request.Header.Get(headerRequestID); rid == "" {
		rid = createRID(request)
	}
	ctx = context.WithValue(ctx, fieldRequestID, rid)
	request.NewRequest(ctx)
	request.SetHeader(headerRequestID, rid)
}

func getRIDFromCtx(ctx context.Context) string {
	return cast.ToString(ctx.Value(fieldRequestID))
}
