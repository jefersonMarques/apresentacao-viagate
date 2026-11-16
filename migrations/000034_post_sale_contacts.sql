create table if not exists post_sale_contacts (
  contract_id uuid primary key references contracts(id),
  contact_mode text not null check (contact_mode in ('self','team')),
  contact_name text not null,
  contact_phone text not null,
  contact_email citext,
  updated_by_signer_id uuid references contract_signers(id),
  updated_by_user_id uuid references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists post_sale_contact_history (
  id bigint generated always as identity primary key,
  contract_id uuid not null references contracts(id),
  contact_mode text not null check (contact_mode in ('self','team')),
  contact_name text not null,
  contact_phone text not null,
  contact_email citext,
  actor_type text not null check (actor_type in ('customer','user')),
  actor_signer_id uuid references contract_signers(id),
  actor_user_id uuid references users(id),
  ip_address inet,
  user_agent text,
  created_at timestamptz not null default now()
);

create index if not exists post_sale_contact_history_contract_idx
  on post_sale_contact_history (contract_id, created_at desc, id desc);
