package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"tuxee/internal/identity"
	"tuxee/internal/update"
)

type updateFixture struct {
	Failure, Slow bool
	Latest        string
}
type updateTransport func(*http.Request) (*http.Response, error)

func (f updateTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Only the test executable substitutes official downloads; production has no URL override.
func updateTestWorker() {
	var fixture updateFixture
	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), "fixture.json"))
	if err != nil || json.Unmarshal(data, &fixture) != nil {
		os.Exit(2)
	}
	binary := "#!/bin/sh\necho 'Tuxee 1.1.0'\n"
	http.DefaultTransport = updateTransport(func(r *http.Request) (*http.Response, error) {
		var body string
		switch r.URL.String() {
		case "https://api.github.com/repos/agent0ai/tuxee/releases/latest":
			body = fmt.Sprintf(`{"tag_name":%q}`, fixture.Latest)
		case "https://github.com/agent0ai/tuxee/releases/download/v1.1.0/checksums.txt":
			body = fmt.Sprintf("%x  tuxee-linux-%s\n", sha256.Sum256([]byte(binary)), runtime.GOARCH)
			if fixture.Failure {
				body = strings.Repeat("0", 64) + "  tuxee-linux-" + runtime.GOARCH
			}
		case "https://github.com/agent0ai/tuxee/releases/download/v1.1.0/tuxee-linux-" + runtime.GOARCH:
			if fixture.Slow {
				_ = os.WriteFile(filepath.Join(os.Getenv("HOME"), "downloading"), nil, 0600)
				<-r.Context().Done()
				return nil, r.Context().Err()
			}
			body = binary
		default:
			return nil, fmt.Errorf("unexpected download %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})
	if update.Worker() != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func updateServer(t *testing.T, user identity.User, fixture updateFixture) (*Server, string) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("credential-dropped worker requires root")
	}
	// Direct /tmp child lets credential-dropped workers traverse the fixture.
	dir, err := os.MkdirTemp("", "tuxee-update-api-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "tuxee")
	if err := os.WriteFile(target, []byte("#!/bin/sh\necho 'Tuxee 1.0.0'\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if fixture.Latest == "" {
		fixture.Latest = "v1.1.0"
	}
	data, _ := json.Marshal(fixture)
	if err := os.WriteFile(filepath.Join(dir, "fixture.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	user.Home = dir
	s := New("test-instance", nil, nil, nil, "", fstest.MapFS{})
	s.Updates = update.NewChecker("1.0.0")
	s.updatePath = target
	s.Restart = func() {}
	ctx, cancel := context.WithCancel(context.Background())
	s.logins[sha256.Sum256([]byte("token"))] = &login{user: user, csrf: "csrf", ctx: ctx, cancel: cancel, expires: time.Now().Add(time.Hour)}
	t.Cleanup(s.Close)
	return s, target
}
func updateRequest(s *Server, method, path string, authenticated, csrf bool, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://tuxee.test"+path, nil)
	r.Header.Set("Origin", origin)
	if authenticated {
		r.AddCookie(&http.Cookie{Name: s.cookieName, Value: "token"})
	}
	if csrf {
		r.Header.Set("X-Tuxee-CSRF", "csrf")
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestUpdateChecksRequireSessionAndCSRF(t *testing.T) {
	s, target := updateServer(t, identity.User{UID: uint32(os.Getuid()), GID: uint32(os.Getgid())}, updateFixture{})
	for _, tc := range []struct {
		method, path string
		auth, csrf   bool
		origin       string
		code         int
	}{
		{"GET", "/api/update", false, false, "", 401},
		{"POST", "/api/update", true, false, "", 403},
		{"POST", "/api/update/apply", false, false, "http://tuxee.test", 401},
		{"POST", "/api/update/apply", true, false, "http://tuxee.test", 403},
		{"POST", "/api/update/apply", true, true, "http://evil.test", 403},
		{"GET", "/api/update/apply", true, true, "", 404},
		{"PUT", "/api/update", true, true, "", 404},
	} {
		if w := updateRequest(s, tc.method, tc.path, tc.auth, tc.csrf, tc.origin); w.Code != tc.code {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body)
		}
	}
	w := updateRequest(s, "GET", "/api/update", true, false, "")
	var status struct {
		Current, Command        string
		CanApply, RestartNeeded bool
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || status.Current != "1.0.0" || !status.CanApply || status.RestartNeeded || status.Command != "'"+target+"' update" {
		t.Fatal(w.Code, status, w.Body)
	}
	t.Setenv("PATH", filepath.Dir(target))
	w = updateRequest(s, "GET", "/api/update", true, false, "")
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil || status.Command != "tuxee update" {
		t.Fatal(status, err)
	}
	s.updateApplying.Store(true)
	if w := updateRequest(s, "POST", "/api/update/apply", true, true, ""); w.Code != 409 {
		t.Fatal(w.Code, w.Body)
	}
	s.updateApplying.Store(false)
}

func TestUpdateApplyIsDisabled(t *testing.T) {
	s, target := updateServer(t, identity.User{UID: uint32(os.Getuid()), GID: uint32(os.Getgid())}, updateFixture{})
	before, _ := os.ReadFile(target)
	w := updateRequest(s, "POST", "/api/update/apply", true, true, "")
	if w.Code != 500 || !strings.Contains(w.Body.String(), update.ErrDisabled.Error()) {
		t.Fatal(w.Code, w.Body)
	}
	after, _ := os.ReadFile(target)
	if string(before) != string(after) || s.closed.Load() {
		t.Fatal("disabled update changed installation or restarted")
	}
}

func TestUpdateRequiresRestartBeforeInstalling(t *testing.T) {
	s, target := updateServer(t, identity.User{UID: uint32(os.Getuid()), GID: uint32(os.Getgid())}, updateFixture{})
	s.Restart = nil
	before, _ := os.ReadFile(target)
	w := updateRequest(s, "POST", "/api/update/apply", true, true, "")
	if w.Code != 503 {
		t.Fatal(w.Code, w.Body)
	}
	after, _ := os.ReadFile(target)
	if string(before) != string(after) || s.closed.Load() {
		t.Fatal("unavailable restart changed installation")
	}
	w = updateRequest(s, "GET", "/api/update", true, false, "")
	var status struct{ CanApply bool }
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil || status.CanApply {
		t.Fatal(w.Body, err)
	}
}
