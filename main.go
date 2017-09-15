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

	"github.com/go-fsnotify/fsnotify"
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

func retCap(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q\n", html.EscapeString(r.URL.Path))
	fmt.Fprintf(w, "ip: %q\n", html.EscapeString(r.RemoteAddr))
	fmt.Fprintf(w, "forward: %q\n", html.EscapeString(r.Header.Get("X-Forwarded-For")))
}

func main() {
	viper.SetConfigType("toml")
	viper.SetConfigName("proxyscript")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath(".")
	viper.Set("Verbose", true)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	//	viper.WatchConfig()
	//	viper.OnConfigChange(func(in fsnotify.Event) {
	//		log.Println("Le fichier de configuration a changé :")
	//	})

	fichier := viper.GetString("data.corres")
	hostport := fmt.Sprintf("%s:%s", viper.GetString("listen.host"), viper.GetString("listen.port"))
	http.HandleFunc("/", retCap)
	go watchfile(fichier)
	log.Fatal(http.ListenAndServe(hostport, nil))
	log.Println("Sortie du programme")
}
