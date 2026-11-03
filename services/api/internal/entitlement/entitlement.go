package entitlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"clientpulse/services/api/internal/repository"
)

var (
	ErrClientLimitExceeded    = errors.New("organization has reached its subscription plan limit for clients")
	ErrAssetLimitExceeded     = errors.New("organization has reached its subscription plan limit for assets")
	ErrMonitorLimitExceeded   = errors.New("organization has reached its subscription plan limit for monitors")
	ErrHeartbeatLimitExceeded = errors.New("organization has reached its subscription plan limit for heartbeats")
	ErrMemberLimitExceeded    = errors.New("organization has reached its subscription plan limit for team members")
	ErrCheckIntervalTooFast   = errors.New("check interval is faster than allowed by your subscription plan")
	ErrBrandingNotIncluded    = errors.New("custom branding is not included in your organization's subscription plan")
	ErrReportsNotIncluded     = errors.New("public reports are not included in your organization's subscription plan")
)

// Service coordinates central entitlement evaluation and quota validation.
type Service struct {
	pool             *pgxpool.Pool
	subscriptionRepo *repository.SubscriptionRepository
}

// NewService constructs a new EntitlementService.
func NewService(pool *pgxpool.Pool, subscriptionRepo *repository.SubscriptionRepository) *Service {
	return &Service{
		pool:             pool,
		subscriptionRepo: subscriptionRepo,
	}
}

// CheckClientCreation validates whether the organization is entitled to create a new client.
func (s *Service) CheckClientCreation(ctx context.Context, orgID string) error {
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil || sub.Limits.MaxClients < 0 {
		return nil // Unlimited or unconstrained legacy tenant
	}

	// Concurrency lock and count
	var count int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM clients 
		WHERE organization_id = $1 AND archived_at IS NULL
	`, orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count clients: %w", err)
	}

	if count >= sub.Limits.MaxClients {
		return fmt.Errorf("%w: limit is %d (currently %d)", ErrClientLimitExceeded, sub.Limits.MaxClients, count)
	}
	return nil
}

// CheckAssetCreation validates whether the organization is entitled to register a new asset.
func (s *Service) CheckAssetCreation(ctx context.Context, orgID string) error {
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil || sub.Limits.MaxAssets < 0 {
		return nil // Unlimited
	}

	var count int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM assets 
		WHERE organization_id = $1 AND archived_at IS NULL
	`, orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count assets: %w", err)
	}

	if count >= sub.Limits.MaxAssets {
		return fmt.Errorf("%w: limit is %d (currently %d)", ErrAssetLimitExceeded, sub.Limits.MaxAssets, count)
	}
	return nil
}

// CheckMonitorInterval validates that an updated check interval conforms to plan limits.
func (s *Service) CheckMonitorInterval(ctx context.Context, orgID string, intervalSeconds int) error {
	if intervalSeconds <= 0 {
		return nil
	}
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}
	if sub.Limits.MinCheckIntervalSeconds > 0 && intervalSeconds < sub.Limits.MinCheckIntervalSeconds {
		return fmt.Errorf("%w: minimum permitted is %ds (requested %ds)", ErrCheckIntervalTooFast, sub.Limits.MinCheckIntervalSeconds, intervalSeconds)
	}
	return nil
}

// CheckMonitorCreation validates monitor counts and check interval against plan entitlements.
func (s *Service) CheckMonitorCreation(ctx context.Context, orgID string, intervalSeconds int) error {
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Unconstrained
	}

	// Check check interval
	if intervalSeconds > 0 && sub.Limits.MinCheckIntervalSeconds > 0 && intervalSeconds < sub.Limits.MinCheckIntervalSeconds {
		return fmt.Errorf("%w: minimum permitted is %ds (requested %ds)", ErrCheckIntervalTooFast, sub.Limits.MinCheckIntervalSeconds, intervalSeconds)
	}

	if sub.Limits.MaxMonitors < 0 {
		return nil // Unlimited
	}

	var count int
	err = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM monitors WHERE organization_id = $1", orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count monitors: %w", err)
	}

	if count >= sub.Limits.MaxMonitors {
		return fmt.Errorf("%w: limit is %d (currently %d)", ErrMonitorLimitExceeded, sub.Limits.MaxMonitors, count)
	}
	return nil
}

// CheckHeartbeatCreation validates heartbeat quotas against plan entitlements.
func (s *Service) CheckHeartbeatCreation(ctx context.Context, orgID string) error {
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil || sub.Limits.MaxHeartbeats < 0 {
		return nil // Unlimited
	}

	var count int
	err = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM heartbeat_monitors WHERE organization_id = $1", orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count heartbeats: %w", err)
	}

	if count >= sub.Limits.MaxHeartbeats {
		return fmt.Errorf("%w: limit is %d (currently %d)", ErrHeartbeatLimitExceeded, sub.Limits.MaxHeartbeats, count)
	}
	return nil
}

// CheckMemberAddition validates team member limits against plan entitlements.
func (s *Service) CheckMemberAddition(ctx context.Context, orgID string) error {
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil || sub.Limits.MaxMembers < 0 {
		return nil // Unlimited
	}

	var count int
	err = s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM organization_memberships WHERE organization_id = $1", orgID).Scan(&count)
	if err != nil {
		return fmt.Errorf("count members: %w", err)
	}

	if count >= sub.Limits.MaxMembers {
		return fmt.Errorf("%w: limit is %d (currently %d)", ErrMemberLimitExceeded, sub.Limits.MaxMembers, count)
	}
	return nil
}

// CheckCustomBranding validates if custom branding is enabled in the plan.
func (s *Service) CheckCustomBranding(ctx context.Context, orgID string) error {
	sub, err := s.subscriptionRepo.GetActiveSubscriptionForOrg(ctx, orgID)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil // Legacy unconstrained
	}
	if !sub.Limits.CustomBrandingEnabled {
		return ErrBrandingNotIncluded
	}
	return nil
}
