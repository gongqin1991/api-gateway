#!/bin/bash

source ./funcs.sh
source ./zip_funcs.sh
source ./deploy_funcs.sh

#工作目录
wk_name=servc
#运行文件
exec_file=main
#部署环境
deploy_dev=$1
#目标服务器
target_server=$2

#解压源码
do_unzip $wk_name --rm
#判断工作目录是否存在
assert_dir $wk_name

#编译源码
echo ">>build workspace,$wk_name $deploy_dev"
cd $wk_name && go_build $wk_name $deploy_dev
#判断运行文件是否存在
assert_file $exec_file

#清除源码文件
rm -rf *.go
rm -rf go.*

#打包必要文件
cd ..
cp docker_ports.py $wk_name/ && rm -rf docker_ports.py
cp Dockerfile $wk_name/ && rm -rf Dockerfile

#压缩目录
do_zip $wk_name --rm
#重命名部署脚本
mv deploy_target.sh deploy.sh
#拷贝到运行服务器上
rsyncpkg=$wkname.tar.gz
deployfile=deploy.sh
push_to_server $target_server /var/www/backend/ $wk_name.tar.gz -p 50100 --rm
push_to_server $target_server /var/www/backend/ funcs.sh -p 50100 --rm
push_to_server $target_server /var/www/backend/ docker_funcs.sh -p 50100 --rm
push_to_server $target_server /var/www/backend/ deploy_funcs.sh -p 50100 --rm
push_to_server $target_server /var/www/backend/ zip_funcs.sh -p 50100 --rm
push_to_server $target_server /var/www/backend/ deploy.sh -p 50100 --rm
#执行下步部署脚本
echo ">>startup..."
exec_remote_deploy $target_server /var/www/backend/ $deployfile -p 50100 -n 1 $deploy_dev --rm

