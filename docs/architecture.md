# Arquitetura da plataforma comercial

Este documento registra os caminhos canônicos do projeto para evitar que uma nova funcionalidade crie uma segunda implementação do mesmo comportamento.

## Princípios

- Regras de negócio ficam em `internal/`; templates apenas apresentam dados e compõem formulários.
- Rotas e handlers HTTP ficam em `internal/httpapp/` e delegam regras persistentes aos stores/serviços do domínio.
- Schema PostgreSQL evolui somente por novas migrations em `migrations/`. Migrations aplicadas não são editadas.
- Assets do navegador são pequenos módulos por responsabilidade, sem bundler e sem dependências globais desnecessárias.
- Antes de criar um componente, procurar o componente canônico equivalente em `web/templates/`.

## Templates canônicos

### Layouts

- `Base`: HTML base e comportamento global mínimo.
- `AuthLayout`: login, convite e recuperação de senha.
- `PublicFlowLayout`: onboarding, assinatura, ativação e verificação pública.
- `AdminLayout`: shell administrativo, navegação e listas.
- `AdminEditorLayout`: editores administrativos e controles de formulário.

### Componentes administrativos

- `AdminPageHeader`: cabeçalho de páginas/listas.
- `AdminListToolbar`, `AdminDataTable`, `AdminActionMenu`: listas administrativas.
- `AdminEditorHeader`, `AdminFormSection`: estrutura de editores.
- `AdminImageUpload`: upload de imagens comerciais.
- `FlashMessages`: feedback de formulários administrativos.

### Componentes públicos

- `PublicFlowHeader`
- `PublicFlashMessages`
- `PublicDataGrid`
- `PublicDataField`

## JavaScript canônico

- `core.js`: clipboard, estado de processamento de submits e comportamento realmente global.
- `form-controls.js`: máscaras, normalização de campos, consulta de CNPJ e upload de imagem.
- `admin-shell.js`: sidebar e navegação administrativa.
- `admin-list.js`: busca, filtros, contador, ações e responsividade das listas.
- `proposal-editor.js`: comportamento exclusivo do editor de proposta.
- `proposal-money.js`: entrada monetária do editor de proposta.
- `contract-template.js`: interação exclusiva do editor de modelo de contrato.
- `activation.js`: campos repetíveis da ativação.
- `contract-verification.js`: verificação local de documento.

Não adicionar JavaScript inline em templates quando o comportamento puder ser expresso por atributos `data-*` e um módulo já existente.

## Assets V1 ainda ativos

A apresentação institucional pública ainda usa o renderer visual V1 de forma intencional. O backend Go expõe somente os arquivos registrados em `registerV1VisualAssets` e o diretório `assets/` usado pelo renderer.

A cadeia principal é:

```text
PublicPresentationPage
  -> /v1/presentation-content.html
  -> styles.css + script.js
  -> executive-v2.js
  -> presentation-bootstrap.js
  -> story / mode / personalization / social / host bridge
```

Esses arquivos não devem ser removidos apenas por estarem fora de `web/`. Eles são compatibilidade visual ativa até uma futura migração do renderer.

A proposta pública reutiliza somente os cinco CSS preservados em `proposal/` e servidos explicitamente por `registerV1VisualAssets`.

## Legado removido

A implementação estática/Supabase anterior não faz parte da aplicação Go. O diretório `supabase/`, HTML/JS antigos da proposta e viewers estáticos não são fonte de verdade nem caminho de deploy.

## Regra para novas páginas

1. Escolher o layout pelo contexto (`Auth`, `PublicFlow`, `Admin` ou `AdminEditor`).
2. Reutilizar componentes existentes para cabeçalho, mensagens, seções, uploads e listas.
3. Criar JavaScript novo apenas quando o comportamento não couber em um módulo canônico existente.
4. Não acessar banco, S3 ou serviços externos diretamente do navegador.
5. Adicionar teste de domínio/handler quando houver regra nova.
6. Executar `task check` antes de considerar a alteração pronta.
