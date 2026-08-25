alter table public.proposal_versions
  drop constraint if exists proposal_versions_pricing_model_check;

alter table public.proposal_versions
  add constraint proposal_versions_pricing_model_check
  check (pricing_model in ('per_item', 'bundle', 'item_and_bundle', 'custom'));

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
  from public.proposal_versions version
  join public.proposals proposal on proposal.id = version.proposal_id
  where version.public_token = proposal_token
    and version.published_at is not null
  limit 1;
$$;

revoke all on function public.get_public_proposal(uuid) from public;
grant execute on function public.get_public_proposal(uuid) to anon, authenticated;
