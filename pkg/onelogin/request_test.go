package onelogin

import (
	"encoding/json"
	"net/url"
	"testing"
	"time"
)

func TestMappingsSyncParam_Setup(t *testing.T) {
	params := url.Values{}
	p := &MappingsSyncParam{}
	p.setup(&params)

	if got := params.Get("mappings"); got != "sync" {
		t.Errorf("expected mappings=sync, got mappings=%s", got)
	}
}

func TestUserUpdatePayload_Marshal(t *testing.T) {
	ts := time.Now().Unix()
	payload := UserUpdatePayload{
		CustomAttributes: map[string]interface{}{
			"c1_last_action": ts,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	attrs, ok := result["custom_attributes"].(map[string]interface{})
	if !ok {
		t.Fatal("expected custom_attributes to be a map")
	}
	if _, ok := attrs["c1_last_action"]; !ok {
		t.Error("expected c1_last_action key in custom_attributes")
	}
}
