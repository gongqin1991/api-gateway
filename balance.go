package main

import (
	"github.com/spf13/viper"
	"github.com/zeromicro/go-zero/core/hash"
	"strings"
	"sync"
)

type (
	balance struct {
		servers []BusinessService
		ip      string
	}
	Balancer struct {
		policy     string
		mu         sync.Mutex
		lastOffset int32
	}
)

var balancer = &Balancer{}

func (blr *Balancer) Setup() {
	policy := viper.GetString("proxy.balance.policy")
	if policy == "" {
		policy = "robin" //轮询
	}
	blr.policy = policy
}

func (blr *Balancer) Balance(request *DirectorRequest) {
	services := make([]BusinessService, 0)
	valid := servicelist.ValidServices()
	for i := range valid {
		serv := valid[i]
		if request.MatchPath(serv.Path) && serv.Gateway {
			services = append(services, serv)
		}
	}
	if len(services) == 0 {
		return
	}

	request.Abort()

	if len(services) == 1 {
		request.SetHost(services[0].Addr())
		return
	}

	bla := balance{
		servers: services,
	}
	if blr.policy == "ip-hash" {
		addr := request.RemoteAddr
		bla.ip = strings.Split(addr, ":")[0]
	}
	serv := balancer.Next(bla)
	request.SetHost(serv.Addr())
}

func (blr *Balancer) Next(b balance) BusinessService {
	switch blr.policy {
	case "ip-hash":
		hash0 := hashIp(b.ip)
		N := len(b.servers)
		index := hash0 % N
		return b.servers[index]
	default:
		blr.mu.Lock()
		defer blr.mu.Unlock()
		index := blr.lastOffset
		if int(index) >= len(b.servers) {
			index = 0
		}
		t := b.servers[index]
		index++
		blr.lastOffset = index
		return t
	}
}

func hashIp(ip string) int {
	hash0 := hash.Hash([]byte(ip))
	return int(hash0 >> 2)
}
