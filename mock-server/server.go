package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

var (
	folder = flag.String("folder", ".", "The location of the responses and routes files")
	port   = flag.String("port", "5050", "The port to listen on")
)

type param struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Mandatory bool   `json:"mandatory"`
}

type route struct {
	Path       string  `json:"path"`       // Path we should listen to
	Method     string  `json:"method"`     // Method we expect
	Parameters []param `json:"parameters"` // Parameters we should accept and check
	Request    string  `json:"request"`    // Request body we should get
	Status     int     `json:"status"`
	Response   string  `json:"response"` // Response file - assumed to be in the same folder
	Headers    string  `json:"headers"`  // Headers to return - can also include cookies, etc. - JSON format
}

type header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func printAndExit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		printAndExit("%v", err)
	}
}

type handler struct {
	routes []route
}

func checkParam(p *param, r *http.Request) error {
	val := r.Form.Get(p.Name)
	if p.Mandatory && val == "" {
		return fmt.Errorf("Param [%s] was not provided", p.Name)
	}
	switch p.Type {
	case "int":
		_, err := strconv.Atoi(val)
		return err
	case "bool":
		_, err := strconv.ParseBool(val)
		return err
	}
	return nil
}

func (h *handler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	fmt.Printf("Received %s\n", r.URL.Path)
	for _, rp := range h.routes {
		if rp.Path == r.URL.Path && rp.Method == r.Method {
			fmt.Printf("Found handler - %s\n", rp.Path)
			for _, p := range rp.Parameters {
				err := checkParam(&p, r)
				if err != nil {
					rw.WriteHeader(http.StatusBadRequest)
					rw.Write([]byte(err.Error()))
					return
				}
			}
			if rp.Request != "" {
				reqdata, err := ioutil.ReadFile(filepath.Join(*folder, rp.Request))
				check(err)
				actual := make([]byte, len(reqdata))
				_, err = r.Body.Read(actual)
				if err != nil && err != io.EOF {
					rw.WriteHeader(http.StatusBadRequest)
					rw.Write([]byte(err.Error()))
					return
				}
				if !bytes.Equal(reqdata, actual) {
					rw.WriteHeader(http.StatusBadRequest)
					rw.Write([]byte(fmt.Sprintf("Wrong request.\nExpected:\n[%s]\nGot:\n[%s]\n", string(reqdata), string(actual))))
					return
				}
			}
			if rp.Response != "" {
				data, err := ioutil.ReadFile(filepath.Join(*folder, rp.Response))
				check(err)
				_, err = rw.Write(data)
				check(err)
			}
			if rp.Headers != "" {
				headersData, err := ioutil.ReadFile(filepath.Join(*folder, rp.Headers))
				check(err)
				var headers []header
				err = json.Unmarshal(headersData, &headers)
				check(err)
				for _, h := range headers {
					rw.Header().Add(h.Name, h.Value)
				}
			}
			status := rp.Status
			if status == 0 {
				status = 200
			}
			rw.WriteHeader(rp.Status)
			return
		}
	}
	rw.WriteHeader(http.StatusNotFound)
	_, err := rw.Write([]byte("Not found"))
	check(err)
}

func main() {
	flag.Parse()
	routesFile := filepath.Join(*folder, "routes.json")
	routesData, err := ioutil.ReadFile(routesFile)
	check(err)
	var routes []route
	err = json.Unmarshal(routesData, &routes)
	check(err)
	h := &handler{routes: routes}
	err = http.ListenAndServe(":"+*port, h)
	check(err)
}
