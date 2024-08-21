package main

import (
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
)

type Dao struct {
	redis *redis.Client
	err   error
}

var dao = &Dao{}

func (dao *Dao) Setup() {
	dao.redis, dao.err = redisSetup()
}

func redisSetup() (*redis.Client, error) {
	c := redis.NewClient(&redis.Options{
		Addr:     viper.GetString("common.redis.addr"),
		Password: viper.GetString("common.redis.pass"),
		DB:       viper.GetInt("common.redis.db"),
	})
	_, err := c.Ping().Result()
	return c, err
}
