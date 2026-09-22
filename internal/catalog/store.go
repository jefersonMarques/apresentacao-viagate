package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCatalogInUse           = errors.New("catalog item is in use")
	ErrCatalogHasProducts     = errors.New("category has products")
	ErrCatalogDependencyInUse = errors.New("product is required by dependency")
	ErrCatalogNotFound        = errors.New("catalog item not found")
	ErrDependencyCycle        = errors.New("product dependency cycle")
	ErrInvalidDependency      = errors.New("invalid product dependency")
)

type ManagedCategory struct {
	ID          string
	Code        string
	Name        string
	Description string
	IsActive    bool
	IsArchived  bool
	IsDeleted   bool
	SortOrder   int
	Products    []ManagedProduct
}

type ManagedProduct struct {
	ID               string
	CategoryID       string
	CategoryCode     string
	Code             string
	Name             string
	Description      string
	Unit             string
	IsActive         bool
	IsArchived       bool
	IsDeleted        bool
	SortOrder        int
	DependencyGroups []DependencyGroup
}

type DependencyGroup struct {
	ID               string
	MatchMode        string
	SortOrder        int
	RequiredProducts []DependencyProduct
}

type DependencyProduct struct {
	ID           string
	Code         string
	Name         string
	CategoryName string
	IsActive     bool
	IsArchived   bool
	IsDeleted    bool
}

type DependencyInput struct {
	MatchMode          string
	RequiredProductIDs []string
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListAdmin(ctx context.Context, showDeleted bool) ([]ManagedCategory, error) {
	rows, err := s.pool.Query(ctx, `
		select
			id::text,code,name,coalesce(description,''),is_active,
			archived_at is not null,deleted_at is not null,sort_order
		from product_categories
		where $1 or deleted_at is null
		order by sort_order,name
	`, showDeleted)
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
			&category.IsArchived,
			&category.IsDeleted,
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
			coalesce(p.description,''),coalesce(p.unit,''),p.is_active,
			p.archived_at is not null,p.deleted_at is not null,p.sort_order
		from products p
		join product_categories c on c.id=p.category_id
		where ($1 or p.deleted_at is null)
		  and ($1 or c.deleted_at is null)
		order by c.sort_order,c.name,p.sort_order,p.name
	`, showDeleted)
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
			&product.IsArchived,
			&product.IsDeleted,
			&product.SortOrder,
		); err != nil {
			return nil, err
		}
		if index, ok := indexes[product.CategoryID]; ok {
			categories[index].Products = append(categories[index].Products, product)
		}
	}
	if err := productRows.Err(); err != nil {
		return nil, err
	}
	if err := s.loadDependencies(ctx, categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func (s *Store) ListForProposal(ctx context.Context, selectedCodes []string) ([]ManagedCategory, error) {
	rows, err := s.pool.Query(ctx, `
		select
			c.id::text,c.code,c.name,coalesce(c.description,''),c.is_active,
			c.archived_at is not null,c.deleted_at is not null,c.sort_order,
			p.id::text,p.category_id::text,p.code,p.name,coalesce(p.description,''),coalesce(p.unit,''),
			p.is_active,p.archived_at is not null,p.deleted_at is not null,p.sort_order
		from product_categories c
		join products p on p.category_id=c.id
		where (
			c.is_active=true and c.archived_at is null and c.deleted_at is null
			and p.is_active=true and p.archived_at is null and p.deleted_at is null
		)
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
			&category.IsArchived,
			&category.IsDeleted,
			&category.SortOrder,
			&product.ID,
			&product.CategoryID,
			&product.Code,
			&product.Name,
			&product.Description,
			&product.Unit,
			&product.IsActive,
			&product.IsArchived,
			&product.IsDeleted,
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := s.loadDependencies(ctx, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Store) ProductForProposal(ctx context.Context, code, proposalID string) (ManagedCategory, ManagedProduct, error) {
	var category ManagedCategory
	var product ManagedProduct
	err := s.pool.QueryRow(ctx, `
		select
			c.id::text,c.code,c.name,coalesce(c.description,''),c.is_active,
			c.archived_at is not null,c.deleted_at is not null,c.sort_order,
			p.id::text,p.category_id::text,p.code,p.name,coalesce(p.description,''),coalesce(p.unit,''),
			p.is_active,p.archived_at is not null,p.deleted_at is not null,p.sort_order
		from products p
		join product_categories c on c.id=p.category_id
		where p.code=$1
		  and (
			(
				c.is_active=true and c.archived_at is null and c.deleted_at is null
				and p.is_active=true and p.archived_at is null and p.deleted_at is null
			)
			or (
				nullif($2,'') is not null
				and exists(
					select 1
					from proposal_versions v
					join proposal_items pi on pi.proposal_version_id=v.id
					where v.proposal_id=nullif($2,'')::uuid
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
		&category.IsArchived,
		&category.IsDeleted,
		&category.SortOrder,
		&product.ID,
		&product.CategoryID,
		&product.Code,
		&product.Name,
		&product.Description,
		&product.Unit,
		&product.IsActive,
		&product.IsArchived,
		&product.IsDeleted,
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
		where id=$1 and deleted_at is null
	`, id, name, strings.TrimSpace(description), isActive, sortOrder)
	if err != nil {
		return "", err
	}
	if result.RowsAffected() == 0 {
		return "", ErrCatalogNotFound
	}
	return id, nil
}

func (s *Store) SaveProduct(
	ctx context.Context,
	id, categoryID, name, description, unit string,
	isActive bool,
	sortOrder int,
	dependencies []DependencyInput,
) (string, error) {
	name = strings.TrimSpace(name)
	categoryID = strings.TrimSpace(categoryID)
	if name == "" || categoryID == "" {
		return "", fmt.Errorf("product name and category are required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var categoryExists bool
	if err := tx.QueryRow(ctx, `
		select exists(select 1 from product_categories where id=$1 and deleted_at is null)
	`, categoryID).Scan(&categoryExists); err != nil {
		return "", err
	}
	if !categoryExists {
		return "", ErrCatalogNotFound
	}

	productID := strings.TrimSpace(id)
	if productID == "" {
		if err := tx.QueryRow(ctx, `
			insert into products(category_id,code,name,description,unit,is_active,sort_order)
			select
				c.id,
				'product-' || substr(replace(gen_random_uuid()::text,'-',''),1,12),
				$2,nullif($3,''),nullif($4,''),$5,$6
			from product_categories c
			where c.id=$1 and c.deleted_at is null
			returning id::text
		`, categoryID, name, strings.TrimSpace(description), strings.TrimSpace(unit), isActive, sortOrder).Scan(&productID); err != nil {
			return "", err
		}
	} else {
		result, err := tx.Exec(ctx, `
			update products
			set category_id=$2,name=$3,description=nullif($4,''),unit=nullif($5,''),is_active=$6,sort_order=$7,updated_at=now()
			where id=$1 and deleted_at is null
		`, productID, categoryID, name, strings.TrimSpace(description), strings.TrimSpace(unit), isActive, sortOrder)
		if err != nil {
			return "", err
		}
		if result.RowsAffected() == 0 {
			return "", ErrCatalogNotFound
		}
	}

	if err := replaceDependencies(ctx, tx, productID, dependencies); err != nil {
		return "", err
	}
	if err := ensureNoDependencyCycles(ctx, tx); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return productID, nil
}

func (s *Store) UpdateProductLifecycle(ctx context.Context, productID, action string) error {
	switch action {
	case "archive":
		return execLifecycle(ctx, s.pool, `
			update products set archived_at=coalesce(archived_at,now()),updated_at=now()
			where id=$1 and deleted_at is null
		`, productID)
	case "unarchive":
		return execLifecycle(ctx, s.pool, `
			update products set archived_at=null,updated_at=now()
			where id=$1 and deleted_at is null
		`, productID)
	case "restore":
		return execLifecycle(ctx, s.pool, `
			update products p
			set deleted_at=null,updated_at=now()
			where p.id=$1 and p.deleted_at is not null
			  and exists(
				select 1 from product_categories c
				where c.id=p.category_id and c.deleted_at is null
			  )
		`, productID)
	case "delete":
		var code string
		if err := s.pool.QueryRow(ctx, `select code from products where id=$1 and deleted_at is null`, productID).Scan(&code); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCatalogNotFound
			}
			return err
		}
		var used bool
		if err := s.pool.QueryRow(ctx, `
			select exists(
				select 1
				from proposal_items
				where coalesce(metadata->>'catalog_id',metadata->>'product_code','')=$1
			)
		`, code).Scan(&used); err != nil {
			return err
		}
		if used {
			return ErrCatalogInUse
		}
		var required bool
		if err := s.pool.QueryRow(ctx, `
			select exists(
				select 1
				from product_dependencies d
				join product_dependency_groups g on g.id=d.group_id
				join products owner on owner.id=g.product_id
				where d.required_product_id=$1
				  and owner.deleted_at is null
			)
		`, productID).Scan(&required); err != nil {
			return err
		}
		if required {
			return ErrCatalogDependencyInUse
		}
		return execLifecycle(ctx, s.pool, `
			update products set deleted_at=now(),updated_at=now()
			where id=$1 and deleted_at is null
		`, productID)
	default:
		return fmt.Errorf("invalid lifecycle action")
	}
}

func (s *Store) UpdateCategoryLifecycle(ctx context.Context, categoryID, action string) error {
	switch action {
	case "archive":
		return execLifecycle(ctx, s.pool, `
			update product_categories set archived_at=coalesce(archived_at,now()),updated_at=now()
			where id=$1 and deleted_at is null
		`, categoryID)
	case "unarchive":
		return execLifecycle(ctx, s.pool, `
			update product_categories set archived_at=null,updated_at=now()
			where id=$1 and deleted_at is null
		`, categoryID)
	case "restore":
		return execLifecycle(ctx, s.pool, `
			update product_categories set deleted_at=null,updated_at=now()
			where id=$1 and deleted_at is not null
		`, categoryID)
	case "delete":
		var code string
		if err := s.pool.QueryRow(ctx, `select code from product_categories where id=$1 and deleted_at is null`, categoryID).Scan(&code); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrCatalogNotFound
			}
			return err
		}
		var hasProducts bool
		if err := s.pool.QueryRow(ctx, `
			select exists(select 1 from products where category_id=$1 and deleted_at is null)
		`, categoryID).Scan(&hasProducts); err != nil {
			return err
		}
		if hasProducts {
			return ErrCatalogHasProducts
		}
		var used bool
		if err := s.pool.QueryRow(ctx, `
			select exists(
				select 1
				from proposal_items
				where metadata->>'category_code'=$1
			)
		`, code).Scan(&used); err != nil {
			return err
		}
		if used {
			return ErrCatalogInUse
		}
		return execLifecycle(ctx, s.pool, `
			update product_categories set deleted_at=now(),updated_at=now()
			where id=$1 and deleted_at is null
		`, categoryID)
	default:
		return fmt.Errorf("invalid lifecycle action")
	}
}

func (s *Store) ValidateSelection(ctx context.Context, selectedCodes []string) error {
	if len(selectedCodes) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		select
			p.code,p.name,g.id::text,g.match_mode,g.sort_order,
			rp.code,rp.name
		from products p
		join product_dependency_groups g on g.product_id=p.id
		join product_dependencies d on d.group_id=g.id
		join products rp on rp.id=d.required_product_id
		where p.code = any($1::text[])
		order by p.name,g.sort_order,g.id,rp.name
	`, selectedCodes)
	if err != nil {
		return err
	}
	defer rows.Close()

	type validationGroup struct {
		ProductCode string
		ProductName string
		GroupID     string
		MatchMode   string
		Required    []DependencyProduct
	}
	groups := []validationGroup{}
	indexes := map[string]int{}
	for rows.Next() {
		var productCode, productName, groupID, matchMode, requiredCode, requiredName string
		var sortOrder int
		if err := rows.Scan(
			&productCode,
			&productName,
			&groupID,
			&matchMode,
			&sortOrder,
			&requiredCode,
			&requiredName,
		); err != nil {
			return err
		}
		_ = sortOrder
		key := productCode + ":" + groupID
		index, ok := indexes[key]
		if !ok {
			index = len(groups)
			indexes[key] = index
			groups = append(groups, validationGroup{
				ProductCode: productCode,
				ProductName: productName,
				GroupID:     groupID,
				MatchMode:   matchMode,
			})
		}
		groups[index].Required = append(groups[index].Required, DependencyProduct{
			Code: requiredCode,
			Name: requiredName,
		})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	selected := map[string]bool{}
	for _, code := range selectedCodes {
		selected[code] = true
	}
	for _, group := range groups {
		matched := 0
		names := make([]string, 0, len(group.Required))
		for _, required := range group.Required {
			names = append(names, required.Name)
			if selected[required.Code] {
				matched++
			}
		}
		valid := matched == len(group.Required)
		if group.MatchMode == "any" {
			valid = matched > 0
		}
		if valid {
			continue
		}
		connector := " e "
		if group.MatchMode == "any" {
			connector = " ou "
		}
		return fmt.Errorf("O produto %q requer %s.", group.ProductName, strings.Join(names, connector))
	}
	return nil
}

func replaceDependencies(ctx context.Context, tx pgx.Tx, productID string, inputs []DependencyInput) error {
	if _, err := tx.Exec(ctx, `delete from product_dependency_groups where product_id=$1`, productID); err != nil {
		return err
	}

	for groupIndex, input := range inputs {
		mode := strings.TrimSpace(input.MatchMode)
		if mode != "any" && mode != "all" {
			return ErrInvalidDependency
		}

		requiredIDs := uniqueNonBlank(input.RequiredProductIDs)
		if len(requiredIDs) == 0 {
			continue
		}
		for _, requiredID := range requiredIDs {
			if requiredID == productID {
				return ErrInvalidDependency
			}
			var valid bool
			if err := tx.QueryRow(ctx, `
				select exists(
					select 1
					from products
					where id=$1
					  and is_active=true
					  and archived_at is null
					  and deleted_at is null
				)
			`, requiredID).Scan(&valid); err != nil {
				return err
			}
			if !valid {
				return ErrInvalidDependency
			}
		}

		var groupID string
		if err := tx.QueryRow(ctx, `
			insert into product_dependency_groups(product_id,match_mode,sort_order)
			values($1,$2,$3)
			returning id::text
		`, productID, mode, groupIndex).Scan(&groupID); err != nil {
			return err
		}
		for _, requiredID := range requiredIDs {
			if _, err := tx.Exec(ctx, `
				insert into product_dependencies(group_id,required_product_id)
				values($1,$2)
			`, groupID, requiredID); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureNoDependencyCycles(ctx context.Context, tx pgx.Tx) error {
	var hasCycle bool
	if err := tx.QueryRow(ctx, `
		with recursive edges(product_id,required_product_id) as (
			select g.product_id,d.required_product_id
			from product_dependency_groups g
			join product_dependencies d on d.group_id=g.id
		),
		walk(root_id,current_id) as (
			select product_id,required_product_id from edges
			union
			select w.root_id,e.required_product_id
			from walk w
			join edges e on e.product_id=w.current_id
		)
		select exists(select 1 from walk where root_id=current_id)
	`).Scan(&hasCycle); err != nil {
		return err
	}
	if hasCycle {
		return ErrDependencyCycle
	}
	return nil
}

func (s *Store) loadDependencies(ctx context.Context, categories []ManagedCategory) error {
	productIndexes := map[string][2]int{}
	for categoryIndex := range categories {
		for productIndex := range categories[categoryIndex].Products {
			productIndexes[categories[categoryIndex].Products[productIndex].ID] = [2]int{categoryIndex, productIndex}
		}
	}
	if len(productIndexes) == 0 {
		return nil
	}

	rows, err := s.pool.Query(ctx, `
		select
			g.id::text,g.product_id::text,g.match_mode,g.sort_order,
			rp.id::text,rp.code,rp.name,rc.name,rp.is_active,
			rp.archived_at is not null,rp.deleted_at is not null
		from product_dependency_groups g
		join product_dependencies d on d.group_id=g.id
		join products rp on rp.id=d.required_product_id
		join product_categories rc on rc.id=rp.category_id
		order by g.product_id,g.sort_order,g.id,rc.sort_order,rc.name,rp.sort_order,rp.name
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	groupIndexes := map[string]int{}
	for rows.Next() {
		var groupID, productID, matchMode string
		var sortOrder int
		var required DependencyProduct
		if err := rows.Scan(
			&groupID,
			&productID,
			&matchMode,
			&sortOrder,
			&required.ID,
			&required.Code,
			&required.Name,
			&required.CategoryName,
			&required.IsActive,
			&required.IsArchived,
			&required.IsDeleted,
		); err != nil {
			return err
		}
		position, ok := productIndexes[productID]
		if !ok {
			continue
		}
		product := &categories[position[0]].Products[position[1]]
		groupKey := productID + ":" + groupID
		groupIndex, ok := groupIndexes[groupKey]
		if !ok {
			groupIndex = len(product.DependencyGroups)
			groupIndexes[groupKey] = groupIndex
			product.DependencyGroups = append(product.DependencyGroups, DependencyGroup{
				ID:        groupID,
				MatchMode: matchMode,
				SortOrder: sortOrder,
			})
		}
		product.DependencyGroups[groupIndex].RequiredProducts = append(
			product.DependencyGroups[groupIndex].RequiredProducts,
			required,
		)
	}
	return rows.Err()
}

func execLifecycle(ctx context.Context, pool *pgxpool.Pool, query, id string) error {
	result, err := pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrCatalogNotFound
	}
	return nil
}

func uniqueNonBlank(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
