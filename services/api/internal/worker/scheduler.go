package worker

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"clientpulse/services/api/internal/checker"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/ssrf"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SchedulerConfig struct {
	WorkerID      string
	Concurrency   int
	PollPeriod    time.Duration
	RetentionDays int
}

type Scheduler struct {
	pool          *pgxpool.Pool
	monitorRepo   *repository.MonitorRepository
	resultRepo    *repository.CheckResultRepository
	checker       *checker.Checker
	workerID      string
	concurrency   int
	pollPeriod    time.Duration
	retentionDays int
	logger        *slog.Logger
}

func NewScheduler(
	pool *pgxpool.Pool,
	monitorRepo *repository.MonitorRepository,
	resultRepo *repository.CheckResultRepository,
	guard *ssrf.Guard,
	cfg SchedulerConfig,
	logger *slog.Logger,
) *Scheduler {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	if cfg.PollPeriod <= 0 {
		cfg.PollPeriod = 1 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = 90
	}

	chk := checker.NewChecker(guard, cfg.WorkerID)

	return &Scheduler{
		pool:          pool,
		monitorRepo:   monitorRepo,
		resultRepo:    resultRepo,
		checker:       chk,
		workerID:      cfg.WorkerID,
		concurrency:   cfg.Concurrency,
		pollPeriod:    cfg.PollPeriod,
		retentionDays: cfg.RetentionDays,
		logger:        logger,
	}
}

// SetTestTransport sets a custom transport on the internal checker for controlled testing.
func (s *Scheduler) SetTestTransport(rt http.RoundTripper) {
	s.checker.SetTestTransport(rt)
}

// Run executes the main scheduling loop until the context is canceled.
func (s *Scheduler) Run(ctx context.Context) error {
	s.logger.Info("starting monitoring scheduler daemon",
		"worker_id", s.workerID,
		"concurrency", s.concurrency,
		"poll_period", s.pollPeriod.String(),
		"retention_days", s.retentionDays,
	)

	// Controlled background cleanup routine
	runPrune := func() {
		pruneCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		count, err := s.resultRepo.PruneExpiredCheckResults(pruneCtx, s.retentionDays, 500)
		if err != nil {
			s.logger.Error("failed to prune expired check results", "error", err)
		} else if count > 0 {
			s.logger.Info("pruned expired check results", "pruned_rows", count, "retention_days", s.retentionDays)
		}
		// Also prune expired aggregates older than 400 days (13+ months)
		aggCount, err := s.resultRepo.PruneExpiredAggregates(pruneCtx, 400)
		if err != nil {
			s.logger.Error("failed to prune expired aggregates", "error", err)
		} else if aggCount > 0 {
			s.logger.Info("pruned expired aggregates older than 13 months", "pruned_rows", aggCount)
		}
	}

	// Prune old results periodically (every 6 hours)
	pruneTicker := time.NewTicker(6 * time.Hour)
	defer pruneTicker.Stop()

	// Initial prune at startup
	go runPrune()

	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup

	ticker := time.NewTicker(s.pollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler received stop signal, draining active jobs...")
			wg.Wait()
			s.logger.Info("all active monitoring jobs completed")
			return nil

		case <-pruneTicker.C:
			go runPrune()

		case <-ticker.C:
			// Claim due jobs
			claimBatchSize := s.concurrency
			claimCtx, claimCancel := context.WithTimeout(ctx, 5*time.Second)
			jobs, err := s.monitorRepo.ClaimDueMonitors(claimCtx, s.workerID, claimBatchSize, 60)
			claimCancel()

			if err != nil {
				if ctx.Err() == nil {
					s.logger.Warn("failed to claim due monitors", "error", err)
				}
				continue
			}

			if len(jobs) == 0 {
				continue
			}

			s.logger.Debug("claimed due monitoring jobs", "count", len(jobs))

			for _, job := range jobs {
				// Acquire concurrency slot
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					return nil
				}

				wg.Add(1)
				go func(j *repository.ClaimedMonitorJob) {
					defer func() {
						<-sem
						wg.Done()
					}()
					s.processJob(ctx, j)
				}(job)
			}
		}
	}
}

// processJob executes a single monitor check and persists the result.
func (s *Scheduler) processJob(ctx context.Context, job *repository.ClaimedMonitorJob) {
	checkTimeout := time.Duration(job.TimeoutSeconds+5) * time.Second
	checkCtx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()

	chkCfg := checker.CheckConfig{
		TargetURL:         job.TargetURL,
		Method:            job.Method,
		TimeoutSeconds:    job.TimeoutSeconds,
		ExpectedStatusMin: job.ExpectedStatusMin,
		ExpectedStatusMax: job.ExpectedStatusMax,
		FollowRedirects:   job.FollowRedirects,
		MaxRedirects:      job.MaxRedirects,
		ExpectedText:      job.ExpectedText,
		JSONPointer:       job.JSONPointer,
		JSONExpectedValue: job.JSONExpectedValue,
		ScheduledAt:       job.ScheduledAt,
	}

	execResult := s.checker.Execute(checkCtx, chkCfg)

	// Persist Check Result
	persistCtx, persistCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer persistCancel()

	_, err := s.resultRepo.CreateCheckResult(persistCtx, repository.CreateCheckResultParams{
		OrganizationID:   job.OrganizationID,
		MonitorID:        job.ID,
		ScheduledAt:      job.ScheduledAt,
		StartedAt:        execResult.StartedAt,
		CompletedAt:      execResult.CompletedAt,
		LatencyMs:        execResult.LatencyMs,
		Result:           execResult.Result,
		HTTPStatusCode:   execResult.HTTPStatusCode,
		ErrorCategory:    execResult.ErrorCategory,
		ErrorMessage:     execResult.ErrorMessage,
		TLSVersion:       execResult.TLSVersion,
		TLSCipher:        execResult.TLSCipher,
		TLSCertIssuer:    execResult.TLSCertIssuer,
		TLSCertSubject:   execResult.TLSCertSubject,
		TLSCertNotBefore: execResult.TLSCertNotBefore,
		TLSCertNotAfter:  execResult.TLSCertNotAfter,
		TLSDaysRemaining: execResult.TLSDaysRemaining,
		TLSIsValid:       execResult.TLSIsValid,
		ResolvedIP:       execResult.ResolvedIP,
		WorkerID:         s.workerID,
		ExecutionType:    "scheduled",
	})
	if err != nil {
		s.logger.Error("failed to persist check result", "monitor_id", job.ID, "error", err)
	}

	// Update Monitor state and drift-free next check via state transition machine
	isSuccess := execResult.Result == "success"

	newMonitorStatus, updateErr := s.monitorRepo.RecordScheduledCheckOutcome(
		persistCtx,
		job.OrganizationID,
		job.ID,
		isSuccess,
		execResult.CompletedAt,
		job.ScheduledAt,
		job.IntervalSeconds,
	)
	if updateErr != nil {
		s.logger.Error("failed to update monitor after check", "monitor_id", job.ID, "error", updateErr)
	} else {
		s.logger.Info("monitoring check executed and recorded",
			"monitor_id", job.ID,
			"target_url", job.TargetURL,
			"result", execResult.Result,
			"status", newMonitorStatus,
			"latency_ms", execResult.LatencyMs,
		)
	}
}

