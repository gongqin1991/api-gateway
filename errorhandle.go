package main

import (
	"net/http"
	"net/http/httputil"
	"strings"
)

func ErrorHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request, error) {
	return func(resp http.ResponseWriter, request *http.Request, err error) {
		log := logger.WithContext(request.Context())
		log.Errorf("proxy hook result:%v", err)
		errorHandle(resp, &DirectorRequest{Request: request}, log)
	}
}

func errorHandle(resp http.ResponseWriter, request *DirectorRequest, log *Logger) {
	retry := false
	//因为微服务内网ip变动导致的请求失败，可以重试
	if host, slots := request.Host, strings.Split(request.Host, ":"); len(slots) == 2 {
		hostport := slots[1]
		for _, serv := range servicelist.ValidServices() {
			if request.MatchPath(serv.Path) && serv.Port != "" && serv.Port[1:] != hostport {
				log.Info("old host:", host, "new host:", serv.Addr())
				retry = true
				break
			}
		}
	}
	if retry {
		resp.WriteHeader(http.StatusGatewayTimeout)
		_, _ = resp.Write([]byte("connect timeout"))
	} else {
		resp.WriteHeader(http.StatusBadGateway)
		_, _ = resp.Write([]byte("internal error"))
	}
}
