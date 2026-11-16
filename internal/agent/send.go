package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/dodozg/daljinac/internal/metrics"
)

// ReportURL builds the report endpoint from the server's base URL. Plain HTTP
// is refused except for loopback addresses, so a token can never travel
// unencrypted across a network.
func ReportURL(base string) (string, error) {
	u, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return "", err
	}
	switch u.Scheme {
	case "https":
	case "http":
		host := u.Hostname()
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return "", errors.New("server URL must use https (plain http is allowed only for localhost)")
		}
	default:
		return "", fmt.Errorf("server URL must start with https://, got %q", base)
	}
	if u.Host == "" {
		return "", fmt.Errorf("server URL %q has no host", base)
	}
	return u.String() + "/api/v1/report", nil
}

// Send posts one report.
func Send(ctx context.Context, c *http.Client, reportURL, token string, r metrics.Report) error {
	body, err := json.Marshal(r)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reportURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("server answered %s: %s", resp.Status, strings.TrimSpace(string(msg)))
	}
	return nil
}
