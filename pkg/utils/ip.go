package utils

import "net"

func IsPublicIP(IP net.IP) bool {
	//如果ip是环回地址、链路本地组播地址、链路本地单播地址 返回false
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	//tcp/ip协议中，保留了三个IP地址区域作为私有地址，其地址范围如下：
	//10.0.0.0/8：     10.0.0.0 ～ 10.255.255.255
	//172.16.0.0/12：   172.16.0.0 ～ 172.31.255.255
	//192.168.0.0/16：  192.168.0.0 ～ 192.168.255.255
	if ip4 := IP.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}
