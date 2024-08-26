#!/bin/bash

#打包项目
function do_zip() {
  argv=$#
  args=($*)
  if [ $argv -lt 1 ];then
    echo "bad parameter!!!"
    exit 1
  fi
  project=$1
  declare rm_files
  for((i=1;i<$argv;i++));do
      case "${args[i]}" in
        "--rm")
          rm_files=$project
          ;;
      esac
  done
  tar --no-xattrs -czvf $project.tar.gz $project
  if [ -d $rm_files ];then
      rm -rf $rm_files
  fi
}

#解压项目
function do_unzip() {
  argv=$#
  args=($*)
  if [ $argv -lt 1 ];then
    echo "bad parameter!!!"
    exit 1
  fi
  project=$1
  declare rm_files
  for((i=1;i<$argv;i++));do
    case "${args[i]}" in
      "--rm")
        rm_files=$project.tar.gz
        ;;
    esac
  done
  rm -rf $project
  tar --no-xattrs -zxvf $project.tar.gz
  if [ -f $rm_files ];then
    rm -rf $rm_files
  fi
}