package netdata

import (
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

type NetData struct {
	ID     uint64 `json:"id"`
	Device string `json:"device"`

	CreateTime time.Time `json:"create_time"` // RFC3339 格式
	PackSize   int32     `json:"pack_size"`

	SrcIP string `json:"src_ip"`
	DstIP string `json:"dst_ip"`
}

func NetDataFromPacket(device string, uID uint64, packet gopacket.Packet) NetData {
	d := NetData{
		ID:         uID,
		Device:     device,
		CreateTime: packet.Metadata().Timestamp,
		PackSize:   int32(packet.Metadata().Length),
	}
	netLayerUpdate(packet, &d)
	updateLength(packet, &d)
	return d
}

const (
	IPVersion4 = 4
	IPVersion6 = 6
)

func netLayerUpdate(packet gopacket.Packet, d *NetData) {
	if iPv4Layer := packet.Layer(layers.LayerTypeIPv4); iPv4Layer != nil {
		l, _ := iPv4Layer.(*layers.IPv4)
		d.SrcIP = l.SrcIP.String()
		d.DstIP = l.DstIP.String()
	}
	if iPv6Layer := packet.Layer(layers.LayerTypeIPv6); iPv6Layer != nil {
		l, _ := iPv6Layer.(*layers.IPv6)
		d.SrcIP = l.SrcIP.String()
		d.DstIP = l.DstIP.String()
	}
}

const (
	MSS                 = 1460
	NotTCPPayloadLength = 46 // 以太网 18 + IP头 20 + UDP头 8
)

func updateLength(packet gopacket.Packet, d *NetData) {
	if packet.Metadata().Length <= 1518 {
		return
	}
	var paylodLength int
	var headerLength int
	// TODO: MTU 的 1500 好像是对 IP层限制的，而不是TCP层限制的
	if packet.Layer(layers.LayerTypeTCP) != nil {
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		l, _ := tcpLayer.(*layers.TCP)

		paylodLength = len(l.BaseLayer.Payload)
		// TCP 层及更底层的所有协议层的Header之和
		headerLength = packet.Metadata().Length - paylodLength
		headerLength += 4
	} else {
		headerLength = NotTCPPayloadLength
		paylodLength = packet.Metadata().Length - headerLength
	}

	reassemblyPacketCount := paylodLength / MSS
	if paylodLength%MSS != 0 {
		reassemblyPacketCount++
	}
	d.PackSize += int32(headerLength * (reassemblyPacketCount - 1))

}
