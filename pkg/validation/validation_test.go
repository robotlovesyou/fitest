package validation_test

import (
	"testing"

	"github.com/robotlovesyou/fitest/pkg/validation"
	"github.com/stretchr/testify/require"
)

type testAllowedRunes struct {
	Value string `validate:"allowed-runes"`
}

func TestAllowedRunesPassesValidString(t *testing.T) {
	v := validation.New()
	err := v.Struct(&testAllowedRunes{
		Value: "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ0123456789-_ ",
	})
	require.NoError(t, err)
}

func TestAllowedRunesFailesInvalidString(t *testing.T) {
	v := validation.New()
	err := v.Struct(&testAllowedRunes{
		Value: "\";'*%@!",
	})
	require.Error(t, err)
}
