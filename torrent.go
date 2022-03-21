package main

import (
	"fmt"
	"strings"
	"strconv"
	"encoding/hex"
	"sync"	
	"io/ioutil"
	"bytes"

	bencode "github.com/jackpal/bencode-go"

)

type Torrent struct {
	magnet_link string
	display_name string
	info_hash []byte

	trackers []Tracker
	peers []Peer
	max_peers int

	metadata_size int // in bytes, given by first extended handshake
	metadata_raw []byte
	metadata_pieces []byte // array of [1/0, 1/0,...] denoting whether we have the piece or not
	metadata Metadata

	metadata_mx sync.Mutex // to ensure that that we only trigger "building" the metadata once
}

type Metadata_Piece struct {
	piece_num int
	data []byte
}

// for simplicity, only magnet links will be supported for now
func new_torrent(magnet_link string, max_peers int) (*Torrent) {
	var torrent Torrent
	torrent.magnet_link = magnet_link
	torrent.max_peers = max_peers
	torrent.parse_magnet_link()

//	var m sync.Mutex
//	torrent.metadata_mx = &m
	return &torrent
}

// only supporting udp links
func (torrent *Torrent) parse_magnet_link() {
	data := strings.Split(torrent.magnet_link, "&")
	for i := 0; i < len(data); i++ {
		switch(data[i][:2]) {
		case "dn":
			torrent.display_name = strings.Replace(data[i][3:], "%20", " ", -1)
		case "tr":
			tracker_link := data[i][3:] // cut off the tr=
			tracker_len := len(tracker_link)
			index := 0

			for index < tracker_len {
				if strings.Compare(string(tracker_link[index]), "%") == 0 {
					token, err := hex.DecodeString(string(tracker_link[index+1:index+3]))
					if err != nil {
						panic(err)
					}
					tracker_link = string(tracker_link[0:index]) + string(token) + string(tracker_link[index+3:])
					tracker_len -= 2
				}
				index++
			}
			if tracker_link[0:3] == "udp" {
				if strings.Contains(tracker_link, "announce") {
					tracker_link = tracker_link[:len(tracker_link) - len("/announce")]
				}
				new_tracker := new_tracker(tracker_link[6:])
				torrent.trackers = append(torrent.trackers, *new_tracker)
			}
		default:
			hash, err := hex.DecodeString(data[i][strings.LastIndex(data[i], ":")+1:])
			if err != nil {
				panic(err)
			}
			torrent.info_hash = hash
		}
	}

}

func (torrent Torrent) print_info() {
	fmt.Println("Name: " + torrent.display_name)
	fmt.Println("Magnet: " + torrent.magnet_link)
	fmt.Println("Trackers:")
	for i := 0; i < len(torrent.trackers); i++ {
		fmt.Println(" -- " + torrent.trackers[i].link)
	}
	fmt.Println("Known peers:")
	if len(torrent.peers) == 0 {
		fmt.Println(" -- None")
	} else {
		for i := 0; i < len(torrent.peers); i++ {
			fmt.Println(" -- " + torrent.peers[i].ip)
		}
	}
	if torrent.metadata_size != 0 {
		fmt.Println("Metadata size: " + strconv.Itoa(torrent.metadata_size) + " (" + strconv.Itoa(torrent.num_metadata_pieces()) + " pieces)")
	}
}

func (torrent *Torrent) find_peers() {
	var wg sync.WaitGroup

	// TODO: fix bad trackers?
	for i := 1; i < len(torrent.trackers); i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup, tracker Tracker) {
			defer wg.Done()
			fmt.Println("Connecting to " + tracker.link)

			err := tracker.connect()
			if err != nil {
				return
			}

			tracker.set_connection_id()
			if err != nil {
				return
			}

			tracker.announce(torrent)
			if err != nil {
				return
			}

			tracker.disconnect()
		} (&wg, torrent.trackers[i])
	}
	wg.Wait()

	// trim excess peers
	if len(torrent.peers) > torrent.max_peers {
		torrent.peers = torrent.peers[0:torrent.max_peers]
	}
}

func metadata_constructor(ch chan Metadata_Piece, metadata_raw *map[int][]byte, pieces *[]int) {
	for len(*metadata_raw) < len(*pieces) {
		piece := <-ch
		if piece.data == nil {
			continue
		}
		fmt.Println(*pieces)
		(*metadata_raw)[piece.piece_num] = piece.data
		(*pieces)[piece.piece_num] = 1
	}
}

//func (torrent *Torrent) get_metadata() {
//	// first let's find an alive peer to find the size of the file

//	var metadata_peers []Peer
//	pieces := make([]int, torrent.num_metadata_pieces()) // array of [0, 0, 0, 0, 0, 0, ...] denoting the pieces we have

//	var handshake_wg sync.WaitGroup
//	// populate array with peers who can send metadata
//	for i := 0; i < len(torrent.peers)-1;  i++ {
//		handshake_wg.Add(1)
//		go func(wg *sync.WaitGroup, metadata_peers *[]Peer, peer Peer) {
//			fmt.Println("Connecting to " + peer.ip)
//			defer wg.Done()
//			peer.connect()
//			peer.perform_handshake(torrent)
//			if peer.uses_extended {
//				_, supports_metadata := peer.extensions["ut_metadata"]
//				if supports_metadata {
//					*metadata_peers = append(*metadata_peers, peer)
//					return // don't disconnect from them if they have the info we need, obviously we're going to need to keep ALL of these connections alive eventually, but this is the initial step
//				}
//			}
//			torrent.peers[i].disconnect()
//		} (&handshake_wg, &metadata_peers, torrent.peers[i])
//	}

//	handshake_wg.Wait()

//	fmt.Println("Found " + strconv.Itoa(len(metadata_peers)) + " peers willing to send metadata")
//	fmt.Println(metadata_peers)

//	for i := 0; i < torrent.num_metadata_pieces(); i++ {
//		pieces = append(pieces, 0)
//	}

//	rand.Seed(time.Now().UnixNano())

//	// start a goroutine for each metadata peer (might not work if peers > pieces)
//	var metadata_collect sync.WaitGroup
//	for i := 0; i < len(metadata_peers); i++ {
//		metadata_collect.Add(1)
//		//go metadata_peers[i].request_metadata(&(torrent.metadata_raw), &pieces, &metadata_collect, torrent.metadata_size)
//	}

//	metadata_collect.Wait()

//	err := os.WriteFile("metadata.torrent", torrent.metadata_raw, 0644)
//	if err != nil {
//		panic(err)
//	}
//}

// assumes the filename is "metadata.torrent", which of course will not be valid in the future if there are multiple torrents
func (torrent *Torrent) parse_metadata_file() {
	data, err := ioutil.ReadFile("metadata.torrent")
	if err != nil {
		panic(err)
	}
	
	var result = Metadata{"", "", 0, "", 0, 0}
	reader := bytes.NewReader(data)
	err = bencode.Unmarshal(reader, &result)
	if err != nil {
		panic(err)
	}
	torrent.metadata = result
	torrent.display_name = torrent.metadata.Name
	fmt.Println(result)
//	return &result

}
func (torrent *Torrent) start_download() {
	// ensure we have the metadata
	torrent.parse_metadata_file()

	// get num_want peers
	torrent.find_peers()

	var wg sync.WaitGroup

	for i := 0; i < len(torrent.peers); i++ {
		wg.Add(1)
		go torrent.peers[i].run(torrent, &wg)
	}

	wg.Wait()
}
