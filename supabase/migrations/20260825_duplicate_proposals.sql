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
  duplicated_client_id uuid;
  duplicated_contact_id uuid;
  duplicated_proposal_id uuid;
  duplicated_version_id uuid;
  validity_days integer;
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
  returning id into duplicated_client_id;

  if source_proposal.contact_id is not null then
    select *
    into source_contact
    from public.client_contacts
    where id = source_proposal.contact_id
      and client_id = source_client.id;

    if found then
      insert into public.client_contacts (
        client_id,
        name,
        role,
        email,
        phone
      ) values (
        duplicated_client_id,
        source_contact.name,
        source_contact.role,
        source_contact.email,
        source_contact.phone
      )
      returning id into duplicated_contact_id;
    end if;
  end if;

  validity_days := case
    when source_proposal.valid_until is null then null
    else greatest(source_proposal.valid_until - source_proposal.created_at::date, 0)
  end;

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
    duplicated_client_id,
    duplicated_contact_id,
    source_proposal.salesperson_id,
    source_proposal.title || ' - Cópia',
    'draft',
    0,
    case
      when validity_days is null then null
      else current_date + validity_days
    end,
    auth.uid()
  )
  returning id into duplicated_proposal_id;

  select *
  into source_version
  from public.proposal_versions
  where proposal_id = source_proposal.id
  order by version_number desc
  limit 1;

  if found then
    insert into public.proposal_versions (
      proposal_id,
      version_number,
      content,
      pricing_model,
      minimum_invoice,
      setup_fee,
      conditions,
      created_by
    ) values (
      duplicated_proposal_id,
      1,
      source_version.content,
      source_version.pricing_model,
      source_version.minimum_invoice,
      source_version.setup_fee,
      source_version.conditions,
      auth.uid()
    )
    returning id into duplicated_version_id;

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
      duplicated_version_id,
      item.label,
      item.unit,
      item.price,
      item.is_optional,
      item.sort_order,
      item.metadata
    from public.proposal_version_items item
    where item.proposal_version_id = source_version.id
    order by item.sort_order, item.created_at;
  end if;

  return duplicated_proposal_id;
end;
$$;

revoke all on function public.duplicate_proposal(uuid) from public;
grant execute on function public.duplicate_proposal(uuid) to authenticated;
