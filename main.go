package main

import (
	"bufio"
	"fmt"
	vlc "github.com/adrg/libvlc-go/v3"
	"github.com/gen2brain/beeep"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
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

	songs := make(map[string]struct{})
	if e := filepath.WalkDir("/Volumes/MPExtern/Music", func(s string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(s, ".mp4") {
			return nil
		}

		//fmt.Printf("Read %s\n", s)
		songs[s] = struct{}{}
		return nil

	}); e != nil {
		log.Fatal(e)
	}

	// Add in second range by 'abusing' random behaviour of map
	for song, _ := range songs {
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

			/* Get media location.
			location, err := media.Location()
			if err != nil {
				log.Println(err)
				break
			}*/
			log.Println("Next up:", song)
			if err := beeep.Notify("Next up", song, "assets/information.png"); err != nil {
				log.Fatal(err)
			}

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
		cmds := map[string]string{"n": "Next song", "p": "previous song", "t": "Pause or resume"}
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
			} else {
				fmt.Printf("Unsupported cmd, possible=\n")
				fmt.Printf("%+v\n", cmds)
			}
		}

	}()

	<-quit
}
