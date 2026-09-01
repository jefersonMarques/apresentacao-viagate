# ViaGate — Plataforma Comercial

Sistema comercial da ViaGate para apresentação institucional, proposta, aceite, onboarding do cliente, geração de contrato e assinatura eletrônica.

Esta branch substitui a arquitetura do protótipo estático/Supabase por um backend próprio em Go.

## Arquitetura

- **Backend:** Go 1.24+
- **UI:** templ, HTML server-side e JavaScript progressivo
- **Banco:** PostgreSQL
- **Arquivos privados:** S3 ou serviço S3-compatible
- **E-mail transacional:** Brevo
- **PDF:** Chromium headless
- **Consulta CNPJ:** provider desacoplado; BrasilAPI é a implementação inicial

O navegador nunca possui credenciais do banco ou do S3. Toda autorização e regra de negócio passa pelo backend Go.

As convenções de manutenção, layouts, componentes canônicos, módulos JavaScript e a fronteira do renderer V1 ainda ativo estão documentados em [`docs/architecture.md`](docs/architecture.md). Antes de criar um segundo mecanismo de formulário, lista, upload ou comportamento de navegador, use esse documento como referência.

## Fluxo comercial

```text
Comercial
  ↓
Apresentação / Proposta versionada
  ↓
Cliente abre a proposta
  ↓
Aceite da versão exata
  ↓
Dados pessoais do responsável
  ↓
Onboarding + CNPJ + apólice
  ↓
Revisão interna
  ├─ correção solicitada → cliente retoma o cadastro
  └─ aprovado
       ↓
Contrato gerado a partir de Markdown versionado
       ↓
PDF + SHA-256
       ↓
OTP por e-mail / Brevo
       ↓
Assinatura eletrônica
       ↓
Dados para ativação
       ↓
Implantação interna
       ↓
Operação liberada
```

A biometria facial e a prova de vida já existem como modos previstos no domínio de verificação de identidade, mas não estão habilitadas nesta versão.

## Segurança e imutabilidade

O sistema mantém separadas as versões de proposta, aceite, onboarding, template de contrato, contrato e assinatura.

Uma proposta publicada não é alterada retroativamente. O aceite registra a versão e o SHA-256 do conteúdo aceito. O contrato é gerado a partir dos dados aprovados e de uma versão específica do template. Após geração, seu conteúdo e hash ficam protegidos contra alteração. Eventos de auditoria e assinatura são append-only.

Ao final da assinatura são produzidos:

```text
contract.pdf
evidence.pdf
evidence.json
manifest.json
signed-package.zip
```

O relatório registra, entre outros dados, identidade declarada do responsável, CPF, e-mail, OTP, texto de consentimento, data/hora, IP, user-agent, sessão e SHA-256 do documento.

## Perfis iniciais

- `commercial` — cria e acompanha os próprios materiais comerciais;
- `operations` — revisa onboarding e documentos;
- `legal` — administra modelos de contrato;
- `super_admin` — visão global e gestão de usuários/permissões.

Novos usuários são criados por convite individual e expirável. Não existe cadastro público.

## Pré-requisitos locais

Instale localmente:

- Go 1.24 ou superior;
- PostgreSQL;
- Chromium/Chrome compatível com execução headless;
- um serviço S3-compatible, como MinIO, ou um bucket S3 real.

O Brevo é necessário para testar entrega real de convites, OTPs e notificações. Sem chave do Brevo o servidor pode ser iniciado em desenvolvimento, mas os e-mails permanecerão na outbox com retry.

## Configuração

Copie `.env.example` para `.env` e ajuste os valores. A aplicação lê variáveis de ambiente; o arquivo `.env` serve como referência e pode ser carregado pelos scripts de desenvolvimento.

Exemplo local com PostgreSQL e MinIO:

```env
APP_ENV=development
APP_ADDR=:8080
APP_BASE_URL=http://localhost:8080
DATABASE_URL=postgres://viagate:viagate@localhost:5432/viagate?sslmode=disable
CHROMIUM_PATH=chromium
TRUST_PROXY_HEADERS=false
REQUIRE_ONBOARDING_REVIEW=true

S3_REGION=us-east-1
S3_BUCKET=viagate-commercial
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
S3_USE_PATH_STYLE=true
S3_SERVER_SIDE_ENCRYPTION=none

BREVO_API_KEY=
BREVO_SENDER_EMAIL=naoresponda@viagate.com.br
BREVO_SENDER_NAME=ViaGate

BOOTSTRAP_ADMIN_EMAIL=seu-email@viagate.com.br
BOOTSTRAP_ADMIN_NAME=Super Admin
```

`S3_SERVER_SIDE_ENCRYPTION=none` é aceito somente para desenvolvimento/teste. Produção exige `AES256` ou `aws:kms`.

Quando o Go estiver atrás de Nginx ou outro proxy reverso confiável que sobrescreva `X-Forwarded-For`/`X-Real-IP`, configure:

```env
TRUST_PROXY_HEADERS=true
```

Não habilite essa opção se o backend puder ser acessado diretamente por clientes externos.

## PostgreSQL

Crie um banco limpo para a nova plataforma. Exemplo:

```sql
create user viagate with password 'viagate';
create database viagate owner viagate;
```

Depois carregue `DATABASE_URL` no ambiente e execute:

```bash
go run ./cmd/migrate up
```

O runner aplica os arquivos de `migrations/` em ordem, cada migration dentro de uma transação, e registra o SHA-256 em `schema_migrations`.

Se uma migration já aplicada for modificada posteriormente, o runner interrompe a execução. Mudanças de schema devem ser feitas por uma nova migration, nunca editando migrations já utilizadas em um ambiente persistente.

Não existe `down` automático para migrations que envolvem documentos comerciais/jurídicos imutáveis.

## S3 / MinIO

O bucket configurado em `S3_BUCKET` deve existir antes de iniciar a aplicação.

Para MinIO local, um exemplo usando o cliente `mc` é:

```bash
mc alias set local http://localhost:9000 minioadmin minioadmin
mc mb --ignore-existing local/viagate-commercial
```

O bucket deve permanecer privado. O sistema grava apenas as chaves no PostgreSQL e gera URLs temporárias para downloads autorizados.

Principais prefixos:

```text
onboarding/{onboarding_id}/insurance_policy/...
contracts/{onboarding_id}/...
contracts/{contract_id}/final/evidence.pdf
contracts/{contract_id}/final/signed-package.zip
```

## Primeiro Super Admin

Na primeira execução configure:

```env
BOOTSTRAP_ADMIN_EMAIL=seu-email@viagate.com.br
BOOTSTRAP_ADMIN_NAME=Seu Nome
```

Se ainda não houver Super Admin, o sistema cria um convite de ativação. Em desenvolvimento a URL inicial também é registrada no log do servidor; com Brevo configurado, ela é enviada por e-mail.

Depois de ativar o primeiro acesso, remova `BOOTSTRAP_ADMIN_EMAIL` da configuração permanente.

## Modelo de contrato

Antes de testar um onboarding completo, entre como Super Admin/Jurídico e crie pelo menos um modelo de contrato, marcando-o como padrão.

Os modelos são Markdown versionado e aceitam variáveis controladas, por exemplo:

```text
{client.legal_name}
{client.cnpj}
{client.address}
{representative.name}
{representative.cpf}
{proposal.minimum_invoice}
{viagate.legal_name}
{viagate.cnpj}
```

Cada salvamento gera uma nova versão. Contratos já gerados continuam ligados à versão antiga.

## Revisão do onboarding

O padrão é:

```env
REQUIRE_ONBOARDING_REVIEW=true
```

Nesse modo, depois que o cliente envia o cadastro e a apólice, os dados ficam bloqueados e entram na fila de revisão. Operações/Super Admin pode:

- marcar em revisão;
- solicitar correção, enviando ao cliente um novo link seguro;
- aprovar o cadastro.

Somente a aprovação permite gerar e enviar o contrato.

Para ambientes de demonstração pode ser usado `REQUIRE_ONBOARDING_REVIEW=false`; o sistema então aprova o cadastro automaticamente antes de gerar o contrato.

## Desenvolvimento

Baixe as dependências e gere os componentes templ:

```bash
go mod download
go run github.com/a-h/templ/cmd/templ@v0.3.943 generate
```

Aplique as migrations:

```bash
go run ./cmd/migrate up
```

Inicie:

```bash
go run ./cmd/server
```

Acesse:

```text
http://localhost:8080/login
```

Endpoints operacionais:

```text
GET /healthz   # processo HTTP vivo
GET /readyz    # PostgreSQL e bucket S3 acessíveis
```

## Makefile

Em ambientes com `make`:

```bash
make deps
make migrate-up
make dev
make test
make check
```

`make check` gera os templates, executa `go vet`, testes e build dos comandos principais.

## Ordem recomendada para o primeiro teste

1. iniciar PostgreSQL;
2. iniciar/configurar S3 ou MinIO e criar o bucket privado;
3. carregar as variáveis de ambiente;
4. executar `go run ./cmd/migrate up`;
5. configurar e ativar o primeiro Super Admin;
6. criar um modelo de contrato padrão;
7. criar/publicar uma proposta;
8. aceitar a proposta como cliente;
9. preencher onboarding e enviar a apólice;
10. revisar e aprovar no painel administrativo;
11. abrir o link recebido via Brevo e assinar com OTP;
12. complementar os dados para ativação;
13. validar `contract.pdf`, `evidence.pdf` e `signed-package.zip`.

## Produção

A V1 está preparada como release candidate de produção. O procedimento operacional completo está em [`docs/production.md`](docs/production.md).

A release inclui:

- bundle Linux reproduzível com binários e assets de runtime;
- checksum SHA-256 do bundle;
- comando `preflight` para validar configuração, PostgreSQL, S3 e Chromium;
- template de ambiente de produção em `deploy/viagate.env.example`;
- unit systemd em `deploy/viagate-commercial.service`;
- geração automática do bundle pelo GitHub Actions após o CI;
- procedimento documentado de migration, healthcheck, smoke test e rollback.

Antes de liberar clientes reais, ainda é obrigatório executar no ambiente de produção o preflight, backup do PostgreSQL, migrations, healthchecks e o smoke test funcional descrito no documento de produção. O deploy deve usar exatamente o SHA que passou no CI.
