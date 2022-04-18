package password

import "golang.org/x/crypto/bcrypt"

// Hasher wraps x/crypto/bcrypt in a user.PasswordHasher compliant interface
type Hasher struct {
	cost int
}

// Hash the provided password, or return an error
func (h Hasher) Hash(plain string) (hash string, err error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	hash = string(hashed)
	return
}

// Compare the provided hash and plaintext passwords
func (h Hasher) Compare(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// New creates a new hasher
func New() Hasher {
	return Hasher{cost: bcrypt.DefaultCost}
}

// NewWeak creates a new hasher suitable for testing, but not production since it will hash quickly, but not very securely
func NewWeak() Hasher {
	return Hasher{cost: bcrypt.MinCost}
}
