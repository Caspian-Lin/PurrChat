package migrate

import (
	"net/url"
	"testing"
)

func TestAdminRoleDSN(t *testing.T) {
	dsn, err := AdminRoleDSN(AdminRoleConfig{
		Host: "localhost", Port: "5432", Database: "purrchat", AdminUser: "postgres",
		AdminPassword: "p@ss:word", Role: "purrchat_app",
	})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if password != "p@ss:word" || parsed.Query().Get("options") != "-c role=purrchat_app" {
		t.Fatalf("admin credential or role was not encoded correctly: %s", dsn)
	}
}

func TestAdminRoleDSNRejectsUnsafeRole(t *testing.T) {
	_, err := AdminRoleDSN(AdminRoleConfig{Role: `app; RESET ROLE`})
	if err == nil {
		t.Fatal("expected unsafe role to be rejected")
	}
}
