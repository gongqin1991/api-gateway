package main

import (
	"github.com/spf13/viper"
	"golang.org/x/time/rate"
	"net/http"
	"time"
)

type RateLimiter struct {
	enable  bool
	count   int
	limiter *rate.Limiter
}

var rLimiter = &RateLimiter{}

func (l *RateLimiter) Setup() {
	enable := viper.GetInt("proxy.rate.limit.enable")
	limitCount := viper.GetInt("proxy.rate.limit.count-second")
	l.enable = enable == ON && limitCount > 0
	l.count = limitCount
	l.limiter = rate.NewLimiter(rate.Every(time.Second), limitCount)
}

func (l *RateLimiter) RateLimit(request *DirectorRequest) {
	if !l.enable || l.limiter.Allow() {
		//不做检查或者验证
		return
	}
	log := logger.WithContext(request.Context())
	request.Abort()
	//验证失败方法处理
	//集群模式下，此节点达到访问次数上线，转移到其他节点
	if nodeAddr := ""; cluster && request.FetchReplica(log, &nodeAddr) {
		request.SetHost(nodeAddr)
		if gateway := viper.GetString("cluster.local.ip"); gateway != "" {
			request.AddHeader(HandleApiGateway, gateway)
		}
		return
	}
	request.Method = http.MethodGet
	request.SetPath("/rate/limit/err")
}
