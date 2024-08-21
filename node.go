package main

import (
	"awesomeProject/pkg/cache"
	"context"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"time"
)

var replSet = &ReplicaSpec{
	ReplicaSet: &cache.MemCache[replicaNode]{LState: cache.LockSync},
}

var replPin = &replicaSetPin{
	Message: onPinMessage,
}

func MeetCluster(ctx context.Context) {
	listen := viper.GetString("cluster.local.port")
	//开启接收握手协议的tcp服务
	//监听本地服务出现异常，则本节点不加入集群
	server := connServer{
		handler: recvReplicaNode,
	}
	if err := server.Listen(listen); err != nil {
		logger.Fatalf("cluster listen error,err:%v", err)
		return
	}
	//握手其他节点
	go replPin.Go(ctx)
	//处理其他节点握手请求
	go server.Run(ctx)
}

func recvReplicaNode(ctx context.Context, request handshakeBody, response handshakeBody) {
	remoteIp := cast.ToString(ctx.Value("ip"))
	item := replicaNode{
		IP:        remoteIp,
		RefreshAt: time.Now().UnixMilli(),
	}
	if err := request.DecodeStruct(&item); err != nil {
		logger.Errorf("decode request error,err:%v", err)
	} else {
		logger.Info("read:", item)
		replSet.ReplicaSet.Set(remoteIp, item)
	}
}

func onPinMessage() map[string]string {
	localPort := viper.GetString("cluster.local.port")
	proxyPort := viper.GetString("proxy.listen")
	registerPort := viper.GetString("register.listen")
	msg := map[string]string{
		"port":          localPort,
		"register_port": registerPort,
		"proxy_port":    proxyPort,
	}
	return msg
}
