package extender

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/service"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	readHeaderTimeout = 5 * time.Second
	shutdownTimeout   = 5 * time.Second
	maxRequestBytes   = 10 << 20
	// sessionStaleAfter bounds a round's age only while no close notification arrives.
	sessionStaleAfter = 2 * time.Second
	// closeSilenceLimit: a longer silence means close notifications stopped (volcano runs a round every second).
	closeSilenceLimit = 10 * time.Second
)

// settings holds the admin-facing configuration a decision needs, read once per scheduling round
// together with the cluster view.
type settings struct {
	config *model.SchedulerExtenderConfig
	quotas *service.QueueQuotaSet
}

// sessionState freezes one scheduling round the way volcano freezes its own snapshot at session
// open, so every job asked during that round is judged against identical data.
type sessionState struct {
	settings *settings
	snap     *snapshot // nil while the plugin switch is off
	builtAt  time.Time
}

// Server answers volcano's extender callbacks. It runs inside the backend process because every
// decision reads the manager's informer caches and the in-process reservation ledger, which also
// makes a single backend replica a hard requirement.
type Server struct {
	reader        client.Reader
	quotaService  *service.QueueQuotaService
	configService *service.ConfigService
	accumulator   *sessionAccumulator
	address       string
	logger        logr.Logger

	mu          sync.Mutex
	session     *sessionState
	lastCloseAt time.Time
	// firstAskSinceClose starts the silence clock at the first ask, so a volcano restart does not look like a missing verb.
	firstAskSinceClose time.Time
	closeSilenceWarned bool
}

func New(
	reader client.Reader,
	quotaService *service.QueueQuotaService,
	configService *service.ConfigService,
	address string,
) *Server {
	return &Server{
		reader:        reader,
		quotaService:  quotaService,
		configService: configService,
		accumulator:   newSessionAccumulator(),
		address:       address,
		logger:        ctrl.Log.WithName("volcano-extender"),
	}
}

// NeedLeaderElection is false: the endpoint must answer whenever the pod serves traffic, otherwise
// volcano's calls time out and every enqueue decision silently falls back to permit.
func (s *Server) NeedLeaderElection() bool {
	return false
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/"+JobEnqueueableVerb, s.handleJobEnqueueable)
	mux.HandleFunc("/"+OnSessionCloseVerb, s.handleSessionClose)

	srv := &http.Server{
		Addr:              s.address,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			s.logger.Error(err, "failed to shut down extender endpoint")
		}
	}()

	s.logger.Info("volcano extender endpoint listening",
		"address", s.address, "verbs", []string{JobEnqueueableVerb, OnSessionCloseVerb})
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// currentSession returns the frozen view of the round in progress, rebuilding it when the previous
// round was closed or when its close notification never arrived.
func (s *Server) currentSession(ctx context.Context) (*sessionState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := utils.GetLocalTime()
	if s.firstAskSinceClose.IsZero() {
		s.firstAskSinceClose = now
	}
	if s.session != nil && (now.Sub(s.lastCloseAt) < closeSilenceLimit || now.Sub(s.session.builtAt) < sessionStaleAfter) {
		return s.session, nil
	}

	config, err := s.configService.GetSchedulerExtenderConfig(ctx)
	if err != nil {
		return nil, err
	}
	quotas, err := s.quotaService.LoadQuotaSet(ctx, config)
	if err != nil {
		return nil, err
	}

	state := &sessionState{
		settings: &settings{config: config, quotas: quotas},
		builtAt:  utils.GetLocalTime(),
	}
	if config.SchedulerExtenderEnabled {
		if !s.closeSilenceWarned && now.Sub(s.firstAskSinceClose) >= closeSilenceLimit {
			s.closeSilenceWarned = true
			s.logger.Info("volcano asks without sending onSessionClose, so the cluster view is rebuilt on a timer; "+
				"set extender.onSessionCloseVerb in the volcano scheduler configmap",
				"asksWithoutClose", now.Sub(s.firstAskSinceClose).Round(time.Second), "rebuildEvery", sessionStaleAfter)
		}
		snap, buildErr := s.buildSnapshot(ctx, state.settings)
		if buildErr != nil {
			return nil, buildErr
		}
		s.accumulator.sweep(snap)
		state.snap = snap
	}

	s.session = state
	return state, nil
}

// invalidateSession drops the frozen view at the end of a scheduling round. The reservation ledger
// deliberately survives: volcano writes this round's admissions to Kubernetes at that very moment,
// and our own cache observes them only milliseconds later.
func (s *Server) invalidateSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = nil
	s.lastCloseAt = utils.GetLocalTime()
	s.firstAskSinceClose = time.Time{}
	s.closeSilenceWarned = false
}

func (s *Server) handleJobEnqueueable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req jobEnqueueableRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes)).Decode(&req); err != nil {
		s.logger.Error(err, "failed to decode job enqueueable request")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	result := s.decide(r.Context(), req.Job)
	switch {
	case result.status == voteReject:
		s.logger.Info("rejecting job enqueue", result.fields...)
	case result.err != nil:
		// A failure-driven abstention silently disables quota and blocking, so it must be visible.
		s.logger.Error(result.err, "abstaining from job enqueue", result.fields...)
	case len(result.fields) > 0:
		// Scope and capability skips repeat every round for the same objects; keep them out of the
		// default log level.
		s.logger.V(1).Info("abstaining from job enqueue", result.fields...)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(jobEnqueueableResponse{Status: result.status}); err != nil {
		s.logger.Error(err, "failed to write job enqueueable response")
	}
}

// handleSessionClose marks the end of a scheduling round. Volcano sends an empty body and discards
// whatever comes back, so there is nothing to parse and nothing to answer with.
func (s *Server) handleSessionClose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.invalidateSession()
	w.WriteHeader(http.StatusOK)
}
