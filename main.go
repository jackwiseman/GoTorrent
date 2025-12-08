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
var download bool

func init() {
	// flag.BoolVar(&seed, "seed", false, "continue seeding after download")
	flag.BoolVar(&download, "download", false, "enable downloading")
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

	config := models.NewConfig(connections)

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

		fmt.Println(torrentFile.String())
		torrentFile.PrintFileInfo()

		torr = models.NewTorrentFromFile(torrentFile, config)
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

		torr = models.NewTorrentFromMagnet(magnetLink, *config)
	}

	if len(torr.GetTrackers()) == 0 {
		fmt.Println("No trackers found in the torrent file or magnet link, DHT is not currently supported, exiting.")
		return
	}

	if download {
		torr.StartDownload()
	}
}
