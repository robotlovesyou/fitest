package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCanGetConfiguredRPCPort(t *testing.T) {
	t.Setenv(RpcPortVar, "1234")
	port, err := rpcPort()
	require.NoError(t, err)
	require.Equal(t, int32(1234), port)
}

func TestErrorReturnedWithMisconfiguredRPCPort(t *testing.T) {
	t.Setenv(RpcPortVar, "bad value")
	_, err := rpcPort()
	require.Error(t, err)
}

func TestCanGetConfiguredHealthcheckPort(t *testing.T) {
	t.Setenv(HealthPortVar, "1234")
	port, err := healthcheckPort()
	require.NoError(t, err)
	require.Equal(t, int32(1234), port)
}

func TestErrorReturnedWithMisconfiguredHealthcheckPort(t *testing.T) {
	t.Setenv(HealthPortVar, "bad value")
	_, err := rpcPort()
	require.Error(t, err)
}

func TestCanGetConfiguredDatabaseURI(t *testing.T) {
	t.Setenv(DatabaseURIVar, "databaseURI")
	require.Equal(t, "databaseURI", databaseURI())
}
