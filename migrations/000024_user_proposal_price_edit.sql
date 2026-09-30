insert into role_permissions(role_id,permission_id)
select r.id,p.id
from roles r
join permissions p on p.code='proposal.price.edit'
where r.code='user'
on conflict do nothing;
