grant select, insert, update, delete on table public.salespeople to authenticated;
grant select, insert, update, delete on table public.clients to authenticated;
grant select, insert, update, delete on table public.client_contacts to authenticated;
grant select, insert, update, delete on table public.proposals to authenticated;
grant select, insert, update, delete on table public.proposal_versions to authenticated;
grant select, insert, update, delete on table public.proposal_version_items to authenticated;

revoke all on table public.salespeople from anon;
revoke all on table public.clients from anon;
revoke all on table public.client_contacts from anon;
revoke all on table public.proposals from anon;
revoke all on table public.proposal_versions from anon;
revoke all on table public.proposal_version_items from anon;
