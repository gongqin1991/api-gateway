package main

import (
	parallel2 "awesomeProject/pkg/parallel"
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	path2 "path"
	"path/filepath"
	"syscall"
	"time"
)

const (
	localNet = "localhost"
	hostName = "[none]"
)

const (
	_ = iota
	ON
)

var (
	//配置文件路径
	cfgPath string
	//运行环境
	environment string
)

var (
	//debug环境
	debug bool
	//集群环境
	cluster bool
	//服务缓存
	serviceCache bool
	//二级网关
	secondary bool
)

func init() {
	if environment == "debug" {
		debug = true
	}
}

func init() {
	flag.StringVar(&cfgPath, "cfg", "", "input config file")
}

func loadConf() (err error) {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = hostName
	}
	fileType := path2.Ext(cfgPath)
	fmt.Println("hostname=", hostname)
	fmt.Println("cfg file path=", cfgPath)
	fmt.Println("cfg file type=", fileType)
	if fileType != ".toml" && fileType != ".yaml" {
		err = errors.New("only support toml or yaml file")
		return
	}
	filename := filepath.Base(cfgPath)
	path := cfgPath[:len(cfgPath)-len(filename)]
	fmt.Println("cfg file name=", filename)
	fmt.Println("cfg file dir=", path)
	viper.SetConfigType(fileType[1:])
	if debug {
		viper.SetConfigFile(cfgPath)
	} else {
		viper.AddConfigPath(path)
		viper.SetConfigFile(filename)
	}
	viper.SetDefault("hostname", hostname)
	if err = viper.ReadInConfig(); err != nil {
		return
	}
	defer func() {
		if enable := viper.GetInt("cluster.enable"); enable == ON {
			cluster = true
		}
		if enable := viper.GetInt("service.cache.enable"); enable == ON {
			serviceCache = true
		}
		if enable := viper.GetInt("secondary.enable"); enable == ON {
			secondary = true
		}
	}()
	defer viper.WatchConfig()
	return
}

func main() {
	fmt.Println("environment:", environment)
	flag.Parse()
	if cfgPath == "" {
		fmt.Println("no config file")
		os.Exit(1)
	}
	//加载配置文件
	if err := loadConf(); err != nil {
		fmt.Println("load config file error,", err)
		os.Exit(1)
	}
	fmt.Println("serviceCache:", serviceCache)
	fmt.Println("cluster:", cluster)
	fmt.Println("secondary:", secondary)
	if err := logger.Setup(); err != nil {
		fmt.Println("log init error,", err)
		os.Exit(1)
	}
	if dao.Setup(); dao.err != nil && !debug {
		fmt.Println("dao init error", dao.err)
		os.Exit(1)
	}

	execGroup := parallel2.NewExecGroup()
	cancelGroup := parallel2.NewCancelGroup()

	//服务层面初始化
	cors.Setup()
	balancer.Setup()
	rLimiter.Setup()
	servicelist.Setup()
	replSet.Setup()
	replPin.Setup()
	//redis缓存微服务，防止容器重启或者重新部署期间丢失请求
	if serviceCache {
		LoadServices()
	}
	//反向代理
	cancelGroup.Add(startProxyServer())
	//服务注册
	cancelGroup.Add(startRegisterServer())
	//缓存服务
	if serviceCache {
		execGroup.Add(CacheServices)
	}
	//二级网关服务
	if secondary {
		execGroup.Add(SecondaryRegistry)
	}
	//加入到集群
	//通过tcp协议进行节点握手
	if cluster {
		ctx, cancel := context.WithCancel(context.TODO())
		cancelGroup.Add(cancel)
		go MeetCluster(ctx)
	}
	//优雅退出
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGKILL)
	s := <-quit
	cancelGroup.Cancel()
	execGroup.Close()
	logger.Destroy()
	fmt.Println(">>exit for", s.String())
	time.Sleep(time.Second * 2)
}

func startProxyServer() context.CancelFunc {
	middlewares := &DirectorMiddleware{}
	middlewares.Use(cors.PreCheck)
	middlewares.Use(RequestID, DumpRequest)
	middlewares.Use(rLimiter.RateLimit)
	middlewares.Use(balancer.Balance)
	purl, _ := url.Parse(viper.GetString("proxy.target")) //target
	p := httputil.NewSingleHostReverseProxy(purl)
	p.ErrorLog = logger.ErrorLog
	p.Director = ProxyDirector(p, middlewares)
	p.ErrorHandler = ErrorHandler(p)
	p.ModifyResponse = ModifyResponse(p)
	srv := &http.Server{
		Addr:     viper.GetString("proxy.listen"),
		Handler:  p,
		ErrorLog: logger.ErrorLog,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("proxy server listen error: %s\n", err)
			os.Exit(1)
		}
	}()
	return func() {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		go srv.Shutdown(ctx)
	}
}

func startRegisterServer() context.CancelFunc {
	serv := gin.Default()
	if !debug {
		gin.SetMode(gin.ReleaseMode)
	}
	serv.POST("/ping", registerService)
	serv.GET("/service/list", registeredServices)
	serv.POST("/service/match", getService)
	serv.GET("/rate/limit/err", reachLimit)
	serv.OPTIONS("/no/content", noContent)
	serv.NoRoute(noFound)
	srv := &http.Server{
		Addr:    viper.GetString("register.listen"),
		Handler: serv,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("api server listen error: %s\n", err)
			os.Exit(1)
		}
	}()
	return func() {
		ctx, _ := context.WithTimeout(context.Background(), time.Second)
		go srv.Shutdown(ctx)
	}
}

func serviceExpired(expireAt int64) bool {
	return expireAt <= 0 || time.Now().UnixMilli() >= expireAt
}
