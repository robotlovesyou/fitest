package password_test

import (
	"testing"

	"github.com/robotlovesyou/fitest/pkg/password"
	"github.com/stretchr/testify/require"
)

func TestNewWeakCreatesValidHashes(t *testing.T) {
	pwd := "password"
	nw := password.NewWeak()
	hash, err := nw.Hash(pwd)
	require.NoError(t, err)
	require.True(t, nw.Compare(hash, pwd))
}

func TestNewCreatesValidHashes(t *testing.T) {
	pwd := "password"
	n := password.New()
	hash, err := n.Hash(pwd)
	require.NoError(t, err)
	require.True(t, n.Compare(hash, pwd))
}
