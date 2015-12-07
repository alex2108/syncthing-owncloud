package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time" 
	"os"
	"os/exec"
)



var since_events = 0
var startTime = "-"
var config Config
var c = make(chan string,10000)


// config for connection to syncthing
type Config struct {
	url         string
	ApiKey      string
	insecure    bool
	occpath     string
	ocuser      string
	apikeyStdin bool
}


func readEvents() error {


	type eventData struct {
		Folder     string        `json:"folder"`
		Item       string        `json:"item"`
	}
	type event struct {
		ID   int       `json:"id"`
		Type string    `json:"type"`
		Time time.Time `json:"time"`
		Data eventData `json:"data"`
	}

	res, err := query_syncthing(fmt.Sprintf("%s/rest/events?since=%d", config.url, since_events))

	if err != nil { //usually connection error -> continue
		//log.Println(err)
		return err
	} else {
		var events []event
		err = json.Unmarshal([]byte(res), &events)
		if err != nil {
			//log.Println(err)
			return err
		}

		for _, event := range events {
			// handle different events
			if event.Type == "ItemFinished" && event.Data.Folder == "cloud" {
				log.Println("folder:",event.Data.Folder,"file",event.Data.Item)
				c <- event.Data.Item
			} 
			since_events = event.ID
		}

	}

	return nil
}

func main_loop() {
	for {
		err := readEvents()
		if err != nil {
			defer initialize()
			time.Sleep(5 * time.Second)
			log.Println("error while reading events:",err)
			return
		}
		
		
	}

}

func externalRunner() {
	for file := range c {
		out,err := exec.Command("php", "-f",config.occpath,"files:scan","--path="+config.ocuser+"/files/"+file).Output()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%s",out)
	}
}





func main() {
	url := flag.String("target", "http://localhost:8384", "Target Syncthing instance")
	apikey := flag.String("api", "", "syncthing api key")
	occpath := flag.String("occpath", "", "path to owncloud occ command")
	ocuser := flag.String("ocuser", "", "owncloud user")
	insecure := flag.Bool("i", false, "skip verification of SSL certificate")
	apikeyStdin := flag.Bool("apikey-from-stdin", false, "use api key from stdin")
	flag.Parse()

	config.url = *url
	config.insecure = *insecure
	config.ApiKey = *apikey
	config.occpath = *occpath
	config.ocuser = *ocuser
	config.apikeyStdin = *apikeyStdin
	
	
	if config.apikeyStdin {
		log.Println("Enter api key:")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		
		if err != nil {
			log.Println("Error reading api key from stdin")
			log.Fatal(err)
		}
		config.ApiKey = input
	}
	
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	
	log.Println("starting externalRunner")
	go externalRunner()
	initialize()
}


