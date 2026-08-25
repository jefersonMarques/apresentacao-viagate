# ViaGate — Hub Comercial

Área comercial para gerar, publicar e acompanhar apresentações institucionais e propostas comerciais.

## Estrutura

- `/apresentacao/` — redireciona para o Hub Comercial autenticado.
- `/apresentacao/proposta/` — Hub Comercial protegido por Supabase Auth.
- `/apresentacao/view.html?token=...` — apresentação institucional publicada e personalizada quando o projeto está montado em `/apresentacao/`.
- `/apresentacao/proposta/view.html?token=...` — proposta comercial publicada quando o projeto está montado em `/apresentacao/`.
- `presentation-content.html` — template institucional reutilizado pelas apresentações geradas.
- `supabase/migrations/` — banco, RLS, versionamento, publicação, analytics e Storage.

Os links públicos são montados de forma relativa ao Hub Comercial. Assim, em desenvolvimento o projeto também pode ser servido diretamente na raiz sem depender do prefixo `/apresentacao/`.

## Hub Comercial

Depois do login, o comercial pode gerar dois tipos de material:

### Apresentação institucional

O conteúdo institucional permanece padronizado, mas cada link publicado pode ter personalização própria:

- vendedor responsável;
- foto, cargo, telefone, WhatsApp e e-mail do vendedor;
- slide final de contato ativado ou desativado;
- empresa do cliente opcional;
- contato do cliente opcional;
- logo do cliente opcional;
- identificação do cliente ativada ou desativada.

Cada publicação possui token público próprio e versão imutável.

### Proposta comercial

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

Ao alterar uma versão publicada, uma nova versão em rascunho é criada. O link antigo continua representando a versão publicada anteriormente.

## Imagens comerciais

Fotos de vendedores e logos de clientes são enviados diretamente para o Supabase Storage. O usuário não precisa informar URL manualmente.

Bucket utilizado:

```text
commercial-assets
```

Estrutura dos objetos:

```text
commercial-assets/
├── salespeople/{auth_user_id}/{uuid}.{ext}
└── clients/{auth_user_id}/{uuid}.{ext}
```

Regras atuais:

- bucket público somente para leitura dos arquivos publicados;
- upload, alteração e exclusão somente por usuário autenticado dentro da própria pasta;
- PNG, JPG, WEBP e SVG;
- limite de 2 MB por arquivo;
- nome físico gerado com UUID;
- arquivos substituídos não são apagados automaticamente, porque versões publicadas antigas podem continuar utilizando a imagem anterior.

## Estatísticas de leitura

Apresentações e propostas registram os seguintes eventos pelo token publicado:

- `open` — link válido foi aberto;
- `start` — visitante iniciou a apresentação/proposta;
- `slide_view` — primeiro acesso daquele visitante a cada slide;
- `complete` — visitante chegou ao último slide.

O painel classifica o material como:

- **Não aberta** — nenhuma abertura registrada;
- **Aberta** — o link foi aberto, mas a apresentação ainda não avançou de forma relevante;
- **Em leitura** — o visitante iniciou ou avançou pelos slides;
- **Lida** — pelo menos uma sessão chegou ao último slide.

Também são exibidos:

- número de aberturas;
- progresso máximo alcançado;
- data/hora da última abertura;
- quantidade geral de materiais publicados e lidos.

A sessão de leitura utiliza somente um UUID aleatório salvo em `sessionStorage`. Não é utilizado fingerprinting do dispositivo.

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
```

As migrations do Hub Comercial adicionam:

- `presentations`;
- `presentation_versions`;
- `shared_document_events`;
- RLS para apresentações e eventos;
- isolamento de clientes, propostas, versões e itens por usuário autenticado;
- `publish_presentation_version`;
- `get_public_presentation`;
- `track_shared_document_event`;
- `get_my_shared_document_stats`;
- endurecimento de `publish_proposal_version` para impedir publicação de material de outro comercial;
- bucket `commercial-assets` e políticas de upload por usuário.

O papel Postgres `anon` não possui `SELECT` direto nas tabelas comerciais ou de analytics. Links públicos acessam somente RPCs limitadas por token publicado. As imagens do bucket são públicas porque fazem parte de materiais públicos compartilhados por token.

### Autenticação

Os usuários são criados diretamente no Supabase Authentication. Não existe cadastro público na aplicação.

Configure como redirect permitido para recuperação de senha:

```text
https://viagate.com.br/apresentacao/proposta/
```

Para desenvolvimento local com o projeto montado em `/apresentacao/`, adicione também:

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

A Publishable Key é própria para aplicações web. A segurança dos dados depende das políticas RLS e da sessão autenticada.

Nunca coloque `sb_secret_...` no repositório ou no navegador.

## Servidor

Não é necessário backend próprio nesta versão. Supabase Auth, PostgREST, Storage e RPCs atendem o fluxo atual.

Se surgir uma função que realmente exija backend próprio, ela deverá ser implementada em Go.

## Executar localmente

### Com o junction `/apresentacao/`

Sirva o diretório pai que contém o junction:

```bash
python -m http.server 8080
```

Acesse:

```text
http://localhost:8080/apresentacao/
```

### Servindo o repositório diretamente

Também é possível executar o servidor dentro da pasta do projeto:

```bash
python -m http.server 8080
```

Nesse caso acesse:

```text
http://localhost:8080/
```

Os links gerados serão ajustados automaticamente para a estrutura usada no acesso atual.
