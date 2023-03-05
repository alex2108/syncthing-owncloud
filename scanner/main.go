// Copyright (C) 2023 Jip de Beer, Alexander Graf.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var sinceEvents = 0
var startTime = "-"
var config Config

var paths = []string{}
var syncthingFolders = []string{}
var syncthingFolderToNextcloudFolderMap = make(map[string]string)
var syncthingFolderToNextcloudUserMap = make(map[string]string)
var mu sync.Mutex

const maxSameParentCount = 9

// config for connection to syncthing
type Config struct {
	url         string
	ApiKey      string
	insecure    bool
	occpath     string
	apikeyStdin bool
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func getScanPath(filePath string, syncthingFolderID string) string {

	nextcloudUserName := syncthingFolderToNextcloudUserMap[syncthingFolderID]
	nextcloudFolderName := syncthingFolderToNextcloudFolderMap[syncthingFolderID]

	if filePath == "." {
		return nextcloudUserName + "/files/" + nextcloudFolderName
	} else {
		return nextcloudUserName + "/files/" + nextcloudFolderName + "/" + filePath
	}
}

func readEvents() error {

	type eventData struct {
		Folder string `json:"folder"`
		Path   string `json:"path"`
		Type   string `json:"type"`
		Action string `json:"action"`
	}
	type event struct {
		ID   int       `json:"id"`
		Type string    `json:"type"`
		Time time.Time `json:"time"`
		Data eventData `json:"data"`
	}

	res, err := querySyncthing(fmt.Sprintf("%s/rest/events/disk?since=%d", config.url, sinceEvents))

	if err != nil { // usually connection error -> continue
		log.Println(err)
		return err
	}

	var events []event
	folderEvents := make(map[string][]event)

	err = json.Unmarshal([]byte(res), &events)
	if err != nil {
		log.Println(err)
		return err
	}

	// Filter the folder events for the configured syncthingFolders
	for _, event := range events {
		folder := event.Data.Folder
		if contains(syncthingFolders, folder) {
			folderEvents[folder] = append(folderEvents[folder], event)
		}

		diff := event.ID - sinceEvents
		if diff > 1 {
			log.Println(sinceEvents, "->", event.ID)
			log.Println("ERROR: missed events:", diff)
		}

		sinceEvents = event.ID
	}

	mu.Lock()

	for _, folder := range syncthingFolders {

		lastParentScanPath := ""
		sameParentCounter := 1

		for _, event := range folderEvents[folder] {
			// log.Println("sinceEvents:", sinceEvents)

			pathFromEvent := event.Data.Path
			scanPath := getScanPath(pathFromEvent, folder)
			parentScanPath := getScanPath(filepath.Dir(pathFromEvent), folder)

			if event.Data.Action == "deleted" {

				// Scan only the parent folder
				// Scanning each file individually takes way longer (in case of many changes) and doesn't handle file deletions
				// The downside is that we now also rescan sibling files which haven't changed

				paths = uniqueAppend(paths, parentScanPath)
			} else {

				if event.Data.Type == "file" {
					// Don't append if parent directory is already listed,
					// since changes to this file will be picked up when scanning the parent directory
					if !contains(paths, parentScanPath) {
						paths = uniqueAppend(paths, scanPath)
					}
				} else {
					// If it's a directory, then scan this path
					paths = uniqueAppend(paths, scanPath)
				}
			}

			if lastParentScanPath == parentScanPath {
				sameParentCounter++
			} else {
				sameParentCounter = 1
			}

			if sameParentCounter == maxSameParentCount {

				// If we detected so many changes inside the same folder,
				// it may be faster to scan the entire folder instead of each file individually

				x := len(paths) - 1
				lastIndex := len(paths) - maxSameParentCount

				if lastIndex < 0 {
					lastIndex = 0
				}

				// Loop over the list of paths and remove the last which shared the same parent
				for x >= lastIndex {

					if matched, _ := path.Match(parentScanPath, filepath.Dir(paths[x])); matched {
						// Remove the last path from list of paths
						paths = paths[:len(paths)-1]
					} else {
						// The last path in the list of paths, does not have the same parent
						// Due to de-duplication we may run out of paths to remove sooner
						// Break out of the loop...
						break
					}
					x--
				}

				// Scan the parent folder instead
				paths = uniqueAppend(paths, parentScanPath)
			}

			lastParentScanPath = parentScanPath
		}

	}

	mu.Unlock()

	return nil
}

func mainLoop() {
	for {
		err := readEvents()
		if err != nil {
			defer initialize()
			time.Sleep(5 * time.Second)
			log.Println("error while reading events:", err)
			return
		}
	}

}

// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func uniqueAppend(slice []string, item string) []string {
	if !contains(slice, item) {
		return append(slice, item)
	}

	return slice
}

func externalRunner() {
	for {
		// There is more work to do
		if len(paths) > 0 {

			mu.Lock()
			pathToScan := paths[0] // get the 0 index element from slice
			paths = paths[1:]      // remove the 0 index element from slice
			mu.Unlock()

			log.Println("Start PHP scan:", pathToScan)
			out, err := exec.Command("php", "-f", config.occpath, "files:scan", "--path="+pathToScan, "--shallow").Output()
			if err != nil {
				log.Fatal(err)
			}
			log.Printf("%s", out)
		} else {
			// Wait and check if there are paths to process in the next round
			time.Sleep(5 * time.Second)
		}
	}
}

func main() {
	url := flag.String("target", "http://localhost:8384", "Target Syncthing instance")
	apikey := flag.String("api", "", "syncthing api key")
	occpath := flag.String("occpath", "", "path to nextcloud occ command")
	insecure := flag.Bool("i", false, "skip verification of SSL certificate")
	apikeyStdin := flag.Bool("apikey-from-stdin", false, "use api key from stdin")

	var mappingList arrayFlags
	flag.Var(&mappingList, "mapping", "Triple of nextcloud username, nextcloud external storage folder name and syncthing folder id, separated by the / character.")

	flag.Parse()

	if len(mappingList) == 0 {
		log.Fatal("The -mapping flag is missing!")
	}

	for _, mapping := range mappingList {
		s := strings.Split(mapping, "/")
		if len(s) != 3 {
			log.Fatal("The -mapping flag should consist of three values, separated by the / character.")
		}

		nextcloudUserName := s[0]
		nextcloudFolderName := s[1]
		syncthingFolderID := s[2]

		syncthingFolders = append(syncthingFolders, syncthingFolderID)
		syncthingFolderToNextcloudFolderMap[syncthingFolderID] = nextcloudFolderName
		syncthingFolderToNextcloudUserMap[syncthingFolderID] = nextcloudUserName
	}

	config.url = *url
	config.insecure = *insecure
	config.ApiKey = *apikey
	config.occpath = *occpath
	config.apikeyStdin = *apikeyStdin

	if config.apikeyStdin {
		log.Println("Enter api key:")
		reader := bufio.NewReader(os.Stdin)
		input, _, err := reader.ReadLine()

		if err != nil {
			log.Println("Error reading api key from stdin")
			log.Fatal(err)
		}
		config.ApiKey = string(input)
	}

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	go externalRunner()
	initialize()
}
