#!/bin/sh

source ./funcs.sh

#工作目录
wkname=servc
#服务名称
msname=servregcenter
#部署环境（mp|nginx|primary|replica）
deploydev=$1

#解压运行环境
clean_unzip_project $wkname
#判断工作目录是否存在
if [ ! -d $wkname ];then
  echo "!!!no exist directory,name:$wkname"
  exit 1
fi

unset LC_CTYPE
export LANG=zh_CN.UTF-8

#进入工作目录
cd $wkname
#时间区域
cp /usr/share/zoneinfo/Asia/Shanghai .
#从cfg.toml配置文件获取容器映射端口
ports=$(python3 docker_ports.py)
echo "ports:$ports"
##清除历史镜像容器
#remove_image_container $msname $deploydev
##制做镜像
#echo ">>docker build..."
#make_image $msname $deploydev
##运行容器
#echo ">>docker run..."
#run_container $msname $deploydev $ports
#清除本地资源
rm -rf ../funcs.sh