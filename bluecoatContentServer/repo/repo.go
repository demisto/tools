package repo

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/boltdb/bolt"
	"github.com/demisto/tools/bluecoatContentServer/domain"
)

const (
	bucket   = "rules"
	security = "security"
)

var (
	// ErrNotFound is returned when a queried item is not found
	ErrNotFound = errors.New("Item not found")
)

type Repo struct {
	db *bolt.DB // db to store the messages
}

// New repo
func New(dbFile string) (r *Repo, err error) {
	err = os.MkdirAll(filepath.Dir(dbFile), 0755)
	if err != nil {
		return
	}
	db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 30 * time.Second})
	if err != nil {
		return
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}
		_, err = tx.CreateBucketIfNotExists([]byte(security))
		return err
	})
	if err != nil {
		db.Close()
	} else {
		r = &Repo{db: db}
	}
	return
}

// Close the repo
func (r *Repo) Close() error {
	return r.db.Close()
}

// User retrieval
func (r *Repo) User(username string) (u *domain.User, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(security))
		data := b.Get([]byte(username))
		if data == nil {
			return ErrNotFound
		}
		u = &domain.User{}
		return json.Unmarshal(data, u)
	})
	return
}

// SaveUser in the DB
func (r *Repo) SaveUser(u *domain.User) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(security))
		data, err := json.Marshal(u)
		if err != nil {
			return err
		}
		return b.Put([]byte(u.User), data)
	})
}

// AddRule to the db
func (r *Repo) AddRule(rule *domain.Rule) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		data, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		return b.Put([]byte(rule.Key()), data)
	})
}

// RemoveRule from the db
func (r *Repo) RemoveRule(rule *domain.Rule) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.Delete([]byte(rule.Key()))
	})
}

// Rules returned from the DB
func (r *Repo) Rules() (rules []*domain.Rule, err error) {
	rules = make([]*domain.Rule, 0)
	err = r.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		return b.ForEach(func(k, v []byte) error {
			rule := &domain.Rule{}
			err := json.Unmarshal(v, rule)
			if err != nil {
				return err
			}
			rules = append(rules, rule)
			return nil
		})
	})
	return
}
