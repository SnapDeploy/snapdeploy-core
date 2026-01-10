package worker

import (
	"context"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"snapdeploy-core/internal/domain/deployment"
	"snapdeploy-core/internal/infrastructure/cleanup"
)

// TTLWorker is a background worker that periodically checks for and cleans up expired deployments
type TTLWorker struct {
	deploymentRepo deployment.DeploymentRepository
	cleanupService *cleanup.CleanupService
	interval       time.Duration
	batchSize      int32

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// TTLWorkerConfig holds configuration for the TTL worker
type TTLWorkerConfig struct {
	// Interval between cleanup checks (default: 5 minutes)
	Interval time.Duration
	// BatchSize is the maximum number of expired deployments to process per cycle
	BatchSize int32
}

// DefaultTTLWorkerConfig returns the default configuration
func DefaultTTLWorkerConfig() TTLWorkerConfig {
	interval := 5 * time.Minute
	batchSize := int32(10)

	// Override from environment
	if envInterval := os.Getenv("CLEANUP_INTERVAL_MINUTES"); envInterval != "" {
		if minutes, err := strconv.Atoi(envInterval); err == nil && minutes > 0 {
			interval = time.Duration(minutes) * time.Minute
		}
	}

	if envBatch := os.Getenv("CLEANUP_BATCH_SIZE"); envBatch != "" {
		if size, err := strconv.Atoi(envBatch); err == nil && size > 0 {
			batchSize = int32(size)
		}
	}

	return TTLWorkerConfig{
		Interval:  interval,
		BatchSize: batchSize,
	}
}

// NewTTLWorker creates a new TTL worker
func NewTTLWorker(
	deploymentRepo deployment.DeploymentRepository,
	cleanupService *cleanup.CleanupService,
	config TTLWorkerConfig,
) *TTLWorker {
	return &TTLWorker{
		deploymentRepo: deploymentRepo,
		cleanupService: cleanupService,
		interval:       config.Interval,
		batchSize:      config.BatchSize,
	}
}

// Start begins the background worker
func (w *TTLWorker) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.wg.Add(1)

	go w.run()

	log.Printf("[TTLWorker] Started with interval=%v, batchSize=%d", w.interval, w.batchSize)
}

// Stop gracefully stops the worker and waits for it to finish
func (w *TTLWorker) Stop() {
	if w.cancel != nil {
		log.Printf("[TTLWorker] Stopping...")
		w.cancel()
	}
	w.wg.Wait()
	log.Printf("[TTLWorker] Stopped")
}

// run is the main worker loop
func (w *TTLWorker) run() {
	defer w.wg.Done()

	// Run immediately on startup
	w.processExpiredDeployments()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			log.Printf("[TTLWorker] Received shutdown signal")
			return
		case <-ticker.C:
			w.processExpiredDeployments()
		}
	}
}

// processExpiredDeployments finds and cleans up expired deployments
func (w *TTLWorker) processExpiredDeployments() {
	ctx, cancel := context.WithTimeout(w.ctx, 5*time.Minute)
	defer cancel()

	log.Printf("[TTLWorker] Checking for expired deployments...")

	// Find expired deployments
	expired, err := w.deploymentRepo.FindExpired(ctx, w.batchSize)
	if err != nil {
		log.Printf("[TTLWorker] Error finding expired deployments: %v", err)
		return
	}

	if len(expired) == 0 {
		log.Printf("[TTLWorker] No expired deployments found")
		return
	}

	log.Printf("[TTLWorker] Found %d expired deployments to cleanup", len(expired))

	// Process each expired deployment
	for _, dep := range expired {
		select {
		case <-ctx.Done():
			log.Printf("[TTLWorker] Context cancelled, stopping processing")
			return
		default:
		}

		w.processExpiredDeployment(ctx, dep)
	}
}

// processExpiredDeployment handles cleanup for a single expired deployment
func (w *TTLWorker) processExpiredDeployment(ctx context.Context, dep *deployment.Deployment) {
	log.Printf("[TTLWorker] Processing expired deployment: %s (project: %s, expired at: %v)",
		dep.ID().String(), dep.ProjectID().String(), dep.ExpiresAt())

	// Cleanup AWS resources
	if w.cleanupService != nil {
		if err := w.cleanupService.CleanupDeployment(ctx, dep); err != nil {
			log.Printf("[TTLWorker] Error cleaning up deployment %s: %v", dep.ID().String(), err)
			// Continue to update status even if cleanup had errors
		}
	}

	// Update deployment status to EXPIRED
	if err := dep.UpdateStatus(deployment.StatusExpired); err != nil {
		log.Printf("[TTLWorker] Error updating deployment status to EXPIRED: %v", err)
		return
	}

	// Save the updated deployment
	if err := w.deploymentRepo.Save(ctx, dep); err != nil {
		log.Printf("[TTLWorker] Error saving deployment after expiration: %v", err)
		return
	}

	log.Printf("[TTLWorker] Successfully expired deployment: %s", dep.ID().String())
}

// ForceCleanup forces an immediate cleanup check (useful for testing)
func (w *TTLWorker) ForceCleanup(ctx context.Context) {
	w.processExpiredDeployments()
}


