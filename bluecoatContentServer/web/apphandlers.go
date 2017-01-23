package web

import (
	"bytes"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/tools/bluecoatContentServer/domain"
)

// dbHandler returns the file with the bluecoat rules
func (ac *AppContext) dbHandler(w http.ResponseWriter, r *http.Request) {
	rules, err := ac.r.Rules()
	if err != nil {
		log.WithError(err).Warn("Unable to load the rules from the DB")
		writeError(w, ErrInternalServer)
		return
	}
	m := make(map[string][]string)
	for _, rule := range rules {
		if _, ok := m[rule.Category]; !ok {
			m[rule.Category] = make([]string, 0)
		}
		m[rule.Category] = append(m[rule.Category], rule.URL)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Disposition", "attachment; filename=database.txt")
	var b bytes.Buffer
	for k, v := range m {
		fmt.Fprintf(&b, "define category %s\n", k)
		for _, u := range v {
			fmt.Fprintf(&b, "%s\n", u)
		}
		b.WriteString("end\n")
	}
	_, err = b.WriteTo(w)
	if err != nil {
		log.WithError(err).Info("Unable to write the full file")
	}
}

// addRule to the DB
func (ac *AppContext) addRule(w http.ResponseWriter, r *http.Request) {
	rule := getRequestBody(r).(*domain.Rule)
	err := ac.r.AddRule(rule)
	if err != nil {
		log.WithError(err).Warnf("Unable to save rule %v", rule)
		writeError(w, ErrInternalServer)
		return
	}
	writeOK(w)
}

// removeRule from the DB
func (ac *AppContext) removeRule(w http.ResponseWriter, r *http.Request) {
	rule := getRequestBody(r).(*domain.Rule)
	err := ac.r.RemoveRule(rule)
	if err != nil {
		log.WithError(err).Warnf("Unable to save rule %v", rule)
		writeError(w, ErrInternalServer)
		return
	}
	writeOK(w)
}
