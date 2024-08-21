#!/bin/sh

source scripts/funcs.sh

scpdir=scripts
cfgdir=configs
#部署环境（mp|nginx|primary|replica）
deploydev=$1
#部署平台
platform=$2
#运行配置文件
runcfg=cfg.toml
#部署目录名称
pkgname=servc

#仅仅替换线上nginx配置文件
if [ $deploydev == "nginx" ];then
   echo "replace nginx config location"
   cd $cfgdir
   sshc=root@106.75.49.211
   scp -P 50100 nginx/dev.location $sshc:/opt/nginx/nginx/conf/modules/
   scp -P 50100 nginx/mp.location $sshc:/opt/nginx/nginx/conf/modules/
   exit 0
fi

#重命名配置文件
cp $cfgdir/$platform.cfg.toml $runcfg
#重命名部署脚本
cp $scpdir/deploy_remote.sh deploy.sh
#判断配置文件是否存在
if [ ! -f $runcfg ];then
  echo "!!!no exist file,name:$runcfg"
  exit 1
fi

#打包必要文件
mkdir -p $pkgname
cp -afp ../pkg $pkgname/
cp -afp ../pkg $pkgname/
cp -p $runcfg $pkgname/
cp -p $scpdir/Dockerfile $pkgname/
cp -p $scpdir/deploy_docker.sh $pkgname/
cp -p $scpdir/docker_ports.py $pkgname/
cp -p *.go $pkgname/
cp -p Makefile $pkgname/
#修改引用包名
#打包后，业务代码引用cache和parallel路径有变化
correct_pkg $pkgname
#压缩项目
zip_project $pkgname
#拷贝到部署服务器上
rsyncpkg=$pkgname.tar.gz
deployfile=deploy.sh
echo ">>rsync $rsyncpkg to remote server"
push_project_to_server root@106.75.108.101 /tmp/ $rsyncpkg $deployfile scripts/funcs.sh
#执行下步部署脚本
echo ">>deploy start"
next_deploy root@106.75.108.101 /tmp/ $deployfile $deploydev $platform
#清理本地部署资源
echo ">>deploy end,clean local resources"
make clean
