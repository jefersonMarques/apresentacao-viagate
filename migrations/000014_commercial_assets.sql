create table if not exists commercial_assets (
  id uuid primary key default gen_random_uuid(),
  owner_user_id uuid not null references users(id),
  category text not null check (category in ('salesperson_photo','client_logo')),
  storage_key text not null unique,
  original_filename text not null,
  mime_type text not null check (mime_type in ('image/jpeg','image/png','image/webp')),
  size_bytes bigint not null check (size_bytes > 0 and size_bytes <= 2097152),
  sha256 bytea not null,
  created_at timestamptz not null default now()
);

create index if not exists commercial_assets_owner_idx
  on commercial_assets(owner_user_id, created_at desc);

create index if not exists commercial_assets_category_idx
  on commercial_assets(category, created_at desc);
