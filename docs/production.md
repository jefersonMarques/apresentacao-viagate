# Produção — ViaGate Comercial

Este documento descreve a promoção da plataforma para um host Linux com systemd e proxy reverso. O processo não depende de Docker.

## 1. Dependências obrigatórias

O host de produção precisa ter:

- Linux x86_64;
- PostgreSQL acessível pelo backend;
- bucket S3 privado e acessível;
- Chrome, Chromium ou Edge compatível com execução headless;
- acesso HTTPS público através de proxy reverso;
- credenciais válidas do Brevo;
- DNS apontando para o proxy antes da abertura para usuários reais.

A geração de contrato e os PDFs comerciais dependem do navegador headless. Por isso a release inclui o comando `preflight`, que verifica configuração, PostgreSQL, S3 e o executável do navegador antes do restart.

## 2. Layout recomendado no servidor

```text
/opt/viagate-commercial/
  releases/
    <version>/
  current -> /opt/viagate-commercial/releases/<version>

/etc/viagate-commercial/
  viagate.env

/var/lib/viagate-commercial/
```

Crie o usuário do serviço e os diretórios uma única vez:

```bash
sudo useradd --system --home /var/lib/viagate-commercial --create-home --shell /usr/sbin/nologin viagate || true
sudo mkdir -p /opt/viagate-commercial/releases /etc/viagate-commercial
sudo chown -R viagate:viagate /opt/viagate-commercial /var/lib/viagate-commercial
sudo chmod 750 /etc/viagate-commercial
```

O arquivo `/etc/viagate-commercial/viagate.env` deve ser legível apenas por root e pelo grupo do serviço:

```bash
sudo cp deploy/viagate.env.example /etc/viagate-commercial/viagate.env
sudo chown root:viagate /etc/viagate-commercial/viagate.env
sudo chmod 640 /etc/viagate-commercial/viagate.env
sudo nano /etc/viagate-commercial/viagate.env
```

Nunca envie esse arquivo para o Git.

## 3. Configuração de produção

Valores obrigatórios ou críticos:

```env
APP_ENV=production
APP_ADDR=127.0.0.1:8080
APP_BASE_URL=https://SEU-DOMINIO
DATABASE_URL=...
CHROMIUM_PATH=...
TRUST_PROXY_HEADERS=true
REQUIRE_ONBOARDING_REVIEW=true
VIAGATE_LEGAL_NAME=...
VIAGATE_CNPJ=...
S3_BUCKET=...
S3_SERVER_SIDE_ENCRYPTION=AES256
BREVO_API_KEY=...
```

`APP_BASE_URL` precisa ser HTTPS. A aplicação recusa inicialização em produção quando a URL não é pública/HTTPS, quando os dados legais obrigatórios estão ausentes, quando o Brevo não está configurado ou quando a criptografia do S3 está desativada.

`TRUST_PROXY_HEADERS=true` só deve ser usado quando `APP_ADDR` não estiver publicamente acessível e o proxy reverso for o único caminho até o Go.

### PostgreSQL

Se PostgreSQL estiver no mesmo host, mantenha-o escutando apenas em loopback ou socket local. Para PostgreSQL remoto/gerenciado, use TLS e a política de validação adequada ao provedor.

Antes de qualquer migration de produção, faça backup:

```bash
pg_dump --format=custom --file=/var/backups/viagate-before-$(date +%Y%m%d-%H%M%S).dump "$DATABASE_URL"
```

As migrations são forward-only. Não existe `down` automático para documentos comerciais e jurídicos.

### S3

O bucket precisa existir antes do deploy e deve permanecer privado. Em produção use `AES256` ou `aws:kms`. Se usar KMS, configure também `S3_KMS_KEY_ID`.

### Chromium

Confirme o executável:

```bash
which chromium || which chromium-browser || which google-chrome || which google-chrome-stable
```

Aponte `CHROMIUM_PATH` para o executável real. O `preflight` falha se o navegador não for localizado.

## 4. Gerar uma release

O bundle Linux contém os três binários e todos os arquivos necessários em runtime:

```text
server
migrate
preflight
migrations/
web/assets/
assets/
proposal/
presentation-content.html
*.css e *.js da apresentação V1
VERSION
GIT_SHA
```

Localmente/CI:

```bash
bash ./scripts/build-release.sh
```

Saída:

```text
dist/viagate-commercial-<sha>-linux-amd64.tar.gz
dist/viagate-commercial-<sha>-linux-amd64.tar.gz.sha256
```

O GitHub Actions também gera esse bundle como artifact depois que o CI passa.

## 5. Instalar a release sem downtime desnecessário

Envie o `.tar.gz` para o servidor e confira o checksum:

```bash
sha256sum -c viagate-commercial-<version>-linux-amd64.tar.gz.sha256
```

Extraia em uma release nova, sem tocar no `current` ainda:

```bash
VERSION=<version>
sudo tar -xzf viagate-commercial-${VERSION}-linux-amd64.tar.gz -C /opt/viagate-commercial/releases/
sudo chown -R viagate:viagate /opt/viagate-commercial/releases/viagate-commercial-${VERSION}-linux-amd64
```

Defina:

```bash
RELEASE=/opt/viagate-commercial/releases/viagate-commercial-${VERSION}-linux-amd64
```

Carregue a configuração e execute o preflight usando a release nova:

```bash
sudo -u viagate bash -c '
  set -a
  source /etc/viagate-commercial/viagate.env
  set +a
  cd "'"$RELEASE"'"
  ./preflight
'
```

O resultado esperado é:

```text
ok postgres
ok s3
ok browser
production preflight passed
```

## 6. Aplicar migrations

Somente depois de backup + preflight:

```bash
sudo -u viagate bash -c '
  set -a
  source /etc/viagate-commercial/viagate.env
  set +a
  cd "'"$RELEASE"'"
  ./migrate up
'
```

O runner valida os hashes das migrations já aplicadas e interrompe se detectar alteração indevida em migration histórica.

## 7. Ativar a release

Troque o symlink de forma atômica:

```bash
sudo ln -sfn "$RELEASE" /opt/viagate-commercial/current.new
sudo mv -Tf /opt/viagate-commercial/current.new /opt/viagate-commercial/current
```

Instale o service na primeira implantação:

```bash
sudo cp deploy/viagate-commercial.service /etc/systemd/system/viagate-commercial.service
sudo systemctl daemon-reload
sudo systemctl enable viagate-commercial
```

Inicie/reinicie:

```bash
sudo systemctl restart viagate-commercial
sudo systemctl status viagate-commercial --no-pager
```

Logs:

```bash
journalctl -u viagate-commercial -f
```

## 8. Healthcheck antes de liberar tráfego

No próprio servidor:

```bash
curl -fsS http://127.0.0.1:8080/healthz -o /dev/null
curl -fsS http://127.0.0.1:8080/readyz -o /dev/null
```

`/healthz` confirma o processo HTTP. `/readyz` confirma PostgreSQL e S3.

Depois teste externamente via HTTPS:

```bash
curl -fsS https://SEU-DOMINIO/healthz -o /dev/null
curl -fsS https://SEU-DOMINIO/readyz -o /dev/null
```

## 9. Proxy reverso

Encaminhe o domínio HTTPS para:

```text
127.0.0.1:8080
```

Preserve o `Host` original e sobrescreva corretamente `X-Forwarded-For`/`X-Real-IP`. Force HTTPS no proxy. A aplicação em produção também envia HSTS e headers de segurança.

Não exponha a porta `8080` à internet quando `TRUST_PROXY_HEADERS=true`.

## 10. Smoke test obrigatório

Antes de considerar a release aberta para clientes reais, valide em produção:

1. login administrativo;
2. criação/publicação de apresentação;
3. abertura do link público da apresentação;
4. PDF da apresentação, incluindo a página de métricas dinâmicas;
5. criação/publicação de proposta;
6. abertura pública da proposta;
7. PDF da proposta;
8. aceite da proposta;
9. retomada pelo mesmo link da proposta;
10. onboarding e upload de apólice;
11. revisão/aprovação interna;
12. recebimento do e-mail do contrato;
13. abertura do PDF exato do contrato;
14. envio e confirmação do OTP;
15. download do contrato assinado, evidência e pacote final;
16. ativação do cliente;
17. notificação interna de ativação pendente;
18. mudança para implantação interna e resolução do alerta.

Esse smoke test valida, em conjunto, PostgreSQL, S3, Brevo, Chromium, jobs internos, sessão, links públicos e fluxo jurídico.

## 11. Primeiro Super Admin

Se o banco de produção for novo, configure temporariamente:

```env
BOOTSTRAP_ADMIN_EMAIL=...
BOOTSTRAP_ADMIN_NAME=...
```

Inicie a aplicação, receba e conclua o convite. Depois remova esses dois valores da configuração permanente e reinicie o serviço.

## 12. Rollback

Mantenha pelo menos a release anterior em `/opt/viagate-commercial/releases`.

Se a nova release falhar antes de migrations, volte o symlink e reinicie:

```bash
sudo ln -sfn /opt/viagate-commercial/releases/<release-anterior> /opt/viagate-commercial/current.new
sudo mv -Tf /opt/viagate-commercial/current.new /opt/viagate-commercial/current
sudo systemctl restart viagate-commercial
```

Se migrations já foram aplicadas, não presuma que um binário antigo é compatível com o schema novo. Nesse caso, avalie a migration específica. Se houver incompatibilidade, o rollback seguro exige restauração do backup PostgreSQL e coordenação com os objetos S3 criados após o deploy.

## 13. Backups e observabilidade

Antes de abrir produção, defina:

- backup automático do PostgreSQL;
- teste periódico de restauração;
- versionamento/retenção do bucket S3 quando suportado;
- retenção dos logs do `journald`;
- monitoramento externo de `/healthz` e `/readyz`;
- alerta para indisponibilidade do serviço;
- acompanhamento da outbox de e-mails e falhas de finalização de contrato.

## 14. Critério de liberação

A release só deve receber tráfego real quando todos os itens abaixo estiverem verdadeiros:

- CI verde no SHA exato;
- checksum do bundle validado;
- backup realizado;
- `preflight` aprovado;
- migrations aplicadas sem erro;
- `/healthz` e `/readyz` aprovados;
- HTTPS e proxy validados;
- smoke test completo aprovado;
- primeiro Super Admin ativo;
- modelo de contrato padrão revisado e publicado;
- sender do Brevo validado;
- bucket privado e criptografado.
