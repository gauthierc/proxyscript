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
	cap  map[string]*Fichiercap
	path string
}

type Fichiercap struct {
	nom   string
	data  []byte
	count int
	path  string
}

// Initialisation d'un fichier .cap
func NewCap(nomfic string, path string) (*Fichiercap, error) {
	// test existance du fichier
	if _, err := os.Stat(path + nomfic); os.IsNotExist(err) {
		log.Println("Erreur NewCap ", err)
		return nil, err
	}
	file := &Fichiercap{}
	file.nom = nomfic
	file.path = path
	file.data = make([]byte, 100)
	file.count = 0
	return file, nil
}

// Chargement en mémoire du fichier .cap
func (file *Fichiercap) LoadCapFile() error {
	f, err := os.Open(file.path + file.nom)
	if err != nil {
		log.Println("Erreur LoadCapFile ", err)
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
		log.Println("Erreur UpdateCapFile ", err)
		return err
	}
	log.Println("Rechargement du fichier cap", file.path+file.nom)
	return nil

}

// Surveillance des modification du fichier .cap
func (file *Fichiercap) WatchCapFile() error {
	err := file.LoadCapFile()
	if err != nil {
		log.Println("Erreur WatchCapFile Load", err)
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
			case event := <-watcher.Events:
				log.Printf("Modification FichierCap %v\n", event)
				done <- true
			case err := <-watcher.Events:
				log.Printf("%v\n", err)
				done <- true
			}
		}
	}()
	log.Println("WatchCapFile ", file.path+file.nom)
	if err := watcher.Add(file.path + file.nom); err != nil {
		log.Println("Erreur WatchCapFile Add", err)
		return err
	}
	<-done
	time.Sleep(time.Millisecond * 100)
	return err
}

// Initialisation d'un fichier .csv
func NewCsv(nomfic string, path string) *Fichiercsv {
	if _, err := os.Stat(nomfic); os.IsNotExist(err) {
		log.Fatal("Erreur NewCsv ", err)
	}
	file := &Fichiercsv{}
	file.nom = nomfic
	file.path = path
	file.data = make(map[string]string)
	file.cap = make(map[string]*Fichiercap)
	file.LoadCsvFile()
	return file
}

// Chargement en mémoire du fichier .csv
func (file *Fichiercsv) LoadCsvFile() error {
	f, err := os.Open(file.nom)
	if err != nil {
		log.Println("Erreur LoadCsvFile ", err)
		return err
	}
	defer f.Close()
	// Suppression des entrées précédentes si elles existent
	for key, _ := range file.data {
		delete(file.data, key)
	}
	for key, _ := range file.cap {
		delete(file.cap, key)
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
		if row[1] != "" {
			file.data[row[0]] = row[1]
			if file.cap[row[1]] == nil {
				file.cap[row[1]], err = NewCap(row[1], file.path)
				if err != nil {
					log.Println("Erreur LoadCsvFile NewCap ", err)
					return err
				} else {
					file.cap[row[1]].LoadCapFile()
					go file.cap[row[1]].WatchCapFile()
				}
			}
		}
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
	if err != nil {
		log.Fatal(err)
	}
	hostport := fmt.Sprintf("%s:%s", viper.GetString("listen.host"), viper.GetString("listen.port"))
	reload := make(chan bool)
	file := NewCsv(viper.GetString("data.corres"), viper.GetString("data.repcap"))
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
