alter table notification_outbox
  add column if not exists kind text not null default 'generic',
  add column if not exists expires_at timestamptz,
  add column if not exists sensitive boolean not null default false;

create index if not exists notification_outbox_expiry_idx
  on notification_outbox(status, expires_at)
  where expires_at is not null and status in ('pending','processing');

create table if not exists contract_finalization_jobs (
  contract_id uuid primary key references contracts(id) on delete cascade,
  status text not null default 'pending' check (status in ('pending','processing','completed','failed')),
  attempts integer not null default 0,
  available_at timestamptz not null default now(),
  processing_at timestamptz,
  completed_at timestamptz,
  last_error text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists contract_finalization_jobs_pending_idx
  on contract_finalization_jobs(status, available_at)
  where status in ('pending','failed');

create table if not exists activation_profiles (
  id uuid primary key default gen_random_uuid(),
  contract_id uuid not null unique references contracts(id),
  client_id uuid not null references clients(id),
  status text not null default 'pending' check (status in ('pending','in_progress','completed','under_internal_setup','activated')),
  finance_responsible_name text,
  finance_responsible_phone text,
  finance_responsible_email citext,
  submitted_at timestamptz,
  activated_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists activation_profiles_status_idx
  on activation_profiles(status, updated_at desc);

create table if not exists activation_goods (
  id uuid primary key default gen_random_uuid(),
  activation_id uuid not null references activation_profiles(id) on delete cascade,
  description text not null,
  sort_order integer not null default 0
);

create index if not exists activation_goods_activation_idx
  on activation_goods(activation_id, sort_order, id);

create table if not exists activation_system_users (
  id uuid primary key default gen_random_uuid(),
  activation_id uuid not null references activation_profiles(id) on delete cascade,
  name text not null,
  phone text,
  email citext not null,
  sort_order integer not null default 0
);

create index if not exists activation_system_users_activation_idx
  on activation_system_users(activation_id, sort_order, id);

create table if not exists activation_access_tokens (
  id uuid primary key default gen_random_uuid(),
  activation_id uuid not null references activation_profiles(id) on delete cascade,
  token_hash bytea not null unique,
  access_type text not null default 'owner' check (access_type in ('owner','delegate')),
  section text not null default 'all' check (section in ('all','finance','goods','users')),
  name text,
  email citext,
  created_by_signer_id uuid references contract_signers(id),
  expires_at timestamptz not null,
  last_used_at timestamptz,
  revoked_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists activation_access_tokens_activation_idx
  on activation_access_tokens(activation_id, expires_at desc);

create table if not exists password_reset_tokens (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash bytea not null unique,
  expires_at timestamptz not null,
  used_at timestamptz,
  created_at timestamptz not null default now()
);

create index if not exists password_reset_tokens_user_idx
  on password_reset_tokens(user_id, expires_at desc);

create or replace function protect_activation_identity()
returns trigger language plpgsql as $$
begin
  if new.contract_id is distinct from old.contract_id or new.client_id is distinct from old.client_id then
    raise exception 'activation contract identity is immutable';
  end if;
  return new;
end;
$$;

drop trigger if exists activation_profiles_identity_immutable on activation_profiles;
create trigger activation_profiles_identity_immutable
before update on activation_profiles
for each row execute function protect_activation_identity();

insert into permissions(code,name) values
 ('activation.read_all','Ver todas as ativações'),
 ('activation.manage','Gerenciar implantação e ativação')
on conflict (code) do nothing;

insert into role_permissions(role_id,permission_id)
select r.id,p.id
from roles r
join permissions p on p.code in ('activation.read_all','activation.manage')
where r.code='super_admin'
on conflict do nothing;

insert into role_permissions(role_id,permission_id)
select r.id,p.id
from roles r
join permissions p on p.code in ('activation.read_all','activation.manage')
where r.code='operations'
on conflict do nothing;

insert into role_permissions(role_id,permission_id)
select r.id,p.id
from roles r
join permissions p on p.code='audit.read'
where r.code='legal'
on conflict do nothing;
