#!/bin/bash

#编译源码
function go_build() {
  project=$1
  platform=$2
  #go mod初始化
  export PATH=$PATH:/usr/local/go/bin
  export GO111MODULE=on
  export GOPROXY=https://goproxy.cn
  go mod init $project
  go mod tidy
  #编译源码
  make build DELIVER=$platform
}

#判断文件是否存在，不存在退出
function assert_file() {
  file=$1
  if [ ! -f $file ];then
    echo "no exist file,filename:$file"
    exit 1
  fi
}

#判断目录是否存在，不存在退出
function assert_dir() {
  file=$1
  if [ ! -d $file ];then
    echo "no exist directory,dirname:$file"
    exit 1
  fi
}