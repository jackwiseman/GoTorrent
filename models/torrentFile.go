package models

import (
	"fmt"
	"os"

	bencode "github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce     string `bencode:"announce"`
	Comment      string `bencode:"comment,omitempty"`
	CreatedBy    string `bencode:"created by,omitempty"`
	CreationDate int    `bencode:"creation date,omitempty"`
	Info         struct {
		Length      int    `bencode:"length"`
		Name        string `bencode:"name"`
		PieceLength int    `bencode:"piece length"`
		Pieces      string `bencode:"pieces"` // this is a byte array of SHA1 hashes of the pieces
	} `bencode:"info"`
}

// create a new torrent file given a file path
func NewTorrentFile(file string) (*TorrentFile, error) {
	fmt.Println(file)

	b, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	defer b.Close()

	tf := &TorrentFile{}

	err = bencode.Unmarshal(b, tf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal torrent file: %w", err)
	}

	return tf, nil
}
