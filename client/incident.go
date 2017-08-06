package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"time"
)

// Target ...
type Label struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Attachment ...
type Attachment struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

// CustomFields ...
type CustomFields map[string]interface{}

// Incident details.
// An incident can be opened by us algorithmically or arrive from an external source like SIEM.
// If you add fields, make sure to add them to the mapping as well
type Incident struct {
	// ID of the incident
	ID string `json:"id"`
	// Version ...
	Version int64 `json:"version"`
	// Modified timestamp
	Modified time.Time `json:"modified"`
	// Type of the incident
	Type string `json:"type"`
	// Name of the incident
	Name string `json:"name"`
	// Status ...
	Status int `json:"status"`
	// Reason for the resolve
	Reason string `json:"reason"`
	// When was this created
	Created time.Time `json:"created"`
	// When this incident has really occurred
	Occurred time.Time `json:"occurred"`
	// When was this closed
	Closed time.Time `json:"closed"`
	// The severity of the incident
	Level int `json:"severity"`
	// Investigation that was opened as a result of the incoming event
	Investigation string `json:"investigationId"`
	// The targets involved
	Labels []Label `json:"labels"`
	// Attachments
	Attachments []Attachment `json:"attachment"`
	// The details of the incident - reason, etc.
	Details string `json:"details"`
	//Duration incident was open
	OpenDuration int64 `json:"openDuration"`
	//The user ID that closed this investigation
	ClosingUserID string `json:"closingUserId"`
	// The user that activated this investigation
	ActivatingUserID string `json:"activatingingUserId,omitempty"`
	//The user who owns this incident
	OwnerID string `json:"owner"`
	// When was this activated
	Activated time.Time `json:"activated,omitempty"`
	// The reason for archiving the incident
	ArchiveReason string `json:"archiveReason"`
	// The associated playbook for this incident
	PlaybookID string `json:"playbookId"`
	// When was this activated
	DueDate time.Time `json:"dueDate,omitempty"`
	// Should we automagically create the investigation
	CreateInvestigation bool `json:"createInvestigation"`
	// This field must have empty json key
	CustomFields `json:""`
}

type idVersion struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

type updateSeverity struct {
	idVersion
	Severity int `json:"severity"`
}

// Order struct holds a sort field and the direction of sorting
type Order struct {
	Field string `json:"field"`
	Asc   bool   `json:"asc"`
}

// IncidentFilter allows for very simple filtering.
type IncidentFilter struct {
	Page              int       `json:"page,omitempty"`
	Size              int       `json:"size,omitempty"`
	Sort              []Order   `json:"sort,omitempty"`
	ID                []string  `json:"id,omitempty"`                // list of IDs to filter by
	Type              []string  `json:"type,omitempty"`              // list of sources
	Name              []string  `json:"name,omitempty"`              // list of sources
	Status            []int     `json:"status,omitempty"`            // list of statuses we are interested in
	NotStatus         []int     `json:"notStatus,omitempty"`         // list of statuses we are not interested in
	Reason            []string  `json:"reason,omitempty"`            // The reason for resolve
	FromDate          time.Time `json:"fromDate,omitempty"`          // filter from date
	ToDate            time.Time `json:"toDate,omitempty"`            // filter to date
	FromClosedDate    time.Time `json:"fromClosedDate,omitempty"`    // filter from date
	ToClosedDate      time.Time `json:"toClosedDate,omitempty"`      // filter to date
	FromActivatedDate time.Time `json:"fromActivatedDate,omitempty"` // filter from date
	ToActivatedDate   time.Time `json:"toActivatedDate,omitempty"`   // filter to date
	FromDueDate       time.Time `json:"fromDueDate,omitempty"`       // filter from date
	ToDueDate         time.Time `json:"toDueDate,omitempty"`         // filter to date
	Level             []int     `json:"level,omitempty"`             // filter based on severity
	Investigation     []string  `json:"investigation,omitempty"`     // list of investigations we would like to filter by
	Systems           []string  `json:"systems,omitempty"`           // list of systems affected
	Files             []string  `json:"files,omitempty"`             // list of files affected
	Urls              []string  `json:"urls,omitempty"`              // list of urls affected
	Users             []string  `json:"users,omitempty"`             // list of users affected
	Details           string    `json:"details,omitempty"`           // details for the query
	AndOp             bool      `json:"andOp,omitempty"`             // should all fields match or at least one
	Query             string    `json:"query,omitempty"`             // free query string
	TotalOnly         bool      `json:"totalOnly"`                   // should return only total with no body
}

type SearchIncidentsData struct {
	Filter       IncidentFilter `json:"filter"`
	FilterByUser bool           `json:"userFilter"`
	FetchInsight bool           `json:"fetchInsights"`
}

// IncidentSearchResponse is the response from the search
type IncidentSearchResponse struct {
	Total int64      `json:"total"`
	Data  []Incident `json:"data"`
}

// CreateIncident in Demisto
func (c *Client) CreateIncident(inc *Incident, account string) (*Incident, error) {
	data, err := json.Marshal(inc)
	if err != nil {
		return nil, err
	}
	res := &Incident{}

	url := "incident"
	if account != "" {
		url = fmt.Sprintf("acc_%s/incident", account)
	}
	err = c.req("POST", url, "", bytes.NewBuffer(data), res)
	return res, err
}

// Incidents search based on provided filter
func (c *Client) Incidents(filter *IncidentFilter) (*IncidentSearchResponse, error) {
	data, err := json.Marshal(&SearchIncidentsData{Filter: *filter, FilterByUser: false, FetchInsight: false})
	if err != nil {
		return nil, err
	}
	res := &IncidentSearchResponse{}
	err = c.req("POST", "incidents/search", "", bytes.NewBuffer(data), res)
	return res, err
}

type investigation struct {
	idVersion
	Investigation *Investigation `json:"investigation"`
}

// IncidentAddAttachment adds an attachment to a given incident
func (c *Client) IncidentAddAttachment(inc *Incident, file io.Reader, name, comment string, account string) (*Incident, error) {
	b := &bytes.Buffer{}
	writer := multipart.NewWriter(b)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	if comment != "" {
		part, err := writer.CreateFormField("comment")
		if err != nil {
			return nil, err
		}
		_, err = part.Write([]byte(comment))
		if err != nil {
			return nil, err
		}
	}
	writer.Close()
	res := &Incident{}
	url := "incident/upload/" + inc.ID
	if account != "" {
		url = fmt.Sprintf("acc_%s/%s", account, url)
	}
	err = c.req("POST", url, writer.FormDataContentType(), b, res)
	return res, err
}

// Investigate a given incident, returns ID and version of invetigation created.
func (c *Client) Investigate(incidentID string, incidentVersion int64) (*Investigation, error) {
	data, err := json.Marshal(&idVersion{ID: incidentID, Version: incidentVersion})
	if err != nil {
		return nil, err
	}
	res := &investigation{}
	err = c.req("POST", "incident/investigate", "", bytes.NewBuffer(data), res)
	return res.Investigation, err
}
