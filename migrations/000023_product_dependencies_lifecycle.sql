alter table product_categories
  add column if not exists archived_at timestamptz,
  add column if not exists deleted_at timestamptz;

alter table products
  add column if not exists archived_at timestamptz,
  add column if not exists deleted_at timestamptz;

create table if not exists product_dependency_groups (
  id uuid primary key default gen_random_uuid(),
  product_id uuid not null references products(id) on delete cascade,
  match_mode text not null check (match_mode in ('any','all')),
  sort_order integer not null default 0,
  created_at timestamptz not null default now()
);

create table if not exists product_dependencies (
  id uuid primary key default gen_random_uuid(),
  group_id uuid not null references product_dependency_groups(id) on delete cascade,
  required_product_id uuid not null references products(id),
  created_at timestamptz not null default now(),
  unique(group_id, required_product_id)
);

create index if not exists product_dependency_groups_product_idx
  on product_dependency_groups(product_id, sort_order);

create index if not exists product_dependencies_required_idx
  on product_dependencies(required_product_id);

create index if not exists product_categories_deleted_idx
  on product_categories(deleted_at)
  where deleted_at is not null;

create index if not exists products_deleted_idx
  on products(deleted_at)
  where deleted_at is not null;
