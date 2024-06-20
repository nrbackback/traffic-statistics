#!/bin/bash
# 该脚本作为参考，实际使用中可能需要修改网络接口
fping --interface ens3 -a -r 0 -g 10.50.4.0/24      
curl --interface ens3 baidu.com
curl --interface ens3 tencent.com
curl --interface ens3 weibo.com
curl --interface ens3 gitee.com
curl --interface ens3 gitlab.com
curl --interface ens3 bitbucket.org
curl --interface ens3 github.com
curl --interface ens3 google.com
curl --interface ens3 youtube.com
curl --interface ens3 facebook.com
curl --interface ens3 facebook.com
