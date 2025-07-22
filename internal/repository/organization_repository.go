package repository

import (
	"database/sql"
	"fmt"
	"github.com/dinerozz/web-behavior-backend/internal/model/request"
	"github.com/dinerozz/web-behavior-backend/internal/model/response"
	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type OrganizationRepository struct {
	db *sqlx.DB
}

func NewOrganizationRepository(db *sqlx.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

func (r *OrganizationRepository) CreateOrganization(org *request.CreateOrganization, creatorID uuid.UUID) (response.Organization, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return response.Organization{}, err
	}
	defer tx.Rollback()

	query := `INSERT INTO organizations (name, description) 
              VALUES ($1, $2) 
              RETURNING id, name, description, created_at, updated_at`

	var organization response.Organization
	var description sql.NullString

	err = tx.QueryRow(query, org.Name, org.Description).Scan(
		&organization.ID,
		&organization.Name,
		&description,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	)
	if err != nil {
		return response.Organization{}, err
	}

	if description.Valid {
		organization.Description = &description.String
	}

	accessQuery := `INSERT INTO user_organization_access (user_id, organization_id, role) VALUES ($1, $2, 'admin')`
	_, err = tx.Exec(accessQuery, creatorID, organization.ID)
	if err != nil {
		return response.Organization{}, err
	}

	if err = tx.Commit(); err != nil {
		return response.Organization{}, err
	}

	return organization, nil
}

func (r *OrganizationRepository) GetOrganizationByID(orgID uuid.UUID) (response.Organization, error) {
	query := `SELECT id, name, description, created_at, updated_at FROM organizations WHERE id = $1`

	var organization response.Organization
	var description sql.NullString

	err := r.db.QueryRow(query, orgID).Scan(
		&organization.ID,
		&organization.Name,
		&description,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	)
	if err != nil {
		return response.Organization{}, err
	}

	if description.Valid {
		organization.Description = &description.String
	}

	return organization, nil
}

func (r *OrganizationRepository) GetOrganizationWithMembers(orgID uuid.UUID) (response.OrganizationWithMembers, error) {
	// Get organization
	org, err := r.GetOrganizationByID(orgID)
	if err != nil {
		return response.OrganizationWithMembers{}, err
	}

	// Get members
	membersQuery := `
		SELECT uoa.user_id, u.username, uoa.role, uoa.created_at
		FROM user_organization_access uoa
		JOIN users u ON u.id = uoa.user_id
		WHERE uoa.organization_id = $1
		ORDER BY uoa.created_at ASC`

	rows, err := r.db.Query(membersQuery, orgID)
	if err != nil {
		return response.OrganizationWithMembers{}, err
	}
	defer rows.Close()

	var members []response.OrganizationMember
	for rows.Next() {
		var member response.OrganizationMember
		err := rows.Scan(&member.UserID, &member.Username, &member.Role, &member.JoinedAt)
		if err != nil {
			return response.OrganizationWithMembers{}, err
		}
		members = append(members, member)
	}

	return response.OrganizationWithMembers{
		ID:          org.ID,
		Name:        org.Name,
		Description: org.Description,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
		Members:     members,
	}, nil
}

func (r *OrganizationRepository) UpdateOrganization(orgID uuid.UUID, org *request.UpdateOrganization) (response.Organization, error) {
	query := `UPDATE organizations 
              SET name = $1, description = $2, updated_at = CURRENT_TIMESTAMP 
              WHERE id = $3 
              RETURNING id, name, description, created_at, updated_at`

	var organization response.Organization
	var description sql.NullString

	err := r.db.QueryRow(query, org.Name, org.Description, orgID).Scan(
		&organization.ID,
		&organization.Name,
		&description,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	)
	if err != nil {
		return response.Organization{}, err
	}

	if description.Valid {
		organization.Description = &description.String
	}

	return organization, nil
}

func (r *OrganizationRepository) DeleteOrganization(orgID uuid.UUID) error {
	query := `DELETE FROM organizations WHERE id = $1`
	result, err := r.db.Exec(query, orgID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization not found")
	}

	return nil
}

func (r *OrganizationRepository) GetUserOrganizations(userID uuid.UUID) (response.UserOrganizations, error) {
	query := `
		SELECT o.id, o.name, o.description, uoa.role, uoa.created_at
		FROM user_organization_access uoa
		JOIN organizations o ON o.id = uoa.organization_id
		WHERE uoa.user_id = $1
		ORDER BY uoa.created_at ASC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return response.UserOrganizations{}, err
	}
	defer rows.Close()

	var organizations []response.UserOrgAccess
	for rows.Next() {
		var org response.UserOrgAccess
		var description sql.NullString
		err := rows.Scan(&org.ID, &org.Name, &description, &org.Role, &org.JoinedAt)
		if err != nil {
			return response.UserOrganizations{}, err
		}

		if description.Valid {
			org.Description = &description.String
		}

		organizations = append(organizations, org)
	}

	return response.UserOrganizations{
		UserID:        userID,
		Organizations: organizations,
	}, nil
}

func (r *OrganizationRepository) AddUserToOrganization(orgID, userID uuid.UUID, role string) error {
	query := `INSERT INTO user_organization_access (user_id, organization_id, role) VALUES ($1, $2, $3)`
	_, err := r.db.Exec(query, userID, orgID, role)
	return err
}

func (r *OrganizationRepository) RemoveUserFromOrganization(orgID, userID uuid.UUID) error {
	query := `DELETE FROM user_organization_access WHERE organization_id = $1 AND user_id = $2`
	result, err := r.db.Exec(query, orgID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user access not found")
	}

	return nil
}

func (r *OrganizationRepository) UpdateUserRole(orgID, userID uuid.UUID, role string) error {
	query := `UPDATE user_organization_access SET role = $1 WHERE organization_id = $2 AND user_id = $3`
	result, err := r.db.Exec(query, role, orgID, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user access not found")
	}

	return nil
}

func (r *OrganizationRepository) CheckUserAccess(orgID, userID uuid.UUID) (string, error) {
	query := `SELECT role FROM user_organization_access WHERE organization_id = $1 AND user_id = $2`
	var role string
	err := r.db.QueryRow(query, orgID, userID).Scan(&role)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user does not have access to this organization")
		}
		return "", err
	}
	return role, nil
}

func (r *OrganizationRepository) IsUserOrgAdmin(orgID, userID uuid.UUID) (bool, error) {
	role, err := r.CheckUserAccess(orgID, userID)
	if err != nil {
		return false, err
	}
	return role == "admin", nil
}
