package main

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"flag"
	"fmt"
	"hash"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/demisto/tools/client"
	"github.com/michielbuddingh/spamsum"
)

func message() {
	fmt.Println("Demisto uploader to create investigations and entries from directories")
	fmt.Println("======================================================================")
}

var (
	path          = flag.String("f", ".", "Folder to recursively iterage")
	username      = flag.String("u", "", "Username to login to the server")
	password      = flag.String("p", "", "Password to login to the server")
	server        = flag.String("s", "", "Demisto server URL")
	investigation = flag.String("investigation", "", "If provided, investigation ID to use instead of creating investigations")
	regex         = flag.String("regex", "", "Regex to filter files and folders. If provided, only files matching the regex will be evaluated and metadata uploaded.")
	verbose       = flag.Bool("v", true, "Verbose mode - should we print directories we are handling")
	extraVerbose  = flag.Bool("vv", false, "Very verbose - should we print details about every file")
	limit         = flag.Int("limit", -1, "Count of files we should limit ourselves to")
	test          = flag.Bool("test", false, "Should we just iterate on the files without uploading them")
	account      = flag.String("account", "", "When in MT env, define an account to create the incident in")
)

var (
	r *regexp.Regexp
	c *client.Client
	u *client.User
)

func printAndExit(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func check(err error) {
	if err != nil {
		printAndExit("%v", err)
	}
}

func checkFlags() {
	if !*test {
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
	if *regex != "" {
		var err error
		r, err = regexp.Compile(*regex)
		if err != nil {
			printAndExit("Invalid regex - %v\n", err)
		}
	}
}

func login() {
	var err error
	c, err = client.New(*username, *password, *server)
	if err != nil {
		printAndExit("Error creating the client - %v\n", err)
	}
	u, err = c.Login()
	if err != nil {
		printAndExit("Error creating the client - %v\n", err)
	}
	fmt.Printf("Logged in successfully with user %s [%s %s]\n", u.Username, u.Name, u.Email)
}

func logout() {
	if err := c.Logout(); err != nil {
		printAndExit("Unable to logout - %v\n", err)
	}
}

// fileInfo holds information about a file.
type fileInfo struct {
	Created     int64
	CreatedStr  string
	Accessed    int64
	AccessedStr string
	Changed     int64
	ChangedStr  string
	Path        string `json:"1. Path"`
	Type        string
	Size        int64 `json:"2. Size"`
	Mode        string
	MD5         string `json:"3. MD5"`
	SHA1        string `json:"4. SHA1"`
	SHA256      string `json:"5. SHA256"`
	SHA512      string `json:"6. SHA512"`
	SSDeep      string `json:"7. SSDeep"`
}

func (f *fileInfo) String() string {
	return fmt.Sprintf("%s - [Created: %s, Accessed: %s, Changed: %s, Size: %v, Mode: %s] - [MD5: %s, SHA1: %s, SHA256: %s, SHA512: %s, SSDEEP: %s]",
		f.Path, f.CreatedStr, f.AccessedStr, f.ChangedStr, f.Size, f.Mode, f.MD5, f.SHA1, f.SHA256, f.SHA512, f.SSDeep)
}

func currInvestigationName(prefix, path string) string {
	if strings.HasPrefix(path, prefix) {
		actualPath := path[len(prefix):]
		if actualPath[0] == '/' || actualPath[0] == os.PathSeparator {
			actualPath = actualPath[1:]
		}
		parts := strings.Split(actualPath, string(os.PathSeparator))
		if len(parts) > 0 {
			return parts[0]
		}
	}
	return ""
}

func main() {
	flag.Parse()
	checkFlags()
	if !*test {
		login()
		defer logout()
	}
	// First, iterate on all directories in the top level directory.
	// For each directory, create an investigation and then collect all sub directories
	topLevel, err := os.Open(*path)
	check(err)
	topLevelEntries, err := topLevel.Readdir(0)
	check(err)
	count := 0
	for i := range topLevelEntries {
		if topLevelEntries[i].IsDir() {
			currName := topLevelEntries[i].Name()
			var subDirsList []string
			var currInvestigation *client.Investigation
			if !*test {
				inc, err := c.CreateIncident(&client.Incident{Type: "Malware", Name: currName, Status: 0, Level: 1, Labels: []client.Label{{Value: currName, Type: "Host"}}}, *account)
				check(err)
				if *verbose {
					fmt.Printf("Incident %s created with ID %s\n", currName, inc.ID)
				}
				currInvestigation, err = c.Investigate(inc.ID, inc.Version)
				check(err)
			} else {
				fmt.Printf("Incident %s would have been created\n", currName)
				currInvestigation = &client.Investigation{Name: currName}
			}
			err := filepath.Walk(filepath.Join(*path, currName), func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					fmt.Fprintf(os.Stderr, "Skipping %s - %v", filePath, err)
					// Just ignore the ones we have no permission to see
					return nil
				}
				if info.IsDir() {
					subDirsList = append(subDirsList, filePath)
				}
				return nil
			})
			check(err)
			sort.Strings(subDirsList)
			for _, k := range subDirsList {
				if !*test {
					info, err := os.Stat(k)
					check(err)
					_, err = c.AddEntryToInvestigation(currInvestigation.ID, createItem(k, info), "table")
					check(err)
				} else {
					fmt.Printf("Would create entry for %s\n", k)
				}
				dirNode, err := os.Open(k)
				check(err)
				dirFiles, err := dirNode.Readdir(0)
				check(err)
				var list []*fileInfo
				for i := range dirFiles {
					if dirFiles[i].IsDir() {
						continue
					}
					list = append(list, createItem(filepath.Join(k, dirFiles[i].Name()), dirFiles[i]))
					count++
				}
				if len(list) > 0 {
					if !*test {
						_, err = c.AddEntryToInvestigation(currInvestigation.ID, list, "table")
						check(err)
					} else {
						fmt.Printf("Would create entry %v\n", list)
					}
				}
				if *limit > 0 && count >= *limit {
					printAndExit("Limit of %v reached", *limit)
				}
			}
		}
	}
}

func createItem(folder string, info os.FileInfo) *fileInfo {
	item := &fileInfo{
		Changed:    info.ModTime().Unix(),
		ChangedStr: info.ModTime().String(),
		Type:       "File",
		Path:       folder,
		Size:       info.Size(),
		Mode:       info.Mode().String(),
	}
	if !info.IsDir() {
		// File type
		ext := filepath.Ext(info.Name())
		fileTypeResult := mime.TypeByExtension(ext)
		if len(fileTypeResult) == 0 && len(ext) > 0 {
			fileTypeResult = ext[1:]
		}
		if fileTypeResult != "" {
			item.Type = fileTypeResult
		}
	} else {
		item.Type = "Folder"
	}
	addOSFileInfo(item, info)
	if info.Mode().IsRegular() {
		addHashes(item.Path, item)
	}
	return item
}

type hashWrapper struct {
	hashList []hash.Hash
}

// Write ...
func (hw *hashWrapper) Write(p []byte) (n int, err error) {
	for _, hash := range hw.hashList {
		n, err = hash.Write(p)
		if err != nil {
			return
		}

		if n < len(p) {
			return 0, errors.New("Cannot write entrie file")
		}
	}
	return
}

// addHashes for (type, size, md5, sha1, sha256, sha512, spam sum...)
func addHashes(filePath string, item *fileInfo) {
	writers := &hashWrapper{
		[]hash.Hash{md5.New(), sha1.New(), sha256.New(), sha512.New()},
	}
	// Update entry file metadata
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not compute hashes for %s - %v\n", filePath, err)
		return
	}
	defer file.Close()

	_, err = io.Copy(writers, file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not compute hashes for %s - %v\n", filePath, err)
		return
	}

	var md5Result []byte
	var sha1Result []byte
	var sha256Result []byte
	var sha512Result []byte

	// Spamsum (SSDeep)
	var ssdeep = ""
	sum, err := spamsum.HashReadSeeker(file, item.Size)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not compute SSDeep for %s - %v\n", filePath, err)
	} else {
		ssdeep = sum.String()
	}

	item.MD5 = fmt.Sprintf("%x", writers.hashList[0].Sum(md5Result))
	item.SHA1 = fmt.Sprintf("%x", writers.hashList[1].Sum(sha1Result))
	item.SHA256 = fmt.Sprintf("%x", writers.hashList[2].Sum(sha256Result))
	item.SHA512 = fmt.Sprintf("%x", writers.hashList[3].Sum(sha512Result))
	item.SSDeep = ssdeep
}
