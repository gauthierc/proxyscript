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

type couple struct {
	ip      string
	fichier string
}

type fichierCsv struct {
	nom  string
	data map[int]couple
	pac  map[string]*fichierPac
	path string
}

type fichierPac struct {
	nom   string
	data  []byte
	count int
	path  string
}

// newPac : Initialisation d'un fichier .pac
func newPac(nomfic string, path string) (*fichierPac, error) {
	// test existance du fichier
	if _, err := os.Stat(path + nomfic); os.IsNotExist(err) {
		log.Println("Erreur newPac ", err)
		return nil, err
	}
	file := &fichierPac{}
	file.nom = nomfic
	file.path = path
	file.data = make([]byte, 5000)
	file.count = 0
	return file, nil
}

// LoadPacFile : Chargement en mémoire du fichier .pac
func (file *fichierPac) LoadPacFile() error {
	f, err := os.Open(file.path + file.nom)
	if err != nil {
		log.Println("Erreur LoadPacFile ", err)
		return err
	}
	defer f.Close()
	file.count, err = f.Read(file.data)
	return err
}

// UpdatePacFile : Mise à jour en mémoire du fichier .pac
func (file *fichierPac) UpdatePacFile(nom string) error {
	file.nom = nom
	err := file.LoadPacFile()
	if err != nil {
		log.Println("Erreur UpdatePacFile ", err)
		return err
	}
	log.Println("Rechargement du fichier pac", file.path+file.nom)
	return nil

}

// WatchPacFile : Surveillance des modifications du fichier .pac
func (file *fichierPac) WatchPacFile() error {
	err := file.LoadPacFile()
	if err != nil {
		log.Println("Erreur WatchPacFile Load", err)
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
				if event.Name != "" {
					log.Printf("Modification FichierPac %v\n", event)
					done <- true
				}
			}
		}
	}()
	log.Println("WatchPacFile ", file.path+file.nom)
	err = watcher.Add(file.path + file.nom)
	if err != nil {
		log.Println("Erreur WatchPacFile Add", err)
		return err
	}
	<-done
	time.Sleep(time.Millisecond * 100)
	return err
}

// newCsv : Initialisation d'un fichier .csv
func newCsv(nomfic string, path string) *fichierCsv {
	if _, err := os.Stat(nomfic); os.IsNotExist(err) {
		log.Fatal("Erreur newCsv ", err)
	}
	file := &fichierCsv{}
	file.nom = nomfic
	file.path = path
	file.data = make(map[int]couple)
	file.pac = make(map[string]*fichierPac)
	file.LoadCsvFile()
	return file
}

// LoadCsvFile : Chargement en mémoire du fichier .csv
func (file *fichierCsv) LoadCsvFile() error {
	f, err := os.Open(file.nom)
	if err != nil {
		log.Println("Erreur LoadCsvFile ", err)
		return err
	}
	defer f.Close()
	// Suppression des entrées précédentes si elles existent
	for key := range file.data {
		delete(file.data, key)
	}
	csvr := csv.NewReader(f)
	i := 0
	for {
		row, err := csvr.Read()
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return err
		}
		if row[1] != "" {
			file.data[i] = couple{row[0], row[1]}
			i = i + 1
			// Je verifie que le fichier n'est pas déjà en mémoire
			if file.pac[row[1]] == nil {
				file.pac[row[1]], err = newPac(row[1], file.path)
				if err != nil {
					log.Println("Erreur LoadCsvFile newPac ", err)
				} else {
					file.pac[row[1]].LoadPacFile()
					go file.pac[row[1]].WatchPacFile()
				}
			}
		}
	}
}

// Mise à jour en mémoire du fichier .csv
func (file *fichierCsv) UpdateCsvFile(nom string) error {
	file.nom = nom
	err := file.LoadCsvFile()
	if err != nil {
		return err
	}
	log.Println("Rechargement du fichier", file.nom)
	return nil
}

// Retourne le nom du fichier pac en fonction de l'ip
func (file *fichierCsv) PacforIP(rip string) (string, error) {
	ip, _, err := net.ParseCIDR(rip + "/32")
	if err != nil {
		return "", err
	}

	for i := 0; i < len(file.data); i++ {
		_, ipnet, _ := net.ParseCIDR(file.data[i].ip)
		if ipnet.Contains(ip) {
			return file.data[i].fichier, nil
		}
	}
	return "", nil
}

// Surveillance des modifications du fichier .csv
func (file *fichierCsv) WatchCsvFile(reloadconf chan bool) error {
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
	//	log.Printf("%v\n", file.data)

	go func() {
		for {
			select {
			case <-reloadconf:
				log.Println("Modification du fichier de config")
				done <- true
			case event := <-watcher.Events:
				if event.Name != "" {
					log.Printf("Modification %v\n", event)
					done <- true
				}
			}
		}
	}()
	err = watcher.Add(file.nom)
	if err != nil {
		log.Println("Erreur", err)
		return err
	}
	<-done
	time.Sleep(time.Millisecond * 100)
	return err
}

// Retourne le fichier pac en fonction de l'ip
func (file *fichierCsv) handlerRetPac(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	var ip string
	forwardfor := html.EscapeString(r.Header.Get("X-Forwarded-For"))
	if forwardfor != "" {
		ip = forwardfor
	} else {
		ip = html.EscapeString(strings.Split(r.RemoteAddr, ":")[0])
	}
	nomfic, err := file.PacforIP(ip)
	if err != nil {
		log.Println("Aucun fichier pour l'ip ", ip)
		http.Error(w, "Fichier pac inexistant", http.StatusNotFound)
		return
	}
	if file.pac[nomfic] != nil {
		fmt.Fprintf(w, "// fichier: %s\n", nomfic)
		fmt.Fprintf(w, "%s\n", file.pac[nomfic].data[:file.pac[nomfic].count])
		log.Printf("%s - GET \"%s\" %s %s IpSource:%s\n", ip, r.URL.Path, nomfic, r.UserAgent(), r.RemoteAddr)
	} else {
		http.Error(w, "Fichier pac inexistant", http.StatusNotFound)
		log.Printf("%s - GET \"%s\" --PAS DE FICHIER pac-- %s IpSource:%s\n", ip, r.URL.Path, r.UserAgent(), r.RemoteAddr)
	}
}

func main() {
	log.Println("Lancement de proxyscript")
	viper.SetConfigType("toml")
	viper.SetConfigName("proxyscript")
	viper.AddConfigPath("./config/")
	viper.AddConfigPath("/home/proxyscript/config/")
	viper.Set("Verbose", true)
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostport := fmt.Sprintf("%s:%s", viper.GetString("listen.host"), viper.GetString("listen.port"))
	reload := make(chan bool)
	file := newCsv(viper.GetString("data.corres"), viper.GetString("data.reppac"))
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

	http.HandleFunc("/", file.handlerRetPac)
	go func() {
		for {
			file.WatchCsvFile(reload)
		}
	}()
	log.Println("Ecoute du serveur http de proxyscript")
	log.Fatal(http.ListenAndServe(hostport, nil))
	log.Println("Sortie du programme")
}
