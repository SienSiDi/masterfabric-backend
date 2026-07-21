package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
)

// mockUserRepo is an in-memory UserRepository for unit tests.
type mockUserRepo struct {
	byEmail map[string]*iammodel.User
	byID    map[uuid.UUID]*iammodel.User
	// nextID lets tests pre-set IDs deterministically
	nextID uuid.UUID
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		byEmail: make(map[string]*iammodel.User),
		byID:    make(map[uuid.UUID]*iammodel.User),
		nextID:  uuid.New(),
	}
}

func (m *mockUserRepo) Create(_ context.Context, email, passwordHash string) (*iammodel.User, error) {
	if _, exists := m.byEmail[email]; exists {
		// mimic a domain conflict — return nil; the use case also checks FindByEmail first
		return nil, errMockConflict
	}
	u := &iammodel.User{
		ID:           m.nextID,
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now().UTC(),
	}
	m.nextID = uuid.New()
	m.byEmail[email] = u
	m.byID[u.ID] = u
	return u, nil
}

func (m *mockUserRepo) FindByEmail(_ context.Context, email string) (*iammodel.User, error) {
	if u, ok := m.byEmail[email]; ok {
		return u, nil
	}
	return nil, errMockNotFound
}

func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*iammodel.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, errMockNotFound
}

func (m *mockUserRepo) UpdateEmail(_ context.Context, id uuid.UUID, newEmail string) (*iammodel.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, errMockNotFound
	}
	// check for email conflict
	for _, existing := range m.byEmail {
		if existing.Email == newEmail && existing.ID != id {
			return nil, errMockConflict
		}
	}
	delete(m.byEmail, u.Email)
	u.Email = newEmail
	m.byEmail[newEmail] = u
	return u, nil
}

func (m *mockUserRepo) UpdatePassword(_ context.Context, id uuid.UUID, newPasswordHash string) error {
	u, ok := m.byID[id]
	if !ok {
		return errMockNotFound
	}
	u.PasswordHash = newPasswordHash
	return nil
}
