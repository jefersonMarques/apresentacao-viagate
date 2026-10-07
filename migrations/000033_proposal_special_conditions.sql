create table if not exists proposal_special_conditions (
    id uuid primary key default gen_random_uuid(),
    condition_text text not null,
    group_codes text[] not null default '{}'::text[],
    is_active boolean not null default true,
    sort_order integer not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    constraint proposal_special_conditions_text_nonempty check (btrim(condition_text) <> '')
);

create unique index if not exists proposal_special_conditions_text_unique
    on proposal_special_conditions (lower(btrim(condition_text)));

create index if not exists proposal_special_conditions_active_sort_idx
    on proposal_special_conditions (is_active, sort_order, created_at);

insert into proposal_special_conditions(condition_text,group_codes,is_active,sort_order)
values
('O retorno da pesquisa cadastral ocorre em até 10 minutos após a conclusão da biometria, salvo indisponibilidade de fontes externas.',array['score'],true,10),
('A autorização biométrica está incluída no fluxo da análise cadastral.',array['score'],true,20),
('A operação pode utilizar link web e aplicativo conforme a configuração comercial contratada.',array['score','logistics'],true,30),
('Após a formalização comercial poderá ser criado grupo de atendimento para implantação e acompanhamento da operação.','{}'::text[],true,40),
('Vitimologia por estado possui precificação conforme abrangência solicitada e prazo de retorno estimado de até 3 horas úteis.',array['authentication'],true,50),
('Os recursos logísticos complementam a gestão de risco, mas não substituem integralmente um processo completo de gerenciamento de risco.',array['logistics','monitoring'],true,60),
('Integrações dependem de documentação técnica, disponibilidade e homologação dos sistemas envolvidos.',array['monitoring','logistics','authentication'],true,70),
('Customizações fora do escopo contratado serão previamente analisadas e, quando aplicável, orçadas por hora técnica.','{}'::text[],true,80),
('Despesas extraordinárias de deslocamento, alimentação e hospedagem não estão incluídas, quando aplicáveis.','{}'::text[],true,90),
('Biometria Facial com Prova de Vida e Geolocalização Inclusa no Cadastro e na Consulta',array['score'],true,100),
('Vitimologia Integrada Inclusa no Cadastro',array['score','authentication'],true,110)
on conflict do nothing;
