package emailtemplates

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const PurposeProposalShare = "proposal_share"

type Template struct {
	ID              string
	Purpose         string
	Name            string
	Description     string
	SubjectTemplate string
	HTMLTemplate    string
	TextTemplate    string
	IsActive        bool
	IsDefault       bool
}

type SaveInput struct {
	ID              string
	Purpose         string
	Name            string
	Description     string
	SubjectTemplate string
	HTMLTemplate    string
	TextTemplate    string
	IsActive        bool
	MakeDefault     bool
	UserID          string
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context, purpose string) ([]Template, error) {
	rows, err := s.pool.Query(ctx, "select id::text,purpose,name,coalesce(description,''),subject_template,html_template,text_template,is_active,is_default from email_templates where purpose=$1 order by is_default desc,is_active desc,name", purpose)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Template{}
	for rows.Next() {
		var item Template
		if err := rows.Scan(&item.ID, &item.Purpose, &item.Name, &item.Description, &item.SubjectTemplate, &item.HTMLTemplate, &item.TextTemplate, &item.IsActive, &item.IsDefault); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ListActive(ctx context.Context, purpose string) ([]Template, error) {
	rows, err := s.pool.Query(ctx, "select id::text,purpose,name,coalesce(description,''),subject_template,html_template,text_template,is_active,is_default from email_templates where purpose=$1 and is_active=true order by is_default desc,name", purpose)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Template{}
	for rows.Next() {
		var item Template
		if err := rows.Scan(&item.ID, &item.Purpose, &item.Name, &item.Description, &item.SubjectTemplate, &item.HTMLTemplate, &item.TextTemplate, &item.IsActive, &item.IsDefault); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ByID(ctx context.Context, id string) (Template, error) {
	var item Template
	err := s.pool.QueryRow(ctx, "select id::text,purpose,name,coalesce(description,''),subject_template,html_template,text_template,is_active,is_default from email_templates where id=$1", id).Scan(
		&item.ID,
		&item.Purpose,
		&item.Name,
		&item.Description,
		&item.SubjectTemplate,
		&item.HTMLTemplate,
		&item.TextTemplate,
		&item.IsActive,
		&item.IsDefault,
	)
	return item, err
}

func (s *Store) Default(ctx context.Context, purpose string) (Template, error) {
	var item Template
	err := s.pool.QueryRow(ctx, "select id::text,purpose,name,coalesce(description,''),subject_template,html_template,text_template,is_active,is_default from email_templates where purpose=$1 and is_active=true order by is_default desc,name limit 1", purpose).Scan(
		&item.ID,
		&item.Purpose,
		&item.Name,
		&item.Description,
		&item.SubjectTemplate,
		&item.HTMLTemplate,
		&item.TextTemplate,
		&item.IsActive,
		&item.IsDefault,
	)
	return item, err
}

func (s *Store) Save(ctx context.Context, input SaveInput) (string, error) {
	input.ID = strings.TrimSpace(input.ID)
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Purpose = strings.TrimSpace(input.Purpose)
	if input.Name == "" || input.Purpose == "" {
		return "", fmt.Errorf("name and purpose are required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	templateID := input.ID
	if templateID == "" {
		if err := tx.QueryRow(ctx, "insert into email_templates(purpose,name,description,subject_template,html_template,text_template,is_active,is_default,created_by,updated_by) values($1,$2,nullif($3,''),$4,$5,$6,$7,false,$8,$8) returning id::text", input.Purpose, input.Name, input.Description, input.SubjectTemplate, input.HTMLTemplate, input.TextTemplate, input.IsActive, input.UserID).Scan(&templateID); err != nil {
			return "", err
		}
	} else {
		command, err := tx.Exec(ctx, "update email_templates set name=$3,description=nullif($4,''),subject_template=$5,html_template=$6,text_template=$7,is_active=$8,updated_by=$9,updated_at=now() where id=$1 and purpose=$2", templateID, input.Purpose, input.Name, input.Description, input.SubjectTemplate, input.HTMLTemplate, input.TextTemplate, input.IsActive, input.UserID)
		if err != nil {
			return "", err
		}
		if command.RowsAffected() != 1 {
			return "", pgx.ErrNoRows
		}
	}

	if !input.IsActive {
		if _, err := tx.Exec(ctx, "update email_templates set is_default=false where id=$1", templateID); err != nil {
			return "", err
		}
	} else if input.MakeDefault {
		if _, err := tx.Exec(ctx, "update email_templates set is_default=false where purpose=$1 and id<>$2", input.Purpose, templateID); err != nil {
			return "", err
		}
		if _, err := tx.Exec(ctx, "update email_templates set is_default=true where id=$1", templateID); err != nil {
			return "", err
		}
	}

	var hasDefault bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from email_templates where purpose=$1 and is_active=true and is_default=true)", input.Purpose).Scan(&hasDefault); err != nil {
		return "", err
	}
	if !hasDefault {
		if _, err := tx.Exec(ctx, "update email_templates set is_default=true where id=(select id from email_templates where purpose=$1 and is_active=true order by updated_at desc,name limit 1)", input.Purpose); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return templateID, nil
}
