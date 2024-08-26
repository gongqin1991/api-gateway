package main

import (
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"net/http"
	"regexp"
)

type (
	corsRule struct {
		Match         string `mapstructure:"match"`
		MathPath      string `mapstructure:"match-path"`
		AllowOrigin   string `mapstructure:"allow-origin"`
		AllowHeaders  string `mapstructure:"allow-headers"`
		AllowMethods  string `mapstructure:"allow-methods"`
		ExposeHeaders string `mapstructure:"expose-headers"`
	}
	Cors struct {
		enable bool
		rules  []corsRule
	}
)

var cors = &Cors{}

func (c *Cors) Setup() {
	enable := viper.GetInt("proxy.cors.enable")
	rules := make([]corsRule, 0)
	if val := viper.Get("proxy.cors.rules"); val != nil {
		for _, rule := range val.([]interface{}) {
			dst := corsRule{}
			_ = mapstructure.Decode(rule, &dst)
			rules = append(rules, dst)
		}
	}
	c.rules = rules
	c.enable = enable == ON && len(rules) > 0
}

func matchService(services []BusinessService, expr, request string) bool {
	if expr == "" || expr == "*" {
		return true
	}
	//微服务名称匹配规则被设置
	//微服务名称没有匹配上则不需要匹配
	got := false
	for _, serv := range services {
		if serv.Name != "" && regexpMatch(expr, serv.Name) && regexpMatch(serv.Path, request) {
			got = true
			break
		}
	}
	return got
}

func (c *Cors) PreCheck(request *DirectorRequest) {
	if !c.enable {
		return
	}
	match := false
	services := servicelist.ValidServices()
	for _, rule := range c.rules {
		if matchService(services, rule.Match, request.Path()) &&
			regexpMatch(rule.MathPath, request.Path()) {
			match = true
			break
		}
	}
	if match && request.Method == http.MethodOptions {
		request.SetPath("/no/content")
		request.Abort()
	}
}

func matchAll(rule corsRule) bool {
	return (rule.Match == "" || rule.Match == "*") && (rule.MathPath == "" || rule.MathPath == "*")
}

func regexpMatch(expr, request string) bool {
	if expr == "" || expr == "*" {
		return true
	}
	reg, err := regexp.Compile(expr)
	if err != nil {
		return false
	}
	return reg.MatchString(request)
}

func (c *Cors) Check(resp *http.Response) {
	if !c.enable {
		return
	}
	request := DirectorRequest{
		Request: resp.Request,
	}
	services := servicelist.ValidServices()
	var (
		allRule  *corsRule
		bestRule *corsRule
	)
	for i := range c.rules {
		rule := c.rules[i]
		if matchAll(rule) {
			allRule = &rule
			continue
		}
		if matchService(services, rule.Match, request.Path()) &&
			regexpMatch(rule.MathPath, request.Path()) {
			bestRule = &rule
		}
	}
	if bestRule != nil {
		bestRule.AddHeader(resp.Header)
		return
	}
	if allRule != nil {
		allRule.AddHeader(resp.Header)
	}
}

func (r corsRule) AddHeader(header http.Header) {
	header.Add("Access-Control-Allow-Origin", r.AllowOrigin)
	header.Add("Access-Control-Allow-Headers", r.AllowHeaders)
	header.Add("Access-Control-Allow-Methods", r.AllowMethods)
	exposeHeaders := r.ExposeHeaders
	if exposeHeaders != "" {
		header.Add("Access-Control-Expose-Headers", exposeHeaders)
	}
	header.Add("Access-Control-Allow-Credentials", "true")
}
