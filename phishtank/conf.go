// Package conf provides basic configuration handling from a file exposing a single global struct with all configuration.
package main

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"sync"

	"github.com/Sirupsen/logrus"
)

// syncFile definition
type syncFile struct {
	// Name of the file
	Name string
	// URL to load file from - can be empty if manually placed
	URL string
	// Interval to sync in min
	Interval int
}

// options anonymous struct holds the global configuration options for the server
var options struct {
	// The address to listen on
	Address string
	// SSL configuration
	SSL struct {
		// The certificate file
		Cert string
		// The private key file
		Key string
	}
	// Path to the directory to load files
	Path string
	// Files to sync
	Files []syncFile
}

// logWriter pipe writer to wrap around standard logger. It is configured in main.
var logWriter *io.PipeWriter

// load configuration from a file.
func load(filename string) error {
	options, err := ioutil.ReadFile(filename)
	if err != nil {
		logrus.WithError(err).Warn("Could not open config file")
		return err
	} else {
		err = json.Unmarshal(options, &options)
		if err != nil {
			logrus.WithError(err).Warn("Could not parse config file")
			return err
		}
	}
	return nil
}

// defaults for the options
func defaults() {
	options.Address = ":9090"
	options.Path = "."
	options.Files = []syncFile{{Name: "phishtank", URL: "http://data.phishtank.com/data/online-valid.json", Interval: 60 * 24}}
}

type PhishDetails struct {
	IPAddress         string `json:"ip_address"`
	CIDRBlock         string `json:"cidr_block"`
	AnnouncingNetwork string `json:"announcing_network"`
	RIR               string `json:"arin"`
	Country           string `json:"country"`
	DetailTime        string `json:"detail_time"`
}

type Phish struct {
	PhishID          string         `json:"phish_id"`
	URL              string         `json:"url"`
	PhishDetailURL   string         `json:"phish_detail_url,omitempty"`
	SubmissionTime   string         `json:"submission_time,omitempty"`
	Verified         string         `json:"verified,omitempty"`
	VerificationTime string         `json:"verification_time,omitempty"`
	Online           string         `json:"online,omitempty"`
	Details          []PhishDetails `json:"details,omitempty"`
	Target           string         `json:"target,omitempty"`
}

// fileData map
type fileData struct {
	data      map[string][]Phish
	dataByURL map[string]map[string]*Phish
	m         sync.RWMutex
}

// Set the data for a specific file
func (d *fileData) Set(name string, data []Phish) {
	d.m.Lock()
	defer d.m.Unlock()
	d.data[name] = data
	d.dataByURL[name] = make(map[string]*Phish)
	for i := range data {
		d.dataByURL[name][data[i].URL] = &data[i]
	}
}

// Get the data for a specific file
func (d *fileData) Get(name string) []Phish {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.data[name]
}

// Get the data from a file for a specific URL
func (d *fileData) GetURL(name, url string) *Phish {
	d.m.RLock()
	defer d.m.RUnlock()
	if d.dataByURL != nil && d.dataByURL[name] != nil {
		return d.dataByURL[name][url]
	}
	return nil
}

var loadedData = fileData{data: make(map[string][]Phish), dataByURL: make(map[string]map[string]*Phish)}
