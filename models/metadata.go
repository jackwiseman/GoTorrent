package models

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"gotorrent/utils"
	"math/rand"
	"os"
	"strconv"

	"github.com/jackpal/bencode-go"
	"github.com/rs/zerolog/log"
)

// Metadata stores the torrent's metadata
// This is the same as the info dictionary in the metainfo file (.torrent file)
type Metadata struct {
	Name     string `bencode:"name"`
	PieceLen int    `bencode:"piece length"`
	Pieces   string `bencode:"pieces"`
	// contains one of the following, where 'length' means there is one file, and 'files' means there are multiple, only single file downloads will be allowed for the moment
	Length int            `bencode:"length,omitempty"`
	Files  []MetadataFile `bencode:"files,omitempty"`
}

// MetadataFile is a subset of Metadata for use in bencoding, since a torrent can contain multiple files
type MetadataFile struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

func (md *Metadata) String() string {
	s := "Name: " + md.Name + "\nPiece length: " + strconv.Itoa(md.PieceLen) + "\nLength: " + strconv.Itoa(md.Length)
	return s
}

func (metadata *Metadata) marshal() ([]byte, error) {
	var b bytes.Buffer
	err := bencode.Marshal(&b, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	return b.Bytes(), nil
}

func (torrent *Torrent) verifyMetadata() error {
	// we reverse the bencode, and then hash the pieces, comparing with the infohash to ensure we have valid metadata in the struct
	b, err := torrent.metadata.marshal()
	if err != nil {
		return err
	}

	hasher := sha1.New()
	hasher.Write(b)
	hash := hasher.Sum(nil)

	log.Info().Msg(fmt.Sprintf("Metadata struct reports metadata length of %d bytes", len(b)))

	log.Info().Msg(fmt.Sprintf("Computed hash: %x", hash))
	log.Info().Msg(fmt.Sprintf("Expected hash: %x", torrent.infoHash[:]))

	if !bytes.Equal(hash, torrent.infoHash[:]) {
		return fmt.Errorf("metadata verification failed: hash mismatch")
	}
	return nil
}

func (torrent *Torrent) getRandMetadataPiece() (int, error) {
	hasAllMetadata, err := torrent.hasAllMetadata()
	if err != nil {
		return -1, err
	}
	if hasAllMetadata {
		return -1, nil
	}

	for {
		testPiece := rand.Intn(torrent.numMetadataPieces())
		hasPiece, err := torrent.hasMetadataPiece(testPiece)
		if err != nil {
			return -1, err
		}
		if !hasPiece {
			return testPiece, nil
		}
	}
}

func (torrent *Torrent) hasMetadataPiece(pieceNum int) (bool, error) {
	if len(torrent.metadataPieces) == 0 {
		return false, nil
	}
	return utils.BitIsSet(torrent.metadataPieces, pieceNum)
}

func (torrent *Torrent) setMetadataPiece(pieceNum int, metadataPiece []byte) error {
	hasAll, err := torrent.hasAllMetadata()
	if err != nil {
		return err
	}
	if hasAll {
		return nil
	}
	// insert into raw byte array
	startIndex := pieceNum * BlockLen
	if startIndex > len(torrent.metadataRaw) {
		return errors.New("metadata piece is out of bounds")
	}

	temp := torrent.metadataRaw[pieceNum*BlockLen+len(metadataPiece):]
	torrent.metadataRaw = append(torrent.metadataRaw[0:pieceNum*BlockLen], metadataPiece...)
	torrent.metadataRaw = append(torrent.metadataRaw, temp...)

	// set as "have"
	utils.SetBit(&torrent.metadataPieces, pieceNum)

	return nil
}

func (torrent *Torrent) buildMetadataFile() error {
	err := os.WriteFile("metadata.torrent", torrent.metadataRaw, 0644)
	fmt.Println("Received metadata")
	if err != nil {
		panic(err)
	}
	return nil
}
