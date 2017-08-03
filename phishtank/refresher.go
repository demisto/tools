package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
)

type refresher struct {
	f    syncFile
	stop chan bool
}

func newRefresher(f syncFile) *refresher {
	return &refresher{f: f, stop: make(chan bool)}
}

func (r *refresher) refresh() {
	for {
		logrus.Infof("Loading file %s from URL %s", r.f.Name, r.f.URL)
		resp, err := http.Get(r.f.URL)
		if err != nil {
			logrus.WithError(err).Warnf("Unable to refresh file %s - URL %s", r.f.Name, r.f.URL)
			return
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= 300 {
			logrus.Warnf("Wrong http return code, got : %d", resp.StatusCode)
			return
		}
		logrus.Infof("Decoding file %s from URL %s", r.f.Name, r.f.URL)
		data := make([]Phish, 0)
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&data)
		resp.Body.Close()
		if err != nil {
			logrus.WithError(err).Warnf("Unable to load file %s - URL %s", r.f.Name, r.f.URL)
			return
		}
		loadedData.Set(r.f.Name, data)
		logrus.Infof("File %s from URL %s loaded", r.f.Name, r.f.URL)
		select {
		case <-r.stop:
			logrus.Infof("Received shutdown signal for refresh of %s", r.f.Name)
			return
		case <-time.After(time.Minute * time.Duration(r.f.Interval)):
		}
	}
}

func (r *refresher) Close() error {
	r.stop <- true
	return nil
}
