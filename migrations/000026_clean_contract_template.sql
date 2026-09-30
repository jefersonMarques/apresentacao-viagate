-- Create a clean version of the active/default contract template without
-- changing any already generated contract or historical template version.
--
-- The template preserves the legal clauses currently in use, but:
--   * uses a neutral title;
--   * never interpolates product booleans;
--   * renders the commercial table only from the accepted proposal snapshot;
--   * hides optional fields when empty;
--   * avoids fixed empty price tables.

with payload(markdown) as (
  values ($contract$
# CONTRATO PARTICULAR DE PRESTAÇÃO DE SERVIÇOS

Pelo presente instrumento particular, de um lado:

**I – {viagate.legal_name}**, pessoa jurídica de direito privado, inscrita no CNPJ sob o nº {viagate.cnpj}, doravante denominada simplesmente **CONTRATADA**;

E, de outro lado:

**II – {client.legal_name}**{% if client.trade_name %}, nome fantasia **{client.trade_name}**{% endif %}, pessoa jurídica de direito privado, inscrita no CNPJ sob o nº {client.cnpj}, com sede em {client.address}, neste ato representada por **{representative.name}**, CPF nº {representative.cpf}{% if representative.role %}, {representative.role}{% endif %}, e-mail {representative.email}, telefone {representative.phone}, doravante denominada simplesmente **CONTRATANTE**;

CONTRATADA e CONTRATANTE, em conjunto denominadas **PARTES**, resolvem celebrar o presente Contrato Particular de Prestação de Serviços, mediante as cláusulas e condições seguintes.

## 1. OBJETO

### 1.1.

O objeto do presente Contrato é a prestação de serviços pela CONTRATADA à CONTRATANTE, relativos ao fornecimento de informações cadastrais, análises, recomendações, autorizações, ferramentas tecnológicas e demais serviços relacionados à gestão de riscos em operações de transporte de cargas.

As pesquisas confirmatórias serão realizadas a partir das informações fornecidas pela CONTRATANTE, observadas as autorizações, bases legais e regras aplicáveis ao tratamento dos dados.

Parágrafo Único. A CONTRATADA poderá realizar pesquisas e análises concernentes a:

1. conduta social e informações cadastrais de motoristas, terceiros, agregados, frotistas e demais profissionais relacionados à operação;
2. dados relativos a veículos e seus respectivos proprietários;
3. informações necessárias à avaliação e ao gerenciamento de risco da operação contratada.

### 1.2. Produtos e serviços contratados

Os produtos, serviços, unidades, condições e valores efetivamente contratados são os constantes da tabela comercial da proposta, reproduzida na cláusula 2.1 deste instrumento.

{% if products.cargo_score %}
**Cargo Score.** A ferramenta CARGO SCORE é uma tecnologia destinada à consulta e recomendação de profissionais e veículos dentro das regras de gerenciamento de riscos aplicáveis à operação da CONTRATANTE, permitindo a indicação de aptidão e autorização conforme os parâmetros definidos para a prestação do serviço.

O banco de dados utilizado pelo Cargo Score poderá ser formado por dados provenientes do CARGO TRUCK e de outras fontes legítimas utilizadas pela CONTRATADA, observadas as autorizações e bases legais aplicáveis.

Quando aplicável ao modelo contratado, a CONTRATANTE terá acesso à recomendação e à liberação resultantes da análise efetuada pela ferramenta, sem que isso implique necessariamente acesso integral à ficha cadastral ou às fontes utilizadas na análise.
{% endif %}

### 1.3. Operação

**Tipo de operação:** {operation.type}

### 1.4. Informações securitárias

Quando aplicável à operação da CONTRATANTE, ficam registradas as seguintes informações:

| Informação | Valor |
|---|---|
| Seguradora | {insurance.insurer} |
| Início de vigência da apólice | {insurance.policy_start_date} |
| Fim de vigência da apólice | {insurance.policy_end_date} |
{% if insurance.broker_company %}| Corretora | {insurance.broker_company} |{% endif %}
{% if insurance.broker_producer %}| Produtor / Corretor responsável | {insurance.broker_producer} |{% endif %}

A prestação dos serviços deverá observar, quando aplicável, as regras estabelecidas na cláusula de gerenciamento de riscos da apólice de seguro contratada pela CONTRATANTE e, na ausência desta, o Projeto de Gerenciamento de Riscos aplicável à operação.

## 2. PREÇO E FORMA DE PAGAMENTO

### 2.1. Modelo comercial

**Modelo de precificação:** {proposal.pricing_model}

**Tabela comercial da proposta:**

{proposal.pricing_table}

### 2.2. Condições adicionais

**Fatura mínima mensal:** {proposal.minimum_invoice}

**Taxa de implantação / setup:** {proposal.setup_fee}

{% if proposal.valid_until %}**Proposta válida até:** {proposal.valid_until}{% endif %}

### 2.3.

Serviços extras relativos a customizações, novas funcionalidades da plataforma, integrações ou adaptações solicitadas pela CONTRATANTE e não contempladas no objeto deste Contrato dependerão de orçamento específico e aceite prévio da CONTRATANTE.

### 2.4.

Os valores estabelecidos no presente Contrato poderão ser corrigidos anualmente pelo índice IGP-M/FGV e, na hipótese de sua extinção, pelo índice que vier a substituí-lo ou por outro que venha a ser convencionado entre as PARTES.

### 2.5.

No preço dos serviços não estão contemplados eventuais custos de hospedagem, deslocamento, alimentação e demais despesas dos prepostos da CONTRATADA necessárias à execução de atividades presenciais, salvo previsão expressa em contrário na proposta comercial.

### 2.6.

Quando previamente autorizadas, as despesas extraordinárias deverão ser reembolsadas pela CONTRATANTE no prazo máximo de 10 (dez) dias corridos após o recebimento dos respectivos comprovantes.

### 2.7.

O atraso no pagamento implicará multa de 2% (dois por cento) sobre o valor em atraso, acrescida de juros de 1% (um por cento) ao mês, calculados pro rata die.

### 2.8.

A cobrança dos serviços iniciará conforme as condições comerciais aceitas e a efetiva disponibilização ou utilização dos serviços contratados.

### 2.9.

Os pagamentos deverão ser realizados até o dia 10 (dez) do mês subsequente ao da prestação dos serviços, mediante nota fiscal e meio de cobrança disponibilizado pela CONTRATADA, salvo condição diversa expressamente prevista na proposta comercial.

### 2.10. Reanálise

A reanálise de ficha compreende a retificação de informações passíveis de correção, quando houver preenchimento incorreto que tenha impactado o resultado da análise.

Parágrafo Único. A reanálise não compreende, quando tecnicamente ou juridicamente inviável, a alteração de campos primários de identificação, como CPF e placa de veículo, hipótese em que poderá ser necessária nova pesquisa.

## 3. OBRIGAÇÕES DAS PARTES

### 3.1.

A CONTRATADA deverá prestar os serviços descritos neste Contrato e na proposta comercial com zelo, qualidade e observância dos parâmetros aplicáveis à operação.

### 3.2.

A CONTRATADA fornecerá à CONTRATANTE, conforme os produtos contratados, recomendações, autorizações, resultados de pesquisas, alertas ou demais informações por meio de seus sistemas, integrações, e-mail ou outro canal formal disponibilizado.

Parágrafo Único. Quando a cobertura securitária da operação estiver condicionada à autorização formal da gerenciadora de riscos, a CONTRATANTE somente deverá liberar motorista, veículo ou viagem após a respectiva autorização, observando as obrigações estabelecidas pela seguradora, pela apólice e pelo Projeto de Gerenciamento de Riscos aplicável.

### 3.3.

A CONTRATADA disponibilizará à CONTRATANTE orientações relativas à operacionalização das ferramentas contratadas.

### 3.4.

A CONTRATANTE reconhece que determinados serviços poderão fornecer somente recomendação, status, autorização ou resultado consolidado, sem disponibilização integral da ficha cadastral ou das fontes consultadas.

### 3.5.

A CONTRATANTE compromete-se a fornecer informações corretas, completas e atualizadas, observando as orientações operacionais da CONTRATADA.

### 3.6.

A CONTRATANTE deverá comunicar à CONTRATADA qualquer ocorrência, anormalidade ou divergência relevante relacionada aos serviços contratados assim que dela tomar conhecimento.

### 3.7.

Quando previsto na proposta comercial ou necessário à implantação, a CONTRATADA poderá realizar treinamento específico sobre normas, procedimentos e utilização das ferramentas contratadas.

### 3.8.

Despesas de deslocamento, hospedagem, alimentação e locomoção relacionadas a treinamentos ou atividades presenciais observarão o disposto na cláusula 2 e as condições da proposta comercial.

## 4. VIGÊNCIA

### 4.1.

O presente Contrato entra em vigor na data de sua assinatura ou aceite e vigorará pelo prazo de 12 (doze) meses, renovando-se automaticamente por iguais períodos, salvo manifestação expressa de qualquer das PARTES nos termos deste Contrato.

### 4.2.

Para fins de registro comercial e eletrônico, considera-se como data de aceite da proposta:

**Data de aceite:** {proposal.accepted_at}

## 5. RESOLUÇÃO CONTRATUAL

### 5.1.

O presente Contrato poderá ser rescindido por qualquer uma das PARTES mediante aviso prévio e expresso com antecedência mínima de 30 (trinta) dias, sem multa contratual pelo pedido de cancelamento, permanecendo exigíveis os valores correspondentes aos serviços já prestados, consumidos ou contratualmente devidos.

### 5.2.

Qualquer uma das PARTES poderá resolver o Contrato independentemente de aviso prévio, multa ou indenização quando ocorrer:

1. imperícia, negligência ou imprudência grave na execução do objeto ou prestação em desacordo material com as especificações contratadas;
2. descumprimento de obrigação contratual não regularizado após notificação da PARTE prejudicada dentro do prazo estabelecido;
3. revelação ou utilização indevida de informação sigilosa a terceiros não autorizados.

## 6. COMPORTAMENTO IDÔNEO, ÉTICA E ANTICORRUPÇÃO

### 6.1.

Durante a relação contratual, as PARTES deverão observar a legislação anticorrupção aplicável e pautar sua atuação em princípios legais, éticos e de boa-fé.

### 6.2.

As PARTES declaram conhecer e cumprir as normas anticorrupção aplicáveis ao objeto deste Contrato, comprometendo-se a não praticar qualquer ato que possa constituir violação dessas normas.

### 6.3.

As PARTES comprometem-se a não dar, oferecer, pagar, prometer pagar ou autorizar, direta ou indiretamente, qualquer vantagem indevida a agente público ou privado com a finalidade de influenciar ato ou decisão ou obter vantagem ilícita.

### 6.4.

O descumprimento das obrigações desta cláusula poderá ensejar a resolução imediata do Contrato, sem prejuízo das medidas legais cabíveis.

## 7. CONFIDENCIALIDADE

### 7.1.

As PARTES, por si, seus empregados, prepostos, agentes, representantes e subcontratados, obrigam-se a manter sigilo sobre dados, materiais, documentos, especificações técnicas ou comerciais, métodos, processos, inovações e demais informações confidenciais a que tenham acesso em razão deste Contrato.

### 7.2.

Não serão consideradas confidenciais as informações que:

1. sejam ou se tornem públicas sem violação deste Contrato;
2. já estivessem legitimamente em posse da PARTE receptora antes de sua divulgação;
3. tenham sido recebidas legitimamente de terceiros sem obrigação de sigilo;
4. tenham sido expressamente identificadas pela PARTE reveladora como não confidenciais.

### 7.3.

O descumprimento do dever de confidencialidade facultará à PARTE prejudicada rescindir o Contrato, sem prejuízo do direito de pleitear perdas e danos.

### 7.4.

Durante a vigência deste Contrato, qualquer PARTE poderá solicitar a devolução ou eliminação de documento confidencial previamente compartilhado, ressalvadas as hipóteses de guarda obrigatória por lei, regulação, auditoria ou exercício regular de direitos.

### 7.5.

O dever de confidencialidade vigorará durante a relação contratual e permanecerá aplicável pelo prazo de 5 (cinco) anos após o seu término, ressalvadas obrigações legais ou contratuais específicas de prazo superior.

### 7.6.

Não será considerada violação de confidencialidade a divulgação exigida por lei, ordem judicial, autoridade administrativa competente ou procedimento arbitral, desde que observados os limites da determinação aplicável.

## 8. PROTEÇÃO DE DADOS PESSOAIS E LGPD

### 8.1.

Na execução deste Contrato, as PARTES observarão a Lei nº 13.709/2018 — Lei Geral de Proteção de Dados Pessoais (LGPD) — e demais normas aplicáveis ao tratamento de dados pessoais.

### 8.2.

A CONTRATANTE declara estar ciente de que deve utilizar os serviços e dados disponibilizados pela CONTRATADA exclusivamente para finalidades legítimas e compatíveis com a operação contratada, observando as bases legais aplicáveis.

### 8.3.

Em relação aos dados pessoais tratados no âmbito deste Contrato, as PARTES comprometem-se a:

1. tratar os dados somente para finalidades legítimas, específicas e compatíveis com a contratação;
2. adotar medidas técnicas e administrativas adequadas para proteção contra acessos não autorizados, perda, destruição, alteração, comunicação ou tratamento inadequado ou ilícito;
3. limitar o acesso aos dados às pessoas que necessitem deles para execução das atividades relacionadas ao Contrato;
4. comunicar à outra PARTE incidentes de segurança relevantes nos termos e prazos exigidos pela legislação aplicável;
5. colaborar, quando cabível, com o atendimento dos direitos dos titulares;
6. manter os dados somente pelo período necessário às finalidades do tratamento ou para cumprimento de obrigação legal, regulatória ou exercício regular de direitos.

### 8.4.

Quando necessário à execução da operação, os dados poderão ser compartilhados, dentro das bases legais aplicáveis, entre CONTRATANTE, CONTRATADA, seguradora, corretora, gerenciadora de riscos e demais agentes legitimamente envolvidos na operação.

## 9. DISPOSIÇÕES GERAIS

### 9.1.

O presente Contrato obriga as PARTES e seus sucessores, nos limites permitidos pela legislação aplicável.

### 9.2.

As informações relativas às condições financeiras e comerciais deste Contrato são consideradas confidenciais e não deverão ser divulgadas a terceiros não autorizados, ressalvadas as hipóteses legais ou necessárias à execução da contratação.

### 9.3.

A CONTRATANTE declara estar ciente de que os serviços disponibilizados pela CONTRATADA constituem ferramentas auxiliares de análise e gerenciamento de riscos e não representam garantia absoluta contra roubo, furto, desaparecimento, apropriação indébita, estelionato, acidentes ou outros eventos relacionados à operação de transporte.

### 9.4.

Para deslocamento às unidades da CONTRATANTE, a CONTRATADA deverá observar os procedimentos de acesso, segurança e autorização informados pela CONTRATANTE.

### 9.5.

Caso sejam necessários serviços, módulos ou operações não contemplados originalmente neste Contrato ou na proposta comercial, as PARTES poderão formalizar nova proposta, aditivo ou instrumento específico.

### 9.6.

A disponibilização das plataformas da CONTRATADA a transportadores subcontratados ou terceiros envolvidos na operação poderá depender de termo de adesão, cadastro ou instrumento específico.

### 9.7.

Após a execução do cronograma inicialmente previsto, eventuais pendências de implantação poderão ser objeto de cronograma complementar e, quando aplicável, orçamento adicional previamente aprovado.

### 9.8.

O presente Contrato não estabelece vínculo empregatício, sociedade, associação, representação exclusiva ou subordinação hierárquica entre as PARTES.

### 9.9.

A proposta comercial aceita pela CONTRATANTE integra este Contrato para todos os fins, inclusive quanto aos produtos, preços, condições comerciais e escopo contratado.

### 9.10.

Resultados de consultas provenientes de fontes distintas poderão apresentar divergências de conteúdo, atualização, cobertura territorial ou disponibilidade.

### 9.11.

Pesquisas ou consultas que dependam de processamento externo, bases estaduais, órgãos públicos ou terceiros estarão sujeitas aos respectivos horários, disponibilidade e prazos de resposta.

### 9.12.

Na ocorrência de indisponibilidade ou inconsistência sistêmica de uma fonte, a CONTRATADA poderá utilizar outros meios legítimos e tecnicamente adequados para execução dos serviços, quando compatível com o produto contratado.

### 9.13.

É de responsabilidade da CONTRATANTE armazenar os documentos inerentes à sua operação de transporte que sejam exigidos por lei, seguradora, PGR ou procedimento interno, incluindo, quando aplicável, CNH, documentos dos veículos e registros relacionados à ANTT.

### 9.14.

É de responsabilidade da CONTRATANTE acompanhar os vencimentos das liberações cadastrais e solicitar as respectivas renovações quando necessárias.

### 9.15.

A CONTRATANTE deverá verificar a correspondência entre as características físicas dos veículos e seus respectivos documentos antes de utilizar as informações em sua operação.

### 9.16.

É vedado o compartilhamento indevido de telas, relatórios, credenciais, dados ou informações disponibilizados nos sistemas da CONTRATADA, especialmente quando isso representar violação de confidencialidade, proteção de dados, propriedade intelectual ou direito de terceiros.

### 9.17.

Para dirimir dúvidas ou controvérsias oriundas deste Contrato, as PARTES elegem o foro da Comarca de Curitiba, Estado do Paraná, ressalvadas as hipóteses legais de competência obrigatória.

## 10. ACEITE E ASSINATURAS

Por estarem de acordo com as condições deste instrumento e da proposta comercial correspondente, as PARTES formalizam o presente Contrato.

**Data de aceite:** {proposal.accepted_at}

**CONTRATANTE**

**Razão social:** {client.legal_name}

{% if client.trade_name %}**Nome fantasia:** {client.trade_name}{% endif %}

**CNPJ:** {client.cnpj}

**Representante:** {representative.name}

**CPF:** {representative.cpf}

{% if representative.role %}**Cargo:** {representative.role}{% endif %}

**E-mail:** {representative.email}

**Telefone:** {representative.phone}

$contract$::text)
),
target as (
  select id,created_by,current_version+1 as next_version
  from contract_templates
  where is_active=true
  order by is_default desc,created_at
  limit 1
),
inserted as (
  insert into contract_template_versions(
    contract_template_id,
    version_number,
    markdown,
    template_hash,
    created_by
  )
  select target.id,
         target.next_version,
         payload.markdown,
         digest(payload.markdown,'sha256'),
         target.created_by
  from target
  cross join payload
  returning contract_template_id,version_number
)
update contract_templates t
set current_version=inserted.version_number,
    description=coalesce(nullif(t.description,''),'Contrato padrão ViaGate com condições comerciais dinâmicas'),
    updated_at=now()
from inserted
where t.id=inserted.contract_template_id;
