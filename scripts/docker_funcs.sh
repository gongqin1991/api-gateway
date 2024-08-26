#!/bin/bash

#清除老的docker镜像和容器
function remove_image_container() {
    image=$1:$2
    container=$1.$2
    containers=$(docker ps -aq --no-trunc --filter ancestor=$image)
    if [ $containers ];then
      docker rm -f $containers
      docker rmi -f $image
    fi
}

#制做docker镜像
function make_image() {
    docker build -t $1:$2 . --rm
}

#运行容器
function run_container() {
    image=$1:$2
    container=$1.$2
    ports=$(python3 docker_ports.py)
    docker run -d --restart=always $ports -v /var/www/backend/logs/servc:/root/logs --name $container $image
}