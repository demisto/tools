package client

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"time"

	"github.com/demisto/server/domain"
)

// Target ...
type Target struct {
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
	Targets []Target `json:"targets"`
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
}

type idVersion struct {
	ID      string `json:"id"`
	Version int64  `json:"version"`
}

type updateSeverity struct {
	idVersion
	Severity domain.Severity `json:"severity"`
}

// CreateIncident in Demisto
func (c *Client) CreateIncident(inc *Incident) (*Incident, error) {
	data, err := json.Marshal(inc)
	if err != nil {
		return nil, err
	}
	res := &Incident{}
	err = c.req("POST", "incident", "", bytes.NewBuffer(data), res)
	return res, err
}

type investigation struct {
	idVersion
	Investigation *Investigation `json:"investigation"`
}

// IncidentAddAttachment adds an attachment to a given incident
func (c *Client) IncidentAddAttachment(inc *Incident, file io.Reader, name, comment string) (*Incident, error) {
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
	err = c.req("POST", "incident/upload/"+inc.ID, writer.FormDataContentType(), b, res)
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
