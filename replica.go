package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"sync"
)

type ReplicaNode struct {
	IP         string
	Port       string `mapstructure:"port"`
	RPort      string `mapstructure:"register_port"`
	PPort      string `mapstructure:"proxy_port"`
	refreshAt  int64  //刷新时间戳
	FullReject string `mapstructure:"full_reject"` //是否达到流量上线 1:拒绝访问
}

type ReplicaSpec struct {
	SearchIP       map[string]*ReplicaNode
	Nodes          []*ReplicaNode
	serviceExpires int64
	mu             sync.Mutex
}

func (spec *ReplicaSpec) Setup() {
	expire := viper.GetInt64("cluster.replicaSet.handshake.service-expire")
	spec.serviceExpires = expire
	spec.SearchIP = make(map[string]*ReplicaNode)
	spec.Nodes = make([]*ReplicaNode, 0)
}

func (rep ReplicaNode) HasService(path string) bool {
	var (
		bs  []byte
		err error
	)
	urlStr := rep.serviceMatchUrl()
	bs, err = json.Marshal(map[string]any{"path": path})
	var resp *http.Response
	resp, err = http.Post(urlStr, "application/json", bytes.NewBuffer(bs))
	if err != nil {
		logger.Fatalf("get service error,url:%s,path:%s,err:%v", urlStr, path, err)
		return false
	}
	defer func() {
		discard(resp.Body.Close())
	}()
	if resp.StatusCode != http.StatusOK {
		logger.Fatalf("get service error,url:%s,path:%s,status:%v", urlStr, path, resp.Status)
		return false
	}
	result := make(map[string]any)
	bs, err = io.ReadAll(resp.Body)
	err = json.Unmarshal(bs, &result)
	return err == nil && cast.ToInt(result["result"]) == 1
}

func (rep ReplicaNode) serviceMatchUrl() string {
	return fmt.Sprintf("http://%s%s/service/match", rep.IP, rep.RPort)
}

func (rep ReplicaNode) serviceTargetUrl() string {
	return fmt.Sprintf("http://%s%s", rep.IP, rep.PPort)
}

func (spec *ReplicaSpec) ValidReplicaSet() []ReplicaNode {
	nodes := make([]ReplicaNode, 0)
	spec.mu.Lock()
	defer spec.mu.Unlock()
	for _, node := range spec.Nodes {
		if !serviceExpired(node.refreshAt + SECOND*spec.serviceExpires) {
			nodes = append(nodes, *node)
		}
	}
	return nodes
}

func (spec *ReplicaSpec) AddNode(ip string, node *ReplicaNode) {
	node.refreshAt = clock()
	spec.mu.Lock()
	defer spec.mu.Unlock()
	if item, ok := spec.SearchIP[ip]; !ok {
		spec.Nodes = append(spec.Nodes, node)
	} else {
		item.refreshAt = node.refreshAt
		item.FullReject = node.FullReject
		item.IP = node.IP
		item.Port = node.Port
		item.RPort = node.RPort
		item.PPort = node.PPort
	}
	spec.SearchIP[ip] = node
}
