package query

import (
	"encoding/json"
	"strings"

	"github.com/lbzss/elasticsearch-alert/command/alert"
	"github.com/lbzss/elasticsearch-alert/config"
	"github.com/lbzss/elasticsearch-alert/utils"
	"github.com/mitchellh/mapstructure"
)

const hitsDelimiter = "\n----------------------------------------\n"

func (q *QueryHandler) process(respData map[string]interface{}) ([]*alert.Record, []map[string]interface{}, error) {
	if len(q.conditions) != 0 && !config.ConditionsMet(respData, q.conditions) {
		return nil, nil, nil
	}

	records := make([]*alert.Record, 0)
	for _, filter := range q.filters {
		elems := utils.GetAll(respData, filter)
		if elems == nil || len(elems) < 1 {
			continue
		}

		fields, err := q.gatherFields(elems)
		if err != nil {
			return nil, nil, err
		}

		if len(fields) < 1 {
			continue
		}

		record := &alert.Record{
			Filter: filter,
			Fields: fields,
		}

		records = append(records, record)
	}

	body := utils.GetAll(respData, q.bodyField)
	if body == nil {
		return records, nil, nil
	}

	stringfieldHits, hits, err := q.gatherHits(body)
	if err != nil {
		return records, nil, err
	}
	if len(stringfieldHits) > 0 {
		record := &alert.Record{
			Filter:    q.bodyField,
			Text:      strings.Join(stringfieldHits, hitsDelimiter),
			BodyField: true,
		}
		records = append(records, record)
	}
	return records, hits, nil
}

func (q *QueryHandler) gatherHits(body []interface{}) ([]string, []map[string]interface{}, error) {
	stringfieldHits := make([]string, 0, len(body))
	hits := make([]map[string]interface{}, 0, len(body))
	for _, elem := range body {
		hit, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}
		hits = append(hits, hit)

		data, err := json.MarshalIndent(hit, "", "    ")
		if err != nil {
			return nil, nil, err
		}
		stringfieldHits = append(stringfieldHits, string(data))
	}
	return stringfieldHits, hits, nil
}

func (q *QueryHandler) gatherFields(elems []interface{}) ([]*alert.Field, error) {
	fields := make([]*alert.Field, 0, len(elems))
	for _, elem := range elems {
		obj, ok := elem.(map[string]interface{})
		if !ok {
			continue
		}

		field := new(alert.Field)
		if err := mapstructure.Decode(obj, field); err != nil {
			return nil, err
		}

		if field.Key == "" || field.Count < 1 {
			continue
		}
		fields = append(fields, field)
	}
	return fields, nil
}
