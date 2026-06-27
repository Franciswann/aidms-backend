package user

import (
	"errors"
	"testing"

	"github.com/Franciswann/aidms-backend/internal/domain/entity"
	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is an in-memory fake satisfying domainrepo.UserRepository,
// so UserService can be tested without a real database.
type mockUserRepository struct {
	usersByEmail  map[string]*entity.User
	saveErr       error
	findErr       error // when set, forces FindByEmail to fail with this error
	saveCallCount int
}

var _ domainrepo.UserRepository = (*mockUserRepository)(nil)

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{usersByEmail: make(map[string]*entity.User)}
}

func (m *mockUserRepository) Save(u *entity.User) error {
	m.saveCallCount++
	if m.saveErr != nil {
		return m.saveErr
	}
	m.usersByEmail[u.Email] = u
	return nil
}

func (m *mockUserRepository) FindByID(id string) (*entity.User, error) {
	for _, u := range m.usersByEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domainrepo.ErrNotFound
}

func (m *mockUserRepository) FindByEmail(email string) (*entity.User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	u, ok := m.usersByEmail[email]
	if !ok {
		return nil, domainrepo.ErrNotFound
	}
	return u, nil
}

// errDBFailure stands in for "something went wrong talking to the database"
// that is NOT a not-found case - e.g. a dropped connection.
var errDBFailure = errors.New("simulated db failure")

func TestUserService_Register(t *testing.T) {
	tests := []struct {
		name      string
		seedEmail string // if set, pre-populate this email so it's "already registered"
		findErr   error
		saveErr   error
		email     string
		password  string
		wantErr   error // nil means "expect success"
	}{
		{
			name:     "new email registers successfully",
			email:    "new@example.com",
			password: "password123",
			wantErr:  nil,
		},
		{
			name:      "email already registered",
			seedEmail: "exists@example.com",
			email:     "exists@example.com",
			password:  "password123",
			wantErr:   ErrEmailAlreadyRegistered,
		},
		{
			name:     "FindByEmail fails with a non-not-found error",
			findErr:  errDBFailure,
			email:    "whatever@example.com",
			password: "password123",
			wantErr:  errDBFailure,
		},
		{
			name:     "Save fails",
			saveErr:  errDBFailure,
			email:    "new2@example.com",
			password: "password123",
			wantErr:  errDBFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepository()
			if tt.seedEmail != "" {
				repo.usersByEmail[tt.seedEmail] = &entity.User{
					ID:             "existing-id",
					Email:          tt.seedEmail,
					HashedPassword: "existing-hash",
				}
			}
			repo.findErr = tt.findErr
			repo.saveErr = tt.saveErr

			svc := NewUserService(repo, "test-secret")
			got, err := svc.Register(tt.email, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, got)
				if tt.seedEmail != "" {
					// the existing user must be untouched - Save should never
					// have been called for a duplicate-email registration
					assert.Equal(t, 0, repo.saveCallCount)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.email, got.Email)
			assert.NotEmpty(t, got.ID)

			// the password must actually be hashed, not stored as-is, and the
			// hash must be verifiable against the original plaintext password
			assert.NotEqual(t, tt.password, got.HashedPassword)
			assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(got.HashedPassword), []byte(tt.password)))

			// Save must have actually persisted the same user we got back
			saved, ok := repo.usersByEmail[tt.email]
			require.True(t, ok)
			assert.Equal(t, got.ID, saved.ID)
		})
	}
}

func TestUserService_Login(t *testing.T) {
	const (
		seededEmail    = "user@example.com"
		seededPassword = "correct-password"
		jwtSecret      = "test-secret"
	)
	hashed, err := bcrypt.GenerateFromPassword([]byte(seededPassword), bcrypt.DefaultCost)
	require.NoError(t, err)
	seededUserID := "seeded-user-id"

	newSeededRepo := func() *mockUserRepository {
		repo := newMockUserRepository()
		repo.usersByEmail[seededEmail] = &entity.User{
			ID:             seededUserID,
			Email:          seededEmail,
			HashedPassword: string(hashed),
		}
		return repo
	}

	tests := []struct {
		name     string
		repo     *mockUserRepository
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "correct email and password returns a token",
			repo:     newSeededRepo(),
			email:    seededEmail,
			password: seededPassword,
			wantErr:  nil,
		},
		{
			name:     "email does not exist",
			repo:     newSeededRepo(),
			email:    "nobody@example.com",
			password: seededPassword,
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "email exists but password is wrong",
			repo:     newSeededRepo(),
			email:    seededEmail,
			password: "wrong-password",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name: "FindByEmail fails with a non-not-found error",
			repo: func() *mockUserRepository {
				r := newSeededRepo()
				r.findErr = errDBFailure
				return r
			}(),
			email:    seededEmail,
			password: seededPassword,
			wantErr:  errDBFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserService(tt.repo, jwtSecret)
			token, err := svc.Login(tt.email, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Empty(t, token)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, token)

			// the token must actually be valid and carry the right user_id,
			// not just be "some non-empty string"
			parsed, err := jwt.Parse(token, func(*jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			require.NoError(t, err)
			claims, ok := parsed.Claims.(jwt.MapClaims)
			require.True(t, ok)
			assert.Equal(t, seededUserID, claims["user_id"])
		})
	}
}
