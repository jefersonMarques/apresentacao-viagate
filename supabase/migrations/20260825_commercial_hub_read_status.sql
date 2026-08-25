create or replace function public.get_my_shared_document_stats()
returns jsonb
language sql
stable
security definer
set search_path = public
as $$
  with documents as (
    select
      'presentation'::text as document_kind,
      presentation.id as document_id,
      presentation.title,
      nullif(version.content #>> '{client,company_name}', '') as client_name,
      version.id as version_id,
      version.version_number,
      version.public_token,
      version.published_at
    from public.presentations presentation
    join public.presentation_versions version
      on version.presentation_id = presentation.id
     and version.version_number = presentation.current_version
     and version.published_at is not null
    where presentation.created_by = auth.uid()

    union all

    select
      'proposal'::text as document_kind,
      proposal.id as document_id,
      proposal.title,
      client.company_name as client_name,
      version.id as version_id,
      version.version_number,
      version.public_token,
      version.published_at
    from public.proposals proposal
    join public.clients client on client.id = proposal.client_id
    join public.proposal_versions version
      on version.proposal_id = proposal.id
     and version.version_number = proposal.current_version
     and version.published_at is not null
    where proposal.created_by = auth.uid()
  ),
  stats as (
    select
      document.document_kind,
      document.document_id,
      document.title,
      document.client_name,
      document.version_number,
      document.public_token,
      document.published_at,
      count(*) filter (where event.event_type = 'open')::int as opens,
      count(*) filter (where event.event_type = 'start')::int as starts,
      count(*) filter (where event.event_type = 'complete')::int as completions,
      count(distinct event.session_id) filter (where event.event_type = 'open')::int as viewer_sessions,
      min(event.created_at) filter (where event.event_type = 'open') as first_opened_at,
      max(event.created_at) filter (where event.event_type = 'open') as last_opened_at,
      max(event.created_at) filter (where event.event_type = 'start') as last_started_at,
      max(event.created_at) filter (where event.event_type = 'complete') as last_completed_at,
      coalesce(
        max(
          case
            when event.event_type = 'slide_view' and event.slide_total > 0
            then round((event.slide_number::numeric / event.slide_total::numeric) * 100)
            else 0
          end
        ),
        0
      )::int as max_progress
    from documents document
    left join public.shared_document_events event
      on (
        document.document_kind = 'presentation'
        and event.presentation_version_id = document.version_id
      )
      or (
        document.document_kind = 'proposal'
        and event.proposal_version_id = document.version_id
      )
    group by
      document.document_kind,
      document.document_id,
      document.title,
      document.client_name,
      document.version_number,
      document.public_token,
      document.published_at
  )
  select coalesce(jsonb_agg(to_jsonb(stats) order by stats.published_at desc), '[]'::jsonb)
  from stats;
$$;

revoke all on function public.get_my_shared_document_stats() from public;
grant execute on function public.get_my_shared_document_stats() to authenticated;
