package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestForExpectedGreeting(t *testing.T) {
	require.Equal(t, "Hello, World", Greeting())
}
