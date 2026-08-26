alter table public.salespeople
  add column if not exists linkedin_url text,
  add column if not exists instagram_url text;

create or replace function public.enrich_proposal_salesperson_snapshot()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  salesperson_record public.salespeople%rowtype;
  salesperson_content jsonb;
begin
  select salesperson.*
  into salesperson_record
  from public.proposals proposal
  join public.salespeople salesperson on salesperson.id = proposal.salesperson_id
  where proposal.id = new.proposal_id;

  if found then
    salesperson_content := coalesce(new.content -> 'salesperson', '{}'::jsonb)
      || jsonb_build_object(
        'linkedin', coalesce(salesperson_record.linkedin_url, ''),
        'instagram', coalesce(salesperson_record.instagram_url, '')
      );

    new.content := jsonb_set(
      coalesce(new.content, '{}'::jsonb),
      '{salesperson}',
      salesperson_content,
      true
    );
  end if;

  return new;
end;
$$;

drop trigger if exists proposal_versions_enrich_salesperson_snapshot on public.proposal_versions;
create trigger proposal_versions_enrich_salesperson_snapshot
before insert or update of content on public.proposal_versions
for each row execute function public.enrich_proposal_salesperson_snapshot();

create or replace function public.enrich_presentation_salesperson_snapshot()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
declare
  salesperson_record public.salespeople%rowtype;
  salesperson_content jsonb;
begin
  select salesperson.*
  into salesperson_record
  from public.presentations presentation
  join public.salespeople salesperson on salesperson.id = presentation.salesperson_id
  where presentation.id = new.presentation_id;

  if found then
    salesperson_content := coalesce(new.content -> 'salesperson', '{}'::jsonb)
      || jsonb_build_object(
        'linkedin', coalesce(salesperson_record.linkedin_url, ''),
        'instagram', coalesce(salesperson_record.instagram_url, '')
      );

    new.content := jsonb_set(
      coalesce(new.content, '{}'::jsonb),
      '{salesperson}',
      salesperson_content,
      true
    );
  end if;

  return new;
end;
$$;

drop trigger if exists presentation_versions_enrich_salesperson_snapshot on public.presentation_versions;
create trigger presentation_versions_enrich_salesperson_snapshot
before insert or update of content on public.presentation_versions
for each row execute function public.enrich_presentation_salesperson_snapshot();

create or replace function public.duplicate_proposal(source_proposal_id uuid)
returns uuid
language plpgsql
security definer
set search_path = public
as $$
declare
  source_proposal public.proposals%rowtype;
  source_client public.clients%rowtype;
  source_contact public.client_contacts%rowtype;
  source_version public.proposal_versions%rowtype;
  new_client_id uuid;
  new_contact_id uuid;
  new_proposal_id uuid;
  new_version_id uuid;
begin
  if auth.uid() is null then
    raise exception 'Authentication required';
  end if;

  select *
  into source_proposal
  from public.proposals
  where id = source_proposal_id
    and created_by = auth.uid();

  if not found then
    raise exception 'Proposal not found';
  end if;

  select *
  into source_client
  from public.clients
  where id = source_proposal.client_id
    and created_by = auth.uid();

  if not found then
    raise exception 'Proposal client not found';
  end if;

  select *
  into source_version
  from public.proposal_versions
  where proposal_id = source_proposal.id
  order by version_number desc
  limit 1;

  if not found then
    raise exception 'Proposal version not found';
  end if;

  insert into public.clients (
    company_name,
    trade_name,
    document,
    logo_url,
    created_by
  ) values (
    source_client.company_name,
    source_client.trade_name,
    source_client.document,
    source_client.logo_url,
    auth.uid()
  )
  returning id into new_client_id;

  if source_proposal.contact_id is not null then
    select *
    into source_contact
    from public.client_contacts
    where id = source_proposal.contact_id;

    if found then
      insert into public.client_contacts (
        client_id,
        name,
        role,
        email,
        phone
      ) values (
        new_client_id,
        source_contact.name,
        source_contact.role,
        source_contact.email,
        source_contact.phone
      )
      returning id into new_contact_id;
    end if;
  end if;

  insert into public.proposals (
    client_id,
    contact_id,
    salesperson_id,
    title,
    status,
    current_version,
    valid_until,
    created_by
  ) values (
    new_client_id,
    new_contact_id,
    source_proposal.salesperson_id,
    source_proposal.title || ' - Cópia',
    'draft',
    0,
    current_date + 15,
    auth.uid()
  )
  returning id into new_proposal_id;

  insert into public.proposal_versions (
    proposal_id,
    version_number,
    content,
    pricing_model,
    minimum_invoice,
    setup_fee,
    conditions,
    published_at,
    created_by
  ) values (
    new_proposal_id,
    1,
    source_version.content,
    source_version.pricing_model,
    source_version.minimum_invoice,
    source_version.setup_fee,
    source_version.conditions,
    null,
    auth.uid()
  )
  returning id into new_version_id;

  insert into public.proposal_version_items (
    proposal_version_id,
    label,
    unit,
    price,
    is_optional,
    sort_order,
    metadata
  )
  select
    new_version_id,
    item.label,
    item.unit,
    item.price,
    item.is_optional,
    item.sort_order,
    item.metadata
  from public.proposal_version_items item
  where item.proposal_version_id = source_version.id
  order by item.sort_order, item.created_at;

  return new_proposal_id;
end;
$$;

revoke all on function public.duplicate_proposal(uuid) from public;
grant execute on function public.duplicate_proposal(uuid) to authenticated;
