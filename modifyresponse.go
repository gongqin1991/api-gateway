package main

import (
	"net/http"
	"net/http/httputil"
)

func ModifyResponse(p *httputil.ReverseProxy) func(*http.Response) error {
	return func(resp *http.Response) error {
		log := logger.WithContext(resp.Request.Context())
		log.Infof("old header:%v", resp.Header)
		modifyResponse(resp, log)
		return nil
	}
}

func modifyResponse(resp *http.Response, log *Logger) {
	//跨域
	cors.Check(resp)
}
