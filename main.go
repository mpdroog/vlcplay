package main

import (
	"bufio"
	"flag"
	"fmt"
	vlc "github.com/adrg/libvlc-go/v3"
	"errors"
	"io/fs"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func remove(s []string, i int) []string {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}

func main() {
	verbose := false
	wait := false
	path := ""
	flag.StringVar(&path, "p", "/Volumes/MPExtern/Music", "Music folder")
	flag.BoolVar(&verbose, "v", false, "Increase verbosity")
	flag.BoolVar(&wait, "r", true, "Retry every 5sec if path is available")
	flag.Parse()

	// Initialize libVLC. Additional command line arguments can be passed in
	// to libVLC by specifying them in the Init function.
	if err := vlc.Init("--no-video", "--quiet"); err != nil {
		log.Fatal(err)
	}
	defer vlc.Release()

	// Create a new list player.
	player, err := vlc.NewListPlayer()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		player.Stop()
		player.Release()
	}()

	// Create a new media list.
	list, err := vlc.NewMediaList()
	if err != nil {
		log.Fatal(err)
	}
	defer list.Release()

	if wait {
		for {
			_, e := os.Stat(path)
			if errors.Is(e, os.ErrNotExist) {
				fmt.Printf("file[%s] not ready..\n", path)
				time.Sleep(time.Second * 5)
				continue
			}
			if e != nil {
				log.Fatal(e)
			}

			// file exists
			break
		}
	}

	songs := []string{}
	if e := filepath.WalkDir(path, func(s string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(s, "._") {
			// ignore ._ files (meta?)
			return nil
		}
		if !strings.HasSuffix(s, ".mp4") && !strings.HasSuffix(s, ".webm") {
			// Only add playable music/video files
			return nil
		}

		if verbose {
			fmt.Printf("[sync] Read %s\n", s)
		}
		songs = append(songs, s)
		return nil

	}); e != nil {
		log.Fatal(e)
	}
	if verbose {
		fmt.Printf("song.count=%d\n", len(songs))
	}

	// Throw in some random as ranging over map is not random enough
	songsNext := make(map[string]struct{})
	for i := len(songs); i > 0; i-- {
		// pick random
		k := rand.Intn(len(songs))
		song := songs[k]

		songsNext[song] = struct{}{}
		if verbose {
			fmt.Printf("[random] %s\n", song)
		}
		songs = remove(songs, k)
	}

	// Add in second range by 'abusing' random behaviour of map
	for song, _ := range songsNext {
		if err := list.AddMediaFromPath(song); err != nil {
			log.Fatal(err)
		}
	}

	// Set player media list.
	if err = player.SetMediaList(list); err != nil {
		log.Fatal(err)
	}

	// Retrieve player event manager.
	manager, err := player.EventManager()
	if err != nil {
		log.Fatal(err)
	}

	// remember currently played song (for deleting with 'r')
	lastFile := ""

	// Register the media end reached event with the event manager.
	quit := make(chan struct{})
	eventCallback := func(event vlc.Event, userData interface{}) {
		switch event {
		case vlc.MediaListPlayerPlayed:
			close(quit)
		case vlc.MediaListPlayerNextItemSet:
			// Retrieve underlying player.
			p, err := player.Player()
			if err != nil {
				log.Println(err)
				break
			}

			// Retrieve currently playing media.
			media, err := p.Media()
			if err != nil {
				log.Println(err)
				break
			}

			song, e := media.Meta(vlc.MediaTitle)
			if e != nil {
				log.Println(e)
				break
			}

			/* Get media location.*/
			location, err := media.Location()
			if err != nil {
				log.Println(err)
				break
			}
			lastFile = location

			title := song
			title = strings.ReplaceAll(title, ".mp4", "")
			title = strings.ReplaceAll(title, ".webm", "")

			fmt.Printf("\033]1;" + title + " \007")
			log.Println("Now playing:", title)

		default:
			fmt.Printf("Event(%s) data=%+v\n", event, userData)
		}
	}

	var eventIDs []vlc.EventID
	for _, event := range []vlc.Event{vlc.MediaListPlayerPlayed, vlc.MediaListPlayerNextItemSet} {
		eventID, err := manager.Attach(event, eventCallback, nil)
		if err != nil {
			log.Fatal(err)
		}
		eventIDs = append(eventIDs, eventID)
	}
	defer manager.Detach(eventIDs...)

	if err := player.SetPlaybackMode(vlc.Loop); err != nil {
		log.Fatal(err)
	}

	// Start playing the media list.
	if err = player.Play(); err != nil {
		log.Fatal(err)
	}

	go func() {
		cmds := map[string]string{"n": "Next song", "p": "previous song", "t": "Pause or resume", "r": "Remove song"}
		for {
			reader := bufio.NewReader(os.Stdin)
			line, e := reader.ReadString('\n')
			if e != nil {
				log.Fatal(err)
			}
			//fmt.Println("Line=" + line)
			cmd := strings.TrimSpace(line)
			if cmd == "n" {
				if err := player.PlayNext(); err != nil {
					log.Fatal(err)
				}
			} else if cmd == "p" {
				if err := player.PlayPrevious(); err != nil {
					log.Fatal(err)
				}
			} else if cmd == "t" {
				if err := player.TogglePause(); err != nil {
					log.Fatal(err)
				}
				fmt.Printf("Toggle pause\n")
			} else if cmd == "r" {
				fname := lastFile
				if err := player.PlayNext(); err != nil {
					log.Fatal(err)
				}
				if e := os.Remove(fname); e != nil {
					log.Fatal(e)
				}
				fmt.Printf("Removed %s\n", fname)
			} else {
				fmt.Printf("Unsupported cmd, possible=\n")
				fmt.Printf("%+v\n", cmds)
			}
		}

	}()

	<-quit
}
