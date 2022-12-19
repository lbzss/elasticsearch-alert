package config

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var DisableCache bool
var homedirCache string
var cacheLock sync.RWMutex

func Dir() (string, error) {
	if !DisableCache {
		cacheLock.RLock()
		cached := homedirCache
		cacheLock.Unlock()
		if cached != "" {
			return cached, nil
		}
	}

	cacheLock.Lock()
	defer cacheLock.Unlock()

	var result string
	var err error

	if runtime.GOOS == "windows" {
		result, err = dirWindows()
	} else {
		result, err = dirUnix()
	}

	if err != nil {
		return "", err
	}
	homedirCache = result
	return result, nil
}

func Expand(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}

	if path[0] != '~' {
		return path, nil
	}

	if len(path) > 1 && path[1] != '/' && path[1] != '\\' {
		return "", errors.New("cannot expand user-specific home dir")
	}

	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, path[1:]), nil
}

func Reset() {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	homedirCache = ""
}

func dirUnix() (string, error) {
	homeEnv := "HOME"
	if runtime.GOOS == "plan9" {
		homeEnv = "home"
	}

	if home := os.Getenv(homeEnv); home != "" {
		return home, nil
	}

	var stdout bytes.Buffer

	if runtime.GOOS == "darwin" {
		cmd := exec.Command("sh", "-c", `dscl -q . -read /Users/"$(whoami)" NFSHomeDirectory | sed 's/^[^ ]*: //'`)
		cmd.Stdout = &stdout
		if err := cmd.Run(); err == nil {
			result := strings.TrimSpace(stdout.String())
			if result != "" {
				return result, nil
			}
		}
	} else {
		cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			if err != exec.ErrNotFound {
				return "", err
			}
		} else {
			if passwd := strings.TrimSpace(stdout.String()); passwd != "" {
				passwdParts := strings.SplitN(passwd, ":", 7)
				if len(passwdParts) > 5 {
					return passwdParts[5], nil
				}
			}
		}
	}
	// If all else failed, try the shell
	stdout.Reset()
	cmd := exec.Command("sh", "-c", "cd && pwd")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}

func dirWindows() (string, error) {
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}
	if home := os.Getenv("USERPROFILE"); home != "" {
		return home, nil
	}
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		return "", errors.New("HOMEDRIVE, HOMEPATH, or USERPROFILE are blank")
	}
	return home, nil
}
