package main

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strings"
)

// @url path:/ping
// @url method:POST
// @url params struct:main.bisService
// desc:服务注册接口
func registerService(c *gin.Context) {
	req := new(BusinessService)
	if err := c.BindJSON(req); err != nil {
		logger.Error("request parameters", err)
		c.String(http.StatusBadRequest, "bad request")
		return
	}
	serv := *req
	bisKey := req.Name
	if bisKey == "" {
		bisKey = serv.Addr()
	}
	servicelist.AddService(bisKey, &serv)
	c.String(http.StatusOK, "pong")
}

// @url path:/service/list
// @url method:GET
// desc:获取注册服务接口
func registeredServices(c *gin.Context) {
	serv := servicelist.ValidServices()
	c.JSON(http.StatusOK, serv)
}

// @url path:/service/match
// @url method:POST
// @url params struct:main.bisService
// desc:获取匹配的注册服务接口
func getService(c *gin.Context) {
	req := new(BusinessService)
	if err := c.BindJSON(req); err != nil {
		logger.Error("request parameters", err)
		c.String(http.StatusBadRequest, "bad request")
		return
	}
	find := map[string]any{
		"result": 0,
	}
	services := servicelist.ValidServices()
	for _, serv := range services {
		//名称匹配
		if req.Name == serv.Name {
			find["result"] = 1
			find["item"] = serv
			break
		}
		//路径匹配
		if req.Path != "" && strings.Index(req.Path, serv.PrefixPath()) == 0 {
			find["result"] = 1
			find["item"] = serv
			break
		}
	}
	c.JSON(http.StatusOK, find)
}

// @url path:/rate/limit/err
// @url method:GET
// desc:限流错误提示
func reachLimit(c *gin.Context) {
	rid := c.Request.Header.Get(headerRequestID)
	logger.WithFields(rid).Info("reach rate limit")
	c.String(http.StatusForbidden, "reach rate limit")
}

// @url path:/no/content
// @url method:OPTIONS
// desc:跨域
func noContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// desc:没有找到处理路由
func noFound(c *gin.Context) {
	rid := c.Request.Header.Get(headerRequestID)
	method := c.Request.Method
	ctyp := c.Request.Header.Get("Content-Type")
	if ctyp == "" {
		ctyp = c.Request.Header.Get("content-type")
	}
	body := ""
	if method == http.MethodPost && strings.Contains(ctyp, "application/json") {
		bs, err := io.ReadAll(c.Request.Body)
		if err == nil {
			body = string(bs)
		}
	}
	logger.WithFields(rid).Errorf("path:%s,body:%s,resp:no route", c.Request.RequestURI, body)
	c.String(http.StatusNotFound, "no found service")
}
