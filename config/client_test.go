package config

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
)

var (
	r map[string]interface{}
)

func TestClient(t *testing.T) {
	filepath := "examples\\config.json"
	client, err := GetClient(filepath)
	if err != nil {
		t.Log(err)
	}
	res, err := client.Info()
	if err != nil {
		t.Log(err)
	}
	defer res.Body.Close()
	if res.IsError() {
		t.Log(res.String())
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		t.Logf("Error parsing the response body: %s", err)
	}
	t.Logf("Client: %s", elasticsearch.Version)
	t.Logf("Server: %s", r["version"].(map[string]interface{})["number"])
	t.Log(strings.Repeat("~", 37))

	var buf bytes.Buffer
	body := `{
		"query": {
		  "term": {
			"system.syslog.message": "error"
		  }
		}
	  }`
	buf.WriteString(body)
	// if err := json.NewEncoder(&buf).Encode(body); err != nil {
	// 	t.Logf("Error encoding query: %s", err)
	// }

	res, err = client.Search(
		client.Search.WithContext(context.Background()),
		client.Search.WithIndex("test_index"),
		client.Search.WithBody(&buf),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithPretty(),
	)
	if err != nil {
		t.Log(err)
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			t.Logf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and error information.
			t.Logf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		t.Logf("Error parsing the response body: %s", err)
	}
	// Print the response status, number of results, and request duration.
	t.Logf(
		"[%s] %d hits; took: %dms",
		res.Status(),
		int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
		int(r["took"].(float64)),
	)
	// Print the ID and document source for each hit.
	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		t.Logf(" * ID=%s, %s", hit.(map[string]interface{})["_id"], hit.(map[string]interface{})["_source"])
	}

	t.Log(strings.Repeat("=", 37))
}
