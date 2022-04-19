// package health provides a health endpoint for the service
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/robotlovesyou/fitest/pkg/log"
)

const (
	// Timeout for healthcheck. Should be configurable
	CheckTimeout = 5 * time.Second
)

type Monitor interface {
	Name() string
	Check(ctx context.Context) error
}

type Service struct {
	logger   *log.Logger
	monitors []Monitor
}

func New(logger *log.Logger, monitors ...Monitor) *Service {
	return &Service{
		logger:   logger,
		monitors: monitors,
	}
}

type CheckResult struct {
	Name string `json:"name"`
	OK   bool   `json:"ok"`
}

type Result struct {
	OK      bool          `json:"ok"`
	Results []CheckResult `json:"results"`
}

func (svc *Service) collectResults(ctx context.Context) ([]CheckResult, bool) {
	ok := true
	results := make(chan CheckResult)
	for _, m := range svc.monitors {
		go svc.collectResult(ctx, m, results)
	}
	collectedResults := make([]CheckResult, 0, len(svc.monitors))
Loop:
	for len(collectedResults) < len(svc.monitors) {
		select {
		case result := <-results:
			collectedResults = append(collectedResults, result)
			ok = ok && result.OK
		case <-ctx.Done():
			ok = false
			break Loop
		}
	}
	return collectedResults, ok
}

func (svc *Service) collectResult(ctx context.Context, monitor Monitor, out chan<- CheckResult) {
	result := CheckResult{Name: monitor.Name(), OK: true}
	svc.logger.Infof(ctx, "checking health for %s", result.Name)

	if err := monitor.Check(ctx); err != nil {
		svc.logger.Errorf(ctx, err, "error collecting health check for %s", result.Name)
		result.OK = false
	}
	select {
	case <-ctx.Done():
	case out <- result:
	}
}

func getStatus(ok bool) int {
	if ok {
		return http.StatusOK
	}
	return http.StatusInternalServerError
}

func (svc *Service) Handle(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), CheckTimeout)
	defer cancel()

	results, ok := svc.collectResults(ctx)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(getStatus(ok))
	enc := json.NewEncoder(w)
	enc.Encode(&Result{
		OK:      ok,
		Results: results,
	})
}
