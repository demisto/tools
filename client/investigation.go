package client

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/blevesearch/bleve/search"
)

// System ...
type System struct {
	Name string `json:"name"`
	Host string `json:"host"`
	OS   string `json:"os"`
	Arch string `json:"arch,omitempty"`
}

// Investigation contains the investigation of a particular incident.
type Investigation struct {
	// ID ...
	ID string `json:"id"`
	// Version ...
	Version int64 `json:"version"`
	// Modified timestamp
	Modified time.Time `json:"modified"`
	// The name of the investigation, which is unique to the project
	Name string `json:"name"`
	// The users who share this investigation
	Users []string `json:"users"`
	// The status of the investigation
	Status int `json:"status"`
	// The type of the investigation
	Type int `json:"type"`
	// The reason for the status (resolve)
	Reason map[string]string `json:"reason"`
	// When was this created
	Created time.Time `json:"created"`
	// When was this closed
	Closed time.Time `json:"closed,omitempty"`
	//The user ID that closed this investigation
	ClosingUserID string `json:"closingUserId,omitempty"`
	//duration from open to close time
	OpenDuration int64 `json:"openDuration,omitempty"`
	//The user ID that created this investigation
	CreatingUserID string `json:"creatingUserId,omitempty"`
	//User defined free text details
	Details string `json:"details"`
	// The systems involved
	Systems []System `json:"systems"`
}

// ReputationData holds the reputation data (reputation, regex, highlights result)
type ReputationData struct {
	Reputation   int    `json:"reputation"`
	ReputationID string `json:"reputationId"`
	Term         string `json:"term"`
}

// EntryReputation holds the entry reputations and the highlights
type EntryReputation struct {
	ReputationsData []*ReputationData           `json:"reputationsData"`
	Highlights      search.FieldTermLocationMap `json:"highlights"`
}

// FileMetadata ...
type FileMetadata struct {
	Type   string `json:"type"`
	Size   int64  `json:"size"`
	MD5    string `json:"md5"`
	SHA1   string `json:"sha1"`
	SHA256 string `json:"sha256"`
	SHA512 string `json:"sha512"`
	SSDeep string `json:"ssdeep"`
}

// Entry holds a single entry in an investigation. Entries entered at close times by the same user will be combined
type Entry struct {
	// ID ...
	ID string `json:"id"`
	// Version ...
	Version int64 `json:"version"`
	// Modified timestamp
	Modified time.Time `json:"modified"`
	// The type of entry - can be a combination (i.e. note + mention)
	Type int `json:"type"`
	// When it was taken
	Created time.Time `json:"created"`
	// The user who created  the entry
	User string `json:"user"`
	// The contents of the entry
	Contents interface{} `json:"contents"`
	// Holds information on how content is formatted
	ContentsFormat string `json:"format"`
	// The id of the investigation it's belongs to
	InvestigationID string `json:"investigationId"`
	// Filename of associated content
	File string `json:"file"`
	// ParentId the ID of the parent entry
	ParentID string `json:"parentId"`
	// Mark entry as pinned
	Pinned int `json:"pinned"`
	// PinnedID - the ID of the insight for the pinned entry
	PinnedID string `json:"pinnedID"`
	// FileMetadata meta data
	FileMetadata *FileMetadata `json:"fileMetadata"`
	// ParentEntry content - for reference
	ParentEntryContent interface{} `json:"parentContent"`
	// The name of the system associated with this entry
	SystemName string `json:"system"`
	// EntryReputations the reputations calculated by regex match
	EntryReputations []*EntryReputation `json:"reputations"`
	// Category
	Category string `json:"category"`
}

type updateEntry struct {
	Contents        string `json:"contents"`
	ContentsFormat  string `json:"format"`
	InvestigationID string `json:"investigationId"`
}

// AddEntryToInvestigation adds a formatted entry to the investigation
func (c *Client) AddEntryToInvestigation(investigationID string, entryData interface{}, format string) (*Entry, error) {
	entry := updateEntry{InvestigationID: investigationID, ContentsFormat: format}
	contents, err := json.Marshal(entryData)
	if err != nil {
		return nil, err
	}
	entry.Contents = string(contents)
	data, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	res := &Entry{}
	err = c.req("POST", "entryFormatted", bytes.NewBuffer(data), res)
	return res, err
}
