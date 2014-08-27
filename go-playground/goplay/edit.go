// Copyright 2011 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goplay

import (
	"appengine"
	"appengine/datastore"
	"net/http"
	"strings"
	"text/template"
)

const hostname = "ms-playground.appspot.com"

func init() {
	http.HandleFunc("/", edit)
}

var editTemplate = template.Must(template.ParseFiles("goplay/edit.html"))

type editData struct {
	Snippet *Snippet
}

func edit(w http.ResponseWriter, r *http.Request) {
	// Redirect foo.play.golang.org to play.golang.org.
	if strings.HasSuffix(r.Host, "."+hostname) {
		http.Redirect(w, r, "http://"+hostname, http.StatusFound)
		return
	}

	snip := &Snippet{Body: []byte(hello)}
	if strings.HasPrefix(r.URL.Path, "/p/") {
		c := appengine.NewContext(r)
		id := r.URL.Path[3:]
		serveText := false
		if strings.HasSuffix(id, ".go") {
			id = id[:len(id)-3]
			serveText = true
		}
		key := datastore.NewKey(c, "Snippet", id, 0, nil)
		err := datastore.Get(c, key, snip)
		if err != nil {
			if err != datastore.ErrNoSuchEntity {
				c.Errorf("loading Snippet: %v", err)
			}
			http.Error(w, "Snippet not found", http.StatusNotFound)
			return
		}
		if serveText {
			w.Header().Set("Content-type", "text/plain")
			w.Write(snip.Body)
			return
		}
	}
	editTemplate.Execute(w, &editData{snip})
}

const hello = `//The printspectrum tool prints out the spectrum (mz and intensity values) of a
//Thermo RAW File
//
//  Every line of the output is a peak registered by the mass spectrometer
//  characterized by an m/z value in Da and an intensity in the mass spectrometer's unit of abundance
package main

import (
	"bitbucket.org/proteinspector/ms"
	"bitbucket.org/proteinspector/ms/unthermo"
	"flag"
	"fmt"
	"log"
)

func main() {
	var scannumber int
	var filename string

	//Parse arguments
	flag.IntVar(&scannumber, "sn", 1, "the scan number")
	flag.StringVar(&filename, "raw", "small.RAW", "name of the RAW file")
	flag.Parse()

	//open RAW file
	file, err := unthermo.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	//Print the Spectrum at the supplied scan number
	printspectrum(file.Scan(scannumber))
}

//Print m/z and Intensity of every peak in the spectrum
func printspectrum(scan ms.Scan) {
	for _, peak := range scan.Spectrum() {
		fmt.Println(peak.Mz, peak.I)
	}
}
`
