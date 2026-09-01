insert into roles(code,name) values
 ('admin','Admin'),
 ('user','Usuário')
on conflict (code) do update set name=excluded.name;

insert into permissions(code,name) values
 ('proposal.price.edit','Alterar valores de propostas'),
 ('proposal.conditions.edit','Alterar condições comerciais de propostas'),
 ('proposal.publish','Publicar propostas'),
 ('proposal.duplicate','Duplicar propostas'),
 ('proposal.cancel','Cancelar propostas'),
 ('presentation.read_own','Ver próprias apresentações'),
 ('presentation.publish','Publicar apresentações'),
 ('presentation.activity.read','Ver atividade de apresentações'),
 ('customer.read_own','Ver próprios clientes'),
 ('customer.read_all','Ver clientes de outros usuários'),
 ('customer.responses.read_own','Ver respostas dos próprios clientes'),
 ('customer.responses.read_all','Ver respostas de clientes de outros usuários'),
 ('customer.sensitive_data.read','Ver dados pessoais sensíveis de clientes'),
 ('customer.documents.read','Ver documentos enviados pelos clientes'),
 ('contract.read_own','Ver próprios contratos'),
 ('contract.evidence.read','Ver evidências de contratos'),
 ('activity.read_own','Ver próprias ações'),
 ('activity.read_all','Ver ações de outros usuários'),
 ('notification.read','Ver notificações internas'),
 ('notification.receive_others','Receber notificações de outros usuários'),
 ('user.permissions.manage','Gerenciar permissões individuais'),
 ('user.status.manage','Ativar e desativar usuários'),
 ('settings.manage','Gerenciar configurações administrativas'),
 ('audit.technical.read','Ver auditoria e logs técnicos'),
 ('system.technical','Acessar opções técnicas e de desenvolvimento')
on conflict (code) do update set name=excluded.name;

create table if not exists user_permission_overrides (
  user_id uuid not null references users(id) on delete cascade,
  permission_id uuid not null references permissions(id) on delete cascade,
  allowed boolean not null,
  updated_by uuid references users(id),
  updated_at timestamptz not null default now(),
  primary key(user_id,permission_id)
);

create table if not exists user_notification_preferences (
  user_id uuid not null references users(id) on delete cascade,
  event_type text not null,
  scope text not null check (scope in ('own','all')),
  enabled boolean not null default true,
  updated_at timestamptz not null default now(),
  primary key(user_id,event_type,scope)
);

create table if not exists in_app_notifications (
  id uuid primary key default gen_random_uuid(),
  recipient_user_id uuid not null references users(id) on delete cascade,
  owner_user_id uuid references users(id) on delete set null,
  actor_user_id uuid references users(id) on delete set null,
  event_type text not null,
  title text not null,
  body text,
  resource_type text,
  resource_id uuid,
  target_url text,
  dedupe_key text,
  read_at timestamptz,
  created_at timestamptz not null default now()
);

create unique index if not exists in_app_notifications_dedupe_idx
  on in_app_notifications(recipient_user_id,dedupe_key)
  where dedupe_key is not null;

create index if not exists in_app_notifications_unread_idx
  on in_app_notifications(recipient_user_id,created_at desc)
  where read_at is null;

create or replace function effective_user_permissions(target_user uuid)
returns table(permission_code text)
language sql
stable
as $$
  select p.code
  from permissions p
  where coalesce(
    (select upo.allowed
       from user_permission_overrides upo
      where upo.user_id=target_user and upo.permission_id=p.id),
    exists(
      select 1
      from user_roles ur
      join role_permissions rp on rp.role_id=ur.role_id
      where ur.user_id=target_user and rp.permission_id=p.id
    )
  );
$$;

-- Preserve the capabilities of legacy operational/legal profiles before
-- consolidating them into the new User profile.
insert into user_permission_overrides(user_id,permission_id,allowed)
select distinct ur.user_id,p.id,true
from user_roles ur
join roles r on r.id=ur.role_id and r.code='operations'
join permissions p on p.code in ('onboarding.review','activation.read_all','activation.manage','customer.responses.read_all','customer.documents.read')
on conflict (user_id,permission_id) do nothing;

insert into user_permission_overrides(user_id,permission_id,allowed)
select distinct ur.user_id,p.id,true
from user_roles ur
join roles r on r.id=ur.role_id and r.code='legal'
join permissions p on p.code in ('contract.template.manage','contract.read_all','contract.evidence.read','audit.read','customer.responses.read_all','customer.documents.read')
on conflict (user_id,permission_id) do nothing;

-- Every non-superadmin legacy profile becomes User. The old role rows remain
-- only as historical compatibility and are no longer assigned to accounts.
insert into user_roles(user_id,role_id)
select distinct ur.user_id,new_role.id
from user_roles ur
join roles old_role on old_role.id=ur.role_id and old_role.code in ('commercial','operations','legal')
cross join roles new_role
where new_role.code='user'
  and not exists(
    select 1 from user_roles current_ur
    join roles assigned_role on assigned_role.id=current_ur.role_id
    where current_ur.user_id=ur.user_id and assigned_role.code='super_admin'
  )
on conflict do nothing;

delete from user_roles ur
using roles r
where ur.role_id=r.id and r.code in ('commercial','operations','legal');

-- User defaults: commercial work and visibility limited to the user's own
-- portfolio. Price changes intentionally are not included.
insert into role_permissions(role_id,permission_id)
select r.id,p.id
from roles r
join permissions p on p.code in (
 'proposal.create','proposal.read_own','proposal.conditions.edit','proposal.publish','proposal.duplicate',
 'presentation.create','presentation.read_own','presentation.publish','presentation.activity.read',
 'customer.read_own','customer.responses.read_own','customer.documents.read',
 'contract.read_own','contract.evidence.read','activity.read_own','notification.read'
)
where r.code='user'
on conflict do nothing;

-- Admin controls the commercial/operational platform, but does not receive
-- development-only capabilities.
insert into role_permissions(role_id,permission_id)
select r.id,p.id
from roles r
join permissions p on p.code in (
 'proposal.create','proposal.read_own','proposal.read_all','proposal.price.edit','proposal.conditions.edit','proposal.publish','proposal.duplicate','proposal.cancel',
 'presentation.create','presentation.read_own','presentation.read_all','presentation.publish','presentation.activity.read',
 'customer.read_own','customer.read_all','customer.responses.read_own','customer.responses.read_all','customer.sensitive_data.read','customer.documents.read',
 'contract.read_own','contract.read_all','contract.evidence.read','contract.template.manage',
 'onboarding.review','activation.read_all','activation.manage',
 'user.manage','user.permissions.manage','user.status.manage',
 'activity.read_own','activity.read_all','audit.read',
 'notification.read','notification.receive_others','settings.manage'
)
where r.code='admin'
on conflict do nothing;

-- Superadmin always receives every permission known at this schema version.
insert into role_permissions(role_id,permission_id)
select r.id,p.id from roles r cross join permissions p where r.code='super_admin'
on conflict do nothing;
