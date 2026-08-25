create extension if not exists pgcrypto;

create table if not exists public.salespeople (
  id uuid primary key default gen_random_uuid(),
  auth_user_id uuid not null unique references auth.users(id) on delete cascade,
  name text not null,
  role text,
  email text,
  phone text,
  photo_url text,
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.clients (
  id uuid primary key default gen_random_uuid(),
  company_name text not null,
  trade_name text,
  document text,
  logo_url text,
  created_by uuid not null references auth.users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.client_contacts (
  id uuid primary key default gen_random_uuid(),
  client_id uuid not null references public.clients(id) on delete cascade,
  name text not null,
  role text,
  email text,
  phone text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.proposals (
  id uuid primary key default gen_random_uuid(),
  client_id uuid not null references public.clients(id),
  contact_id uuid references public.client_contacts(id),
  salesperson_id uuid not null references public.salespeople(id),
  title text not null default 'Proposta Comercial ViaGate',
  status text not null default 'draft' check (status in ('draft', 'published', 'accepted', 'declined', 'expired')),
  current_version integer not null default 0 check (current_version >= 0),
  valid_until date,
  created_by uuid not null references auth.users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists public.proposal_versions (
  id uuid primary key default gen_random_uuid(),
  proposal_id uuid not null references public.proposals(id) on delete cascade,
  version_number integer not null check (version_number > 0),
  public_token uuid not null unique default gen_random_uuid(),
  content jsonb not null default '{}'::jsonb,
  pricing_model text not null default 'custom' check (pricing_model in ('per_item', 'bundle', 'custom')),
  minimum_invoice numeric(14,2) not null default 0 check (minimum_invoice >= 0),
  setup_fee numeric(14,2) not null default 0 check (setup_fee >= 0),
  conditions jsonb not null default '{"items":[]}'::jsonb,
  published_at timestamptz,
  created_by uuid not null references auth.users(id),
  created_at timestamptz not null default now(),
  unique (proposal_id, version_number)
);

create table if not exists public.proposal_version_items (
  id uuid primary key default gen_random_uuid(),
  proposal_version_id uuid not null references public.proposal_versions(id) on delete cascade,
  label text not null,
  unit text,
  price numeric(14,4) not null default 0 check (price >= 0),
  is_optional boolean not null default false,
  sort_order integer not null default 0,
  metadata jsonb not null default '{}'::jsonb,
  created_at timestamptz not null default now()
);

create index if not exists proposals_client_id_idx on public.proposals(client_id);
create index if not exists proposals_salesperson_id_idx on public.proposals(salesperson_id);
create index if not exists proposal_versions_proposal_id_idx on public.proposal_versions(proposal_id, version_number desc);
create index if not exists proposal_version_items_version_id_idx on public.proposal_version_items(proposal_version_id, sort_order);

create or replace function public.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

drop trigger if exists salespeople_set_updated_at on public.salespeople;
create trigger salespeople_set_updated_at
before update on public.salespeople
for each row execute function public.set_updated_at();

drop trigger if exists clients_set_updated_at on public.clients;
create trigger clients_set_updated_at
before update on public.clients
for each row execute function public.set_updated_at();

drop trigger if exists client_contacts_set_updated_at on public.client_contacts;
create trigger client_contacts_set_updated_at
before update on public.client_contacts
for each row execute function public.set_updated_at();

drop trigger if exists proposals_set_updated_at on public.proposals;
create trigger proposals_set_updated_at
before update on public.proposals
for each row execute function public.set_updated_at();

create or replace function public.prevent_published_version_mutation()
returns trigger
language plpgsql
as $$
begin
  if old.published_at is not null then
    raise exception 'Published proposal versions are immutable';
  end if;

  return case when tg_op = 'DELETE' then old else new end;
end;
$$;

drop trigger if exists proposal_versions_immutable on public.proposal_versions;
create trigger proposal_versions_immutable
before update or delete on public.proposal_versions
for each row execute function public.prevent_published_version_mutation();

alter table public.salespeople enable row level security;
alter table public.clients enable row level security;
alter table public.client_contacts enable row level security;
alter table public.proposals enable row level security;
alter table public.proposal_versions enable row level security;
alter table public.proposal_version_items enable row level security;

drop policy if exists salespeople_select_authenticated on public.salespeople;
create policy salespeople_select_authenticated
on public.salespeople for select
to authenticated
using (true);

drop policy if exists salespeople_insert_own on public.salespeople;
create policy salespeople_insert_own
on public.salespeople for insert
to authenticated
with check (auth_user_id = auth.uid());

drop policy if exists salespeople_update_own on public.salespeople;
create policy salespeople_update_own
on public.salespeople for update
to authenticated
using (auth_user_id = auth.uid())
with check (auth_user_id = auth.uid());

drop policy if exists salespeople_delete_own on public.salespeople;
create policy salespeople_delete_own
on public.salespeople for delete
to authenticated
using (auth_user_id = auth.uid());

drop policy if exists clients_authenticated_all on public.clients;
create policy clients_authenticated_all
on public.clients for all
to authenticated
using (true)
with check (auth.uid() is not null);

drop policy if exists client_contacts_authenticated_all on public.client_contacts;
create policy client_contacts_authenticated_all
on public.client_contacts for all
to authenticated
using (true)
with check (auth.uid() is not null);

drop policy if exists proposals_authenticated_all on public.proposals;
create policy proposals_authenticated_all
on public.proposals for all
to authenticated
using (true)
with check (auth.uid() is not null);

drop policy if exists proposal_versions_select_authenticated on public.proposal_versions;
create policy proposal_versions_select_authenticated
on public.proposal_versions for select
to authenticated
using (true);

drop policy if exists proposal_versions_insert_authenticated on public.proposal_versions;
create policy proposal_versions_insert_authenticated
on public.proposal_versions for insert
to authenticated
with check (created_by = auth.uid() and published_at is null);

drop policy if exists proposal_versions_update_draft on public.proposal_versions;
create policy proposal_versions_update_draft
on public.proposal_versions for update
to authenticated
using (published_at is null)
with check (published_at is null);

drop policy if exists proposal_versions_delete_draft on public.proposal_versions;
create policy proposal_versions_delete_draft
on public.proposal_versions for delete
to authenticated
using (published_at is null);

drop policy if exists proposal_version_items_select_authenticated on public.proposal_version_items;
create policy proposal_version_items_select_authenticated
on public.proposal_version_items for select
to authenticated
using (true);

drop policy if exists proposal_version_items_insert_draft on public.proposal_version_items;
create policy proposal_version_items_insert_draft
on public.proposal_version_items for insert
to authenticated
with check (
  exists (
    select 1
    from public.proposal_versions version
    where version.id = proposal_version_id
      and version.published_at is null
  )
);

drop policy if exists proposal_version_items_update_draft on public.proposal_version_items;
create policy proposal_version_items_update_draft
on public.proposal_version_items for update
to authenticated
using (
  exists (
    select 1
    from public.proposal_versions version
    where version.id = proposal_version_id
      and version.published_at is null
  )
)
with check (
  exists (
    select 1
    from public.proposal_versions version
    where version.id = proposal_version_id
      and version.published_at is null
  )
);

drop policy if exists proposal_version_items_delete_draft on public.proposal_version_items;
create policy proposal_version_items_delete_draft
on public.proposal_version_items for delete
to authenticated
using (
  exists (
    select 1
    from public.proposal_versions version
    where version.id = proposal_version_id
      and version.published_at is null
  )
);

create or replace function public.publish_proposal_version(target_version_id uuid)
returns table (
  public_token uuid,
  published_at timestamptz
)
language plpgsql
security definer
set search_path = public
as $$
declare
  target_version public.proposal_versions%rowtype;
begin
  if auth.uid() is null then
    raise exception 'Authentication required';
  end if;

  select *
  into target_version
  from public.proposal_versions
  where id = target_version_id;

  if not found then
    raise exception 'Proposal version not found';
  end if;

  if target_version.published_at is null then
    update public.proposal_versions
    set published_at = now()
    where id = target_version_id
    returning * into target_version;
  end if;

  update public.proposals
  set status = 'published',
      current_version = target_version.version_number,
      updated_at = now()
  where id = target_version.proposal_id;

  return query
  select target_version.public_token, target_version.published_at;
end;
$$;

create or replace function public.get_public_proposal(proposal_token uuid)
returns jsonb
language sql
stable
security definer
set search_path = public
as $$
  select jsonb_build_object(
    'proposal', jsonb_build_object(
      'id', proposal.id,
      'title', proposal.title,
      'status', proposal.status,
      'current_version', proposal.current_version,
      'valid_until', proposal.valid_until
    ),
    'version', jsonb_build_object(
      'version_number', version.version_number,
      'content', version.content,
      'pricing_model', version.pricing_model,
      'pricing_model_label', case version.pricing_model
        when 'per_item' then 'Análise por item'
        when 'bundle' then 'Análise por conjunto'
        else 'Personalizado'
      end,
      'minimum_invoice', version.minimum_invoice,
      'setup_fee', version.setup_fee,
      'conditions', version.conditions,
      'published_at', version.published_at,
      'items', coalesce(
        (
          select jsonb_agg(
            jsonb_build_object(
              'label', item.label,
              'unit', item.unit,
              'price', item.price,
              'is_optional', item.is_optional,
              'sort_order', item.sort_order,
              'metadata', item.metadata
            ) order by item.sort_order, item.created_at
          )
          from public.proposal_version_items item
          where item.proposal_version_id = version.id
        ),
        '[]'::jsonb
      )
    ),
    'is_expired', proposal.valid_until is not null and proposal.valid_until < current_date
  )
  from public.proposal_versions version
  join public.proposals proposal on proposal.id = version.proposal_id
  where version.public_token = proposal_token
    and version.published_at is not null
  limit 1;
$$;

revoke all on function public.publish_proposal_version(uuid) from public;
grant execute on function public.publish_proposal_version(uuid) to authenticated;

revoke all on function public.get_public_proposal(uuid) from public;
grant execute on function public.get_public_proposal(uuid) to anon, authenticated;
