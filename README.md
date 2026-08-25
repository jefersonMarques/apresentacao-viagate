# ViaGate — Hub Comercial

Área comercial para gerar, publicar e acompanhar apresentações institucionais e propostas comerciais.

## Estrutura

- `/apresentacao/` — redireciona para o Hub Comercial autenticado.
- `/apresentacao/proposta/` — Hub Comercial protegido por Supabase Auth.
- `/apresentacao/view.html?token=...` — apresentação institucional publicada.
- `/apresentacao/proposta/view.html?token=...` — proposta comercial publicada.
- `presentation-content.html` — conteúdo institucional reutilizado pelas apresentações geradas.
- `supabase/migrations/` — banco, RLS, versionamento, publicação, analytics e Storage.

Os links públicos são montados de forma relativa ao Hub Comercial, permitindo executar o projeto com ou sem o prefixo `/apresentacao/`.

## Hub Comercial

A gestão autenticada é separada em:

- **Visão geral** — materiais publicados e acompanhamento de leitura;
- **Apresentações** — criação, publicação e edição das apresentações institucionais;
- **Propostas** — criação, versionamento, publicação e acompanhamento das propostas;
- **Meu perfil** — foto e dados do comercial usados nos materiais publicados.

### Apresentação institucional

Cada publicação possui token próprio e pode configurar:

- vendedor responsável;
- foto, cargo, telefone, WhatsApp e e-mail;
- slide final de contato ativado ou desativado;
- empresa do cliente;
- contato do cliente;
- logo do cliente;
- identificação do cliente ativada ou desativada.

O conteúdo somente é liberado após entrada em tela cheia. Ao sair do fullscreen, a apresentação volta para o estado bloqueado e oferece **Continuar apresentação** ou **Voltar ao início**.

A navegação fica centralizada no rodapé com:

```text
↑   05 / 18   ↓
```

### Proposta comercial

A proposta utiliza o mesmo comportamento de tela cheia da apresentação. O conteúdo permanece bloqueado até o visitante iniciar a apresentação em fullscreen.

Cada proposta possui:

- cliente e contato;
- vendedor responsável;
- logo do cliente;
- cenário operacional considerado;
- solução e escopo;
- modelo de análise cadastral;
- itens de investimento e opcionais;
- fatura mínima;
- implantação;
- condições comerciais;
- validade;
- versões imutáveis após publicação;
- token público aleatório por versão.

Modelos comerciais suportados:

- análise por item;
- análise por conjunto;
- análise por item + conjunto;
- condições específicas.

O viewer comercial usa como referência a estrutura já utilizada pela ViaGate: Score, consultas e autenticação, prevenção, aplicativo/logística, monitoramento de veículos e fatura mínima.

## Imagens comerciais

Fotos de vendedores e logos de clientes são enviados diretamente para o Supabase Storage.

Bucket:

```text
commercial-assets
```

Estrutura:

```text
commercial-assets/
├── salespeople/{auth_user_id}/{uuid}.{ext}
└── clients/{auth_user_id}/{uuid}.{ext}
```

Regras:

- leitura pública dos arquivos usados nos materiais publicados;
- upload, alteração e exclusão somente por usuário autenticado dentro da própria pasta;
- PNG, JPG, WEBP e SVG;
- limite de 2 MB;
- nome físico gerado com UUID;
- arquivos substituídos não são apagados automaticamente, preservando versões já publicadas.

## Estatísticas de leitura

Apresentações e propostas registram:

- `open` — link válido aberto;
- `start` — visitante iniciou o material em tela cheia;
- `slide_view` — primeiro acesso da sessão a cada slide;
- `complete` — sessão chegou ao último slide.

O Hub classifica cada link como:

- **Não aberta** — nenhuma abertura;
- **Aberta** — link aberto, mas apresentação não iniciada;
- **Em leitura** — material iniciado ou parcialmente percorrido;
- **Lida** — pelo menos uma sessão chegou ao último slide.

Também são exibidos:

- número de aberturas;
- progresso máximo;
- primeira e última abertura disponíveis na RPC;
- última inicialização;
- última conclusão;
- quantidade geral de materiais publicados e lidos.

A sessão utiliza um UUID aleatório em `sessionStorage`. Não é utilizado fingerprinting.

## Supabase

### Banco e Storage

Execute as migrations na ordem:

```text
20260825_proposals.sql
20260825_proposals_grants.sql
20260825_proposals_immutability_fix.sql
20260825_commercial_hub_analytics.sql
20260825_commercial_hub_access_control.sql
20260825_commercial_assets_storage.sql
20260825_proposal_pricing_models.sql
20260825_commercial_hub_read_status.sql
```

As migrations adicionam:

- propostas e versões;
- apresentações e versões;
- eventos de leitura;
- RLS e isolamento por usuário;
- publicação por token;
- estatísticas de abertura e conclusão;
- suporte aos modelos por item, conjunto e item + conjunto;
- bucket `commercial-assets` e políticas de upload.

O papel Postgres `anon` não possui `SELECT` direto nas tabelas comerciais ou de analytics. Os viewers públicos acessam somente RPCs limitadas por token publicado.

### Autenticação

Os usuários são criados diretamente no Supabase Authentication. Não existe cadastro público.

Redirect de produção:

```text
https://viagate.com.br/apresentacao/proposta/
```

Para desenvolvimento local com `/apresentacao/`:

```text
http://localhost:8080/apresentacao/proposta/
```

### Frontend

A configuração fica em `proposal/config.js`:

```javascript
export const proposalConfig = Object.freeze({
  supabaseUrl: 'https://SEU-PROJETO.supabase.co',
  supabasePublishableKey: 'sb_publishable_...',
  publicProposalUrl: './view.html',
  publicPresentationUrl: '../view.html',
  loginUrl: './',
  assetBucket: 'commercial-assets',
});
```

Nunca coloque `sb_secret_...` no repositório ou no navegador.

## Servidor

Não é necessário backend próprio nesta versão. Supabase Auth, PostgREST, Storage e RPCs atendem o fluxo atual.

Se futuramente houver necessidade de backend próprio, ele deverá ser implementado em Go.

## Executar localmente

### Com junction `/apresentacao/`

Sirva o diretório pai que contém o junction:

```bash
python -m http.server 8080
```

Acesse:

```text
http://localhost:8080/apresentacao/
```

### Servindo o repositório diretamente

Dentro da pasta do projeto:

```bash
python -m http.server 8080
```

Acesse:

```text
http://localhost:8080/
```
