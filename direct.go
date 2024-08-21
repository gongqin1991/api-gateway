package main

import "github.com/spf13/viper"

func DirectRequest(request *DirectorRequest) {
	log := logger.WithContext(request.Context())
	printService := func(log *Logger, serv businessService) bool {
		log.Info("match path service:", serv)
		return true
	}
	var (
		address string
		urlPath string
	)
	log.Info("find service start...")
	//查找本地路由
	for _, serv := range servicelist.ValidServices() {
		if request.MatchPath(serv.Path) && printService(log, serv) {
			address = serv.Addr()
			urlPath = request.DirectPath(serv.PrefixPath())
			log.Infof("attached service:%v", serv)
			break
		}
	}
	//查找其他节点路由
	if nodeAddr := ""; cluster && address == "" && request.FetchReplica(log, &nodeAddr) {
		address = nodeAddr
	}
	if address == "" {
		log.Info("no service matched!")
	} else if gateway := viper.GetString("cluster.local.ip"); gateway != "" {
		request.AddHeader(HandleApiGateway, gateway)
	}
	request.SetHost(address)
	request.SetPath(urlPath)
}
