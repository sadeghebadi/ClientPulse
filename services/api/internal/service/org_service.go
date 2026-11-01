package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"clientpulse/services/api/internal/auth"
	"clientpulse/services/api/internal/config"
	"clientpulse/services/api/internal/repository"
	"clientpulse/services/api/internal/tenant"
)

var (
	ErrForbidden             = errors.New("forbidden: insufficient permissions")
	ErrCannotDemoteLastOwner = errors.New("cannot demote or remove the last remaining owner")
	ErrInvalidRole           = errors.New("invalid role: must be owner, admin, member, or viewer")
)

type OrgService struct {
	orgRepo     *repository.OrganizationRepository
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
	auditRepo   *repository.AuditRepository
	cfg         *config.Config
}

func NewOrgService(
	orgRepo *repository.OrganizationRepository,
	userRepo *repository.UserRepository,
	sessionRepo *repository.SessionRepository,
	auditRepo *repository.AuditRepository,
	cfg *config.Config,
) *OrgService {
	return &OrgService{
		orgRepo:     orgRepo,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		auditRepo:   auditRepo,
		cfg:         cfg,
	}
}

type CreateOrgRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug,omitempty"`
	Locale   string `json:"locale,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type UpdateOrgRequest struct {
	Name     string `json:"name"`
	Locale   string `json:"locale"`
	Timezone string `json:"timezone"`
}

type AddMemberRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// CreateOrganization provisions a new organization and assigns the creator as owner.
func (s *OrgService) CreateOrganization(ctx context.Context, userID string, req CreateOrgRequest) (*repository.Organization, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("organization name is required")
	}

	slug := slugify(req.Slug)
	if slug == "" {
		slug = slugify(name)
	}
	if slug == "" {
		slug = "agency-" + userID[:8]
	}

	locale := req.Locale
	if locale == "" {
		locale = s.cfg.DefaultLocale
	}
	timezone := req.Timezone
	if timezone == "" {
		timezone = s.cfg.DefaultTimezone
	}

	// Ensure unique slug
	uniqueSlug := slug
	for i := 1; i <= 5; i++ {
		_, err := s.orgRepo.GetOrganizationBySlug(ctx, uniqueSlug)
		if errors.Is(err, repository.ErrOrgNotFound) {
			break
		}
		uniqueSlug = fmt.Sprintf("%s-%d", slug, i)
	}

	org, err := s.orgRepo.CreateOrganization(ctx, name, uniqueSlug, locale, timezone)
	if err != nil {
		randSuffix, _ := auth.GenerateSecureToken(4)
		org, err = s.orgRepo.CreateOrganization(ctx, name, fmt.Sprintf("%s-%s", slug, strings.ToLower(randSuffix)), locale, timezone)
		if err != nil {
			return nil, fmt.Errorf("failed to create organization: %w", err)
		}
	}

	// Add creator as owner
	_, err = s.orgRepo.AddMember(ctx, org.ID, userID, tenant.RoleOwner)
	if err != nil {
		return nil, fmt.Errorf("failed to add owner membership: %w", err)
	}

	_ = s.auditRepo.RecordEvent(ctx, &org.ID, &userID, "org.create", "organization", org.ID, nil, "")

	return org, nil
}

// GetCurrentOrganization validates user membership and returns org details and user role.
func (s *OrgService) GetCurrentOrganization(ctx context.Context, orgID, userID string) (*repository.Organization, string, error) {
	membership, err := s.orgRepo.GetMembership(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrMembershipNotFound) {
			return nil, "", ErrForbidden
		}
		return nil, "", err
	}

	org, err := s.orgRepo.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, "", err
	}

	return org, membership.Role, nil
}

// UpdateOrganization updates organization details if caller has owner or admin role.
func (s *OrgService) UpdateOrganization(ctx context.Context, orgID, callerUserID string, req UpdateOrgRequest) (*repository.Organization, error) {
	membership, err := s.orgRepo.GetMembership(ctx, orgID, callerUserID)
	if err != nil || (membership.Role != tenant.RoleOwner && membership.Role != tenant.RoleAdmin) {
		return nil, ErrForbidden
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("organization name is required")
	}

	locale := req.Locale
	if locale != "en" && locale != "fa" {
		locale = "en"
	}

	timezone := strings.TrimSpace(req.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}

	org, err := s.orgRepo.UpdateOrganization(ctx, orgID, name, locale, timezone)
	if err != nil {
		return nil, err
	}

	_ = s.auditRepo.RecordEvent(ctx, &orgID, &callerUserID, "org.update", "organization", orgID, nil, "")

	return org, nil
}

// SwitchOrganization switches the active organization for the current user's session.
func (s *OrgService) SwitchOrganization(ctx context.Context, sessionID, userID, targetOrgID string) (*repository.Organization, error) {
	// Mandatory membership check: user cannot select an org they do not belong to
	_, err := s.orgRepo.GetMembership(ctx, targetOrgID, userID)
	if err != nil {
		return nil, ErrForbidden
	}

	org, err := s.orgRepo.GetOrganizationByID(ctx, targetOrgID)
	if err != nil {
		return nil, err
	}

	if sessionID != "" {
		if err := s.sessionRepo.SetActiveOrganization(ctx, sessionID, targetOrgID); err != nil {
			return nil, fmt.Errorf("failed to update session active org: %w", err)
		}
	}

	return org, nil
}

// ListUserOrganizations retrieves all organizations the user is a member of.
func (s *OrgService) ListUserOrganizations(ctx context.Context, userID string) ([]repository.Membership, error) {
	return s.orgRepo.ListUserMemberships(ctx, userID)
}

// ListMembers lists all members of the organization if caller belongs to it.
func (s *OrgService) ListMembers(ctx context.Context, orgID, callerUserID string) ([]repository.Membership, error) {
	_, err := s.orgRepo.GetMembership(ctx, orgID, callerUserID)
	if err != nil {
		return nil, ErrForbidden
	}

	return s.orgRepo.ListOrganizationMembers(ctx, orgID)
}

// AddMember adds a user to the organization by email.
func (s *OrgService) AddMember(ctx context.Context, orgID, callerUserID string, req AddMemberRequest) (*repository.Membership, error) {
	callerMem, err := s.orgRepo.GetMembership(ctx, orgID, callerUserID)
	if err != nil || (callerMem.Role != tenant.RoleOwner && callerMem.Role != tenant.RoleAdmin) {
		return nil, ErrForbidden
	}

	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role != tenant.RoleOwner && req.Role != tenant.RoleAdmin && req.Role != tenant.RoleMember && req.Role != tenant.RoleViewer {
		return nil, ErrInvalidRole
	}

	// Admin cannot grant Owner role
	if callerMem.Role != tenant.RoleOwner && req.Role == tenant.RoleOwner {
		return nil, ErrForbidden
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	user, err := s.userRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return nil, errors.New("user with this email does not exist; they must sign up first")
		}
		return nil, err
	}

	membership, err := s.orgRepo.AddMember(ctx, orgID, user.ID, req.Role)
	if err != nil {
		return nil, err
	}

	_ = s.auditRepo.RecordEvent(ctx, &orgID, &callerUserID, "member.add", "user", user.ID, []byte(fmt.Sprintf(`{"role":"%s"}`, req.Role)), "")

	return membership, nil
}

// UpdateMemberRole updates a member's role. Only owners can change roles.
func (s *OrgService) UpdateMemberRole(ctx context.Context, orgID, callerUserID, targetUserID, newRole string) error {
	callerMem, err := s.orgRepo.GetMembership(ctx, orgID, callerUserID)
	if err != nil || callerMem.Role != tenant.RoleOwner {
		return ErrForbidden
	}

	newRole = strings.ToLower(strings.TrimSpace(newRole))
	if newRole != tenant.RoleOwner && newRole != tenant.RoleAdmin && newRole != tenant.RoleMember && newRole != tenant.RoleViewer {
		return ErrInvalidRole
	}

	targetMem, err := s.orgRepo.GetMembership(ctx, orgID, targetUserID)
	if err != nil {
		return err
	}

	// If demoting an owner, ensure at least one other owner remains
	if targetMem.Role == tenant.RoleOwner && newRole != tenant.RoleOwner {
		ownerCount, err := s.orgRepo.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if ownerCount <= 1 {
			return ErrCannotDemoteLastOwner
		}
	}

	if err := s.orgRepo.UpdateMemberRole(ctx, orgID, targetUserID, newRole); err != nil {
		return err
	}

	_ = s.auditRepo.RecordEvent(ctx, &orgID, &callerUserID, "member.role_update", "user", targetUserID, []byte(fmt.Sprintf(`{"new_role":"%s"}`, newRole)), "")

	return nil
}

// RemoveMember removes a member from the organization.
func (s *OrgService) RemoveMember(ctx context.Context, orgID, callerUserID, targetUserID string) error {
	callerMem, err := s.orgRepo.GetMembership(ctx, orgID, callerUserID)
	if err != nil {
		return ErrForbidden
	}

	targetMem, err := s.orgRepo.GetMembership(ctx, orgID, targetUserID)
	if err != nil {
		return err
	}

	// Self-removal is allowed unless user is the only owner
	isSelf := callerUserID == targetUserID
	if !isSelf {
		// Only Owner or Admin can remove other members
		if callerMem.Role != tenant.RoleOwner && callerMem.Role != tenant.RoleAdmin {
			return ErrForbidden
		}
		// Admin cannot remove an Owner
		if targetMem.Role == tenant.RoleOwner && callerMem.Role != tenant.RoleOwner {
			return ErrForbidden
		}
	}

	// If target is Owner, verify another owner exists
	if targetMem.Role == tenant.RoleOwner {
		ownerCount, err := s.orgRepo.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if ownerCount <= 1 {
			return ErrCannotDemoteLastOwner
		}
	}

	if err := s.orgRepo.RemoveMember(ctx, orgID, targetUserID); err != nil {
		return err
	}

	_ = s.auditRepo.RecordEvent(ctx, &orgID, &callerUserID, "member.remove", "user", targetUserID, nil, "")

	return nil
}

