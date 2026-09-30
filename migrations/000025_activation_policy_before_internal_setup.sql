-- Activation data may now be completed before signature, but an insurance
-- policy is required before the preparation can be considered complete.
-- Preserve already-started/activated operations and only reopen historical
-- "completed" profiles that do not have a current policy document.

update activation_profiles a
set status='in_progress',
    submitted_at=null,
    updated_at=now()
from contracts c
where c.id=a.contract_id
  and a.status='completed'
  and not exists (
    select 1
    from uploaded_documents d
    where d.onboarding_id=c.onboarding_id
      and d.document_type='insurance_policy'
      and d.status='uploaded'
  );
