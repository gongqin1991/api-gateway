package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
)

const (
	HandleApiGateway = "handle-api-gateway"
)

const (
	Abort = iota
	Pass
)

type DirectorRequest struct {
	*http.Request
	state int
}

func ProxyDirector(proxy *httputil.ReverseProxy, middleware *DirectorMiddleware) func(*http.Request) {
	old := proxy.Director
	if middleware == nil {
		middleware = new(DirectorMiddleware)
	}
	middleware.Use(DirectRequest)
	return func(request *http.Request) {
		middlewares := middleware.Middleware
		old(request)
		req := &DirectorRequest{
			Request: request,
			state:   Pass,
		}
		middlewares(req)
	}
}

func (p *DirectorRequest) SetHost(host string) {
	if host == "" {
		return
	}
	p.Request.URL.Host = host
	p.Request.Host = host
}

func (p *DirectorRequest) SetPath(path string) {
	if path == "" {
		return
	}
	p.Request.URL.Path = path
	p.Request.URL.RawPath = path
}

func (p *DirectorRequest) DirectPath(prefix string) string {
	return p.Path()[len(prefix):]
}

func (p *DirectorRequest) MatchPath(path string) bool {
	if path == "*" {
		return true
	}
	return path != "" && strings.Index(p.Path(), path) == 0
}

func (p *DirectorRequest) AddHeader(key, value string) {
	p.Header.Add(key, value)
}

func (p *DirectorRequest) SetHeader(key, value string) {
	p.Header.Set(key, value)
}

func (p *DirectorRequest) NewRequest(ctx context.Context) {
	newReq := p.WithContext(ctx)
	*(p.Request) = *newReq
}

func (p *DirectorRequest) FetchReplica(log *Logger, addr *string) bool {
	getServer := false
	var (
		skip        = make([]string, 0)
		skipReplica = func(item replicaNode) bool {
			if item.RPort == "" {
				return true
			}
			for _, ip := range skip {
				if ip == item.IP {
					return true
				}
			}
			return false
		}
	)
	//该请求是从其他节点转发过来的
	//我们再找合适的网关要排除他
	if ips := p.Header.Values(HandleApiGateway); len(ips) > 0 {
		skip = append(skip, ips...)
		log.Infof("handle gateway:%v", ips)
	}
retry:
	replicaSet := make([]replicaNode, 0)
	valid := replSet.ValidReplicaSet()
	for i := range valid {
		replica := valid[i]
		if !skipReplica(replica) && replica.FullReject != "1" {
			replicaSet = append(replicaSet, replica)
		}
	}
	//最近刷新的节点是我们认为最合适的
	var best *replicaNode
	for i := range replicaSet {
		replica := replicaSet[i]
		if best == nil {
			best = &replica
			continue
		}
		if replica.RefreshAt > best.RefreshAt {
			best = &replica
		}
	}
	if best != nil {
		log.Infof("best replica gateway:%v", *best)
		if !best.HasService(p.Path()) {
			//没有匹配该请求路径的微服务
			//重试，并且下次api网关查找跳过此IP
			skip = append(skip, best.IP)
			goto retry
		}
		//查找到合适的api网关并拼接api网关地址
		getServer = true
		*addr = fmt.Sprintf("http://%s%s", best.IP, best.PPort) //url address
	}
	return getServer
}

func (p *DirectorRequest) Path() string {
	return p.RequestURI
}

func (p *DirectorRequest) Abort() {
	p.state = Abort
}
