#!/bin/sh

source ./funcs.sh

#工作目录
wkname=servc
#运行配置文件
runcfg=cfg.toml
#dokcerfile
dockerfi=Dockerfile
#docker_ports.py
dockerpos=docker_ports.py
#服务器名称
sername=$1
#部署平台
platform=$2

#解压源码
clean_unzip_project $wkname
#判断工作目录是否存在
if [ ! -d $wkname ];then
  echo "!!!no exist directory,name:$wkname"
  exit 1
fi

#进入工作目录
cd $wkname
#编译源码
echo ">>build workspace:$(pwd)"
go_build $wkname $sername
#判断运行文件是否存在
mainfile=main
if [ ! -f $mainfile ];then
  echo "!!!build fail,stop deploy"
  exit 1
fi

#重命名部署脚本
mv deploy_docker.sh deploy.sh

#打包必要文件
mkdir -p $wkname
cp -p $mainfile $wkname/
cp -p $runcfg $wkname/
cp -p $dockerfi $wkname/
cp -p $dockerpos $wkname/
#压缩目录
zip_project $wkname
#拷贝到运行服务器上
rsyncpkg=$wkname.tar.gz
deployfile=deploy.sh
workdir=/var/www/backend/
push_project_to_server root@106.75.49.211 $workdir $rsyncpkg $deployfile ../funcs.sh
#执行下步部署脚本
echo ">>docker deploy..."
next_deploy root@106.75.49.211 $workdir $deployfile $sername
#清理本地资源
rm -rf ../funcs.sh

