package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const internalUrl = "http://localhost:4000"
const url = "http://iomics.ugent.be/msplayground" //internalUrl
const name = "/prog.go"
const mypath = "C:\\Users\\compomics\\msplayground" //"/home/pieter/compomics/msplayground/"
const sel_ldr = mypath + "/nacl_sdk/pepper_34/tools/sel_ldr_x86_64"

type limitedCmd struct {
	*exec.Cmd
}

type RunResponse struct {
	Errors string
	Events []Event
}

type Event struct {
	Message string
	Delay   int
}

type prog struct {
	Directory   string
	RunResponse RunResponse
	Downloads   map[string]string
}

//mapping of source strings to executables and output
var progs map[string]prog = make(map[string]prog)

func cache(code string, p prog) {
	progs[code] = p
}

func getCached(code string) (p prog) {
	if _, ok := progs[code]; !ok {
		p = makeProg(code)
		progs[code] = p
	}
	return progs[code]
}

func makeProg(code string) prog {
	dir, err := ioutil.TempDir("", strings.TrimSuffix(strings.TrimPrefix(name, "/"), ".go"))
	if err != nil {
		log.Println("error creating temporary directory:", err)
	}
	ioutil.WriteFile(dir+name, []byte(code), 0644)
	return prog{dir, RunResponse{}, nil}
}

func (cmd *limitedCmd) killAfter(t time.Duration) (err error) {
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-time.After(t):
		if err = cmd.Process.Kill(); err != nil {
			log.Fatal("failed to kill: ", err)
		}
		<-done // allow goroutine to exit
		log.Println("process killed")
	case err = <-done:
	}
	return err
}

func isServed(s string) bool {
	response, err := http.Get(s)
	if err != nil {
		log.Println(err)
		return false
	}
	defer response.Body.Close()

	text, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return false
	}
	return !strings.Contains(string(text), "404 page not found")
}

//compile the source for multiple platforms and architectures
//if successful, serve the working folder on http
//otherwise return with an empty string map so nothing gets back to the client
func (p *prog) compileandserve() {
	ds := make(map[string]string, 6)
	errs := make([]error, 6)

	ds["32-bit Linux"], errs[0] = p.compile("linux", "386")
	ds["64-bit Linux"], errs[1] = p.compile("linux", "amd64")
	ds["32-bit Windows"], errs[2] = p.compile("windows", "386")
	ds["64-bit Windows"], errs[3] = p.compile("windows", "amd64")
	ds["32-bit MacOS X"], errs[4] = p.compile("darwin", "386")
	ds["64-bit MacOS X"], errs[5] = p.compile("darwin", "amd64")

	for _, err := range errs {
		if err != nil {
			p.Downloads = make(map[string]string, 1)
			p.Downloads["error: "+err.Error()] = ""
			return
		}
	}

	fileBasePath := "/" + filepath.Base(p.Directory) + "/"
	for k := range ds {
		ds[k] = url + fileBasePath + filepath.Base(ds[k])
	}

	p.Downloads = ds
	if !isServed(url+fileBasePath) {
		log.Println("serving", p.Directory)
		http.Handle(fileBasePath, http.StripPrefix(fileBasePath, http.FileServer(http.Dir(p.Directory))))
	}
}

//compile cross-platform, in the same directory as the source,
//preconditions are a GOROOT and GOPATH that can do this
func (p *prog) compile(goos string, goarch string) (string, error) {
	oldgoarch := os.Getenv("GOARCH")
	oldgoos := os.Getenv("GOOS")

	os.Setenv("GOARCH", goarch)
	os.Setenv("GOOS", goos)

	defer os.Setenv("GOARCH", oldgoarch)
	defer os.Setenv("GOOS", oldgoos)

	ifile := p.Directory + name
	ofile := strings.TrimSuffix(ifile, ".go") + "-" + goos + "-" + goarch
	if goos == "windows" {
		ofile += ".exe"
	}

	cmd := limitedCmd{exec.Command("go", "build", "-o", ofile, ifile)}

	//only capture stderr, a successful build has no output
	var b bytes.Buffer
	cmd.Stderr = &b

	err := cmd.Start()
	if err != nil {
		log.Println(err)
	}
	cmd.killAfter(5 * time.Second)

	if b.String() != "" {
		err = errors.New(b.String())
	}

	return ofile, err
}

//compile and run prog.go with ./sel_ldr_x86_64
func (p *prog) run() {
	oldgopath := os.Getenv("GOPATH")

	os.Setenv("GOPATH", mypath)

	defer os.Setenv("GOPATH", oldgopath)

	bin, err := p.compile("nacl", "amd64p32")
	if err != nil {
		p.RunResponse = RunResponse{
			Errors: err.Error(),
			Events: []Event{Event{Message: "", Delay: 0}},
		}
		return
	}

	cmd := limitedCmd{exec.Command(sel_ldr, bin)}

	log.Println("running", p.Directory)

	//only capture stdout, as the Go output streams are merged in NaCl
	var out bytes.Buffer
	cmd.Stdout = &out

	err = cmd.Start()
	if err != nil {
		log.Println(err)
	}
	cmd.killAfter(5 * time.Second)

	p.RunResponse = RunResponse{
		Errors: "",
		Events: []Event{Event{Message: out.String(), Delay: 0}},
	}
}

//handle post requests that contain source code. cache the code, check for
//preexisting output, run, and marshal stdout in json as response
var run = func(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		code := r.FormValue("body")

		p := getCached(code)

		if p.RunResponse.Events == nil {
			p.run()
			cache(code, p)
		}

		b, err := json.Marshal(p.RunResponse)
		if err != nil {
			log.Println(err)
		}

		w.Write(b)
	default:
		fmt.Fprint(w, "I only answer to POST requests.")
	}
}

//handle post requests that contain source code, cache the code, check for
//preexisting binaries, compile, serve all we have
//and marshal the download links in json as response
var download = func(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		code := r.FormValue("body")

		p := getCached(code)

		if p.Downloads == nil {
			p.compileandserve()
			cache(code, p)
		}

		b, err := json.Marshal(p.Downloads)
		if err != nil {
			log.Println(err)
		}

		w.Write(b)
	default:
		fmt.Fprint(w, "I only answer to POST requests.")
	}
}

//start up the http server and clean up the created folders when yser quits
func main() {
	http.HandleFunc("/run", run)
	http.HandleFunc("/download", download)
	go http.ListenAndServe(strings.TrimPrefix(internalUrl, "http://"), nil)

	fmt.Println("Terminate with Ctrl+D to clean up temporary directories.")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		//wait for the user to EOF
	}
	//cleanup
	for _, p := range progs {
		os.RemoveAll(p.Directory)
	}
}
