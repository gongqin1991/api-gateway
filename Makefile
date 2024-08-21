.PHONY: build clean local deploy test nginx cluster primary

UNAME := $(shell uname)
ARCH := $(shell arch)
DELIVER ?=mp
PLATFORM ?=main

#uname大写变小写
UNAME := $(shell echo $(UNAME) | tr A-Z a-z)
#x86_64=>amd64
ifeq ($(ARCH),x86_64)
	ARCH := amd64
endif
#远程部署
deploy:
	sh scripts/deploy.sh $(DELIVER) $(PLATFORM)

#本地测试
test:local
	./main --cfg=configs/local.cfg.toml

#仅仅只是替换nginx location文件
nginx:
	make DELIVER=nginx PLATFORM=location

#集群化部署
cluster:
	@echo ">>!!!deploy server 106.75.108.101"
	make PLATFORM=rep101
	@echo ">>!!!deploy server 106.75.49.211"
	make PLATFORM=rep211

#一级网关
primary:
	make DELIVER=primary PLATFORM=primary

secondary:
	make DELIVER=secondary PLATFORM=sec101
	make DELIVER=secondary PLATFORM=sec211


build:
	@echo $(UNAME)
	@echo $(ARCH)
	CGO_ENABLED=0 GOOS=$(UNAME) GOARCH=$(ARCH) go build -o main -tags "nomsgpack" -ldflags "-w -s -X 'main.environment=$(DELIVER)'"

local:clean
	go build -o main -tags "nomsgpack" -ldflags "-w -s -X 'main.environment=debug'"

clean:
	rm -rf main
	rm -rf servc
	rm -rf servc.tar.gz
	rm -rf cfg.toml
	rm -rf deploy.sh


