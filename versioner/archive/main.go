// Copyright (C) 2014 The Syncthing Authors.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Interval struct {
	step int64
	end  int64
}

// The type holds our configuration
type Staggered struct {
	versionsPath string
	repoPath     string
}

func main() {

	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Parse()
	args := flag.Args()

	versionsDir := args[2]

	repoPath := args[0]

	s := Staggered{
		versionsPath: versionsDir,
		repoPath:     repoPath,
	}

	s.Archive(strings.Join(args[0:2], "/"))

}

func (v Staggered) Archive(filePath string) error {

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("not archiving nonexistent file", filePath)
			return nil
		} else {
			log.Println(err)
		}
	}

	_, err = os.Stat(v.versionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("creating versions dir", v.versionsPath)
			os.MkdirAll(v.versionsPath, 0755)
		} else {
			log.Println(err)
			return nil
		}
	}

	log.Println("archiving", filePath)

	file := filepath.Base(filePath)
	inRepoPath, err := filepath.Rel(v.repoPath, filepath.Dir(filePath))
	if err != nil {
		log.Println(err)
		return nil
	}

	dir := filepath.Join(v.versionsPath, inRepoPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Println(err)
		return nil
	}
	modTime := fileInfo.ModTime().Unix()

	ver := file + ".v" + fmt.Sprintf("%010d", modTime)
	dst := filepath.Join(dir, ver)

	log.Println("moving to", dst)

	err = os.Rename(filePath, dst)
	if err != nil {
		log.Println(err)
	}
	err = os.Chtimes(dst, time.Now(), time.Now())
	if err != nil {
		log.Println(err)
	}
	return nil
}
