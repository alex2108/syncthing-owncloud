package main

import (
	"encoding/json"
	"log"
	"time"
)


func getStartTime() (string, error){


	type StStatus struct {
		StartTime string
	}
	out, err := query_syncthing(config.url + "/rest/system/status")
	
	if err != nil {
		log.Println(err)
		return "", err
	}
	var m StStatus
	err = json.Unmarshal([]byte(out), &m)
	if err != nil {
		log.Println(err)
		return "",err
	}

	return m.StartTime, nil


}





func initialize() {
	currentStartTime, err := getStartTime()
	
	
	if err != nil {
		defer initialize()
		time.Sleep(5 * time.Second)
		log.Println("error while reading start time:",err)
		return
	}
	if startTime != currentStartTime {
		log.Println("syncthing restarted at",currentStartTime)
		startTime = currentStartTime
		since_events = 0 
	}
	log.Println("starting main loop")
	main_loop()

}