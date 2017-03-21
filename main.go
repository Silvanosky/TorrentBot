package main

import (
	"log"
	"encoding/json"
	"os"
	"io/ioutil"
	"github.com/tubbebubbe/transmission"
	t411 "github.com/Silvanosky/t411-client/t411client"
	"bufio"
	"time"
	"os/signal"
	"strconv"
)

type Setting struct {
	TransmissionUser string `json:"TransmissionUser"`
	TransmissionPass string `json:"TransmissionPass"`
	TransmissionURL  string `json:"TransmissionUrl"`
	T411User         string `json:"T411User"`
	T411Pass         string `json:"T411Pass"`
}

type Data struct {
	TransmissionID 	 string `json:"TransmissionID"`
	T411ID 		 string `json:"T411ID"`
}

type ListData []Data

func readJson(data interface{}, filename string) (error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer jsonFile.Close()
	jsonData, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(jsonData, data); err != nil {
		log.Printf("Error decoding file: %s for: %v", filename, err)
		return err
	}
	return nil
}

func writeJson(data interface{}, filename string) (error) {
	rankingsJson, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error decoding file: %s for: %v", filename, err)
		return err
	}

	err = ioutil.WriteFile(filename, rankingsJson, 0644)
	if err != nil {
		return err
	}

	return nil
}

func routineCheckLeave(done *bool) {
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
	*done = true
	log.Print("Will shutdown on the next loop !")
}

func main() {

	var settings Setting
	err := readJson(&settings, "config.json")
	if err != nil {
		log.Fatalln("Error JSON:", err)
	}

	t411Map := make(map[string]string)

	data := make(ListData, 0)
	readJson(&data, "data.json")

	for _, d := range data{
		t411Map[d.T411ID] = d.TransmissionID
	}

	log.Println("Adding shutdown hook..")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func(){
		<-c
		log.Println("Saving data..")

		data := make(ListData, 0)
		for k, v := range t411Map{
			data = append(data, Data{v, k})
		}
		writeJson(data, "data.json")
		os.Exit(1)
	}()


	client := transmission.New(settings.TransmissionURL, settings.TransmissionUser, settings.TransmissionPass)
	log.Println("Logged transmission")

	var t411Client *t411.T411

	for t411Client == nil {
		t411Client, err = t411.NewT411ClientWithToken("", settings.T411User, settings.T411Pass, "")
		if err != nil {
			log.Println(err.Error())
		}
	}
	t411Client.KeepRatio(false)
	log.Println("Logged t411")

	done := false
	go routineCheckLeave(&done)

	log.Print("Updating transmission torrents")
	_, err = client.GetTorrents()
	if err != nil {
		log.Panic(err)
	}

	for !done {
		log.Print("Fetching torrents of the day !")
		dayTors, err := t411Client.TorrentsOfToday()
		if err != nil {
			log.Println(err.Error())
			log.Print("Waiting 10 sec and retry!")
			time.Sleep(30 * time.Second)
			continue
		}

		for _, torrent := range *dayTors{
			if _, ok := t411Map[string(torrent.ID)]; ok {
				continue
			}
			if torrent.Owner != "94925822" && torrent.Owner != "99183906" && torrent.Owner != "1348889" && torrent.Owner != "102683783" && torrent.Owner != "103254977" && torrent.Owner != "97121229" && torrent.Owner != "100662827" && torrent.Owner != "5810986"&& torrent.Owner != "1807879" {// Arkhos01 and amzerzo35 profile ID
				continue
			}
			seeders, _ := strconv.Atoi(torrent.Seeders)
			//size, _ := strconv.Atoi(torrent.Size)
			//leechers,_ := strconv.Atoi(torrent.Leechers)

			if seeders == 0 { //Cancel over 18GB 18874368
				continue
			}

			log.Println("Download Torrent:")
			log.Println("   ID:            ", torrent.ID)
			log.Println("   Name:          ", torrent.Name)
			log.Println("   Added:         ", torrent.Added)
			log.Println("   IsVerified:    ", torrent.IsVerified)
			log.Println("   Category:      ", torrent.Categoryname)
			log.Println("   Owner:         ", torrent.Owner)
			log.Println("   Seeders:       ", torrent.Seeders)
			log.Println("   Leechers:      ", torrent.Leechers)
			log.Println("   Size:          ", torrent.Size)

			filename, e := t411Client.DownloadTorrent(&torrent)
			if e != nil {
				log.Println(e.Error())
			}else {
				cmd, _ := transmission.NewAddCmdByFilename(filename)
				tor, e := client.ExecuteAddCommand(cmd)
				if e != nil {
					log.Println(e.Error())
				}else {
					t411Map[string(torrent.ID)] = strconv.Itoa(tor.ID)
				}
			}

		}
		log.Print("Waiting...")
		time.Sleep(5 * time.Minute)
	}
}