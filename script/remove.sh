#!/bin/bash
# 上传到远程和写到文件同时开启时，删除pcap文件的脚本
p=${1:-/work_dir/pcap_dir} # uat
t=${2:-4320} # 3天，保留3天内的pcap文件

cd $p
find . -name "*.pcap" -mmin +$t | xargs rm 
