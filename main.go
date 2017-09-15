package main

import (
	"encoding/csv"
	"fmt"
	"html"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
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

func watchfile(fichiercsv string, reloadconf chan bool) error {
	if _, err := os.Stat(fichiercsv); os.IsNotExist(err) {
		log.Fatal("Erreur ", err)
	}
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
	log.Printf("%v\n", datacsv)

	go func() {
		for {
			select {
			case <-reloadconf:
				log.Println("Modification du fichier de config")
				done <- true
				break
			case event := <-watcher.Events:
				log.Printf("Modification %v\n", event)
				done <- true
			case err := <-watcher.Events:
				log.Printf("%v\n", err)
				done <- true
			}
		}
	}()
	if err := watcher.Add(fichiercsv); err != nil {
		log.Println("Erreur", err)
		return err
	}
	<-done
	time.Sleep(time.Millisecond * 100)
	return err
}

func handlerRetCap(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q\n", html.EscapeString(r.URL.Path))
	ip := ""
	forwardfor := html.EscapeString(r.Header.Get("X-Forwarded-For"))
	if forwardfor != "" {
		ip = forwardfor
	} else {
		ip = html.EscapeString(strings.Split(r.RemoteAddr, ":")[0])
	}
	fmt.Fprintf(w, "ip: %q\n", ip)
}

func main() {
	viper.SetConfigType("toml")
	viper.SetConfigName("proxyscript")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath(".")
	viper.Set("Verbose", true)
	err := viper.ReadInConfig()
	fichier := viper.GetString("data.corres")
	hostport := fmt.Sprintf("%s:%s", viper.GetString("listen.host"), viper.GetString("listen.port"))
	reload := make(chan bool)

	if err != nil {
		log.Fatal(err)
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		err := viper.ReadInConfig()
		if err != nil {
			log.Println(err)
		} else {
			fichiernew := viper.GetString("data.corres")
			if fichiernew != fichier {
				log.Println("Fichier de config a changé.")
				fichier = fichiernew
				reload <- true
			}
		}
	})

	viper.WatchConfig()

	http.HandleFunc("/", handlerRetCap)
	go func() {
		for {
			watchfile(fichier, reload)
			log.Println("Rechargement du fichier", fichier)
		}
	}()
	log.Fatal(http.ListenAndServe(hostport, nil))
	log.Println("Sortie du programme")
}
