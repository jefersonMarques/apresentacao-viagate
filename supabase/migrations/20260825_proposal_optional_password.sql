create table if not exists public.proposal_access_settings (
  proposal_id uuid primary key references public.proposals(id) on delete cascade,
  password_hash text not null,
  updated_at timestamptz not null default now()
);

alter table public.proposal_access_settings enable row level security;

revoke all on table public.proposal_access_settings from public, anon, authenticated;

create or replace function public.set_proposal_access_password(
  target_proposal_id uuid,
  access_password text
)
returns boolean
language plpgsql
security definer
set search_path = public
as $$
declare
  normalized_password text;
begin
  if auth.uid() is null then
    raise exception 'Authentication required';
  end if;

  if not exists (
    select 1
    from public.proposals proposal
    where proposal.id = target_proposal_id
      and proposal.created_by = auth.uid()
  ) then
    raise exception 'Proposal not found';
  end if;

  normalized_password := nullif(btrim(coalesce(access_password, '')), '');

  if normalized_password is null then
    delete from public.proposal_access_settings
    where proposal_id = target_proposal_id;

    return false;
  end if;

  if char_length(normalized_password) < 4 then
    raise exception 'Proposal password must have at least 4 characters';
  end if;

  if char_length(normalized_password) > 72 then
    raise exception 'Proposal password is too long';
  end if;

  insert into public.proposal_access_settings (
    proposal_id,
    password_hash,
    updated_at
  ) values (
    target_proposal_id,
    crypt(normalized_password, gen_salt('bf', 10)),
    now()
  )
  on conflict (proposal_id) do update
  set password_hash = excluded.password_hash,
      updated_at = now();

  return true;
end;
$$;

create or replace function public.get_proposal_access_settings(target_proposal_id uuid)
returns jsonb
language sql
stable
security definer
set search_path = public
as $$
  select jsonb_build_object(
    'requires_password', exists (
      select 1
      from public.proposal_access_settings access
      where access.proposal_id = target_proposal_id
    )
  )
  where exists (
    select 1
    from public.proposals proposal
    where proposal.id = target_proposal_id
      and proposal.created_by = auth.uid()
  );
$$;

create or replace function public.get_public_proposal_secure(
  proposal_token uuid,
  proposal_password text default null
)
returns jsonb
language plpgsql
stable
security definer
set search_path = public
as $$
declare
  resolved_proposal_id uuid;
  stored_password_hash text;
  requires_password boolean := false;
  password_valid boolean := false;
  result jsonb;
begin
  select proposal.id
  into resolved_proposal_id
  from public.proposal_versions version
  join public.proposals proposal on proposal.id = version.proposal_id
  where version.public_token = proposal_token
    and version.published_at is not null
  limit 1;

  if resolved_proposal_id is null then
    return null;
  end if;

  select access.password_hash
  into stored_password_hash
  from public.proposal_access_settings access
  where access.proposal_id = resolved_proposal_id;

  requires_password := stored_password_hash is not null;

  if requires_password then
    password_valid := proposal_password is not null
      and crypt(proposal_password, stored_password_hash) = stored_password_hash;

    if not password_valid then
      return jsonb_build_object(
        'authorized', false,
        'requires_password', true
      );
    end if;
  end if;

  select jsonb_build_object(
    'authorized', true,
    'requires_password', requires_password,
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
        when 'item_and_bundle' then 'Análise por item + conjunto'
        else 'Condições definidas nesta proposta'
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
  into result
  from public.proposal_versions version
  join public.proposals proposal on proposal.id = version.proposal_id
  where version.public_token = proposal_token
    and version.published_at is not null
  limit 1;

  return result;
end;
$$;

revoke all on function public.set_proposal_access_password(uuid, text) from public;
grant execute on function public.set_proposal_access_password(uuid, text) to authenticated;

revoke all on function public.get_proposal_access_settings(uuid) from public;
grant execute on function public.get_proposal_access_settings(uuid) to authenticated;

revoke all on function public.get_public_proposal(uuid) from anon, authenticated;
revoke all on function public.get_public_proposal_secure(uuid, text) from public;
grant execute on function public.get_public_proposal_secure(uuid, text) to anon, authenticated;
