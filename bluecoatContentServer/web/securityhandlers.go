package web

import (
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	lru "github.com/hashicorp/golang-lru"
)

var bruteForceMap *lru.Cache

func initBruteForceMap(sleep bool) {
	tmpBruteForceMap, err := lru.New(100)
	if err != nil {
		log.WithError(err).Error("Failed creating brute force lru map sleep:", sleep)
		if sleep {
			time.Sleep(time.Second * 10)
		}
	} else {
		bruteForceMap = tmpBruteForceMap
	}
}

func (ac *AppContext) preventBruteForce(key string) {
	var count int
	// This is just for safety if for some reason we fail to create the map in the initialization phase
	if bruteForceMap == nil {
		initBruteForceMap(true)
	}
	countInter, exists := bruteForceMap.Get(key)
	if exists {
		count, _ = countInter.(int)
		count++
	}
	if count <= 0 {
		count = 1
	}
	if count > 5 {
		time.Sleep(time.Second * 60 * time.Duration(count-5))
	} else if count > 2 {
		time.Sleep(time.Second * 10)
	}
	bruteForceMap.Add(key, count)
}

func (ac *AppContext) resetBruteForce(key string) {
	bruteForceMap.Remove(key)
}

func (ac *AppContext) handleLoginError(w http.ResponseWriter, user string) {
	ac.preventBruteForce(user)
	writeError(w, ErrCredentials)
}
