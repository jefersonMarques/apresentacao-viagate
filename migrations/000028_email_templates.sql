create table if not exists email_templates (
  id uuid primary key default gen_random_uuid(),
  purpose text not null,
  name text not null,
  description text,
  subject_template text not null,
  html_template text not null,
  text_template text not null default '',
  is_active boolean not null default true,
  is_default boolean not null default false,
  created_by uuid references users(id),
  updated_by uuid references users(id),
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create unique index if not exists email_templates_default_purpose_idx
  on email_templates(purpose)
  where is_active and is_default;

create index if not exists email_templates_purpose_idx
  on email_templates(purpose, is_active, name);

insert into email_templates(
  purpose,
  name,
  description,
  subject_template,
  html_template,
  text_template,
  is_active,
  is_default
)
select
  'proposal_share',
  'Proposta comercial ViaGate',
  'Modelo padrão para compartilhar propostas comerciais pelo cliente de e-mail.',
  'Proposta ViaGate — {client.display_name}',
  $email$
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;max-width:640px;margin:0 auto;background:#ffffff;border:1px solid #d9e2e8;font-family:Arial,Helvetica,sans-serif;color:#102637">
  <tr>
    <td style="background:#071827;padding:24px 28px;border-bottom:4px solid #ff6b18">
      <div style="color:#ff6b18;font-size:11px;font-weight:700;letter-spacing:2px">VIAGATE</div>
      <div style="margin-top:7px;color:#ffffff;font-size:22px;font-weight:700">Proposta comercial</div>
    </td>
  </tr>
  <tr>
    <td style="padding:30px 28px">
      <p style="margin:0 0 16px;font-size:15px;line-height:1.6">Olá, {contact.name}.</p>
      <p style="margin:0 0 12px;font-size:14px;line-height:1.65">Preparei a proposta comercial da ViaGate para <strong>{client.display_name}</strong>.</p>
      <p style="margin:0 0 24px;color:#536875;font-size:13px;line-height:1.65">No link abaixo estão a solução, os produtos, valores e condições comerciais desta negociação.</p>
      <table role="presentation" cellspacing="0" cellpadding="0">
        <tr>
          <td style="background:#ff6b18">
            <a href="{proposal.url}" style="display:inline-block;padding:13px 20px;color:#ffffff;text-decoration:none;font-size:13px;font-weight:700">Abrir proposta</a>
          </td>
        </tr>
      </table>
      <p style="margin:22px 0 0;color:#82919a;font-size:11px;line-height:1.6">Link direto:<br><a href="{proposal.url}" style="color:#536875;word-break:break-all">{proposal.url}</a></p>
    </td>
  </tr>
  <tr>
    <td style="padding:22px 28px;border-top:1px solid #e2e8ec;background:#f8fafb">
      <div style="color:#102637;font-size:14px;font-weight:700">{salesperson.name}</div>
      <div style="margin-top:3px;color:#6f8290;font-size:12px">{salesperson.job_title}</div>
      <div style="margin-top:7px;color:#536875;font-size:12px">{salesperson.phone} · {salesperson.email}</div>
    </td>
  </tr>
</table>
$email$,
  $text$
Olá, {contact.name}.

Preparei a proposta comercial da ViaGate para {client.display_name}.

Acesse a proposta:
{proposal.url}

{salesperson.name}
{salesperson.job_title}
{salesperson.phone}
{salesperson.email}
$text$,
  true,
  true
where not exists (
  select 1
  from email_templates
  where purpose='proposal_share'
);
