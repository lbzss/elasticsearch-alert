package query

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/lbzss/elasticsearch-alert/command/alert"
	"github.com/lbzss/elasticsearch-alert/config"
	"github.com/robfig/cron"
)

type QueryHandlerConfig struct {
	Name         string
	AlertMethods []alert.Method
	Client       *elasticsearch.Client
	ESUrl        string
	QueryData    map[string]interface{}
	QueryIndex   string
	Schedule     string
	BodyField    string
	Filters      []string
	Conditions   []config.Condition
}

type QueryHandler struct {
	StopCh chan struct{}

	name         string
	hostname     string
	alertMethods []alert.Method
	client       *elasticsearch.Client
	esURL        string
	queryIndex   string
	queryData    map[string]interface{}
	schedule     cron.Schedule
	bodyField    string
	filters      []string
	conditions   []config.Condition
}

// TODO
func NewQueryHandler(config *QueryHandlerConfig) (*QueryHandler, error) {
	if config == nil {
		config = &QueryHandlerConfig{}
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error getting hostname, %v", err.Error())
	}

	schedule, err := cron.Parse(config.Schedule)
	if err != nil {
		return nil, fmt.Errorf("error parsing cron schedule: %v", err)
	}

	// if config.Client == nil {
	// 	config.Client, err = config.
	// }

	return &QueryHandler{
		StopCh: make(chan struct{}),

		name:         config.Name,
		hostname:     hostname,
		alertMethods: config.AlertMethods,
		client:       config.Client,
		esURL:        config.ESUrl,
		queryIndex:   config.QueryIndex,
		queryData:    config.QueryData,
		schedule:     schedule,
		bodyField:    config.BodyField,
		filters:      config.Filters,
		conditions:   config.Conditions,
	}, nil
}

func validateConfig(config *QueryHandlerConfig) error {
	var allErrors *multierror.Error
	if config.Name == "" {
		allErrors = multierror.Append(allErrors, errors.New("no rule name provided"))
	}

	config.ESUrl = strings.TrimRight(config.ESUrl, "/")
	if config.ESUrl == "" {
		allErrors = multierror.Append(allErrors, errors.New("no Elasticsearch URL provided"))
	}

	if config.QueryIndex == "" {
		allErrors = multierror.Append(allErrors, errors.New("no Elasticsearch Index provided"))
	}

	if len(config.AlertMethods) < 1 {
		allErrors = multierror.Append(allErrors, errors.New("at least one alert method must be specified"))
	}

	if config.QueryData == nil || len(config.QueryData) < 1 {
		allErrors = multierror.Append(allErrors, errors.New("no query body provided"))
	}
	return allErrors.ErrorOrNil()
}
