package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	iammodel "github.com/masterfabric/masterfabric_backend/internal/domain/iam/model"
	iamrepo "github.com/masterfabric/masterfabric_backend/internal/domain/iam/repository"
	domainerr "github.com/masterfabric/masterfabric_backend/internal/shared/errors"
)

var _ iamrepo.RoleRepository = (*RoleRepository)(nil)

type RoleRepository struct {
	pool *pgxpool.Pool
}

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{pool: pool}
}

func (r *RoleRepository) FindByName(ctx context.Context, name string) (*iammodel.Role, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, permissions FROM roles WHERE name = $1
	`, name)
	role, err := scanRole(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domainerr.New(domainerr.CodeNotFound, "role not found", err)
		}
		return nil, domainerr.New(domainerr.CodeInternal, "find role by name", fmt.Errorf("query role: %w", err))
	}
	return role, nil
}

func (r *RoleRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]iammodel.Role, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT r.id, r.name, r.permissions
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`, userID)
	if err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "find roles by user", fmt.Errorf("query roles: %w", err))
	}
	defer rows.Close()
	var roles []iammodel.Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, domainerr.New(domainerr.CodeInternal, "scan role", err)
		}
		roles = append(roles, *role)
	}
	if err := rows.Err(); err != nil {
		return nil, domainerr.New(domainerr.CodeInternal, "iterate roles", err)
	}
	return roles, nil
}

func (r *RoleRepository) AssignToUser(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`, userID, roleID)
	if err != nil {
		return domainerr.New(domainerr.CodeInternal, "assign role to user", fmt.Errorf("insert user_roles: %w", err))
	}
	return nil
}

type roleScanner interface {
	Scan(dest ...any) error
}

func scanRole(row roleScanner) (*iammodel.Role, error) {
	var r iammodel.Role
	var permsRaw []byte
	if err := row.Scan(&r.ID, &r.Name, &permsRaw); err != nil {
		return nil, err
	}
	var perms []string
	if err := json.Unmarshal(permsRaw, &perms); err != nil {
		return nil, fmt.Errorf("unmarshal permissions: %w", err)
	}
	r.Permissions = make([]iammodel.Permission, 0, len(perms))
	for _, p := range perms {
		r.Permissions = append(r.Permissions, iammodel.Permission(p))
	}
	return &r, nil
}
