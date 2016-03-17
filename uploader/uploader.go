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

	"github.com/Sirupsen/logrus"
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
)

var (
	r *regexp.Regexp
)

func check() {
	/*
		if *username == "" {
			fmt.Fprintln(os.Stderr, "Please provide the username")
			os.Exit(1)
		}
		if *password == "" {
			fmt.Fprintln(os.Stderr, "Please provide the password")
			os.Exit(1)
		}
		if *server == "" {
			fmt.Fprintln(os.Stderr, "Please provide the Demisto server URL")
			os.Exit(1)
		}*/
	if *regex != "" {
		var err error
		r, err = regexp.Compile(*regex)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid regex - %v\n", err)
			os.Exit(1)
		}
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
	Path        string
	Type        string
	Size        int64
	Mode        string
	MD5         string
	SHA1        string
	SHA256      string
	SHA512      string
	SSDeep      string
}

func (f *fileInfo) String() string {
	return fmt.Sprintf("%s - [Created: %s, Accessed: %s, Changed: %s, Size: %v, Mode: %s] - [MD5: %s, SHA1: %s, SHA256: %s, SHA512: %s, SSDEEP: %s]",
		f.Path, f.CreatedStr, f.AccessedStr, f.ChangedStr, f.Size, f.Mode, f.MD5, f.SHA1, f.SHA256, f.SHA512, f.SSDeep)
}

func main() {
	flag.Parse()
	check()
	var list []*fileInfo
	err := filepath.Walk(*path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Skipping %s - %v", filePath, err)
			// Just ignore the ones we have no permission to see
			return nil
		}
		skip := false
		if r != nil {
			if !r.MatchString(filePath) {
				skip = true
			}
		}
		if !skip {
			item := createItem(filePath, info)
			list = append(list, item)
			if info.IsDir() {
				if *verbose {
					fmt.Printf("%s\n", filePath)
				}
			} else if *extraVerbose {
				fmt.Println(item)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error iterating %s - %v", *path, err)
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
		logrus.WithError(err).Errorf("Could not compute SSDeep")
	} else {
		ssdeep = sum.String()
	}

	item.MD5 = fmt.Sprintf("%x", writers.hashList[0].Sum(md5Result))
	item.SHA1 = fmt.Sprintf("%x", writers.hashList[1].Sum(sha1Result))
	item.SHA256 = fmt.Sprintf("%x", writers.hashList[2].Sum(sha256Result))
	item.SHA512 = fmt.Sprintf("%x", writers.hashList[3].Sum(sha512Result))
	item.SSDeep = ssdeep
}
