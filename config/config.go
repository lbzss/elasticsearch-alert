package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elastic/go-elasticsearch/v8"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	envConfigFile     string = "GO_ELASTICSEARCH_ALERTS_CONFIG_FILE"
	envRulesDir       string = "GO_ELASTICSEARCH_ALERTS_RULES_DIR"
	defaultConfigFile string = "/etc/go-elasticsearch-alerts/config.json"
	defaultRulesDir   string = "/etc/go-elasticsearch-alerts/rules"
)

type Config struct {
	Elasticsearch *ESConfig    `json:"elasticsearch"`
	Rules         []RuleConfig `json:"-"`
}

type ESConfig struct {
	// Server represents the 'elasticsearch.server' field
	// of the main configuration file
	Server *ServerConfig `json:"server"`

	// Client represents the 'elasticsearch.client' field
	// of the main configuration file
	Client *ClientConfig `json:"client"`
}

func (es *ESConfig) validate() error {
	if es.Server == nil {
		return errors.New("no 'elasticsearch.server' field found")
	}

	if es.Server.ElasticsearchURL == "" {
		return errors.New("no 'elasticsearch.server.url' field found")
	}
	return nil
}

type ServerConfig struct {
	// ElasticsearchURL is the URL of your Elasticsearch instance.
	// This value should come from the 'elasticsearch.server.url'
	// field of the main configuration file
	ElasticsearchURL string `json:"url"`
}

type RuleConfig struct {
	Name                 string                 `json:"name"`
	CronSchedule         string                 `json:"schedule"`
	ElasticsearchIndex   string                 `json:"index"`
	ElasticsearchBodyRaw interface{}            `json:"body"`
	ElasticsearchBody    map[string]interface{} `json:"-"`
	Filters              []string               `json:"filters"`
	Outputs              []OutputConfig         `json:"outputs"`
	Conditions           []Condition            `json:"conditions"`
	// BodyField string `json:"body_field"`
}

func (r *RuleConfig) validate() error {
	if r.Name == "" {
		return errors.New("no 'name' field found")
	}
	if r.ElasticsearchIndex == "" {
		return errors.New("no 'index' field found")
	}

	if r.CronSchedule == "" {
		return errors.New("no 'schedule' field found")
	}

	if r.Filters == nil {
		r.Filters = []string{}
	}

	if r.Outputs == nil {
		return errors.New("no 'output' field found")
	}

	if len(r.Outputs) < 1 {
		return errors.New("at least one output must be specified ('outputs')")
	}

	for i, output := range r.Outputs {
		if err := output.validate(); err != nil {
			return fmt.Errorf("error in output %d of rule %s: %v", i+1, r.Name, err)
		}
	}

	for i, condition := range r.Conditions {
		if err := condition.validate(); err != nil {
			return fmt.Errorf("error in condition %d of rule %s: %v", i+1, r.Name, err)
		}
	}
	return nil
}

type OutputConfig struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config"`
}

func (o *OutputConfig) validate() error {
	if o.Type == "" {
		return errors.New("all outputs must have a type specified ('output.type')")
	}

	if o.Config == nil || len(o.Config) < 1 {
		return errors.New("all outputs must have a config field ('output.config')")
	}
	return nil
}

func GetClient(filepath string) (*elasticsearch.Client, error) {
	config, err := decodeConfigFile(filepath)
	if err != nil {
		return nil, err
	}
	client, err := config.NewESClient()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func decodeConfigFile(f string) (*Config, error) {
	var err error
	f, err = homedir.Expand(f)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Clean(f))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	cfg := new(Config)
	err = json.NewDecoder(file).Decode(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
