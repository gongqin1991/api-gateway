#!/bin/sh

#修复源代码引用路径
function correct_pkg() {
  dir=$1
  olddir=$(pwd)
  cd $dir
  files=$(ls -l|awk '{print $9}'|grep '.go'|grep -v grep)
  for each in ${files[*]}
  do
    cat $each | sed 's/awesomeProject/servc/g' > $each.1 && mv $each.1 $each
  done
  cd $olddir
}

#打包项目
function zip_project() {
    project=$1
    tar -czvf $project.tar.gz $project
}

#删除老的项目源码并解压新项目
function clean_unzip_project() {
    project=$1
    rm -rf $project
    tar -zxvf $project.tar.gz
    rm -rf $project.tar.gz
}

#推送项目源码到云端服务器
function push_project_to_server() {
    server=$1
    workspace=$2
    project=$3
    deployfile=$4
    funcfile=$5
    rsync --rsync-path="sudo rsync" --update --remove-source-files -avzhe "ssh -p 50100" --progress -O  $project $server:$workspace
    rsync --rsync-path="sudo rsync" --update --remove-source-files -avzhe "ssh -p 50100" --progress -O  $deployfile $server:$workspace
    rsync --rsync-path="sudo rsync" --update -avzhe "ssh -p 50100" --progress -O  $funcfile $server:$workspace
}

#执行脚本
function next_deploy() {
   server=$1
   workspace=$2
   deployfile=$3
   deploydev=$4
   platform=$5
  ssh -p 50100 $server "cd $workspace && sh $deployfile $deploydev $platform && rm -rf $deployfile"
}

#编译源码
function go_build() {
  project=$1
  platform=$2
  #go mod初始化
  export PATH=$PATH:/usr/local/go/bin
  export GO111MODULE=on
  go mod init $project
  go mod tidy
  #编译源码
  make build DELIVER=$platform
}

#清除老的docker镜像和容器
function remove_image_container() {
    name=$1
    tag=$2
    image=$name:$tag
    container=$name.$tag
    containers=$(docker ps -aq --no-trunc --filter ancestor=$image)
    if [ $containers ];then
      docker rm -f $containers
      docker rmi -f $image
    fi
}

#制做docker镜像
function make_image() {
    name=$1
    tag=$2
    image=$name:$tag
    docker build -t $image . --rm
}

#运行容器
function run_container() {
    name=$1
    tag=$2
    ports=$3
    image=$name:$tag
    container=$name.$tag
    docker run -d --restart=always $ports -v /var/www/backend/logs/servc:/root/logs --name $container $image
}