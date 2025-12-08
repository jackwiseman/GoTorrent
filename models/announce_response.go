package models

import (
	"encoding/binary"
	"errors"
	"net"
)

type AnnounceResponse struct {
	Action        uint32
	TransactionID uint32
	Interval      uint32
	Leechers      uint32
	Seeders       uint32
	Peers         []AnnouncePeer
}

type AnnouncePeer struct {
	IP   net.IP
	Port uint16
}

func UnmarshalAnnounceResponse(data []byte) (*AnnounceResponse, error) {
	if len(data) < 20 {
		return nil, errors.New("response too short")
	}

	resp := &AnnounceResponse{
		Action:        binary.BigEndian.Uint32(data[0:4]),
		TransactionID: binary.BigEndian.Uint32(data[4:8]),
		Interval:      binary.BigEndian.Uint32(data[8:12]),
		Leechers:      binary.BigEndian.Uint32(data[12:16]),
		Seeders:       binary.BigEndian.Uint32(data[16:20]),
	}

	peerData := data[20:]
	numPeers := len(peerData) / 6

	for i := 0; i < numPeers; i++ {
		offset := i * 6
		peer := AnnouncePeer{
			IP:   net.IP(peerData[offset : offset+4]),
			Port: binary.BigEndian.Uint16(peerData[offset+4 : offset+6]),
		}
		resp.Peers = append(resp.Peers, peer)
	}

	return resp, nil
}
