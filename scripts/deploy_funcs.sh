#!/bin/bash

#推送文件到云端服务器
function push_to_server() {
    argv=$#
    args=($*)
    if [ $argv -le 2 ];then
       echo "bad parameter!!!"
       exit 1
    fi
    server=$1
    workspace=$2
    source_file=$3
    declare rm_files
    declare ssh_port
    for((i=3;i<$argv;i++));do
       case "${args[i]}" in
         "--rm")
         rm_files="--remove-source-files"
         ;;
         "-p")
          ssh_port="ssh -p ${args[i+1]}"
          ;;
       esac
    done
    rsync --rsync-path="sudo rsync" --update $rm_files -avzhe "$ssh_port" --progress -O $source_file $server:$workspace
}

#执行远程脚本
function exec_remote_deploy() {
  argv=$#
  args=($*)
  if [ $argv -le 2 ];then
    echo "bad parameter!!!"
    exit 1
  fi
  server=$1
  workspace=$2
  deploy_file=$3
  declare ssh_port
  declare rm_files
  deploy_args=()
  for((i=3;i<$argv;i++));do
      case "${args[i]}" in
        "--rm")
         rm_files="&& rm -rf $deploy_file"
         ;;
        "-p")
         ssh_port="-p ${args[i+1]}"
         ;;
       "-n")
         for((k=0;k<${args[i+1]};k++));do
            deploy_args[k]=${args[i+2+k]}
         done
         ;;
      esac
  done
  ssh $ssh_port $server "cd $workspace && sh $deploy_file ${deploy_args[*]} $rm_files"
}