import {
  buildPublicPresentationUrl,
  buildPublicProposalUrl,
  supabase,
} from './supabase.js';

const hubState = {
  session: null,
  salesProfile: null,
  stats: [],
  presentation: null,
  version: null,
  activeTab: 'overview',
};

function appendStylesheet() {
  if (document.querySelector('link[data-commercial-hub]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './commercial-hub.css?v=20260825-6';
  link.dataset.commercialHub = 'true';
  document.head.appendChild(link);
}

function escapeHtml(value) {
  const element = document.createElement('div');
  element.textContent = String(value ?? '');
  return element.innerHTML;
}

function formatDateTime(value) {
  if (!value) {
    return 'Nunca aberta';
  }

  return new Intl.DateTimeFormat('pt-BR', {
    dateStyle: 'short',
    timeStyle: 'short',
  }).format(new Date(value));
}

function getStat(kind, documentId) {
  return hubState.stats.find((item) => item.document_kind === kind && item.document_id === documentId) ?? null;
}

function getReadingStatus(stat) {
  if (!stat || Number(stat.opens) === 0) {
    return { label: 'Não aberta', className: 'unopened' };
  }

  if (Number(stat.completions) > 0) {
    return { label: 'Lida', className: 'read' };
  }

  if (Number(stat.starts) > 0 || Number(stat.max_progress) > 10) {
    return { label: 'Em leitura', className: 'progress' };
  }

  return { label: 'Aberta', className: 'opened' };
}

function buildStatsText(stat) {
  if (!stat || Number(stat.opens) === 0) {
    return 'O link ainda não foi aberto';
  }

  return `${Number(stat.opens)} abertura(s) · ${Number(stat.max_progress) || 0}% de progresso máximo`;
}

function getPublicUrl(kind, token) {
  if (!token) {
    return '';
  }

  return kind === 'presentation'
    ? buildPublicPresentationUrl(token)
    : buildPublicProposalUrl(token);
}

async function copyPublicLink(kind, token, button = null) {
  const url = getPublicUrl(kind, token);
  if (!url) {
    return;
  }

  try {
    await navigator.clipboard.writeText(url);

    if (button) {
      const original = button.textContent;
      button.textContent = 'Copiado';
      window.setTimeout(() => {
        button.textContent = original;
      }, 1200);
    }
  } catch {
    window.prompt('Copie o link:', url);
  }
}

function ensureProposalPricingModels() {
  const select = document.getElementById('pricingModel');
  if (!select || select.querySelector('option[value="item_and_bundle"]')) {
    return;
  }

  const customOption = select.querySelector('option[value="custom"]');
  const combinedOption = document.createElement('option');
  combinedOption.value = 'item_and_bundle';
  combinedOption.textContent = 'Análise por item + conjunto';

  select.insertBefore(combinedOption, customOption ?? null);

  const perItem = select.querySelector('option[value="per_item"]');
  const bundle = select.querySelector('option[value="bundle"]');

  if (perItem) perItem.textContent = 'Análise por item';
  if (bundle) bundle.textContent = 'Análise por conjunto';
  if (customOption) customOption.textContent = 'Condições específicas';
}

function createHubHeader(title, description, actions = '') {
  return `
    <header class="hub-view-header">
      <div>
        <h2>${escapeHtml(title)}</h2>
        ${description ? `<p>${escapeHtml(description)}</p>` : ''}
      </div>
      ${actions}
    </header>
  `;
}

function ensureHubStructure() {
  const dashboard = document.getElementById('dashboardView');
  const adminHeader = dashboard?.querySelector('.admin-header');
  const dashboardGrid = dashboard?.querySelector('.dashboard-grid');
  const proposalList = document.getElementById('proposalList');
  const profileForm = document.getElementById('salesProfileForm');
  const proposalButton = document.getElementById('newProposalButton');
  const adminMain = document.querySelector('.admin-main');

  if (!dashboard || !adminHeader || !dashboardGrid || !proposalList || !profileForm || !proposalButton || !adminMain) {
    return;
  }

  if (dashboard.dataset.hubReady === 'true') {
    return;
  }

  dashboard.dataset.hubReady = 'true';

  const proposalPanel = proposalList.closest('.panel');
  const profilePanel = profileForm.closest('.panel');
  if (!proposalPanel || !profilePanel) {
    return;
  }

  const eyebrow = adminHeader.querySelector('.admin-eyebrow');
  const title = adminHeader.querySelector('h1');
  const description = adminHeader.querySelector('p');

  if (eyebrow) eyebrow.textContent = 'HUB COMERCIAL';
  if (title) title.textContent = 'Gestão comercial';
  if (description) description.textContent = 'Crie materiais, publique links individuais e acompanhe a leitura de cada envio.';

  const originalActions = Array.from(adminHeader.children).find((child) => child.contains(proposalButton));
  if (originalActions && originalActions !== adminHeader.firstElementChild) {
    originalActions.remove();
  }

  const nav = document.createElement('nav');
  nav.className = 'hub-nav';
  nav.setAttribute('aria-label', 'Seções do Hub Comercial');
  nav.innerHTML = `
    <button type="button" data-hub-tab="overview">Visão geral</button>
    <button type="button" data-hub-tab="presentations">Apresentações</button>
    <button type="button" data-hub-tab="proposals">Propostas</button>
    <button type="button" data-hub-tab="profile">Meu perfil</button>
  `;

  const views = document.createElement('div');
  views.className = 'hub-views';

  const overviewView = document.createElement('section');
  overviewView.className = 'hub-view';
  overviewView.dataset.hubView = 'overview';
  overviewView.innerHTML = `
    ${createHubHeader('Visão geral', 'Acompanhe o que aconteceu depois que cada link foi enviado.')}
    <section class="hub-summary" id="hubSummary"></section>
    <section class="hub-panel">
      <header class="hub-panel-header">
        <div>
          <h3>Acompanhamento dos materiais publicados</h3>
          <p>Abertura, progresso de leitura e último acesso por link.</p>
        </div>
        <button class="link-button" type="button" id="refreshHubButton">Atualizar</button>
      </header>
      <div class="hub-tracking-table" id="hubTrackingTable"></div>
    </section>
  `;

  const presentationsView = document.createElement('section');
  presentationsView.className = 'hub-view';
  presentationsView.dataset.hubView = 'presentations';
  presentationsView.innerHTML = `
    ${createHubHeader(
      'Apresentações',
      'Gere um link da apresentação institucional com identificação do cliente e contato comercial quando necessário.',
      '<button class="primary-button" type="button" id="newPresentationButton">Nova apresentação</button>',
    )}
    <section class="hub-panel">
      <header class="hub-panel-header">
        <div><h3>Apresentações geradas</h3></div>
        <button class="link-button" type="button" id="refreshPresentationButton">Atualizar</button>
      </header>
      <div id="presentationList" class="hub-document-list"></div>
    </section>
  `;

  const proposalsView = document.createElement('section');
  proposalsView.className = 'hub-view';
  proposalsView.dataset.hubView = 'proposals';
  proposalsView.innerHTML = createHubHeader(
    'Propostas comerciais',
    'Crie, versione, publique e acompanhe propostas por cliente.',
  );

  const proposalActions = document.createElement('div');
  proposalActions.className = 'hub-view-actions';
  proposalActions.appendChild(proposalButton);
  proposalsView.querySelector('.hub-view-header')?.appendChild(proposalActions);
  proposalsView.appendChild(proposalPanel);

  const profileView = document.createElement('section');
  profileView.className = 'hub-view';
  profileView.dataset.hubView = 'profile';
  profileView.innerHTML = createHubHeader(
    'Meu perfil comercial',
    'Estes dados são usados no contato final das apresentações e propostas geradas por este usuário.',
  );
  profileView.appendChild(profilePanel);

  views.append(overviewView, presentationsView, proposalsView, profileView);
  dashboardGrid.replaceWith(nav, views);

  const editor = document.createElement('section');
  editor.className = 'hub-editor';
  editor.id = 'presentationEditorView';
  editor.hidden = true;
  editor.innerHTML = `
    <div class="hub-editor-shell">
      <div class="hub-editor-top">
        <div>
          <span class="hub-section-eyebrow">APRESENTAÇÃO INSTITUCIONAL</span>
          <h1 id="presentationEditorTitle">Nova apresentação</h1>
          <p>O conteúdo institucional permanece o mesmo. Configure somente a identificação deste envio.</p>
        </div>
        <button class="ghost-button" type="button" id="closePresentationEditor">Voltar</button>
      </div>

      <form id="presentationForm">
        <input id="presentationId" type="hidden" />
        <input id="presentationVersionId" type="hidden" />

        <section class="hub-editor-card">
          <header><h2>Identificação do envio</h2></header>
          <div class="hub-editor-body">
            <div class="form-grid">
              <div class="form-field full">
                <label for="presentationTitle">Título interno</label>
                <input id="presentationTitle" value="Apresentação Institucional ViaGate" required />
              </div>
              <div class="form-field">
                <label for="presentationClientName">Empresa do cliente</label>
                <input id="presentationClientName" />
              </div>
              <div class="form-field">
                <label for="presentationContactName">Contato do cliente</label>
                <input id="presentationContactName" />
              </div>
              <div class="form-field full">
                <label for="presentationClientLogo">Logo do cliente</label>
                <input id="presentationClientLogo" type="url" />
              </div>
            </div>
          </div>
        </section>

        <section class="hub-editor-card">
          <header><h2>Conteúdo personalizado</h2></header>
          <div class="hub-editor-body">
            <div class="hub-check-grid">
              <label class="hub-check">
                <input id="presentationShowContact" type="checkbox" checked />
                <span><strong>Exibir contato comercial no final</strong><small>Nome, cargo, foto, telefone e e-mail do perfil logado.</small></span>
              </label>
              <label class="hub-check">
                <input id="presentationShowClient" type="checkbox" />
                <span><strong>Identificar o cliente na apresentação</strong><small>Exibe empresa, contato e logo cadastrados acima.</small></span>
              </label>
            </div>
          </div>
        </section>

        <div class="hub-editor-footer">
          <span id="presentationEditorVersion" class="admin-user">Rascunho ainda não salvo</span>
          <div class="hub-actions">
            <button class="ghost-button" id="copyPresentationLink" type="button" hidden>Copiar link</button>
            <button class="secondary-button" type="submit">Salvar rascunho</button>
            <button class="primary-button" id="publishPresentationButton" type="button">Publicar</button>
          </div>
        </div>
        <div class="message-box" id="presentationMessage" hidden></div>
      </form>
    </div>
  `;

  adminMain.appendChild(editor);
  ensureProposalPricingModels();
}

function activateHubTab(tabName) {
  hubState.activeTab = tabName;

  document.querySelectorAll('[data-hub-tab]').forEach((button) => {
    const active = button.dataset.hubTab === tabName;
    button.classList.toggle('active', active);
    button.setAttribute('aria-current', active ? 'page' : 'false');
  });

  document.querySelectorAll('[data-hub-view]').forEach((view) => {
    view.hidden = view.dataset.hubView !== tabName;
  });
}

function showMessage(message, type = '') {
  const element = document.getElementById('presentationMessage');
  if (!element) return;

  element.textContent = message;
  element.className = `message-box${type ? ` ${type}` : ''}`;
  element.hidden = false;
}

function hideMessage() {
  const element = document.getElementById('presentationMessage');
  if (element) element.hidden = true;
}

async function loadSalesProfile() {
  if (!hubState.session?.user) return null;

  const { data, error } = await supabase
    .from('salespeople')
    .select('*')
    .eq('auth_user_id', hubState.session.user.id)
    .maybeSingle();

  if (error) throw error;

  hubState.salesProfile = data;
  return data;
}

function collectPresentationContent() {
  const salesperson = hubState.salesProfile ?? {};

  return {
    salesperson: {
      name: salesperson.name ?? '',
      role: salesperson.role ?? '',
      email: salesperson.email ?? '',
      phone: salesperson.phone ?? '',
      whatsapp: String(salesperson.phone ?? '').replace(/\D/g, ''),
      photo_url: salesperson.photo_url ?? '',
    },
    client: {
      company_name: document.getElementById('presentationClientName')?.value?.trim() ?? '',
      contact_name: document.getElementById('presentationContactName')?.value?.trim() ?? '',
      logo_url: document.getElementById('presentationClientLogo')?.value?.trim() ?? '',
    },
    settings: {
      show_contact_slide: Boolean(document.getElementById('presentationShowContact')?.checked),
      show_client_identity: Boolean(document.getElementById('presentationShowClient')?.checked),
    },
  };
}

function resetPresentationEditor() {
  hubState.presentation = null;
  hubState.version = null;

  document.getElementById('presentationForm')?.reset();
  document.getElementById('presentationId').value = '';
  document.getElementById('presentationVersionId').value = '';
  document.getElementById('presentationTitle').value = 'Apresentação Institucional ViaGate';
  document.getElementById('presentationShowContact').checked = true;
  document.getElementById('presentationShowClient').checked = false;
  document.getElementById('presentationEditorTitle').textContent = 'Nova apresentação';
  document.getElementById('presentationEditorVersion').textContent = 'Rascunho ainda não salvo';
  document.getElementById('copyPresentationLink').hidden = true;
  hideMessage();
}

function openPresentationEditor() {
  resetPresentationEditor();
  document.getElementById('dashboardView').hidden = true;
  document.getElementById('editorView').hidden = true;
  document.getElementById('presentationEditorView').hidden = false;
  window.scrollTo({ top: 0, behavior: 'auto' });
}

function closePresentationEditor() {
  document.getElementById('presentationEditorView').hidden = true;
  document.getElementById('dashboardView').hidden = false;
  activateHubTab('presentations');
}

async function openExistingPresentation(presentationId) {
  const { data: presentation, error: presentationError } = await supabase
    .from('presentations')
    .select('*')
    .eq('id', presentationId)
    .single();

  if (presentationError) throw presentationError;

  const { data: version, error: versionError } = await supabase
    .from('presentation_versions')
    .select('*')
    .eq('presentation_id', presentationId)
    .order('version_number', { ascending: false })
    .limit(1)
    .maybeSingle();

  if (versionError) throw versionError;

  hubState.presentation = presentation;
  hubState.version = version;
  const content = version?.content ?? {};

  document.getElementById('presentationId').value = presentation.id;
  document.getElementById('presentationVersionId').value = version?.id ?? '';
  document.getElementById('presentationTitle').value = presentation.title ?? '';
  document.getElementById('presentationClientName').value = content.client?.company_name ?? '';
  document.getElementById('presentationContactName').value = content.client?.contact_name ?? '';
  document.getElementById('presentationClientLogo').value = content.client?.logo_url ?? '';
  document.getElementById('presentationShowContact').checked = content.settings?.show_contact_slide !== false;
  document.getElementById('presentationShowClient').checked = Boolean(content.settings?.show_client_identity);
  document.getElementById('presentationEditorTitle').textContent = presentation.title;
  document.getElementById('presentationEditorVersion').textContent = version
    ? `Versão ${version.version_number} · ${version.published_at ? 'publicada' : 'rascunho'}`
    : 'Sem versão';

  const copyButton = document.getElementById('copyPresentationLink');
  copyButton.hidden = !version?.published_at;
  copyButton.dataset.token = version?.published_at ? version.public_token : '';

  document.getElementById('dashboardView').hidden = true;
  document.getElementById('editorView').hidden = true;
  document.getElementById('presentationEditorView').hidden = false;
  window.scrollTo({ top: 0, behavior: 'auto' });
}

async function ensurePresentation() {
  const title = document.getElementById('presentationTitle').value.trim();

  if (hubState.presentation?.id) {
    const { data, error } = await supabase
      .from('presentations')
      .update({ title })
      .eq('id', hubState.presentation.id)
      .select()
      .single();

    if (error) throw error;
    hubState.presentation = data;
    return data;
  }

  if (!hubState.salesProfile?.id) {
    throw new Error('Salve seu perfil comercial antes de gerar uma apresentação.');
  }

  const { data, error } = await supabase
    .from('presentations')
    .insert({
      salesperson_id: hubState.salesProfile.id,
      title,
      created_by: hubState.session.user.id,
    })
    .select()
    .single();

  if (error) throw error;

  hubState.presentation = data;
  return data;
}

async function ensureDraftVersion(presentationId) {
  if (hubState.version && !hubState.version.published_at) {
    return hubState.version;
  }

  const { data: latest, error: latestError } = await supabase
    .from('presentation_versions')
    .select('version_number')
    .eq('presentation_id', presentationId)
    .order('version_number', { ascending: false })
    .limit(1)
    .maybeSingle();

  if (latestError) throw latestError;

  const nextVersion = Number(latest?.version_number ?? 0) + 1;
  const { data, error } = await supabase
    .from('presentation_versions')
    .insert({
      presentation_id: presentationId,
      version_number: nextVersion,
      content: collectPresentationContent(),
      created_by: hubState.session.user.id,
    })
    .select()
    .single();

  if (error) throw error;

  hubState.version = data;
  return data;
}

async function savePresentationDraft(event) {
  event?.preventDefault();
  hideMessage();

  try {
    await loadSalesProfile();
    const presentation = await ensurePresentation();
    const version = await ensureDraftVersion(presentation.id);

    const { data, error } = await supabase
      .from('presentation_versions')
      .update({ content: collectPresentationContent() })
      .eq('id', version.id)
      .select()
      .single();

    if (error) throw error;

    hubState.version = data;
    document.getElementById('presentationId').value = presentation.id;
    document.getElementById('presentationVersionId').value = data.id;
    document.getElementById('presentationEditorVersion').textContent = `Versão ${data.version_number} · rascunho`;
    showMessage('Rascunho salvo.', 'success');
    return data;
  } catch (error) {
    showMessage(error.message || 'Não foi possível salvar a apresentação.', 'error');
    return null;
  }
}

async function publishPresentation() {
  const version = await savePresentationDraft();
  if (!version) return;

  const { data, error } = await supabase.rpc('publish_presentation_version', {
    target_version_id: version.id,
  });

  if (error) {
    showMessage('Não foi possível publicar a apresentação.', 'error');
    return;
  }

  const published = Array.isArray(data) ? data[0] : data;
  hubState.version = {
    ...version,
    published_at: published?.published_at,
    public_token: published?.public_token,
  };

  const copyButton = document.getElementById('copyPresentationLink');
  copyButton.hidden = false;
  copyButton.dataset.token = published?.public_token ?? '';
  document.getElementById('presentationEditorVersion').textContent = `Versão ${version.version_number} · publicada`;
  showMessage('Apresentação publicada. O link está disponível para envio.', 'success');
  await refreshHub();
}

async function loadStats() {
  const { data, error } = await supabase.rpc('get_my_shared_document_stats');
  hubState.stats = error || !Array.isArray(data) ? [] : data;
  renderSummary();
  renderTrackingTable();
  decorateProposalRows();
}

function renderSummary() {
  const element = document.getElementById('hubSummary');
  if (!element) return;

  const published = hubState.stats.length;
  const opens = hubState.stats.reduce((total, item) => total + Number(item.opens || 0), 0);
  const read = hubState.stats.filter((item) => Number(item.completions) > 0).length;
  const active = hubState.stats.filter((item) => Number(item.starts) > 0 && Number(item.completions) === 0).length;

  element.innerHTML = `
    <article class="hub-metric"><small>Links publicados</small><strong>${published}</strong></article>
    <article class="hub-metric"><small>Aberturas</small><strong>${opens}</strong></article>
    <article class="hub-metric"><small>Lidos até o final</small><strong>${read}</strong></article>
    <article class="hub-metric"><small>Em leitura</small><strong>${active}</strong></article>
  `;
}

function renderTrackingTable() {
  const element = document.getElementById('hubTrackingTable');
  if (!element) return;

  if (!hubState.stats.length) {
    element.innerHTML = '<div class="hub-empty">Nenhum material publicado ainda.</div>';
    return;
  }

  element.innerHTML = `
    <div class="hub-tracking-head">
      <span>Material</span>
      <span>Leitura</span>
      <span>Aberturas</span>
      <span>Progresso</span>
      <span>Última abertura</span>
      <span></span>
    </div>
    ${hubState.stats.map((stat) => {
      const status = getReadingStatus(stat);
      const typeLabel = stat.document_kind === 'presentation' ? 'Apresentação' : 'Proposta';
      const title = stat.client_name || stat.title || typeLabel;

      return `
        <article class="hub-tracking-row">
          <div><strong>${escapeHtml(title)}</strong><small>${typeLabel} · versão ${Number(stat.version_number) || 1}</small></div>
          <div><span class="hub-reading-status ${status.className}">${status.label}</span></div>
          <div><strong>${Number(stat.opens) || 0}</strong></div>
          <div><strong>${Number(stat.max_progress) || 0}%</strong></div>
          <div><span>${escapeHtml(formatDateTime(stat.last_opened_at))}</span></div>
          <div><button class="link-button" type="button" data-copy-tracking-kind="${stat.document_kind}" data-copy-tracking-token="${stat.public_token}">Copiar link</button></div>
        </article>
      `;
    }).join('')}
  `;

  element.querySelectorAll('[data-copy-tracking-token]').forEach((button) => {
    button.addEventListener('click', () => copyPublicLink(
      button.dataset.copyTrackingKind,
      button.dataset.copyTrackingToken,
      button,
    ));
  });
}

async function loadPresentations() {
  const list = document.getElementById('presentationList');
  if (!list || !hubState.session?.user) return;

  const { data, error } = await supabase
    .from('presentations')
    .select('*')
    .order('updated_at', { ascending: false });

  if (error) {
    list.innerHTML = '<div class="hub-empty">Não foi possível carregar as apresentações.</div>';
    return;
  }

  if (!data?.length) {
    list.innerHTML = '<div class="hub-empty">Nenhuma apresentação criada.</div>';
    return;
  }

  list.innerHTML = data.map((presentation) => {
    const stat = getStat('presentation', presentation.id);
    const status = getReadingStatus(stat);

    return `
      <article class="hub-document-row">
        <div class="hub-document-main">
          <strong>${escapeHtml(presentation.title)}</strong>
          <small>${presentation.status === 'published' ? `Publicada · versão ${presentation.current_version}` : 'Rascunho'}</small>
        </div>
        <div class="hub-document-reading">
          <span class="hub-reading-status ${status.className}">${status.label}</span>
          <small>${escapeHtml(buildStatsText(stat))}</small>
        </div>
        <div class="hub-document-last">
          <strong>${escapeHtml(formatDateTime(stat?.last_opened_at))}</strong>
          <small>Última abertura</small>
        </div>
        <div class="hub-row-actions">
          ${stat?.public_token ? `<button class="link-button" type="button" data-copy-presentation="${stat.public_token}">Copiar link</button>` : ''}
          <button class="link-button" type="button" data-open-presentation="${presentation.id}">Editar</button>
        </div>
      </article>
    `;
  }).join('');

  list.querySelectorAll('[data-open-presentation]').forEach((button) => {
    button.addEventListener('click', () => openExistingPresentation(button.dataset.openPresentation).catch(console.error));
  });

  list.querySelectorAll('[data-copy-presentation]').forEach((button) => {
    button.addEventListener('click', () => copyPublicLink('presentation', button.dataset.copyPresentation, button));
  });
}

function decorateProposalRows() {
  const list = document.getElementById('proposalList');
  if (!list) return;

  list.querySelectorAll('[data-open-proposal]').forEach((button) => {
    const row = button.closest('.proposal-row');
    const documentId = button.dataset.openProposal;
    if (!row) return;

    row.classList.add('hub-proposal-row');
    row.querySelector('.hub-proposal-tracking')?.remove();
    row.querySelector('[data-copy-proposal-tracking]')?.remove();

    const stat = getStat('proposal', documentId);
    const status = getReadingStatus(stat);
    const tracking = document.createElement('div');
    tracking.className = 'hub-proposal-tracking';
    tracking.innerHTML = `
      <span class="hub-reading-status ${status.className}">${status.label}</span>
      <small>${escapeHtml(buildStatsText(stat))}</small>
      <small>${escapeHtml(formatDateTime(stat?.last_opened_at))}</small>
    `;

    const actionCell = row.lastElementChild;
    row.insertBefore(tracking, actionCell);

    if (stat?.public_token && actionCell) {
      const copyButton = document.createElement('button');
      copyButton.className = 'link-button';
      copyButton.type = 'button';
      copyButton.dataset.copyProposalTracking = stat.public_token;
      copyButton.textContent = 'Copiar link';
      copyButton.addEventListener('click', () => copyPublicLink('proposal', stat.public_token, copyButton));
      actionCell.prepend(copyButton);
    }

    const statusBadge = row.querySelector('.status-badge');
    if (statusBadge) {
      const labels = {
        draft: 'Rascunho',
        published: 'Publicada',
        accepted: 'Aceita',
        declined: 'Recusada',
        expired: 'Expirada',
      };
      statusBadge.textContent = labels[statusBadge.textContent.trim()] ?? statusBadge.textContent;
    }
  });
}

async function refreshHub() {
  if (!hubState.session?.user) return;

  await loadStats();
  await loadPresentations();
  decorateProposalRows();
}

function bindHubEvents() {
  document.querySelectorAll('[data-hub-tab]').forEach((button) => {
    button.addEventListener('click', () => activateHubTab(button.dataset.hubTab));
  });

  document.getElementById('newPresentationButton')?.addEventListener('click', openPresentationEditor);
  document.getElementById('closePresentationEditor')?.addEventListener('click', closePresentationEditor);
  document.getElementById('refreshPresentationButton')?.addEventListener('click', refreshHub);
  document.getElementById('refreshHubButton')?.addEventListener('click', refreshHub);
  document.getElementById('presentationForm')?.addEventListener('submit', savePresentationDraft);
  document.getElementById('publishPresentationButton')?.addEventListener('click', publishPresentation);
  document.getElementById('copyPresentationLink')?.addEventListener('click', (event) => {
    copyPublicLink('presentation', event.currentTarget.dataset.token, event.currentTarget);
  });

  document.getElementById('newProposalButton')?.addEventListener('click', () => {
    hubState.activeTab = 'proposals';
  });

  document.getElementById('backToDashboardButton')?.addEventListener('click', () => {
    window.setTimeout(() => activateHubTab('proposals'), 0);
  });

  const proposalList = document.getElementById('proposalList');
  if (proposalList) {
    new MutationObserver(() => {
      window.setTimeout(decorateProposalRows, 0);
    }).observe(proposalList, { childList: true, subtree: true });
  }
}

async function handleAuthenticatedSession(session) {
  hubState.session = session;
  if (!session?.user) return;

  try {
    await loadSalesProfile();
    await refreshHub();
    activateHubTab(hubState.activeTab);
  } catch (error) {
    console.error('Não foi possível carregar o hub comercial.', error);
  }
}

async function initializeCommercialHub() {
  appendStylesheet();
  ensureHubStructure();
  bindHubEvents();
  activateHubTab('overview');

  const { data } = await supabase.auth.getSession();
  await handleAuthenticatedSession(data.session);

  supabase.auth.onAuthStateChange((_event, session) => {
    handleAuthenticatedSession(session);
  });
}

initializeCommercialHub();
