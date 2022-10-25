package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/hashicorp/go-version"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	paths, err := LookPath("go")
	if err != nil {
		fmt.Println(err)
	}

	var oldV *version.Version
	var newest string
	for _, path := range paths {
		v, err := getVersion(path)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if oldV == nil || v.GreaterThan(oldV) {
			oldV = v
			newest = path

		}
	}
	dir := filepath.Dir(newest)
	pathEnv := os.Getenv("PATH")
	err = os.Setenv("PATH", fmt.Sprintf("%s:%s", dir, pathEnv))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// execute whatever using the latest version of Go available on the path
	passThrough()

}

func getVersion(file string) (*version.Version, error) {
	cmd := exec.Command(file, "version")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}
	strs := strings.Split(out.String(), " ")

	return version.NewVersion(strings.TrimPrefix(strs[2], "go"))
}

func passThrough() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("You need to run this as:\nyo gotestsum")
		return
	}

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

}

// LookPath will find all the executables in the PATH
func LookPath(file string) ([]string, error) {
	if strings.Contains(file, "/") {
		err := findExecutable(file)
		if err == nil {
			return []string{file}, nil
		}
		return nil, &Error{file, err}
	}
	path := os.Getenv("PATH")
	var foundExecutables []string
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: currentPath element "" means "."
			dir = "."
		}
		currentPath := filepath.Join(dir, file)
		if err := findExecutable(currentPath); err == nil {
			currentPath, err = filepath.EvalSymlinks(currentPath)
			if err != nil {
				// unable to resolve symbolic link, so just skip it
				continue
			}
			if !filepath.IsAbs(currentPath) {
				currentPath, err = filepath.Abs(currentPath)
				if err != nil {
					// unable to resolve relative path, so just skip it
					continue
				}
			}
			foundExecutables = append(foundExecutables, currentPath)
			continue
		}
	}
	if len(foundExecutables) == 0 {
		return foundExecutables, &Error{file, ErrNotFound}
	}
	return foundExecutables, nil
}

// ErrNotFound is the error resulting if a path search failed to find an executable file.
var ErrNotFound = errors.New("executable file not found in $PATH")

// findExecutable will return an error if the fully qualified file path
// does not exist or is not an executable
func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return fs.ErrPermission
}

// Error is returned by LookPath when it fails to classify a file as an
// executable.
type Error struct {
	// Name is the file name for which the error occurred.
	Name string
	// Err is the underlying error.
	Err error
}

func (e *Error) Error() string {
	return "exec: " + strconv.Quote(e.Name) + ": " + e.Err.Error()
}
