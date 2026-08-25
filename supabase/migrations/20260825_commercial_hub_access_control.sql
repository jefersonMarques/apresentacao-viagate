drop policy if exists clients_authenticated_all on public.clients;
create policy clients_select_own
on public.clients for select
to authenticated
using (created_by = auth.uid());

create policy clients_insert_own
on public.clients for insert
to authenticated
with check (created_by = auth.uid());

create policy clients_update_own
on public.clients for update
to authenticated
using (created_by = auth.uid())
with check (created_by = auth.uid());

create policy clients_delete_own
on public.clients for delete
to authenticated
using (created_by = auth.uid());

drop policy if exists client_contacts_authenticated_all on public.client_contacts;
create policy client_contacts_select_own
on public.client_contacts for select
to authenticated
using (
  exists (
    select 1
    from public.clients client
    where client.id = client_id
      and client.created_by = auth.uid()
  )
);

create policy client_contacts_insert_own
on public.client_contacts for insert
to authenticated
with check (
  exists (
    select 1
    from public.clients client
    where client.id = client_id
      and client.created_by = auth.uid()
  )
);

create policy client_contacts_update_own
on public.client_contacts for update
to authenticated
using (
  exists (
    select 1
    from public.clients client
    where client.id = client_id
      and client.created_by = auth.uid()
  )
)
with check (
  exists (
    select 1
    from public.clients client
    where client.id = client_id
      and client.created_by = auth.uid()
  )
);

create policy client_contacts_delete_own
on public.client_contacts for delete
to authenticated
using (
  exists (
    select 1
    from public.clients client
    where client.id = client_id
      and client.created_by = auth.uid()
  )
);

drop policy if exists proposals_authenticated_all on public.proposals;
create policy proposals_select_own
on public.proposals for select
to authenticated
using (created_by = auth.uid());

create policy proposals_insert_own
on public.proposals for insert
to authenticated
with check (created_by = auth.uid());

create policy proposals_update_own
on public.proposals for update
to authenticated
using (created_by = auth.uid())
with check (created_by = auth.uid());

create policy proposals_delete_own
on public.proposals for delete
to authenticated
using (created_by = auth.uid());

drop policy if exists proposal_versions_select_authenticated on public.proposal_versions;
drop policy if exists proposal_versions_insert_authenticated on public.proposal_versions;
drop policy if exists proposal_versions_update_draft on public.proposal_versions;
drop policy if exists proposal_versions_delete_draft on public.proposal_versions;

create policy proposal_versions_select_own
on public.proposal_versions for select
to authenticated
using (
  exists (
    select 1
    from public.proposals proposal
    where proposal.id = proposal_id
      and proposal.created_by = auth.uid()
  )
);

create policy proposal_versions_insert_own
on public.proposal_versions for insert
to authenticated
with check (
  created_by = auth.uid()
  and published_at is null
  and exists (
    select 1
    from public.proposals proposal
    where proposal.id = proposal_id
      and proposal.created_by = auth.uid()
  )
);

create policy proposal_versions_update_own_draft
on public.proposal_versions for update
to authenticated
using (
  published_at is null
  and exists (
    select 1
    from public.proposals proposal
    where proposal.id = proposal_id
      and proposal.created_by = auth.uid()
  )
)
with check (published_at is null and created_by = auth.uid());

create policy proposal_versions_delete_own_draft
on public.proposal_versions for delete
to authenticated
using (
  published_at is null
  and exists (
    select 1
    from public.proposals proposal
    where proposal.id = proposal_id
      and proposal.created_by = auth.uid()
  )
);

drop policy if exists proposal_version_items_select_authenticated on public.proposal_version_items;
drop policy if exists proposal_version_items_insert_draft on public.proposal_version_items;
drop policy if exists proposal_version_items_update_draft on public.proposal_version_items;
drop policy if exists proposal_version_items_delete_draft on public.proposal_version_items;

create policy proposal_version_items_select_own
on public.proposal_version_items for select
to authenticated
using (
  exists (
    select 1
    from public.proposal_versions version
    join public.proposals proposal on proposal.id = version.proposal_id
    where version.id = proposal_version_id
      and proposal.created_by = auth.uid()
  )
);

create policy proposal_version_items_insert_own_draft
on public.proposal_version_items for insert
to authenticated
with check (
  exists (
    select 1
    from public.proposal_versions version
    join public.proposals proposal on proposal.id = version.proposal_id
    where version.id = proposal_version_id
      and version.published_at is null
      and proposal.created_by = auth.uid()
  )
);

create policy proposal_version_items_update_own_draft
on public.proposal_version_items for update
to authenticated
using (
  exists (
    select 1
    from public.proposal_versions version
    join public.proposals proposal on proposal.id = version.proposal_id
    where version.id = proposal_version_id
      and version.published_at is null
      and proposal.created_by = auth.uid()
  )
)
with check (
  exists (
    select 1
    from public.proposal_versions version
    join public.proposals proposal on proposal.id = version.proposal_id
    where version.id = proposal_version_id
      and version.published_at is null
      and proposal.created_by = auth.uid()
  )
);

create policy proposal_version_items_delete_own_draft
on public.proposal_version_items for delete
to authenticated
using (
  exists (
    select 1
    from public.proposal_versions version
    join public.proposals proposal on proposal.id = version.proposal_id
    where version.id = proposal_version_id
      and version.published_at is null
      and proposal.created_by = auth.uid()
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

  select version.*
  into target_version
  from public.proposal_versions version
  join public.proposals proposal on proposal.id = version.proposal_id
  where version.id = target_version_id
    and proposal.created_by = auth.uid();

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
  where id = target_version.proposal_id
    and created_by = auth.uid();

  return query
  select target_version.public_token, target_version.published_at;
end;
$$;

revoke all on function public.publish_proposal_version(uuid) from public;
grant execute on function public.publish_proposal_version(uuid) to authenticated;