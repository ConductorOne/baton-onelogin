package connector

import (
	"testing"

	"github.com/conductorone/baton-onelogin/pkg/onelogin"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

func newTestUser() *onelogin.User {
	u := &onelogin.User{
		Username:   "jdoe",
		Email:      "jdoe@example.com",
		Firstname:  "Jane",
		Lastname:   "Doe",
		Status:     1,
		Title:      "Staff Engineer",
		Department: "Engineering",
	}
	u.Id = 42
	return u
}

func TestBuildUserProfile_IncludesJobTitleAndDepartment(t *testing.T) {
	user := newTestUser()

	profile, _ := buildUserProfile(resolveDisplayName(user), user)

	if got := profile["job_title"]; got != "Staff Engineer" {
		t.Errorf("expected job_title=%q, got %v", "Staff Engineer", got)
	}
	if got := profile["department"]; got != "Engineering" {
		t.Errorf("expected department=%q, got %v", "Engineering", got)
	}

	// Existing profile keys must keep working.
	if got := profile["login"]; got != "jdoe" {
		t.Errorf("expected login=%q, got %v", "jdoe", got)
	}
	if got := profile["user_id"]; got != "42" {
		t.Errorf("expected user_id=%q, got %v", "42", got)
	}
	if got := profile["first_name"]; got != "Jane" {
		t.Errorf("expected first_name=%q, got %v", "Jane", got)
	}
	if got := profile["last_name"]; got != "Doe" {
		t.Errorf("expected last_name=%q, got %v", "Doe", got)
	}
}

func TestBuildUserProfile_OmitsEmptyJobTitleAndDepartment(t *testing.T) {
	for _, tc := range []struct {
		name       string
		title      string
		department string
	}{
		{name: "empty", title: "", department: ""},
		{name: "whitespace only", title: "   ", department: "\t\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			user := newTestUser()
			user.Title = tc.title
			user.Department = tc.department

			profile, _ := buildUserProfile(resolveDisplayName(user), user)

			if _, ok := profile["job_title"]; ok {
				t.Errorf("expected job_title to be omitted, got %v", profile["job_title"])
			}
			if _, ok := profile["department"]; ok {
				t.Errorf("expected department to be omitted, got %v", profile["department"])
			}
		})
	}
}

func TestBuildUserProfile_TrimsJobTitleAndDepartment(t *testing.T) {
	user := newTestUser()
	user.Title = "  Staff Engineer  "
	user.Department = "  Engineering  "

	profile, _ := buildUserProfile(resolveDisplayName(user), user)

	if got := profile["job_title"]; got != "Staff Engineer" {
		t.Errorf("expected trimmed job_title=%q, got %v", "Staff Engineer", got)
	}
	if got := profile["department"]; got != "Engineering" {
		t.Errorf("expected trimmed department=%q, got %v", "Engineering", got)
	}
}

func TestBuildUserProfile_ManagerFields(t *testing.T) {
	user := newTestUser()
	managerID := 99
	user.ManagerId = &managerID
	user.ManagerEmail = "boss@example.com"

	profile, _ := buildUserProfile(resolveDisplayName(user), user)

	if got := profile["manager_user_id"]; got != "99" {
		t.Errorf("expected manager_user_id=%q, got %v", "99", got)
	}
	if got := profile["manager_email"]; got != "boss@example.com" {
		t.Errorf("expected manager_email=%q, got %v", "boss@example.com", got)
	}
}

func TestParseIntoUserResource_ProfileCarriesJobTitleAndDepartment(t *testing.T) {
	user := newTestUser()

	resource, err := parseIntoUserResource(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := rs.GetUserTrait(resource); err != nil {
		t.Fatalf("unexpected error reading user trait: %v", err)
	}

	fields := resource.GetProfile().GetFields()
	if got := fields["job_title"].GetStringValue(); got != "Staff Engineer" {
		t.Errorf("expected job_title=%q in resource profile, got %q", "Staff Engineer", got)
	}
	if got := fields["department"].GetStringValue(); got != "Engineering" {
		t.Errorf("expected department=%q in resource profile, got %q", "Engineering", got)
	}
}
