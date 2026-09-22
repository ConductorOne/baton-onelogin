package onelogin

import (
	"encoding/json"
	"net/url"
	"strings"
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

func TestUserFields_IncludesTitleAndDepartment(t *testing.T) {
	params := url.Values{}
	prepareUserFilters().setup(&params)

	fields := strings.Split(params.Get("fields"), ",")
	have := make(map[string]bool, len(fields))
	for _, f := range fields {
		have[f] = true
	}

	// The users list endpoint only returns the fields we explicitly request,
	// so title and department must be part of the filter.
	for _, want := range []string{"id", "email", "username", "firstname", "lastname", "status", "group_id", "title", "department"} {
		if !have[want] {
			t.Errorf("expected user fields filter to include %q, got %q", want, params.Get("fields"))
		}
	}
}

func TestUser_UnmarshalTitleAndDepartment(t *testing.T) {
	const payload = `{"id":42,"username":"jdoe","email":"jdoe@example.com","firstname":"Jane","lastname":"Doe","status":1,"title":"Staff Engineer","department":"Engineering"}`

	var user User
	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if user.Title != "Staff Engineer" {
		t.Errorf("expected Title=%q, got %q", "Staff Engineer", user.Title)
	}
	if user.Department != "Engineering" {
		t.Errorf("expected Department=%q, got %q", "Engineering", user.Department)
	}
}

func TestUser_UnmarshalMissingTitleAndDepartment(t *testing.T) {
	const payload = `{"id":42,"username":"jdoe","email":"jdoe@example.com","status":1}`

	var user User
	if err := json.Unmarshal([]byte(payload), &user); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if user.Title != "" {
		t.Errorf("expected empty Title, got %q", user.Title)
	}
	if user.Department != "" {
		t.Errorf("expected empty Department, got %q", user.Department)
	}
}
