package main

import (
	"bytes"
	"encoding/json"
	"github.com/spf13/viper"
	"net/http"
	"time"
)

func SecondaryRegistry(stop <-chan struct{}) {
	regurl := viper.GetString("secondary.primary")
	localip := viper.GetString("common.local-ip")
	proxyPort := viper.GetString("proxy.listen")
	interval := viper.GetInt64("secondary.interval-second")
	tick := time.NewTicker(ToDuration(interval, time.Second))
	if localip == "" {
		localip = localAddress()
	}
	reqBody := map[string]interface{}{
		"port":    proxyPort,
		"host":    localip,
		"path":    "*",
		"gateway": true,
	}
	do := func() {
		bs, _ := json.Marshal(reqBody)
		request, _ := http.NewRequest(http.MethodPost, regurl, bytes.NewReader(bs))
		request.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			logger.Fatalf("secondary register error,err:%v", err)
		} else if resp.StatusCode != http.StatusOK {
			logger.Fatalf("secondary register error,status:%s", resp.Status)
		}
	}
	do()
	Tick(tick, stop, do)
}
