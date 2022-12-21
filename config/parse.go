package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/mitchellh/go-homedir"
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

func ParseConfig() (*Config, error) {
	configFile := defaultConfigFile
	if v := os.Getenv(envConfigFile); v != "" {
		configFile = v
	}

	cfg, err := decodeConfigFile(configFile)
	if err != nil {
		return nil, err
	}

	if cfg.Elasticsearch == nil {
		return nil, fmt.Errorf("no 'elasticsearch' field found in main configuration file %s", configFile)
	}

	if err := cfg.Elasticsearch.validate(); err != nil {
		return nil, fmt.Errorf("error in main configuration file %s: %v", configFile, err)
	}

	rules, err := ParseRules()
	if err != nil {
		return nil, err
	}
	if len(rules) < 1 {
		return nil, errors.New("at least one rule must be specified")
	}
	cfg.Rules = rules
	return cfg, nil
}

func ParseRules() ([]RuleConfig, error) {
	rulesDir := defaultRulesDir
	if v := os.Getenv(envRulesDir); v != "" {
		d, err := homedir.Expand(v)
		if err != nil {
			return nil, fmt.Errorf("error expanding rules directory: %v", err)
		}
		rulesDir = d
	}

	ruleFiles, err := filepath.Glob(filepath.Join(rulesDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("error globbing rules dir: %v", err)
	}

	rules := make([]RuleConfig, 0, len(ruleFiles))
	for _, ruleFile := range ruleFiles {
		file, err := os.Open(filepath.Clean(ruleFile))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("error opening file %s: %v", file.Name(), err)
		}

		dec := json.NewDecoder(file)
		dec.UseNumber()

		var rule RuleConfig
		if err := dec.Decode(file); err != nil {
			file.Close()
			return nil, fmt.Errorf("error JSON-decoding rule file %s: %v", file.Name(), err)
		}
		file.Close()

		rule.ElasticsearchBody, err = parseBody(rule.ElasticsearchBodyRaw)
		if err != nil {
			return nil, fmt.Errorf("error in rule file %s: %v", file.Name(), err)
		}
		rule.ElasticsearchBodyRaw = nil

		if err := rule.validate(); err != nil {
			return nil, fmt.Errorf("error in rule file %s: %v", file.Name(), err)
		}

		rules = append(rules, rule)
	}
	return rules, nil
}

func parseBody(v interface{}) (map[string]interface{}, error) {
	switch b := v.(type) {
	case map[string]interface{}:
		return b, nil
	case string:
		var body map[string]interface{}
		if err := json.NewDecoder(bytes.NewBufferString(b)).Decode(&body); err != nil {
			return nil, fmt.Errorf("error JSON-decoding 'body' field: %v", err)
		}
		return body, nil
	default:
		return nil, errors.New("'body' field must be valid JSON")
	}
}
