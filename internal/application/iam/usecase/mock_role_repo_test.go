package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var (
	errMockNotFound = domainerr.New(domainerr.CodeNotFound, "not found", nil)
	errMockConflict = domainerr.New(domainerr.CodeConflict, "conflict", nil)
)

// mockRoleRepo is an in-memory RoleRepository for unit tests.
type mockRoleRepo struct {
	byName      map[string]*iammodel.Role
	userRoles   map[uuid.UUID][]uuid.UUID
	assignCalls int
}

func newMockRoleRepo() *mockRoleRepo {
	userRole := &iammodel.Role{
		ID:          uuid.New(),
		Name:        "user",
		Permissions: []iammodel.Permission{"own:read", "own:write"},
	}
	return &mockRoleRepo{
		byName:    map[string]*iammodel.Role{"user": userRole},
		userRoles: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockRoleRepo) FindByName(_ context.Context, name string) (*iammodel.Role, error) {
	if r, ok := m.byName[name]; ok {
		return r, nil
	}
	return nil, errMockNotFound
}

func (m *mockRoleRepo) FindByUserID(_ context.Context, userID uuid.UUID) ([]iammodel.Role, error) {
	ids := m.userRoles[userID]
	roles := make([]iammodel.Role, 0, len(ids))
	for _, id := range ids {
		for _, r := range m.byName {
			if r.ID == id {
				roles = append(roles, *r)
			}
		}
	}
	return roles, nil
}

func (m *mockRoleRepo) AssignToUser(_ context.Context, userID, roleID uuid.UUID) error {
	m.assignCalls++
	for _, id := range m.userRoles[userID] {
		if id == roleID {
			return nil
		}
	}
	m.userRoles[userID] = append(m.userRoles[userID], roleID)
	return nil
}

// mockRefreshRepo is an in-memory RefreshTokenRepository for unit tests.
type mockRefreshRepo struct {
	saved     []*iammodel.RefreshToken
	revoked   map[uuid.UUID]bool
	listByErr error // optional: inject an error for ListByUser
}

func newMockRefreshRepo() *mockRefreshRepo {
	return &mockRefreshRepo{revoked: make(map[uuid.UUID]bool)}
}

func (m *mockRefreshRepo) Create(_ context.Context, token *iammodel.RefreshToken) error {
	m.saved = append(m.saved, token)
	return nil
}

func (m *mockRefreshRepo) FindByHash(_ context.Context, hash string) (*iammodel.RefreshToken, error) {
	for _, t := range m.saved {
		if t.TokenHash == hash {
			return t, nil
		}
	}
	return nil, errMockNotFound
}

func (m *mockRefreshRepo) Revoke(_ context.Context, id uuid.UUID) error {
	for _, t := range m.saved {
		if t.ID == id {
			if m.revoked[id] {
				return errMockNotFound // mimic "already revoked" -> NotFound
			}
			m.revoked[id] = true
			now := time.Now().UTC()
			t.RevokedAt = &now
			return nil
		}
	}
	return errMockNotFound
}

func (m *mockRefreshRepo) RevokeAllForUser(_ context.Context, userID uuid.UUID) error {
	for _, t := range m.saved {
		if t.UserID == userID {
			m.revoked[t.ID] = true
			now := time.Now().UTC()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockRefreshRepo) UpdateLastSeen(_ context.Context, id uuid.UUID, at time.Time) error {
	for _, t := range m.saved {
		if t.ID == id {
			t.LastSeenAt = at
			return nil
		}
	}
	return errMockNotFound
}

func (m *mockRefreshRepo) ListByUser(_ context.Context, userID uuid.UUID) ([]iammodel.RefreshToken, error) {
	if m.listByErr != nil {
		return nil, m.listByErr
	}
	out := []iammodel.RefreshToken{}
	for _, t := range m.saved {
		if t.UserID == userID {
			out = append(out, *t)
		}
	}
	return out, nil
}
