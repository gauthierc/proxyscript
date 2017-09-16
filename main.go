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
	cap  map[string]Fichiercap
}

type Fichiercap struct {
	nom   string
	data  []byte
	count int
}

// Initialisation d'un fichier .cap
func NewCap(nomfic string) (*Fichiercap, error) {
	// test existance du fichier
	if _, err := os.Stat(nomfic); os.IsNotExist(err) {
		log.Println("Erreur ", err)
		return nil, err
	}
	file := &Fichiercap{}
	file.nom = nomfic
	file.data = make([]byte, 100)
	file.count = 0
	return file, nil
}

// Chargement en mémoire du fichier .cap
func (file *Fichiercap) LoadCapFile() error {
	f, err := os.Open(file.nom)
	if err != nil {
		return err
	}
	defer f.Close()
	file.count, err = f.Read(file.data)
	if err != nil {
		return err
	}
	return nil
}

// Mise à jour en mémoire du fichier .cap
func (file *Fichiercap) UpdateCapFile(nom string) error {
	file.nom = nom
	err := file.LoadCapFile()
	if err != nil {
		return err
	}
	log.Println("Rechargement du fichier cap", file.nom)
	return nil

}

// Surveillance des modification du fichier .cap
func (file *Fichiercap) WatchfileCap(reloadconf chan bool) error {
	err := file.LoadCapFile()
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

	go func() {
		for {
			select {
			case <-reloadconf:
				log.Println("Modification du fichier de config")
				done <- true
				break
			case event := <-watcher.Events:
				log.Printf("Modification FichierCap %v\n", event)
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

// Initialisation d'un fichier .csv
func NewCsv(nomfic string) *Fichiercsv {
	if _, err := os.Stat(nomfic); os.IsNotExist(err) {
		log.Fatal("Erreur ", err)
	}
	file := &Fichiercsv{}
	file.nom = nomfic
	file.data = make(map[string]string)
	file.LoadCsvFile()
	return file
}

// Chargement en mémoire du fichier .csv
func (file *Fichiercsv) LoadCsvFile() error {
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

// Mise à jour en mémoire du fichier .csv
func (file *Fichiercsv) UpdateCsvFile(nom string) error {
	file.nom = nom
	err := file.LoadCsvFile()
	if err != nil {
		return err
	}
	log.Println("Rechargement du fichier", file.nom)
	return nil
}

// Retourne le nom du fichier cap en fonction de l'ip
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

func (file *Fichiercsv) WatchCsvFile(reloadconf chan bool) error {
	err := file.LoadCsvFile()
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
	file := NewCsv(viper.GetString("data.corres"))
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
				file.UpdateCsvFile(fichier)
				reload <- true
			}
		}
	})

	viper.WatchConfig()

	http.HandleFunc("/", handlerRetCap)
	go func() {
		for {
			file.WatchCsvFile(reload)
		}
	}()
	log.Fatal(http.ListenAndServe(hostport, nil))
	log.Println("Sortie du programme")
}
