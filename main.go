package main

import (
	"flag"
	"fmt"
	"gotorrent/models"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"

	tea "github.com/charmbracelet/bubbletea"
)

// var seed bool
var connections int
var debug bool
var file string
var magnet string
var download bool

type tickMsg time.Time

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
	// Open log file (creates new or truncates existing)
	logFile, err := os.Create("gotorrent.log")
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	// Configure zerolog to write to the file
	log.Logger = log.Output(logFile)

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
		if len(os.Args) < 3 {
			fmt.Printf("Provide a file path\n")
			return
		}

		torrentFile, err := models.NewTorrentFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		log.Info().Msgf("Torrent file loaded: %s", torrentFile.Info.Name)
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

	p := tea.NewProgram(model{torrent: torr}, tea.WithAltScreen())

	if download {
		go torr.StartDownload()
	}

	_, err = p.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start TUI")
	}

	// if download {
	// 	torr.StartDownload()
	// }
}

type model struct {
	torrent *models.Torrent
}

func (m model) Init() tea.Cmd {
	return tick()
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case tickMsg:
		return m, tick()
	}

	return m, nil
}

func (m model) View() string {
	s := "Gotorrent\n\n"
	s += fmt.Sprintf("Name: %s\n", m.torrent.GetName())
	s += fmt.Sprintf("Size: %s\n", m.torrent.GetFileSizePretty())
	s += fmt.Sprintf("Total Peers: %d\n", m.torrent.GetNumPeers())
	if m.torrent.GetNumPeers() > 0 {
		good, bad, unknown := m.torrent.GetPeerStats()
		s += fmt.Sprintf(" - %d good\n - %d bad\n - %d unknown\n", good, bad, unknown)
	}
	s += fmt.Sprintf("Progress: %s\n", m.torrent.GetPiecesDownloaded())
	s += "\nPress q to quit.\n"

	return s
}
