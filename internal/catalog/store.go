package catalog

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ManagedCategory struct {
	ID          string
	Code        string
	Name        string
	Description string
	IsActive    bool
	SortOrder   int
	Products    []ManagedProduct
}

type ManagedProduct struct {
	ID           string
	CategoryID   string
	CategoryCode string
	Code         string
	Name         string
	Description  string
	Unit         string
	IsActive     bool
	SortOrder    int
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListAdmin(ctx context.Context) ([]ManagedCategory, error) {
	rows, err := s.pool.Query(ctx, `
		select id::text,code,name,coalesce(description,''),is_active,sort_order
		from product_categories
		order by sort_order,name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	categories := []ManagedCategory{}
	indexes := map[string]int{}
	for rows.Next() {
		var category ManagedCategory
		if err := rows.Scan(
			&category.ID,
			&category.Code,
			&category.Name,
			&category.Description,
			&category.IsActive,
			&category.SortOrder,
		); err != nil {
			return nil, err
		}
		indexes[category.ID] = len(categories)
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	productRows, err := s.pool.Query(ctx, `
		select
			p.id::text,p.category_id::text,c.code,p.code,p.name,
			coalesce(p.description,''),coalesce(p.unit,''),p.is_active,p.sort_order
		from products p
		join product_categories c on c.id=p.category_id
		order by c.sort_order,c.name,p.sort_order,p.name
	`)
	if err != nil {
		return nil, err
	}
	defer productRows.Close()

	for productRows.Next() {
		var product ManagedProduct
		if err := productRows.Scan(
			&product.ID,
			&product.CategoryID,
			&product.CategoryCode,
			&product.Code,
			&product.Name,
			&product.Description,
			&product.Unit,
			&product.IsActive,
			&product.SortOrder,
		); err != nil {
			return nil, err
		}
		if index, ok := indexes[product.CategoryID]; ok {
			categories[index].Products = append(categories[index].Products, product)
		}
	}
	return categories, productRows.Err()
}

func (s *Store) ListForProposal(ctx context.Context, selectedCodes []string) ([]ManagedCategory, error) {
	rows, err := s.pool.Query(ctx, `
		select
			c.id::text,c.code,c.name,coalesce(c.description,''),c.is_active,c.sort_order,
			p.id::text,p.category_id::text,p.code,p.name,coalesce(p.description,''),coalesce(p.unit,''),p.is_active,p.sort_order
		from product_categories c
		join products p on p.category_id=c.id
		where (c.is_active=true and p.is_active=true)
		   or p.code = any($1::text[])
		order by c.sort_order,c.name,p.sort_order,p.name
	`, selectedCodes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []ManagedCategory{}
	indexes := map[string]int{}
	for rows.Next() {
		var category ManagedCategory
		var product ManagedProduct
		if err := rows.Scan(
			&category.ID,
			&category.Code,
			&category.Name,
			&category.Description,
			&category.IsActive,
			&category.SortOrder,
			&product.ID,
			&product.CategoryID,
			&product.Code,
			&product.Name,
			&product.Description,
			&product.Unit,
			&product.IsActive,
			&product.SortOrder,
		); err != nil {
			return nil, err
		}
		product.CategoryCode = category.Code
		index, ok := indexes[category.ID]
		if !ok {
			index = len(result)
			indexes[category.ID] = index
			result = append(result, category)
		}
		result[index].Products = append(result[index].Products, product)
	}
	return result, rows.Err()
}

func (s *Store) ProductForProposal(ctx context.Context, code, proposalID string) (ManagedCategory, ManagedProduct, error) {
	var category ManagedCategory
	var product ManagedProduct
	err := s.pool.QueryRow(ctx, `
		select
			c.id::text,c.code,c.name,coalesce(c.description,''),c.is_active,c.sort_order,
			p.id::text,p.category_id::text,p.code,p.name,coalesce(p.description,''),coalesce(p.unit,''),p.is_active,p.sort_order
		from products p
		join product_categories c on c.id=p.category_id
		where p.code=$1
		  and (
			(c.is_active=true and p.is_active=true)
			or (
				nullif($2,'') is not null
				and exists(
					select 1
					from proposal_versions v
					join proposal_items pi on pi.proposal_version_id=v.id
					where v.proposal_id=$2::uuid
					  and coalesce(pi.metadata->>'catalog_id',pi.metadata->>'product_code','')=$1
				)
			)
		  )
	`, strings.TrimSpace(code), strings.TrimSpace(proposalID)).Scan(
		&category.ID,
		&category.Code,
		&category.Name,
		&category.Description,
		&category.IsActive,
		&category.SortOrder,
		&product.ID,
		&product.CategoryID,
		&product.Code,
		&product.Name,
		&product.Description,
		&product.Unit,
		&product.IsActive,
		&product.SortOrder,
	)
	if err != nil {
		return ManagedCategory{}, ManagedProduct{}, err
	}
	product.CategoryCode = category.Code
	return category, product, nil
}

func (s *Store) SaveCategory(ctx context.Context, id, name, description string, isActive bool, sortOrder int) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("category name is required")
	}
	if strings.TrimSpace(id) == "" {
		var createdID string
		err := s.pool.QueryRow(ctx, `
			insert into product_categories(code,name,description,is_active,sort_order)
			values(
				'category-' || substr(replace(gen_random_uuid()::text,'-',''),1,12),
				$1,nullif($2,''),$3,$4
			)
			returning id::text
		`, name, strings.TrimSpace(description), isActive, sortOrder).Scan(&createdID)
		return createdID, err
	}
	result, err := s.pool.Exec(ctx, `
		update product_categories
		set name=$2,description=nullif($3,''),is_active=$4,sort_order=$5,updated_at=now()
		where id=$1
	`, id, name, strings.TrimSpace(description), isActive, sortOrder)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}
	return id, nil
}

func (s *Store) SaveProduct(ctx context.Context, id, categoryID, name, description, unit string, isActive bool, sortOrder int) (string, error) {
	name = strings.TrimSpace(name)
	categoryID = strings.TrimSpace(categoryID)
	if name == "" || categoryID == "" {
		return "", fmt.Errorf("product name and category are required")
	}
	if strings.TrimSpace(id) == "" {
		var createdID string
		err := s.pool.QueryRow(ctx, `
			insert into products(category_id,code,name,description,unit,is_active,sort_order)
			values(
				$1,
				'product-' || substr(replace(gen_random_uuid()::text,'-',''),1,12),
				$2,nullif($3,''),nullif($4,''),$5,$6
			)
			returning id::text
		`, categoryID, name, strings.TrimSpace(description), strings.TrimSpace(unit), isActive, sortOrder).Scan(&createdID)
		return createdID, err
	}
	result, err := s.pool.Exec(ctx, `
		update products
		set category_id=$2,name=$3,description=nullif($4,''),unit=nullif($5,''),is_active=$6,sort_order=$7,updated_at=now()
		where id=$1
	`, id, categoryID, name, strings.TrimSpace(description), strings.TrimSpace(unit), isActive, sortOrder)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", pgx.ErrNoRows
	}
	return id, nil
}
