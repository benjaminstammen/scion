//go:build !no_sqlite

// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package hub

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/scion/pkg/store"
	"github.com/GoogleCloudPlatform/scion/pkg/store/sqlite"
)

func newTestServerWithStore(t *testing.T) (*Server, store.Store) {
	t.Helper()
	s, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create sqlite store: %v", err)
	}
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	srv := &Server{store: s}
	return srv, s
}

func TestListMaintenanceOperations(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/maintenance/operations", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceOps(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should have both migrations and operations keys.
	if _, ok := body["migrations"]; !ok {
		t.Error("response missing 'migrations' key")
	}
	if _, ok := body["operations"]; !ok {
		t.Error("response missing 'operations' key")
	}
}

func TestListMaintenanceOperations_NonAdmin(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	member := NewAuthenticatedUser("u1", "member@example.com", "Member", "member", "cli")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/maintenance/operations", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), member))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceOps(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestExecuteMigration_NonAdmin(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	member := NewAuthenticatedUser("u1", "member@example.com", "Member", "member", "cli")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/secret-hub-id-migration/run", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), member))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestExecuteMigration_NotFound(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/nonexistent/run", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteMigration_AlreadyCompleted(t *testing.T) {
	srv, s := newTestServerWithStore(t)

	// Mark the migration as completed.
	op, err := s.GetMaintenanceOperation(context.Background(), "secret-hub-id-migration")
	if err != nil {
		t.Fatalf("failed to get operation: %v", err)
	}
	now := time.Now()
	op.Status = store.MaintenanceStatusCompleted
	op.CompletedAt = &now
	if err := s.UpdateMaintenanceOperation(context.Background(), op); err != nil {
		t.Fatalf("failed to update operation: %v", err)
	}

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/secret-hub-id-migration/run", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteMigration_AlreadyRunning(t *testing.T) {
	srv, s := newTestServerWithStore(t)

	// Mark the migration as running.
	op, err := s.GetMaintenanceOperation(context.Background(), "secret-hub-id-migration")
	if err != nil {
		t.Fatalf("failed to get operation: %v", err)
	}
	now := time.Now()
	op.Status = store.MaintenanceStatusRunning
	op.StartedAt = &now
	if err := s.UpdateMaintenanceOperation(context.Background(), op); err != nil {
		t.Fatalf("failed to update operation: %v", err)
	}

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/secret-hub-id-migration/run", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteMigration_NoSecretBackend(t *testing.T) {
	srv, _ := newTestServerWithStore(t)
	// No secret backend configured → should return error.

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	body := `{"params":{"dryRun":true}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/secret-hub-id-migration/run",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteMigration_OperationNotMigration(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	// Try to run a routine operation through the migrations endpoint.
	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/pull-images/run", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteMigration_MethodNotAllowed(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/maintenance/migrations/secret-hub-id-migration/run", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExecuteMigration_InvalidPath(t *testing.T) {
	srv, _ := newTestServerWithStore(t)

	admin := NewAuthenticatedUser("u1", "admin@example.com", "Admin", "admin", "cli")

	// Missing /run suffix
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/maintenance/migrations/secret-hub-id-migration", nil)
	req = req.WithContext(contextWithIdentity(req.Context(), admin))
	rr := httptest.NewRecorder()
	srv.handleAdminMaintenanceMigrations(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing /run, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestParseMigrationParams(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]interface{}
		wantDry string
	}{
		{
			name:    "empty",
			body:    nil,
			wantDry: "",
		},
		{
			name:    "dryRun true",
			body:    map[string]interface{}{"params": map[string]interface{}{"dryRun": true}},
			wantDry: "true",
		},
		{
			name:    "dryRun false",
			body:    map[string]interface{}{"params": map[string]interface{}{"dryRun": false}},
			wantDry: "",
		},
		{
			name:    "no params key",
			body:    map[string]interface{}{"other": "value"},
			wantDry: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := parseMigrationParams(tt.body)
			if got := params["dryRun"]; got != tt.wantDry {
				t.Errorf("parseMigrationParams() dryRun = %q, want %q", got, tt.wantDry)
			}
		})
	}
}
