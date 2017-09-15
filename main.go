package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/go-fsnotify/fsnotify"
)

func parseLocation(file string) (map[string]*Point, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	csvr := csv.NewReader(f)

	locations := map[string]*Point{}
	for {
		row, err := csvr.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return locations, err
		}

		p := &Point{}
		if p.lat, err = strconv.ParseFloat(row[1], 64); err != nil {
			return nil, err
		}
		if p.lon, err = strconv.ParseFloat(row[2], 64); err != nil {
			return nil, err
		}
		locations[row[0]] = p
	}
}

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Erreur ", err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				fmt.Printf("Event ! %v\n", event)
			case err := <-watcher.Events:
				fmt.Printf("Erreur ! %v\n", err)
			}
		}
	}()
	if err := watcher.Add("./sys/corres.csv"); err != nil {
		fmt.Println("Erreur", err)
	}
	<-done
}
