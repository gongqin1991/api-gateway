#!/bin/bash

source ./funcs.sh
source ./zip_funcs.sh
source ./deploy_funcs.sh

#工作目录
wk_name=servc
#服务名称
ms_name=servregcenter
#部署环境（mp||primary|replica）
deploy_dev=$1

#解压运行环境
do_unzip $wk_name --rm
#判断工作目录是否存在
assert_dir $wk_name

#启动服务
cd $wk_name && make startup_$deploy_dev
#清除本地资源
cd .. && rm -rf *.sh