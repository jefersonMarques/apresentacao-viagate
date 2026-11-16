create table if not exists post_sale_share_links (
  id bigint generated always as identity primary key,
  contract_id uuid not null references contracts(id),
  activation_access_token_id uuid not null unique references activation_access_tokens(id),
  created_by_user_id uuid not null references users(id),
  delivery text not null check (delivery in ('copy','email')),
  created_at timestamptz not null default now()
);

create index if not exists post_sale_share_links_contract_idx
  on post_sale_share_links (contract_id, created_at desc);
