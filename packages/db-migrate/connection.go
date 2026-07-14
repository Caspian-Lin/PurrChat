package migrate

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
)

var postgresRolePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

// AdminRoleConfig connects with an administrator credential but immediately
// assumes Role. PostgreSQL therefore records migrated objects as owned by the
// application role instead of the administrator.
type AdminRoleConfig struct {
	Host          string
	Port          string
	Database      string
	AdminUser     string
	AdminPassword string
	Role          string
}

func AdminRoleDSN(cfg AdminRoleConfig) (string, error) {
	if !postgresRolePattern.MatchString(cfg.Role) {
		return "", fmt.Errorf("invalid PostgreSQL application role %q", cfg.Role)
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.AdminUser, cfg.AdminPassword),
		Host:   net.JoinHostPort(cfg.Host, cfg.Port),
		Path:   cfg.Database,
	}
	query := u.Query()
	query.Set("timezone", "UTC")
	query.Set("options", "-c role="+cfg.Role)
	u.RawQuery = query.Encode()
	return u.String(), nil
}
