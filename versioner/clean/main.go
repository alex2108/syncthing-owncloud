// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)


type Interval struct {
	step int64
	end  int64
}

// The type holds our configuration
type Staggered struct {
	versionsPath  string
	interval      [4]Interval
}

var debug bool = false




func main() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	flag.Parse()
	args:=flag.Args()

	log.Println("cleaning:",args[0])
	
	versionsDir := args[0]
	
	var maxAge int64 = 365


	s := Staggered{
		versionsPath:  versionsDir,
		interval: [4]Interval{
			Interval{30, 3600},               // first hour -> 30 sec between versions
			Interval{3600, 86400},            // next day -> 1 h between versions
			Interval{86400, 592000},          // next 30 days -> 1 day between versions
			Interval{604800, maxAge * 86400}, // next year -> 1 week between versions
		},
	}
	
	s.clean()

	
	
}


func (v Staggered) clean() {


	if debug {
		log.Println("Versioner clean: Cleaning", v.versionsPath)
	}

	_, err := os.Stat(v.versionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			if debug {
				log.Println("creating versions dir", v.versionsPath)
			}
			os.MkdirAll(v.versionsPath, 0755)
		} else {
			log.Println("Versioner: can't create versions dir", err)
		}
	}

	versionsPerFile := make(map[string][]string)
	filesPerDir := make(map[string]int)

	err = filepath.Walk(v.versionsPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.Mode().IsDir() && f.Mode()&os.ModeSymlink == 0 {
			filesPerDir[path] = 0
			if path != v.versionsPath {
				dir := filepath.Dir(path)
				filesPerDir[dir]++
			}
		} else {
			// Regular file, or possibly a symlink.
			extension := filepath.Ext(path)
			dir := filepath.Dir(path)
			name := path[:len(path)-len(extension)]
			//log.Println("name:", name)
			filesPerDir[dir]++
			versionsPerFile[name] = append(versionsPerFile[name], path)
		}

		return nil
	})
	if err != nil {
		log.Println("Versioner: error scanning versions dir", err)
		return
	}

	for _, versionList := range versionsPerFile {
		// List from filepath.Walk is sorted
		v.expire(versionList)
	}

	for path, numFiles := range filesPerDir {
		if numFiles > 0 {
			continue
		}

		if path == v.versionsPath {
			if debug {
				log.Println("Cleaner: versions dir is empty, don't delete", path)
			}
			continue
		}

		if debug {
			log.Println("Cleaner: deleting empty directory", path)
		}
		err = os.Remove(path)
		if err != nil {
			log.Println("Versioner: can't remove directory", path, err)
		}
	}
	if debug {
		log.Println("Cleaner: Finished cleaning", v.versionsPath)
	}
}

func (v Staggered) expire(versions []string) {
	if debug {
		log.Println("Versioner: Expiring versions", versions)
	}
	var prevAge int64
	firstFile := true
	for _, file := range versions {
		fi, err := os.Lstat(file)
		if err != nil {
			log.Println("versioner:", err)
			continue
		}

		if fi.IsDir() {
			log.Printf("non-file %q is named like a file version", file)
			continue
		}
		
		
		
		
		
		versionTimeArchive := fi.ModTime()
		ageArchive := int64(time.Since(versionTimeArchive).Seconds())

		// If the file is older than the max age 
		// time of archive counts here to prevent instant deletion of old files, modification time for intervals
		if lastIntv := v.interval[len(v.interval)-1]; lastIntv.end > 0 && ageArchive > lastIntv.end {
			log.Println("Versioner: File over maximum age -> delete ", file)

			err = os.Remove(file)
			if err != nil {
				log.Printf("Versioner: can't remove %q: %v", file, err)
			}
			continue
		}
		

		versionTimeInt, err := strconv.ParseInt(strings.Replace(filepath.Ext(file), ".v", "", 1), 10, 0)
		if err != nil {
			if debug {
				log.Printf("Versioner: file name %q is invalid: %v", file, err)
			}
			continue
		}
		versionTime := time.Unix(versionTimeInt,0)
		age := int64(time.Since(versionTime).Seconds())
		

		// If it's the first (oldest) file in the list we can skip the interval checks
		if firstFile {
			prevAge = age
			firstFile = false
			continue
		}

		// Find the interval the file fits in
		var usedInterval Interval
		for _, usedInterval = range v.interval {
			if age < usedInterval.end {
				break
			}
		}

		if prevAge-age < usedInterval.step {
			log.Println("too many files in step -> delete", file)
			err = os.Remove(file)
			if err != nil {
				log.Printf("Versioner: can't remove %q: %v", file, err)
			}
			continue
		}

		prevAge = age
	}
}
