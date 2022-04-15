package password

import "golang.org/x/crypto/bcrypt"

// Hasher wraps x/crypto/bcrypt in a user.PasswordHasher compliant interface
type Hasher struct {
	cost int
}

func (h Hasher) Hash(plain string) (hash string, err error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	hash = string(hashed)
	return
}

func (h Hasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func New() Hasher {
	return Hasher{cost: bcrypt.DefaultCost}
}

func NewWeak() Hasher {
	return Hasher{cost: bcrypt.MinCost}
}
