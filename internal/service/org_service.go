// Package service provides business logic services for CYP-Registry.
package service

import (
	"errors"
	"time"

	"cyp-registry/internal/dao"

	"go.uber.org/zap"
)

// OrgService provides organization management services.
type OrgService struct {
	logger *zap.Logger
}

// Organization represents an organization.
type Organization struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	OwnerID     int64     `json:"owner_id"`
	OwnerName   string    `json:"owner_name,omitempty"`
	MemberCount int       `json:"member_count,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// OrgMember represents an organization member.
type OrgMember struct {
	ID        int64     `json:"id"`
	OrgID     int64     `json:"org_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateOrgRequest represents a request to create an organization.
type CreateOrgRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayName string `json:"display_name"`
}

// NewOrgService creates a new OrgService instance.
func NewOrgService(logger *zap.Logger) *OrgService {
	return &OrgService{
		logger: logger,
	}
}

// CreateOrganization creates a new organization.
func (s *OrgService) CreateOrganization(req *CreateOrgRequest, ownerID int64) (*Organization, error) {
	// Check if name already exists
	existing, err := dao.GetOrganizationByName(req.Name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("organization name already exists")
	}

	displayName := req.DisplayName
	if displayName == "" {
		displayName = req.Name
	}

	daoOrg := &dao.Organization{
		Name:        req.Name,
		DisplayName: displayName,
		OwnerID:     ownerID,
	}

	if err := dao.CreateOrganization(daoOrg); err != nil {
		return nil, err
	}

	return &Organization{
		ID:          daoOrg.ID,
		Name:        daoOrg.Name,
		DisplayName: daoOrg.DisplayName,
		OwnerID:     daoOrg.OwnerID,
		CreatedAt:   daoOrg.CreatedAt,
		UpdatedAt:   daoOrg.UpdatedAt,
	}, nil
}

// GetOrganization retrieves an organization by ID.
func (s *OrgService) GetOrganization(id int64) (*Organization, error) {
	daoOrg, err := dao.GetOrganization(id)
	if err != nil {
		return nil, err
	}
	if daoOrg == nil {
		return nil, errors.New("organization not found")
	}

	return s.convertOrg(daoOrg), nil
}

// GetOrganizationByName retrieves an organization by name.
func (s *OrgService) GetOrganizationByName(name string) (*Organization, error) {
	daoOrg, err := dao.GetOrganizationByName(name)
	if err != nil {
		return nil, err
	}
	if daoOrg == nil {
		return nil, errors.New("organization not found")
	}

	return s.convertOrg(daoOrg), nil
}

// ListOrganizations lists all organizations.
func (s *OrgService) ListOrganizations(page, pageSize int) ([]*Organization, int, error) {
	daoOrgs, total, err := dao.ListOrganizations(page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	orgs := make([]*Organization, len(daoOrgs))
	for i, daoOrg := range daoOrgs {
		orgs[i] = s.convertOrg(daoOrg)
	}

	return orgs, total, nil
}

// ListUserOrganizations lists organizations for a user.
func (s *OrgService) ListUserOrganizations(userID int64) ([]*Organization, error) {
	daoOrgs, err := dao.ListUserOrganizations(userID)
	if err != nil {
		return nil, err
	}

	orgs := make([]*Organization, len(daoOrgs))
	for i, daoOrg := range daoOrgs {
		orgs[i] = s.convertOrg(daoOrg)
	}

	return orgs, nil
}

// UpdateOrganization updates an organization.
func (s *OrgService) UpdateOrganization(id int64, displayName string, userID int64) error {
	org, err := dao.GetOrganization(id)
	if err != nil {
		return err
	}
	if org == nil {
		return errors.New("organization not found")
	}

	// Check permission
	if org.OwnerID != userID {
		return errors.New("permission denied")
	}

	org.DisplayName = displayName
	return dao.UpdateOrganization(org)
}

// DeleteOrganization deletes an organization.
func (s *OrgService) DeleteOrganization(id int64, userID int64) error {
	org, err := dao.GetOrganization(id)
	if err != nil {
		return err
	}
	if org == nil {
		return errors.New("organization not found")
	}

	// Check permission
	if org.OwnerID != userID {
		return errors.New("permission denied")
	}

	return dao.DeleteOrganization(id)
}

// AddMember adds a member to an organization.
func (s *OrgService) AddMember(orgID, userID, requestorID int64, role string) error {
	org, err := dao.GetOrganization(orgID)
	if err != nil {
		return err
	}
	if org == nil {
		return errors.New("organization not found")
	}

	// Check permission
	if org.OwnerID != requestorID {
		return errors.New("permission denied")
	}

	if role == "" {
		role = "member"
	}

	return dao.AddOrgMember(orgID, userID, role)
}

// RemoveMember removes a member from an organization.
func (s *OrgService) RemoveMember(orgID, userID, requestorID int64) error {
	org, err := dao.GetOrganization(orgID)
	if err != nil {
		return err
	}
	if org == nil {
		return errors.New("organization not found")
	}

	// Check permission
	if org.OwnerID != requestorID {
		return errors.New("permission denied")
	}

	// Cannot remove owner
	if userID == org.OwnerID {
		return errors.New("cannot remove organization owner")
	}

	return dao.RemoveOrgMember(orgID, userID)
}

// GetMembers retrieves members of an organization.
func (s *OrgService) GetMembers(orgID int64) ([]*OrgMember, error) {
	daoMembers, err := dao.GetOrgMembers(orgID)
	if err != nil {
		return nil, err
	}

	members := make([]*OrgMember, len(daoMembers))
	for i, m := range daoMembers {
		members[i] = &OrgMember{
			ID:        m.ID,
			OrgID:     m.OrgID,
			UserID:    m.UserID,
			Username:  m.Username,
			Role:      m.Role,
			CreatedAt: m.CreatedAt,
		}
	}

	return members, nil
}

func (s *OrgService) convertOrg(daoOrg *dao.Organization) *Organization {
	return &Organization{
		ID:          daoOrg.ID,
		Name:        daoOrg.Name,
		DisplayName: daoOrg.DisplayName,
		OwnerID:     daoOrg.OwnerID,
		CreatedAt:   daoOrg.CreatedAt,
		UpdatedAt:   daoOrg.UpdatedAt,
	}
}
