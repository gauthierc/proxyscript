package main

import (
	"encoding/csv"
	"io"
	"log"
	"net"
	"os"

	"github.com/go-fsnotify/fsnotify"
)

func parseCsvFile(file string) (map[string]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	csvr := csv.NewReader(f)

	filespac := map[string]string{}
	for {
		row, err := csvr.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return filespac, err
		}

		filespac[row[0]] = row[1]
	}
}

func capforIp(rip string, csvmap map[string]string) (string, error) {
	ip, _, err := net.ParseCIDR(rip + "/32")
	if err != nil {
		return "", err
	}
	for key, value := range csvmap {
		_, ipnet, _ := net.ParseCIDR(key)
		if ipnet.Contains(ip) {
			return value, nil
		}
	}
	return "", nil
}

func watchfile(fichiercsv string) error {
	if _, err := os.Stat(fichiercsv); os.IsNotExist(err) {
		log.Fatal("Erreur ", err)
	}
	remoteip := "172.29.143.107"
	datacsv, err := parseCsvFile(fichiercsv)
	if err != nil {
		log.Println("Erreur ", err)
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Erreur ", err)
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	fichiercap, _ := capforIp(remoteip, datacsv)
	log.Println(fichiercap)
	log.Printf("%v\n", datacsv)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				datacsv, _ = parseCsvFile(fichiercsv)
				log.Printf("Modification %v\n", event)
				fichiercap, _ = capforIp(remoteip, datacsv)
				log.Println(fichiercap)
				log.Printf("%v\n", datacsv)
			case err := <-watcher.Events:
				datacsv, _ = parseCsvFile(fichiercsv)
				log.Printf("Erreur  %v\n", err)
				fichiercap, _ = capforIp(remoteip, datacsv)
				log.Println(fichiercap)
				log.Printf("%v\n", datacsv)
			}
		}
	}()
	log.Println("Sortie du gofunc")
	if err := watcher.Add(fichiercsv); err != nil {
		log.Println("Erreur", err)
		return err
	}
	<-done
	return err
}

func main() {
	fichier := "./sys/essai.csv"

	watchfile(fichier)
	log.Println("Sortie du programme")
}
