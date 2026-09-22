create table if not exists product_categories (
  id uuid primary key default gen_random_uuid(),
  code text not null unique,
  name text not null,
  description text,
  is_active boolean not null default true,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create table if not exists products (
  id uuid primary key default gen_random_uuid(),
  category_id uuid not null references product_categories(id),
  code text not null unique,
  name text not null,
  description text,
  unit text,
  is_active boolean not null default true,
  sort_order integer not null default 0,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists products_category_order_idx
  on products(category_id, sort_order, name);

insert into product_categories(code,name,description,is_active,sort_order)
values
  ('score','Cargo Score | Análise cadastral','Pesquisa cadastral com autorização biométrica, validações oficiais e análise de risco para motorista e veículo.',true,10),
  ('authentication','Consultas e autenticação','Consultas pontuais, autenticação de formulários e validações complementares ao processo cadastral.',true,20),
  ('logistics','Cargo Truck | Aplicativo e logística','Aplicativo para cadastro, coletas, entregas, eventos de parada e rastreamento por GPS do smartphone do motorista.',true,30),
  ('prevention','Prevenção','Recursos complementares para gestão preventiva, incluindo multas e histórico veicular completo.',true,40),
  ('monitoring','Monitoramento de veículos | Integração com gerenciadora','Monitoramento satelital via integração com gerenciadora, com opções por veículo, viagem e checklist.',true,50)
on conflict (code) do nothing;

insert into products(category_id,code,name,unit,is_active,sort_order)
select c.id,v.code,v.name,v.unit,true,v.sort_order
from product_categories c
join (
  values
    ('score','score-item-driver-register','Cadastro | Motorista — Frota, agregado e terceiro','cadastro',10),
    ('score','score-item-driver-other','Cadastro | Motorista — Outras funções','cadastro',20),
    ('score','score-item-vehicle-register','Cadastro | Veículos','cadastro',30),
    ('score','score-item-driver-query','Consulta | Motorista — Frota, agregado e terceiro','consulta',40),
    ('score','score-item-vehicle-query','Consulta | Veículos','consulta',50),
    ('score','score-item-reanalysis','Reanálise de campos preenchidos incorretamente','reanálise',60),
    ('score','score-bundle-register','Cadastro | Motorista + veículos + colaboradores','conjunto',70),
    ('score','score-bundle-query','Consulta | Motorista + veículos + colaboradores','conjunto',80),
    ('score','score-bundle-reanalysis','Reanálise | Conjunto','reanálise',90),
    ('authentication','auth-cargo','Cargo Autenticador','consulta',10),
    ('authentication','auth-lawsuits','Pesquisa processo criminal, trabalhista, cível e familiar','consulta',20),
    ('authentication','auth-victimology-state','Vitimologia por estado','estado',30),
    ('authentication','auth-victimology-integrated','Vitimologia integrada','consulta',40),
    ('authentication','auth-antt','Consulta veículos | ANTT','consulta',50),
    ('authentication','auth-on-demand','Avulso | Consultas e autenticação de formulários','consulta',60),
    ('logistics','truck-first-without-score','Cargo Truck | Primeira viagem (sem Score)','viagem',10),
    ('logistics','truck-next-without-score','Cargo Truck | Viagens subsequentes (sem Score)','viagem',20),
    ('logistics','truck-with-score','Cargo Truck | Primeira viagem e subsequentes (com Score)','viagem',30),
    ('prevention','prevention-fines','Sistema Gestor de Multas','veículo',10),
    ('prevention','prevention-history','Histórico Veicular Completo','consulta',20),
    ('monitoring','monitoring-fixed','Veículo Fixo','veículo',10),
    ('monitoring','monitoring-trip','Viagem Avulsa','viagem',20),
    ('monitoring','monitoring-autotrac','ADE Autotrac','veículo',30),
    ('monitoring','monitoring-checklist','Check List','veículo',40)
) as v(category_code,code,name,unit,sort_order) on v.category_code=c.code
on conflict (code) do nothing;
