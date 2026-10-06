-- Use the white ViaGate brand mark directly on the dark proposal email header.
-- Only the untouched seeded template is changed; customized templates remain intact.

update email_templates
set
  html_template=replace(
    html_template,
    '<img src="{brand.logo_url}" alt="ViaGate" style="display:block;max-width:150px;max-height:42px;border:0;background:#ffffff;padding:6px 8px">',
    '<img src="{brand.logo_url}" alt="ViaGate" style="display:block;max-width:170px;max-height:46px;border:0">'
  ),
  updated_at=now()
where purpose='proposal_share'
  and name='Proposta comercial ViaGate'
  and created_by is null
  and updated_by is null
  and html_template like '%background:#ffffff;padding:6px 8px%';
