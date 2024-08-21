package main

import (
	"awesomeProject/pkg/cache"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"io"
	"net/http"
	"time"
)

type replicaNode struct {
	IP         string
	Port       string `mapstructure:"port"`
	RPort      string `mapstructure:"register_port"`
	PPort      string `mapstructure:"proxy_port"`
	RefreshAt  int64  //刷新时间戳
	FullReject string `mapstructure:"full_reject"` //是否达到流量上线 1:拒绝访问
}

type ReplicaSpec struct {
	ReplicaSet *cache.MemCache[replicaNode]
}

func (spec *ReplicaSpec) Setup() {
	expire := viper.GetInt64("cluster.replicaSet.handshake.service-expire")
	spec.ReplicaSet.Expire = cache.ToDuration(expire, time.Second)
	spec.ReplicaSet.Init()
}

func (rep replicaNode) HasService(path string) bool {
	url := fmt.Sprintf("http://%s%s/service/match", rep.IP, rep.RPort)
	params := map[string]any{
		"path": path,
	}
	bs, _ := json.Marshal(params)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bs))
	if err != nil {
		logger.Fatalf("get service error,url:%s,path:%s,err:%v", url, path, err)
		return false
	}
	if resp.StatusCode != http.StatusOK {
		logger.Fatalf("get service error,url:%s,path:%s,status:%v", url, path, resp.Status)
		return false
	}
	result := make(map[string]any)
	defer resp.Body.Close()
	bs, _ = io.ReadAll(resp.Body)
	_ = json.Unmarshal(bs, &result)
	return cast.ToInt(result["result"]) == 1
}

func (spec *ReplicaSpec) ValidReplicaSet() []replicaNode {
	list := make([]replicaNode, 0)
	spec.ReplicaSet.HasNext(func(item replicaNode, expireAt int64) bool {
		if !serviceExpired(expireAt) { //未过期
			list = append(list, item)
		}
		return true
	})
	return list
}
