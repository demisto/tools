package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/demisto/tools/client"
)

var (
	body         = flag.String("body", "", "The file to read the body from")
	subject      = flag.String("subject", "FW: Your Invoice", "The subject for the email")
	target       = flag.String("target", "user@acme.com", "Who is being phished")
	attachment   = flag.String("attachment", "", "The attachment file")
	username     = flag.String("u", "", "Username to login to the server")
	password     = flag.String("p", "", "Password to login to the server")
	server       = flag.String("s", "", "Demisto server URL")
	level        = flag.String("level", "low", "Incident level - low/medium/high/critical")
	incidentType = flag.String("type", "Phishing", "Incident type - default/phishing/malware/...")
)

var (
	c *client.Client
	u *client.User
)

func printAndExit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		printAndExit("%v\n", err)
	}
}

func checkParams() {
	if *username == "" {
		printAndExit("Please provide the username\n")
	}
	if *password == "" {
		printAndExit("Please provide the password\n")
	}
	if *server == "" {
		printAndExit("Please provide the Demisto server URL\n")
	}
	if *body == "" {
		printAndExit("Please provide the body for the email\n")
	}
	if *attachment == "" {
		printAndExit("Please provide the attachment to use\n")
	}

	bInfo, err := os.Stat(*body)
	check(err)
	if !bInfo.Mode().IsRegular() {
		printAndExit("File [%s] must be a regular file\n", *body)
	}
	aInfo, err := os.Stat(*attachment)
	check(err)
	if !aInfo.Mode().IsRegular() {
		printAndExit("File [%s] must be a regular file\n", *attachment)
	}
}

func login() {
	var err error
	c, err = client.New(*username, *password, *server)
	check(err)
	u, err = c.Login()
	check(err)
	fmt.Printf("Logged in successfully with user %s [%s %s]\n", u.Username, u.Name, u.Email)
}

func logout() {
	err := c.Logout()
	check(err)
}

func main() {
	flag.Parse()
	checkParams()
	login()
	defer logout()
	bodyData, err := ioutil.ReadFile(*body)
	check(err)
	levels := map[string]int{"low": 1, "medium": 2, "high": 3, "critical": 4}
	l := levels[*level]
	if l == 0 {
		l = 1
	}
	incident := &client.Incident{Type: *incidentType, Name: *subject, Status: 0, Level: l, Details: string(bodyData),
		Targets: []client.Target{{Value: *target, Type: "Email/from"}},
	}
	inc, err := c.CreateIncident(incident)
	check(err)
	at, err := os.Open(*attachment)
	check(err)
	defer at.Close()
	_, err = c.IncidentAddAttachment(inc, at, filepath.Base(*attachment), "Mail attachment")
	check(err)
}
