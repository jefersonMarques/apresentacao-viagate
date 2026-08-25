alter table public.proposal_versions
  drop constraint if exists proposal_versions_pricing_model_check;

alter table public.proposal_versions
  add constraint proposal_versions_pricing_model_check
  check (pricing_model in ('per_item', 'bundle', 'item_and_bundle', 'custom'));
