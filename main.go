package main

import (
	"flag"
	"fmt"
	"gotorrent/models"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

// var seed bool
var connections int
var debug bool
var file string
var magnet string

func init() {
	// flag.BoolVar(&seed, "seed", false, "continue seeding after download")
	flag.StringVar(&file, "file", "", "path to the .torrent file")
	flag.StringVar(&magnet, "magnet", "", "magnet link to download")
	flag.IntVar(&connections, "connections", 50, "number of connections to use")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.Parse()
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	if magnet != "" && file != "" {
		fmt.Println("Please provide either a file or a magnet link, not both.")
		return
	}

	if magnet == "" && file == "" {
		fmt.Println("Please provide a file path or a magnet link.")
		return
	}

	var torr *models.Torrent

	if file != "" {
		fmt.Println(len(os.Args))
		if len(os.Args) < 3 {
			fmt.Printf("Provide a file path\n")
			return
		}

		torrentFile, err := models.NewTorrentFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		torr = models.NewTorrentFromFile(torrentFile, connections)
	}

	if magnet != "" {
		if len(os.Args) < 2 {
			fmt.Printf("Provide a magnet link\n")
			return
		}

		magnetLink, err := models.NewMagnet(os.Args[2])
		if err != nil {
			panic(err)
		}

		torr = models.NewTorrentFromMagnet(magnetLink, connections)
	}

	torr.StartDownload()
}
