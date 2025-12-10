package models

import (
	"time"

	"github.com/rs/zerolog/log"
)

// ConnectionHandler runs once we have a list of non-duplicate peers, connecting to those peers and managing their connections
type ConnectionHandler struct {
	// contains all peers which are either active or connecting
	activeConns []*Peer
	// associated torrent
	torrent *Torrent
	// channel for peers to notify connection handler that they've disconnected, allows us to not just run in a ticker
	doneChan chan *Peer

	// logger *log.Logger
}

func newConnHandler(torrent *Torrent) *ConnectionHandler {
	var ch ConnectionHandler
	ch.torrent = torrent
	ch.doneChan = make(chan *Peer)
	// ch.logger = log.New(torrent.logFile, "[Connection Handler] ", log.Ltime|log.Lshortfile)
	//	ch.logger.SetOutput(io.Discard)
	return &ch
}

func (ch *ConnectionHandler) run() {
	defer log.Info().Msg("Finished running")
	//	defer ch.logger.Println("Finished running")

	for {
		ch.torrent.peersMx.Lock()
		peers := ch.torrent.peers
		ch.torrent.peersMx.Unlock()

		var badPeers int

		// try check peer list again in 5 seconds if there are none yet
		if len(peers) == 0 {
			time.Sleep(5)
			continue
		}

		// add peers until we reach max connections

		for _, peer := range peers {
			// we've reached max active connections
			if len(ch.activeConns) >= ch.torrent.maxPeers {
				break
			}
			switch peer.status {
			case Bad:
				badPeers++
				if badPeers == len(peers) {
					return
				}
				// if i == len(peers)-1 && badPeers == len(peers) {
				// 	// all peers are bad
				// 	return
				// }
			case Alive, Connecting:
				continue
			default:
				ch.activeConns = append(ch.activeConns, peer)
				go ch.activeConns[len(ch.activeConns)-1].run(ch.doneChan)
			}
		}
		// log.Info().Msg(fmt.Sprintf("Bad: %d Alive: %d Total: %d\n", badPeers, alivePeers, len(ch.torrent.peers)))
		ch.removeConnection(<-ch.doneChan) // block until someone disconnects
	}
}

// remove peer from the connection slice
func (ch *ConnectionHandler) removeConnection(peer *Peer) {
	//	ch.logger.Printf(" - %s", peer.String())
	peer.disconnect()
	if len(ch.activeConns) == 1 {
		ch.activeConns = []*Peer{}
	} else {
		for i := 0; i < len(ch.activeConns); i++ {
			if ch.activeConns[i].ip == peer.ip {
				ch.activeConns[i] = ch.activeConns[len(ch.activeConns)-1]
				ch.activeConns = ch.activeConns[:len(ch.activeConns)-1]
			}
		}
	}
}

// Prints all alive connections
func (ch *ConnectionHandler) String() string {
	if len(ch.activeConns) == 0 {
		return "No peers connected"
	}
	var s string
	for i := 0; i < len(ch.activeConns); i++ {
		s += ch.activeConns[i].ip + " "
	}
	return s
}
