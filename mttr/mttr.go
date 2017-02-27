package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/demisto/tools/client"
)

var (
	username = flag.String("u", "", "Username to login to the server")
	password = flag.String("p", "", "Password to login to the server")
	server   = flag.String("s", "", "Demisto server URL")
	filter   = flag.String("f", "status:=2 and closed:>"+defaultStartDate(), "Filter for the mttr query")
	group    = flag.String("g", "owner", "Which field to group by")
	output   = flag.String("o", "mttr.csv", "Output csv file path")
)

var (
	c *client.Client
	u *client.User
)

func defaultStartDate() string {
	return time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
}

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
	incidents, err := c.Incidents(&client.IncidentFilter{Page: 0, Size: 10000, Query: *filter})
	check(err)
	fmt.Printf("Total number of incidents is: %v\n", incidents.Total)
	mttr := make(map[string]map[string]int64)
	for i := range incidents.Data {
		if incidents.Data[i].Closed.IsZero() {
			continue
		}
		delta := incidents.Data[i].Closed.Sub(incidents.Data[i].Created)
		field := incidents.Data[i].OwnerID
		if *group == "owner" {
			if field == "" {
				field = "dbot"
			}
		} else {
			found := false
			val := reflect.ValueOf(incidents.Data[i])
			typ := val.Type()
			for i := 0; i < typ.NumField(); i++ {
				tag := typ.Field(i).Tag
				jsonTag := tag.Get("json")
				if jsonTag == *group || strings.Title(*group) == typ.Field(i).Name {
					field = val.FieldByName(typ.Field(i).Name).String()
					found = true
					break
				}
			}
			if !found {
				for k, v := range incidents.Data[i].CustomFields {
					if k == *group {
						field = fmt.Sprintf("%v", v)
						found = true
						break
					}
				}
			}
			// If not found then it will default to owner
			if !found {
				field = ""
			}
		}
		if _, ok := mttr[field]; ok {
			mttr[field]["incidents"] = mttr[field]["incidents"] + 1
			mttr[field]["total"] = mttr[field]["total"] + int64(delta)
		} else {
			mttr[field] = map[string]int64{"incidents": 1, "total": int64(delta)}
		}
	}
	f, err := os.Create(*output)
	check(err)
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{strings.Title(*group), "Incidents", "MTTR"})
	for k, v := range mttr {
		w.Write([]string{k, strconv.FormatInt(v["incidents"], 10), strconv.FormatInt(int64(time.Duration(v["total"]/v["incidents"]).Minutes()), 10)})
	}
	w.Flush()
}
