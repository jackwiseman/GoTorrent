package models

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	bencode "github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce     string `bencode:"announce"`
	Comment      string `bencode:"comment,omitempty"`
	CreatedBy    string `bencode:"created by,omitempty"`
	CreationDate int    `bencode:"creation date,omitempty"`
	Info         struct {
		Length      int    `bencode:"length"`
		Name        string `bencode:"name"`         // suggested file name
		PieceLength int    `bencode:"piece length"` // length of each piece in bytes (all are the same size except possibly the last)
		Pieces      string `bencode:"pieces"`       // this is a byte array of SHA1 hashes of the pieces
		Files       []struct {
			Length int      `bencode:"length"`
			Path   []string `bencode:"path"`
		} `bencode:"files,omitempty"`
	} `bencode:"info"`
	Data string // raw bytes of the torrent file as a string
}

// create a new torrent file given a file path
func NewTorrentFile(file string) (*TorrentFile, error) {
	fmt.Println(file)

	// Read the entire file into memory
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	tf := &TorrentFile{
		Data: string(data), // store the raw bytes as a string
	}

	// Unmarshal from the bytes
	err = bencode.Unmarshal(bytes.NewReader(data), tf)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal torrent file: %w", err)
	}

	// fmt.Println(tf.Data)
	fmt.Println(tf.GetInfoHash())

	return tf, nil
}

func (tf *TorrentFile) GetInfoHash() string {
	// the info hash is the SHA1 hash of the bencoded info dictionary
	// Find the "info" key in the bencode data
	index := strings.Index(tf.Data, "4:info")
	if index == -1 {
		return ""
	}

	// Move past "4:info" to the start of the info dictionary
	start := index + 6 // len("4:info")

	// Parse through the bencode to find where the dictionary ends
	end := findBencodeEnd(tf.Data, start)
	if end == -1 {
		return ""
	}

	// Extract just the info dictionary
	infoDict := tf.Data[start:end]

	// Hash it
	hasher := sha1.New()
	hasher.Write([]byte(infoDict))
	hash := hasher.Sum(nil)

	return fmt.Sprintf("%x", hash)
}

// findBencodeEnd finds the end position of a bencode value starting at pos
func findBencodeEnd(data string, pos int) int {
	if pos >= len(data) {
		return -1
	}

	switch data[pos] {
	case 'd', 'l': // dictionary or list
		pos++ // skip 'd' or 'l'
		for pos < len(data) && data[pos] != 'e' {
			// Parse key-value pairs (for dict) or values (for list)
			end := findBencodeEnd(data, pos)
			if end == -1 {
				return -1
			}
			pos = end
		}
		if pos >= len(data) {
			return -1
		}
		return pos + 1 // include the 'e'

	case 'i': // integer
		pos++ // skip 'i'
		end := strings.IndexByte(data[pos:], 'e')
		if end == -1 {
			return -1
		}
		return pos + end + 1

	default: // string (format: <length>:<data>)
		// Read the length
		colonPos := strings.IndexByte(data[pos:], ':')
		if colonPos == -1 {
			return -1
		}

		var length int
		_, err := fmt.Sscanf(data[pos:pos+colonPos], "%d", &length)
		if err != nil {
			return -1
		}

		// Skip past length, colon, and string data
		return pos + colonPos + 1 + length
	}
}

func (tf *TorrentFile) String() string {
	return fmt.Sprintf("TorrentFile(Name: %s\nAnnounce: %s\nComment: %s\nCreated By: %s\nCreation Date: %s\nLength: %d\nPieceLength: %d\n", tf.Info.Name, tf.Announce, tf.Comment, tf.CreatedBy, fmt.Sprint(tf.CreationDate), tf.Info.Length, tf.Info.PieceLength)
}

func (tf *TorrentFile) PrintFileInfo() {
	// if length is present and there is no key 'files', it's a single file torrent
	if len(tf.Info.Files) == 0 {
		fmt.Printf("Single file torrent: %s (%d bytes)\n", tf.Info.Name, tf.Info.Length)
	} else {
		fmt.Printf("Multi-file torrent: %s\n", tf.Info.Name)
		for _, file := range tf.Info.Files {
			path := strings.Join(file.Path, "/")
			fmt.Printf(" - %s (%d bytes)\n", path, file.Length)
		}
	}
}
