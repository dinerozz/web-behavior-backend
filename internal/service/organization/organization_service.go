package organization

import (
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response"
	"github.com/dinerozz/web-behavior-backend/internal/repository"
	"github.com/gofrs/uuid"
)

type OrganizationService struct {
	Repo     *repository.OrganizationRepository
	UserRepo *repository.UserRepository
}

func NewOrganizationService(repo *repository.OrganizationRepository, userRepo *repository.UserRepository) *OrganizationService {
	return &OrganizationService{
		Repo:     repo,
		UserRepo: userRepo,
	}
}

func (s *OrganizationService) CreateOrganization(org *request.CreateOrganization, creatorID uuid.UUID) (response.Organization, error) {
	_, err := s.UserRepo.GetUserById(creatorID)
	if err != nil {
		return response.Organization{}, fmt.Errorf("creator not found: %w", err)
	}

	return s.Repo.CreateOrganization(org, creatorID)
}

func (s *OrganizationService) GetAll() (*[]response.Organization, error) {
	organizations, err := s.Repo.GetAll()
	if err != nil {
		return &[]response.Organization{}, fmt.Errorf("failed to get organizations: %w", err)
	}

	return organizations, nil
}

func (s *OrganizationService) GetOrganizationByID(orgID uuid.UUID, userID uuid.UUID) (response.Organization, error) {
	_, err := s.Repo.CheckUserAccess(orgID, userID)
	if err != nil {
		return response.Organization{}, fmt.Errorf("access denied: %w", err)
	}

	return s.Repo.GetOrganizationByID(orgID)
}

func (s *OrganizationService) GetOrganizationWithMembers(orgID uuid.UUID, userID uuid.UUID) (response.OrganizationWithMembers, error) {
	_, err := s.Repo.CheckUserAccess(orgID, userID)
	if err != nil {
		return response.OrganizationWithMembers{}, fmt.Errorf("access denied: %w", err)
	}

	return s.Repo.GetOrganizationWithMembers(orgID)
}

func (s *OrganizationService) UpdateOrganization(orgID uuid.UUID, org *request.UpdateOrganization, userID uuid.UUID) (response.Organization, error) {
	isAdmin, err := s.Repo.IsUserOrgAdmin(orgID, userID)
	if err != nil {
		return response.Organization{}, fmt.Errorf("access check failed: %w", err)
	}
	if !isAdmin {
		return response.Organization{}, fmt.Errorf("only admins can update organization")
	}

	return s.Repo.UpdateOrganization(orgID, org)
}

func (s *OrganizationService) DeleteOrganization(orgID uuid.UUID, userID uuid.UUID) error {
	isAdmin, err := s.Repo.IsUserOrgAdmin(orgID, userID)
	if err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("only admins can delete organization")
	}

	return s.Repo.DeleteOrganization(orgID)
}

func (s *OrganizationService) GetUserOrganizations(userID uuid.UUID) (response.UserOrganizations, error) {
	return s.Repo.GetUserOrganizations(userID)
}

func (s *OrganizationService) AddUserToOrganization(orgID uuid.UUID, addUserReq *request.AddUserToOrganization, adminUserID uuid.UUID) error {
	isAdmin, err := s.Repo.IsUserOrgAdmin(orgID, adminUserID)
	if err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("only admins can add users to organization")
	}

	userToAddID, err := uuid.FromString(addUserReq.UserID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	_, err = s.UserRepo.GetUserById(userToAddID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	_, err = s.Repo.CheckUserAccess(orgID, userToAddID)
	if err == nil {
		return fmt.Errorf("user is already in this organization")
	}

	if addUserReq.Role != "admin" && addUserReq.Role != "member" && addUserReq.Role != "viewer" {
		return fmt.Errorf("invalid role: must be 'admin', 'member', or 'viewer'")
	}

	return s.Repo.AddUserToOrganization(orgID, userToAddID, addUserReq.Role)
}

func (s *OrganizationService) RemoveUserFromOrganization(orgID, userToRemoveID, adminUserID uuid.UUID) error {
	isAdmin, err := s.Repo.IsUserOrgAdmin(orgID, adminUserID)
	if err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("only admins can remove users from organization")
	}

	if adminUserID == userToRemoveID {
		orgWithMembers, err := s.Repo.GetOrganizationWithMembers(orgID)
		if err != nil {
			return fmt.Errorf("failed to get organization members: %w", err)
		}

		adminCount := 0
		for _, member := range orgWithMembers.Members {
			if member.Role == "admin" {
				adminCount++
			}
		}

		if adminCount == 1 {
			return fmt.Errorf("cannot remove the only admin from organization")
		}
	}

	return s.Repo.RemoveUserFromOrganization(orgID, userToRemoveID)
}

func (s *OrganizationService) UpdateUserRole(orgID, userToUpdateID uuid.UUID, role string, adminUserID uuid.UUID) error {
	isAdmin, err := s.Repo.IsUserOrgAdmin(orgID, adminUserID)
	if err != nil {
		return fmt.Errorf("access check failed: %w", err)
	}
	if !isAdmin {
		return fmt.Errorf("only admins can update user roles")
	}

	if role != "admin" && role != "member" && role != "viewer" {
		return fmt.Errorf("invalid role: must be 'admin', 'member', or 'viewer'")
	}

	if adminUserID == userToUpdateID && role != "admin" {
		orgWithMembers, err := s.Repo.GetOrganizationWithMembers(orgID)
		if err != nil {
			return fmt.Errorf("failed to get organization members: %w", err)
		}

		adminCount := 0
		for _, member := range orgWithMembers.Members {
			if member.Role == "admin" {
				adminCount++
			}
		}

		if adminCount == 1 {
			return fmt.Errorf("cannot demote the only admin from organization")
		}
	}

	return s.Repo.UpdateUserRole(orgID, userToUpdateID, role)
}

func (s *OrganizationService) CheckUserAccess(orgID, userID uuid.UUID) (string, error) {
	return s.Repo.CheckUserAccess(orgID, userID)
}

func (s *OrganizationService) IsUserOrgAdmin(orgID, userID uuid.UUID) (bool, error) {
	return s.Repo.IsUserOrgAdmin(orgID, userID)
}
