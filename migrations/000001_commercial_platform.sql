create extension if not exists pgcrypto;
create extension if not exists citext;

create type user_status as enum ('invited','active','disabled');
create type proposal_status as enum ('draft','published','accepted','expired','cancelled');
create type onboarding_status as enum ('pending','in_progress','submitted','under_review','correction_requested','approved');
create type contract_status as enum ('draft','generated','sent','partially_signed','signed','cancelled');
create type signer_status as enum ('pending','otp_sent','verified','signed','cancelled');
create type identity_mode as enum ('email_otp','phone_otp','document','face','liveness','certificate');
create type identity_status as enum ('pending','verified','failed','not_required');

create table users (
  id uuid primary key default gen_random_uuid(),
  email citext not null unique,
  name text not null,
  password_hash text,
  status user_status not null default 'invited',
  last_login_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table roles (
  id uuid primary key default gen_random_uuid(),
  code text not null unique,
  name text not null
);

create table permissions (
  id uuid primary key default gen_random_uuid(),
  code text not null unique,
  name text not null
);

create table user_roles (
  user_id uuid not null references users(id) on delete cascade,
  role_id uuid not null references roles(id) on delete cascade,
  primary key (user_id, role_id)
);

create table role_permissions (
  role_id uuid not null references roles(id) on delete cascade,
  permission_id uuid not null references permissions(id) on delete cascade,
  primary key (role_id, permission_id)
);

create table user_invitations (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash bytea not null unique,
  expires_at timestamptz not null,
  accepted_at timestamptz,
  created_by uuid references users(id),
  created_at timestamptz not null default now()
);

create table sessions (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references users(id) on delete cascade,
  token_hash bytea not null unique,
  ip_address inet,
  user_agent text,
  expires_at timestamptz not null,
  created_at timestamptz not null default now(),
  revoked_at timestamptz
);

create table clients (
  id uuid primary key default gen_random_uuid(),
  legal_name text not null,
  trade_name text,
  cnpj varchar(14),
  email citext,
  phone text,
  street text,
  street_number text,
  complement text,
  district text,
  city text,
  state char(2),
  postal_code text,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);
create unique index clients_cnpj_unique on clients(cnpj) where cnpj is not null;

create table presentations (
  id uuid primary key default gen_random_uuid(),
  client_id uuid references clients(id),
  title text not null,
  status proposal_status not null default 'draft',
  current_version integer not null default 0,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table presentation_versions (
  id uuid primary key default gen_random_uuid(),
  presentation_id uuid not null references presentations(id) on delete cascade,
  version_number integer not null,
  public_token uuid not null unique default gen_random_uuid(),
  content jsonb not null,
  content_hash bytea not null,
  published_at timestamptz,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  unique(presentation_id, version_number)
);

create table proposals (
  id uuid primary key default gen_random_uuid(),
  client_id uuid not null references clients(id),
  title text not null,
  status proposal_status not null default 'draft',
  current_version integer not null default 0,
  valid_until date,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table proposal_versions (
  id uuid primary key default gen_random_uuid(),
  proposal_id uuid not null references proposals(id) on delete cascade,
  version_number integer not null,
  public_token uuid not null unique default gen_random_uuid(),
  pricing_model text not null,
  content jsonb not null,
  conditions jsonb not null default '[]'::jsonb,
  minimum_invoice numeric(14,2) not null default 0,
  setup_fee numeric(14,2) not null default 0,
  content_hash bytea not null,
  published_at timestamptz,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  unique(proposal_id, version_number)
);

create table proposal_items (
  id uuid primary key default gen_random_uuid(),
  proposal_version_id uuid not null references proposal_versions(id) on delete cascade,
  group_name text not null,
  label text not null,
  unit text,
  price numeric(14,4) not null default 0,
  is_optional boolean not null default false,
  sort_order integer not null default 0,
  metadata jsonb not null default '{}'::jsonb
);

create table proposal_acceptances (
  id uuid primary key default gen_random_uuid(),
  proposal_id uuid not null references proposals(id),
  proposal_version_id uuid not null references proposal_versions(id),
  proposal_hash bytea not null,
  accepted_by_name text not null,
  accepted_by_email citext not null,
  accepted_by_cpf varchar(11) not null,
  accepted_by_phone text not null,
  accepted_by_role text,
  authority_declared boolean not null,
  acceptance_text_version text not null,
  ip_address inet,
  user_agent text,
  session_id uuid not null,
  accepted_at timestamptz not null default now(),
  unique(proposal_version_id)
);

create table onboardings (
  id uuid primary key default gen_random_uuid(),
  proposal_acceptance_id uuid not null unique references proposal_acceptances(id),
  client_id uuid not null references clients(id),
  status onboarding_status not null default 'pending',
  cnpj varchar(14) not null,
  legal_name text not null,
  trade_name text,
  street text,
  street_number text,
  complement text,
  district text,
  city text,
  state char(2),
  postal_code text,
  operation_type text,
  insurer text,
  policy_start_date date,
  policy_end_date date,
  broker_company text,
  broker_producer text,
  company_responsible_name text not null,
  company_responsible_cpf varchar(11) not null,
  company_responsible_phone text not null,
  company_responsible_email citext not null,
  company_responsible_role text,
  company_responsible_authority_declared boolean not null default false,
  finance_responsible_name text,
  finance_responsible_phone text,
  finance_responsible_email citext,
  submitted_at timestamptz,
  approved_at timestamptz,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table onboarding_goods (
  id uuid primary key default gen_random_uuid(),
  onboarding_id uuid not null references onboardings(id) on delete cascade,
  description text not null,
  sort_order integer not null default 0
);

create table onboarding_system_users (
  id uuid primary key default gen_random_uuid(),
  onboarding_id uuid not null references onboardings(id) on delete cascade,
  name text not null,
  phone text,
  email citext not null,
  sort_order integer not null default 0
);

create table uploaded_documents (
  id uuid primary key default gen_random_uuid(),
  onboarding_id uuid not null references onboardings(id) on delete cascade,
  document_type text not null,
  storage_key text not null unique,
  original_filename text not null,
  mime_type text not null,
  size_bytes bigint not null check (size_bytes > 0),
  sha256 bytea not null,
  status text not null default 'uploaded',
  uploaded_at timestamptz not null default now()
);

create table contract_templates (
  id uuid primary key default gen_random_uuid(),
  name text not null,
  description text,
  is_active boolean not null default true,
  current_version integer not null default 0,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table contract_template_versions (
  id uuid primary key default gen_random_uuid(),
  contract_template_id uuid not null references contract_templates(id),
  version_number integer not null,
  markdown text not null,
  template_hash bytea not null,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  unique(contract_template_id, version_number)
);

create table contracts (
  id uuid primary key default gen_random_uuid(),
  onboarding_id uuid not null references onboardings(id),
  proposal_version_id uuid not null references proposal_versions(id),
  template_version_id uuid not null references contract_template_versions(id),
  status contract_status not null default 'draft',
  rendered_markdown text,
  rendered_html text,
  pdf_storage_key text,
  document_sha256 bytea,
  generated_at timestamptz,
  sent_at timestamptz,
  fully_signed_at timestamptz,
  created_by uuid not null references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table contract_signers (
  id uuid primary key default gen_random_uuid(),
  contract_id uuid not null references contracts(id) on delete cascade,
  signer_type text not null check (signer_type in ('client','viagate')),
  name text not null,
  email citext not null,
  cpf varchar(11) not null,
  role text,
  sign_order integer not null default 1,
  status signer_status not null default 'pending',
  signed_at timestamptz,
  signed_document_hash bytea,
  signature_session_id uuid,
  unique(contract_id, signer_type, email)
);

create table signature_challenges (
  id uuid primary key default gen_random_uuid(),
  contract_signer_id uuid not null references contract_signers(id) on delete cascade,
  otp_hash bytea not null,
  expires_at timestamptz not null,
  attempts integer not null default 0,
  verified_at timestamptz,
  created_at timestamptz not null default now()
);

create table identity_verifications (
  id uuid primary key default gen_random_uuid(),
  contract_signer_id uuid not null references contract_signers(id) on delete cascade,
  mode identity_mode not null,
  status identity_status not null default 'pending',
  provider text,
  external_reference text,
  evidence jsonb not null default '{}'::jsonb,
  verified_at timestamptz,
  created_at timestamptz not null default now()
);

create table signature_events (
  id bigint generated always as identity primary key,
  contract_id uuid not null references contracts(id),
  contract_signer_id uuid references contract_signers(id),
  event_type text not null,
  document_hash bytea,
  ip_address inet,
  user_agent text,
  session_id uuid,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table audit_events (
  id bigint generated always as identity primary key,
  actor_user_id uuid references users(id),
  actor_type text not null default 'user',
  event_type text not null,
  resource_type text not null,
  resource_id uuid,
  ip_address inet,
  user_agent text,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create table notification_outbox (
  id uuid primary key default gen_random_uuid(),
  channel text not null default 'email',
  recipient citext not null,
  recipient_name text,
  subject text not null,
  html_body text not null,
  text_body text,
  status text not null default 'pending',
  attempts integer not null default 0,
  available_at timestamptz not null default now(),
  sent_at timestamptz,
  last_error text,
  created_at timestamptz not null default now()
);

create index proposals_created_by_idx on proposals(created_by, updated_at desc);
create index proposal_versions_proposal_idx on proposal_versions(proposal_id, version_number desc);
create index presentations_created_by_idx on presentations(created_by, updated_at desc);
create index onboardings_status_idx on onboardings(status, updated_at desc);
create index contracts_status_idx on contracts(status, updated_at desc);
create index audit_events_resource_idx on audit_events(resource_type, resource_id, created_at desc);
create index notification_outbox_pending_idx on notification_outbox(status, available_at) where status = 'pending';

insert into roles(code,name) values
 ('super_admin','Super Admin'),
 ('commercial','Comercial'),
 ('operations','Operações'),
 ('legal','Jurídico')
on conflict (code) do nothing;

insert into permissions(code,name) values
 ('proposal.create','Criar proposta'),
 ('proposal.read_own','Ver próprias propostas'),
 ('proposal.read_all','Ver todas as propostas'),
 ('presentation.create','Criar apresentação'),
 ('presentation.read_all','Ver todas as apresentações'),
 ('contract.template.manage','Gerenciar modelos de contrato'),
 ('contract.read_all','Ver todos os contratos'),
 ('user.manage','Gerenciar usuários'),
 ('audit.read','Consultar auditoria'),
 ('onboarding.review','Revisar onboarding')
on conflict (code) do nothing;

insert into role_permissions(role_id, permission_id)
select r.id, p.id from roles r cross join permissions p where r.code = 'super_admin'
on conflict do nothing;

insert into role_permissions(role_id, permission_id)
select r.id, p.id from roles r join permissions p on p.code in ('proposal.create','proposal.read_own','presentation.create') where r.code = 'commercial'
on conflict do nothing;

insert into role_permissions(role_id, permission_id)
select r.id, p.id from roles r join permissions p on p.code in ('onboarding.review') where r.code = 'operations'
on conflict do nothing;

insert into role_permissions(role_id, permission_id)
select r.id, p.id from roles r join permissions p on p.code in ('contract.template.manage','contract.read_all') where r.code = 'legal'
on conflict do nothing;
