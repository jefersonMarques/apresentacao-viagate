create table if not exists post_sale_operations (
  contract_id uuid primary key references contracts(id),
  operation_type text check (operation_type in ('normal','avulsa')),
  insurer text,
  policy_start_date date,
  policy_end_date date,
  broker_company text,
  broker_producer text,
  updated_by_user_id uuid references users(id),
  updated_by_access_token_id uuid references activation_access_tokens(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint post_sale_policy_dates_valid check (
    policy_start_date is null or policy_end_date is null
    or policy_end_date >= policy_start_date
  )
);

create table if not exists post_sale_operation_history (
  id bigint generated always as identity primary key,
  contract_id uuid not null references contracts(id),
  operation_type text,
  insurer text,
  policy_start_date date,
  policy_end_date date,
  broker_company text,
  broker_producer text,
  actor_type text not null check (actor_type in ('customer','user')),
  actor_user_id uuid references users(id),
  actor_access_token_id uuid references activation_access_tokens(id),
  actor_name text,
  ip_address inet,
  user_agent text,
  created_at timestamptz not null default now()
);

create index if not exists post_sale_operation_history_contract_idx
  on post_sale_operation_history (contract_id, created_at desc, id desc);
