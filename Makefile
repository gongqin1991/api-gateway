.PHONY: build clean local deploy startup_mp

DELIVER ?=debug #部署服务器
PLATFORM ?=main#部署渠道
CFG_FILETYPE ?=toml #启动配置文件类型
BUILD_SERVER ?=root@106.75.108.101 #编译服务器
TARGET_SERVER ?=root@106.75.49.211 #目标服务器

UNAME := $(shell uname)
ARCH := $(shell arch)

#uname大写变小写
UNAME := $(shell echo $(UNAME) | tr A-Z a-z)
#x86_64=>amd64
ifeq ($(ARCH),x86_64)
	ARCH := amd64
endif

#远程部署
deploy:
	sh scripts/deploy.sh $(DELIVER) $(PLATFORM) $(CFG_FILETYPE) $(BUILD_SERVER) $(TARGET_SERVER)

#本地启动
local:build
	./main --cfg=configs/local.cfg.toml

#编译
build:
	@echo ">>uname:$(UNAME)"
	@echo ">>arch:$(ARCH)"
	CGO_ENABLED=0 GOOS=$(UNAME) GOARCH=$(ARCH) go build -o main -tags "nomsgpack" -ldflags "-w -s -X 'main.environment=$(DELIVER)'" -buildvcs=false

#清除本地文件
clean:
	rm -rf main
	rm -rf servc
	rm -rf servc.tar.gz
	rm -rf cfg.toml
	rm -rf deploy.sh

#启动mp环境服务
startup_mp:
	cp /usr/share/zoneinfo/Asia/Shanghai .
	@echo "$(shell pwd)"
	@source ../docker_funcs.sh && \
	remove_image_container api mp && \
    make_image api mp && \
    run_container api mp
