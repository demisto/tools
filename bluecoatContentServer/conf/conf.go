// Package conf provides basic configuration handling from a file exposing a single global struct with all configuration.
package conf

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/Sirupsen/logrus"
)

// Options anonymous struct holds the global configuration options for the server
var Options struct {
	// The address to listen on
	Address string
	// SSL configuration
	SSL struct {
		// The certificate file
		Cert string
		// The private key file
		Key string
	}
	// DB path
	DB string
}

// The pipe writer to wrap around standard logger. It is configured in main.
var LogWriter *io.PipeWriter

// Load loads configuration from a file.
func Load(filename string) error {
	options, err := ioutil.ReadFile(filename)
	if err != nil {
		logrus.WithError(err).Warn("Could not open config file")
		return err
	} else {
		err = json.Unmarshal(options, &Options)
		if err != nil {
			logrus.WithError(err).Warn("Could not parse config file")
			return err
		}
	}
	return nil
}

func Default() {
	Options.Address = ":9090"
	Options.DB = "bccs.db"
}
