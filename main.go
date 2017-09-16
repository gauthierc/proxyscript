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

type Fichiercsv struct {
	nom  string
	data map[string]string
}

type Fichiercap struct {
	nom  string
	data string
}

func New(nomfic string) *Fichiercsv {
	if _, err := os.Stat(nomfic); os.IsNotExist(err) {
		log.Fatal("Erreur ", err)
	}
	file := &Fichiercsv{}
	file.nom = nomfic
	file.data = make(map[string]string)
	file.ParseCsvFile()
	return file
}

func (file *Fichiercsv) ParseCsvFile() error {
	f, err := os.Open(file.nom)
	if err != nil {
		return err
	}
	defer f.Close()
	for key, _ := range file.data {
		delete(file.data, key)
	}
	csvr := csv.NewReader(f)
	for {
		row, err := csvr.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			}

			return err
		}

		file.data[row[0]] = row[1]
	}
	return nil
}

func (file *Fichiercsv) Update(nom string) error {
	file.nom = nom
	err := file.ParseCsvFile()
	if err != nil {
		return err
	}
	log.Println("Rechargement du fichier", file.nom)
	return nil
}

func (file *Fichiercsv) CapforIp(rip string) (string, error) {
	ip, _, err := net.ParseCIDR(rip + "/32")
	if err != nil {
		return "", err
	}
	for key, value := range file.data {
		_, ipnet, _ := net.ParseCIDR(key)
		if ipnet.Contains(ip) {
			return value, nil
		}
	}
	return "", nil
}

func (file *Fichiercsv) Watchfile(reloadconf chan bool) error {
	err := file.ParseCsvFile()
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
	log.Printf("%v\n", file.data)

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
	if err := watcher.Add(file.nom); err != nil {
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
	hostport := fmt.Sprintf("%s:%s", viper.GetString("listen.host"), viper.GetString("listen.port"))
	reload := make(chan bool)
	file := New(viper.GetString("data.corres"))
	if err != nil {
		log.Fatal(err)
	}
	viper.OnConfigChange(func(e fsnotify.Event) {
		err := viper.ReadInConfig()
		if err != nil {
			log.Println(err)
		} else {
			fichier := viper.GetString("data.corres")
			if fichier != file.nom {
				log.Println("Fichier de config a changé.")
				file.Update(fichier)
				reload <- true
			}
		}
	})

	viper.WatchConfig()

	http.HandleFunc("/", handlerRetCap)
	go func() {
		for {
			file.Watchfile(reload)
		}
	}()
	log.Fatal(http.ListenAndServe(hostport, nil))
	log.Println("Sortie du programme")
}
