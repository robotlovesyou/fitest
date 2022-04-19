package health_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/robotlovesyou/fitest/pkg/health"
	"github.com/robotlovesyou/fitest/pkg/log"
	"github.com/stretchr/testify/require"
)

const (
	testTimeout = 30 * time.Second
	path        = "/healthy"
)

type stubMonitor struct {
	name   string
	result error
}

func (sm *stubMonitor) Name() string {
	return sm.name
}

func (sm *stubMonitor) Check(context.Context) error {
	return sm.result
}

func happyMonitor(name string) *stubMonitor {
	return &stubMonitor{name: name}
}

func sadMonitor(name string, result error) *stubMonitor {
	return &stubMonitor{name: name, result: result}
}

func withService(monitors ...health.Monitor) func(func(context.Context, string)) {
	return func(f func(context.Context, string)) {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		defer cancel()

		lis, err := net.Listen("tcp", "localhost:0")
		if err != nil {
			panic(fmt.Errorf("cannot listen on open port: %w", err))
		}
		logger, err := log.New("health tests")
		if err != nil {
			panic(err)
		}
		service := health.New(logger, monitors...)
		mux := http.NewServeMux()
		mux.HandleFunc(path, service.Handle)
		go func() {
			http.Serve(lis, mux)
		}()
		f(ctx, lis.Addr().String())
	}
}

func TestHealthReturnsOKWithAllHealthyMonitors(t *testing.T) {
	withService(happyMonitor("a"), happyMonitor("b"))(func(ctx context.Context, addr string) {
		var r health.Result
		client := resty.New()
		res, err := client.R().SetResult(&r).SetError(&r).Get(fmt.Sprintf("http://%s%s", addr, path))
		t.Logf("%+v", r)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode())
		require.True(t, r.OK)
		require.Len(t, r.Results, 2)
		for _, res := range r.Results {
			require.True(t, res.OK)
		}
	})
}

func TestHealthReturnsNotOKWithAnUnHealthyMonitor(t *testing.T) {
	withService(happyMonitor("a"), sadMonitor("b", fmt.Errorf("sad")))(func(ctx context.Context, addr string) {
		var r health.Result
		client := resty.New()
		res, err := client.R().SetResult(&r).SetError(&r).Get(fmt.Sprintf("http://%s%s", addr, path))
		t.Logf("%+v", r)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, res.StatusCode())
		require.False(t, r.OK)
		require.Len(t, r.Results, 2)
		require.False(t, r.Results[0].OK == r.Results[1].OK)
	})
}
