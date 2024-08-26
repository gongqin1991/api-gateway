package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"net"
	"sync"
	"time"
)

type BusinessService struct {
	Name      string `json:"ms"`
	Path      string `json:"path"`
	Host      string `json:"host,omitempty"`
	Port      string `json:"port,omitempty"`
	Gateway   bool   `json:"gateway"`
	refreshAt int64
}

type ServiceSpec struct {
	Services       []*BusinessService
	serviceExpires int64
	mu             sync.Mutex
}

var servicelist = &ServiceSpec{}

func (spec *ServiceSpec) Setup() {
	expire := viper.GetInt64("register.service-expire")
	spec.serviceExpires = expire
	spec.Services = make([]*BusinessService, 0)
}

func LoadServices() {
	keyname := viper.GetString("service.cache.name")
	servlistStr, err := dao.redis.Get(keyname).Result()
	if err == nil {
		servlist := make([]*BusinessService, 0)
		discard(json.Unmarshal([]byte(servlistStr), &servlist))
		fmt.Println("recovery services:", servlistStr)
		ts := clock()
		for _, serv := range servlist {
			serv.refreshAt = ts
		}
		servicelist.Services = servlist
	}
}

func CacheServices(stop <-chan struct{}) {
	cacheName := viper.GetString("service.cache.name")
	interval := viper.GetInt64("service.cache.interval-second")
	intervalDur := ToDuration(interval, time.Second)
	tick := time.NewTicker(intervalDur)
	Tick(tick, stop, func() {
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

func (serv BusinessService) Addr() string {
	host := serv.Host
	port := serv.Port
	if host == "" {
		host = localNet
	}
	return host + port
}

func (serv BusinessService) PrefixPath() string {
	if serv.Path != "" {
		return serv.Path
	}
	return "/" + serv.Name
}

func (spec *ServiceSpec) ValidServices() []BusinessService {
	services := make([]BusinessService, 0)
	spec.mu.Lock()
	defer spec.mu.Unlock()
	for _, service := range spec.Services {
		if !serviceExpired(service.refreshAt + SECOND*spec.serviceExpires) {
			services = append(services, *service)
		}
	}
	return services
}

func (spec *ServiceSpec) AddService(bisKey string, serv *BusinessService) {
	serv.refreshAt = clock()
	spec.mu.Lock()
	defer spec.mu.Unlock()
	dict := make(map[string]int)
	var (
		key string
	)
	for i, service := range spec.Services {
		key = service.Name
		if key == "" {
			key = service.Addr()
		}
		dict[key] = i
	}
	if index, ok := dict[bisKey]; ok {
		spec.Services[index] = serv
	} else {
		spec.Services = append(spec.Services, serv)
	}
}
