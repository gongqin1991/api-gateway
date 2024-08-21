package main

import (
	"awesomeProject/pkg/cache"
	"awesomeProject/pkg/parallel"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"net"
	"time"
)

type businessService struct {
	Name    string `json:"ms"`
	Path    string `json:"path"`
	Host    string `json:"host,omitempty"`
	Port    string `json:"port,omitempty"`
	Gateway bool   `json:"gateway"`
}

type ServiceSpec struct {
	Services *cache.MemCache[businessService]
}

var (
	servicelist = &ServiceSpec{
		Services: &cache.MemCache[businessService]{LState: cache.LockSync},
	}
)

func (spec *ServiceSpec) Setup() {
	expire := viper.GetInt64("register.service-expire")
	spec.Services.Expire = cache.ToDuration(expire, time.Second)
	spec.Services.Init()
}

func LoadServices() {
	keyname := viper.GetString("service.cache.name")
	servlistStr, err := dao.redis.Get(keyname).Result()
	if err == nil {
		servlist := make([]businessService, 0)
		_ = json.Unmarshal([]byte(servlistStr), &servlist)
		fmt.Println("recovery services:", servlistStr)
		for _, serv := range servlist {
			servicelist.Services.Set(serv.Name, serv)
		}
	}
}

func CacheServices(stop <-chan struct{}) {
	cacheName := viper.GetString("service.cache.name")
	interval := viper.GetInt64("service.cache.interval-second")
	intervalDur := cache.ToDuration(interval, time.Second)
	tick := time.NewTicker(intervalDur)
	parallel.Tick(tick, stop, func() {
		list := servicelist.ValidServices()
		servBytes, _ := json.Marshal(list)
		servStr := string(servBytes)
		logger.Infof("cache service list:%s", servStr)
		dao.redis.Set(cacheName, servStr, intervalDur)
	})
}

func localAddress() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func (serv businessService) Addr() string {
	host := serv.Host
	port := serv.Port
	if host == "" {
		host = localNet
	}
	return host + port
}

func (serv businessService) PrefixPath() string {
	if serv.Path != "" {
		return serv.Path
	}
	return "/" + serv.Name
}

func (spec *ServiceSpec) ValidServices() []businessService {
	list := make([]businessService, 0)
	spec.Services.HasNext(func(service businessService, expireAt int64) bool {
		if !serviceExpired(expireAt) { //未过期
			list = append(list, service)
		}
		return true
	})
	return list
}
