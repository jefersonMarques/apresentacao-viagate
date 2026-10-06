-- Upgrade only the untouched seeded proposal-share template.
-- User-customized templates are preserved.

update email_templates
set
  description='Modelo comercial ViaGate com produtos dinâmicos da versão publicada da proposta.',
  subject_template='Proposta comercial — {client.display_name}',
  html_template=$email$
<table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="width:100%;max-width:680px;margin:0 auto;background:#ffffff;border:1px solid #d9e2e8;font-family:Arial,Helvetica,sans-serif;color:#102637">
  <tr>
    <td style="background:#071827;padding:22px 28px;border-bottom:4px solid #ff6b18">
      <img src="{brand.logo_url}" alt="ViaGate" style="display:block;max-width:150px;max-height:42px;border:0;background:#ffffff;padding:6px 8px">
      <div style="margin-top:12px;color:#ffffff;font-size:22px;font-weight:700">Proposta comercial</div>
    </td>
  </tr>
  <tr>
    <td style="padding:30px 28px">
      <p style="margin:0 0 18px;font-size:15px;line-height:1.6">{contact.greeting}</p>
      <p style="margin:0 0 14px;font-size:14px;line-height:1.7">Conforme conversamos, encaminho nossa proposta comercial para análise da <strong>{client.display_name}</strong>.</p>

      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="margin:0 0 24px;background:#f7f9fa;border-left:4px solid #ff6b18">
        <tr>
          <td style="padding:16px 18px">
            <div style="font-size:14px;font-weight:700;color:#102637">{proposal.title}</div>
            <div style="margin-top:5px;font-size:12px;color:#6f8290">Validade: {proposal.valid_until}</div>
          </td>
        </tr>
      </table>

      <div style="margin:0 0 10px;font-size:14px;font-weight:700;color:#102637">Produtos e serviços contemplados</div>
      <div style="margin:0 0 26px">{proposal.products_html}</div>

      <div style="margin:0 0 12px;font-size:14px;font-weight:700;color:#102637">Principais diferenciais ViaGate</div>
      <div style="margin:0 0 26px;font-size:13px;line-height:1.8;color:#536875">
        &#8226;&nbsp; Pesquisas criminais de proprietários de veículos, pessoa física e jurídica;<br>
        &#8226;&nbsp; Biometria facial com prova de vida, sem necessidade de cópia de documentos;<br>
        &#8226;&nbsp; Referenciamento junto às principais seguradoras do mercado;<br>
        &#8226;&nbsp; Aplicação ao mercado agro, operações com motoristas terceiros e outros perfis profissionais;<br>
        &#8226;&nbsp; Recursos para redução de riscos e prevenção de fraudes;<br>
        &#8226;&nbsp; Três formas de captura de biometria facial: link, web e aplicativo;<br>
        &#8226;&nbsp; Sistema white label com painel gerencial e relatórios;<br>
        &#8226;&nbsp; Treinamento por vídeo e suporte após a implantação;<br>
        &#8226;&nbsp; Sem multa por cancelamento e sem fatura mínima mensal.
      </div>

      <p style="margin:0 0 18px;font-size:14px;line-height:1.7">Todos os produtos, valores e condições comerciais podem ser consultados diretamente na proposta.</p>

      <table role="presentation" cellspacing="0" cellpadding="0" style="margin:0 0 24px">
        <tr>
          <td style="background:#ff6b18">
            <a href="{proposal.url}" style="display:inline-block;padding:14px 22px;color:#ffffff;text-decoration:none;font-size:13px;font-weight:700">Abrir proposta comercial</a>
          </td>
        </tr>
      </table>

      <p style="margin:0 0 24px;color:#82919a;font-size:11px;line-height:1.6">Link direto:<br><a href="{proposal.url}" style="color:#536875;word-break:break-all">{proposal.url}</a></p>

      <div style="margin:0 0 10px;font-size:14px;font-weight:700;color:#102637">Após a aprovação</div>
      <p style="margin:0 0 8px;font-size:13px;line-height:1.7;color:#536875">Solicitaremos os dados dos usuários responsáveis pela utilização do sistema e do responsável financeiro.</p>
      <p style="margin:0 0 8px;font-size:13px;line-height:1.7;color:#536875">Quando aplicável aos produtos contratados, também serão solicitadas as principais mercadorias transportadas e a cópia da apólice de seguros.</p>
      <p style="margin:0 0 22px;font-size:12px;line-height:1.7;color:#82919a">Exigência de apólice nesta proposta: <strong>{proposal.requires_policy}</strong>.</p>

      <table role="presentation" width="100%" cellspacing="0" cellpadding="0" style="margin:0 0 24px">
        <tr>
          <td style="padding:15px 18px;background:#fff7f2;border:1px solid #ffd9c2;font-size:13px;line-height:1.7;color:#7a3a15">
            <strong>Prazo de implantação:</strong> até 48 horas úteis após a assinatura dos contratos e o recebimento das informações necessárias.
          </td>
        </tr>
      </table>

      <p style="margin:0;font-size:14px;line-height:1.7">Mais uma vez, agradeço pela oportunidade e fico à disposição para quaisquer esclarecimentos.</p>
    </td>
  </tr>
  <tr>
    <td style="padding:22px 28px;border-top:1px solid #e2e8ec;background:#f8fafb">
      <div style="color:#102637;font-size:14px;font-weight:700">{salesperson.name}</div>
      <div style="margin-top:3px;color:#6f8290;font-size:12px">{salesperson.job_title}</div>
      <div style="margin-top:7px;color:#536875;font-size:12px;line-height:1.6">{salesperson.phone}<br>{salesperson.email}</div>
    </td>
  </tr>
</table>
$email$,
  text_template=$text$
{contact.greeting}

Conforme conversamos, encaminho nossa proposta comercial para análise da {client.display_name}.

{proposal.title}
Validade: {proposal.valid_until}

Produtos e serviços contemplados:
{proposal.products_text}

Todos os produtos, valores e condições comerciais estão disponíveis em:
{proposal.url}

Após a aprovação, solicitaremos os dados dos usuários responsáveis pela utilização do sistema e do responsável financeiro.
Quando aplicável aos produtos contratados, também serão solicitadas as principais mercadorias transportadas e a cópia da apólice de seguros.

Exigência de apólice nesta proposta: {proposal.requires_policy}.

Prazo de implantação: até 48 horas úteis após a assinatura dos contratos e o recebimento das informações necessárias.

Mais uma vez, agradeço pela oportunidade e fico à disposição.

{salesperson.name}
{salesperson.job_title}
{salesperson.phone}
{salesperson.email}
$text$,
  updated_at=now()
where purpose='proposal_share'
  and name='Proposta comercial ViaGate'
  and created_by is null
  and updated_by is null;
