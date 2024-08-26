#!/bin/bash

source scripts/funcs.sh
source scripts/zip_funcs.sh
source scripts/deploy_funcs.sh

scp_dir=scripts
cfg_dir=configs
#部署环境
deploy_dev=$1
#部署平台
platform=$2
#配置文件类型（toml|yml|yaml）
cfg_filetype=$3
#编译服务器
build_server=$4
#目标服务器
target_server=$5
#运行配置文件
run_cfg=cfg.$cfg_filetype
#部署目录名称
pkg_name=servc

#重命名配置文件
cp $cfg_dir/$platform.cfg.$cfg_filetype $run_cfg
#重命名部署脚本
cp $scp_dir/deploy_remote.sh deploy.sh
#判断配置文件是否存在
assert_file $run_cfg

#打包必要文件
mkdir -p $pkg_name
cp -p *.go $pkg_name/
cp -p Makefile $pkg_name/
cp -p $run_cfg $pkg_name/ && rm -rf $run_cfg

#压缩项目
do_zip $pkg_name --rm

unset LC_CTYPE
export LANG=zh_CN.UTF-8

#拷贝到部署服务器上
rsyncpkg=$pkg_name.tar.gz
deployfile=deploy.sh
echo ">>rsync $rsyncpkg to remote server"
push_to_server $build_server /tmp/ $rsyncpkg -p 50100 --rm
push_to_server $build_server /tmp/ $deployfile -p 50100 --rm
push_to_server $build_server /tmp/ $scp_dir/funcs.sh -p 50100
push_to_server $build_server /tmp/ $scp_dir/docker_funcs.sh -p 50100
push_to_server $build_server /tmp/ $scp_dir/deploy_funcs.sh -p 50100
push_to_server $build_server /tmp/ $scp_dir/zip_funcs.sh -p 50100
push_to_server $build_server /tmp/ $scp_dir/Dockerfile -p 50100
push_to_server $build_server /tmp/ $scp_dir/deploy_target.sh -p 50100
push_to_server $build_server /tmp/ $scp_dir/docker_ports.py -p 50100

#执行下步部署脚本
echo ">>deploy start"
exec_remote_deploy $build_server /tmp/ $deployfile -p 50100 -n 2 $deploy_dev $target_server
#清理本地部署资源
echo ">>deploy end,clean local resources"
make clean
