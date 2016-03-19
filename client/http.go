package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

const (
	// xsrfTokenKey ...
	xsrfTokenKey = "X-XSRF-TOKEN"
	// xsrfCookieKey ...
	xsrfCookieKey = "XSRF-TOKEN"
)

type credentials struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// Client implements a client for the Demisto server
type Client struct {
	*http.Client
	credentials *credentials
	username    string
	password    string
	server      string
	token       string
}

// User - user data
type User struct {
	ID             string                 `json:"id"`
	Username       string                 `json:"username"`
	Email          string                 `json:"email"`
	Phone          string                 `json:"phone"`
	Name           string                 `json:"name"`
	Roles          map[string][]string    `json:"roles"`
	IsDefaultAdmin bool                   `json:"defaultAdmin"`
	PlaygroundID   string                 `json:"playgroundId"`
	Preferences    map[string]interface{} `json:"preferences"`
	LastLogin      time.Time              `json:"lastLogin"`
	Permissions    map[string][]string    `json:"permissions,omitempty"`
	Homepage       string                 `json:"homepage"`
	Notify         []string               `json:"notify"`
	Image          string                 `json:"image,omitempty"`
}

// New client that does not do anything yet before the login
func New(username, password, server string) (*Client, error) {
	if username == "" || password == "" || server == "" {
		return nil, fmt.Errorf("Please provide all the parameters")
	}
	if !strings.HasSuffix(server, "/") {
		server += "/"
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cookieJar, _ := cookiejar.New(nil)
	c := &Client{Client: &http.Client{Transport: tr, Jar: cookieJar}, credentials: &credentials{User: username, Password: password}, server: server}
	c.Jar = cookieJar
	req, err := http.NewRequest("GET", server, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	for _, element := range resp.Cookies() {
		if element.Name == xsrfCookieKey {
			c.token = element.Value
		}
	}
	return c, nil
}

// handleError will handle responses with status code different from success
func (c *Client) handleError(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Unexpected status code: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return nil
}

func (c *Client) req(method, path string, body io.Reader, result interface{}) error {
	req, err := http.NewRequest(method, c.server+path, body)
	if err != nil {
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-type", "application/json")
	req.Header.Add(xsrfTokenKey, c.token)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if err = c.handleError(resp); err != nil {
		return err
	}
	if result != nil {
		switch result := result.(type) {
		// Should we just dump the response body
		case io.Writer:
			if _, err = io.Copy(result, resp.Body); err != nil {
				return err
			}
		default:
			if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
				return err
			}
		}
	}
	return nil
}

// Login to the Demisto server , and returns statues code
func (c *Client) Login() (*User, error) {
	creds, err := json.Marshal(c.credentials)
	if err != nil {
		return nil, err
	}
	u := &User{}
	err = c.req("POST", "login", bytes.NewBuffer(creds), u)
	return u, err
}

// Logout from the Demisto server
func (c *Client) Logout() error {
	return c.req("POST", "logout", nil, nil)
}
