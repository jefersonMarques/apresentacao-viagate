# Apresentação ViaGate

Apresentação institucional e área de propostas comerciais da ViaGate.

## Estrutura

- `/apresentacao/` — apresentação institucional em modo apresentação.
- `/apresentacao/proposta/` — área comercial protegida por Supabase Auth.
- `/apresentacao/proposta/view.html?token=...` — proposta publicada para o cliente.
- `supabase/migrations/` — estrutura, versionamento, RLS, grants e RPCs das propostas.

## Apresentação institucional

A página principal utiliza uma camada de apresentação sobre o conteúdo institucional existente:

- camada inicial sobre toda a apresentação com blur de fundo;
- botão **Iniciar apresentação** centralizado;
- entrada pela Fullscreen API;
- ao sair da tela cheia, a apresentação é pausada e a camada retorna com **Continuar apresentação**;
- retomada no mesmo slide em que a apresentação foi interrompida;
- navegação por setas, Page Up/Page Down e roda do mouse;
- `Home` para o primeiro slide;
- `End` para o último slide;
- `F` para alternar tela cheia;
- scrollbar e elementos com comportamento de site ocultos;
- controles discretos que desaparecem quando o mouse fica inativo;
- slide narrativo com um exemplo completo de operação;
- slide final exclusivo para o contato comercial;
- métricas exibidas como base consolidada, sem contador simulando tempo real.

O contato exibido no último slide fica em `presentation-contact.js`, separado do conteúdo institucional. Isso permite trocar nome, cargo, foto, telefone, e-mail e WhatsApp do comercial sem alterar os slides.

Exemplo:

```javascript
window.presentationContact = Object.freeze({
  name: 'Antônio Santos',
  role: 'Sócio-Diretor Comercial',
  email: 'antonio.santos@viagate.com.br',
  phone: '(41) 99962-3600',
  whatsapp: '5541999623600',
  photoUrl: './assets/antonio-photo.svg',
});
```

O conteúdo institucional original foi preservado em `presentation-content.html`. O `index.html` funciona apenas como shell de apresentação e carrega as melhorias depois que os slides executivos existentes terminam de ser montados.

## Propostas comerciais

A proposta é um produto separado da apresentação institucional, embora compartilhe a identidade visual.

Cada proposta possui:

- cliente e contato;
- vendedor responsável;
- foto e contatos do vendedor;
- logo do cliente;
- contexto da negociação;
- prioridades do cliente;
- solução e escopo propostos;
- modelo comercial;
- itens de preço e opcionais;
- fatura mínima;
- implantação;
- condições comerciais;
- validade;
- versões imutáveis depois da publicação;
- token público aleatório por versão.

Ao alterar uma versão já publicada, o painel cria uma nova versão em rascunho. A versão publicada anteriormente permanece disponível pelo token original.

## Supabase

### 1. Banco

Execute as migrations do diretório `supabase/migrations/` na ordem abaixo:

```text
20260825_proposals.sql
20260825_proposals_grants.sql
20260825_proposals_immutability_fix.sql
```

As migrations criam:

- `salespeople`
- `clients`
- `client_contacts`
- `proposals`
- `proposal_versions`
- `proposal_version_items`
- políticas RLS
- permissões explícitas para `authenticated`
- imutabilidade de versões publicadas
- `publish_proposal_version`
- `get_public_proposal`

O papel Postgres `anon` não possui `SELECT` direto nas tabelas. A proposta pública é retornada somente pelo RPC `get_public_proposal` quando o token corresponde a uma versão publicada.

### 2. Autenticação

Os usuários são criados diretamente no **Supabase Authentication**, sem cadastro público pela aplicação.

No Supabase Auth, configure a URL abaixo como redirect permitido para recuperação de senha:

```text
https://viagate.com.br/apresentacao/proposta/
```

### 3. Configuração do frontend

Preencha `proposal/config.js` com a URL do projeto e a **Publishable Key** (`sb_publishable_...`):

```javascript
export const proposalConfig = Object.freeze({
  supabaseUrl: 'https://SEU-PROJETO.supabase.co',
  supabasePublishableKey: 'sb_publishable_...',
  publicProposalUrl: '/apresentacao/proposta/view.html',
  loginUrl: '/apresentacao/proposta/',
});
```

A Publishable Key é própria para aplicações web e pode existir no frontend. A segurança dos dados continua sendo controlada pelas políticas RLS e pela sessão do usuário autenticado.

A **Secret Key** (`sb_secret_...`) nunca deve ser colocada no repositório, no JavaScript entregue ao navegador ou em qualquer outro componente público.

## Servidor

Nesta versão não é necessário backend próprio. A aplicação continua estática e utiliza Supabase Auth, PostgREST e RPCs.

Se futuramente houver uma função que realmente exija servidor próprio, o backend deverá ser implementado em Go.

## Executar localmente

A aplicação precisa ser servida por HTTP; não abra os arquivos diretamente com `file://`.

```bash
python -m http.server 8080
```

Depois acesse:

```text
http://localhost:8080/
```

Para testar o caminho utilizado em produção, o servidor local deve reproduzir o prefixo `/apresentacao/` ou o `<base>` do conteúdo institucional deve ser ajustado apenas no ambiente local.
