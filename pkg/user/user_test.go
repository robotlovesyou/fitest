package user_test

import (
	"bytes"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/robotlovesyou/fitest/pkg/password"
	"github.com/robotlovesyou/fitest/pkg/store/userstore"
	"github.com/robotlovesyou/fitest/pkg/user"
	"github.com/robotlovesyou/fitest/pkg/validation"
	"golang.org/x/crypto/bcrypt"
)

const bobbyTables = "Robert'); DROP TABLE Students;--"

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Hand coded stub/mock for UserStore
//// I prefer hand coded stubs where appropriate because the code created by
//// mockgen makes me sad!
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type stubCreate func(context.Context, *userstore.User) (userstore.User, error)
type stubUpdate func(context.Context, *userstore.User) (userstore.User, error)
type stubReadOne func(context.Context, uuid.UUID) (userstore.User, error)
type stubDeleteOne func(context.Context, uuid.UUID) error
type stubFindMany func(context.Context, *userstore.Query) (userstore.Page, error)

type stubUserStore struct {
	stubCreate    stubCreate
	stubUpdate    stubUpdate
	stubReadOne   stubReadOne
	stubDeleteOne stubDeleteOne
	stubFindMany  stubFindMany
}

func newStubUserStore() *stubUserStore {
	return &stubUserStore{
		stubCreate: func(context.Context, *userstore.User) (userstore.User, error) {
			panic("stub create")
		},
		stubUpdate: func(context.Context, *userstore.User) (userstore.User, error) {
			panic("stub update")
		},
		stubReadOne: func(context.Context, uuid.UUID) (userstore.User, error) {
			panic("stub read one")
		},
		stubDeleteOne: func(context.Context, uuid.UUID) error {
			panic("stub delete one")
		},
		stubFindMany: func(context.Context, *userstore.Query) (userstore.Page, error) {
			panic("stub find many")
		},
	}
}

func (store *stubUserStore) Create(ctx context.Context, rec *userstore.User) (userstore.User, error) {
	return store.stubCreate(ctx, rec)
}

func (store *stubUserStore) Update(ctx context.Context, rec *userstore.User) (userstore.User, error) {
	return store.stubUpdate(ctx, rec)
}

func (store *stubUserStore) ReadOne(ctx context.Context, id uuid.UUID) (userstore.User, error) {
	return store.stubReadOne(ctx, id)
}

func (store *stubUserStore) DeleteOne(ctx context.Context, id uuid.UUID) error {
	return store.stubDeleteOne(ctx, id)
}

func (store *stubUserStore) FindMany(ctx context.Context, query *userstore.Query) (userstore.Page, error) {
	return store.stubFindMany(ctx, query)
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////
//// Test helper functions
////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type option interface {
	isoption()
}

type hasherOpt struct {
	hasher user.PasswordHasher
}

func useHasher(hasher user.PasswordHasher) hasherOpt {
	return hasherOpt{hasher: hasher}
}

func (ho hasherOpt) isoption() {}

// badHasher implements user.PasswordHasher and fails for all calls
type badHasher struct{}

func (bh badHasher) Hash(string) (string, error) {
	return "", errors.New("failed")
}

func (bh badHasher) Compare(string, string) bool {
	return false
}

type idGenOpt struct {
	idGenerator user.IDGenerator
}

func useIDGenerator(idGenerator user.IDGenerator) idGenOpt {
	return idGenOpt{idGenerator: idGenerator}
}

func (igo idGenOpt) isoption() {}

func withService(store *stubUserStore, options ...option) func(func(*user.Service)) {
	hasher := user.PasswordHasher(password.NewWeak())
	idGenerator := uuid.NewRandom

	for _, o := range options {
		switch opt := o.(type) {
		case hasherOpt:
			hasher = opt.hasher
		case idGenOpt:
			idGenerator = opt.idGenerator
		}
	}

	return func(f func(service *user.Service)) {
		f(user.New(store, hasher, idGenerator, validation.New()))
	}
}

func emptyID(id [16]byte) bool {
	var emptyID [16]byte
	return compareIDs(id, emptyID)
}

func compareIDs(a [16]byte, b [16]byte) bool {
	return bytes.Equal(a[:], b[:])
}

func checkPasswordHash(hashed, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}
