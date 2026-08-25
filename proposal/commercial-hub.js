import {
  buildPublicPresentationUrl,
  supabase,
} from './supabase.js';

const hubState = {
  session: null,
  salesProfile: null,
  stats: [],
  presentation: null,
  version: null,
};

function appendStylesheet() {
  if (document.querySelector('link[data-commercial-hub]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './commercial-hub.css';
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
    return 'Nunca';
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
    return { label: 'Não aberta', className: '' };
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
  if (!stat) {
    return 'Sem dados de leitura';
  }

  return `${Number(stat.opens) || 0} abertura(s) · ${Number(stat.max_progress) || 0}% máximo · última ${formatDateTime(stat.last_opened_at)}`;
}

function ensureHubStructure() {
  const dashboard = document.getElementById('dashboardView');
  const adminHeader = dashboard?.querySelector('.admin-header');
  const dashboardGrid = dashboard?.querySelector('.dashboard-grid');
  const adminMain = document.querySelector('.admin-main');
  const proposalButton = document.getElementById('newProposalButton');

  if (!dashboard || !adminHeader || !dashboardGrid || !adminMain || document.getElementById('presentationList')) {
    return;
  }

  const eyebrow = adminHeader.querySelector('.admin-eyebrow');
  const title = adminHeader.querySelector('h1');
  const description = adminHeader.querySelector('p');

  if (eyebrow) eyebrow.textContent = 'HUB COMERCIAL';
  if (title) title.textContent = 'Apresentações e propostas';
  if (description) description.textContent = 'Gere links individuais, personalize o material e acompanhe abertura e leitura.';

  if (proposalButton) {
    proposalButton.textContent = 'Nova proposta';
    const actions = document.createElement('div');
    actions.className = 'hub-actions';
    proposalButton.parentNode.insertBefore(actions, proposalButton);
    actions.appendChild(proposalButton);

    const presentationButton = document.createElement('button');
    presentationButton.className = 'secondary-button';
    presentationButton.type = 'button';
    presentationButton.id = 'newPresentationButton';
    presentationButton.textContent = 'Nova apresentação';
    actions.prepend(presentationButton);
  }

  const summary = document.createElement('section');
  summary.className = 'hub-summary';
  summary.id = 'hubSummary';
  dashboard.insertBefore(summary, dashboardGrid);

  const presentationPanel = document.createElement('section');
  presentationPanel.className = 'panel hub-stats-panel';
  presentationPanel.innerHTML = `
    <header class="panel-header">
      <h2>Apresentações geradas</h2>
      <button class="link-button" type="button" id="refreshPresentationButton">Atualizar</button>
    </header>
    <div class="panel-body">
      <div id="presentationList" class="proposal-list"><div class="hub-empty">Carregando apresentações...</div></div>
    </div>
  `;
  dashboardGrid.prepend(presentationPanel);

  const editor = document.createElement('section');
  editor.className = 'hub-editor';
  editor.id = 'presentationEditorView';
  editor.hidden = true;
  editor.innerHTML = `
    <div class="hub-editor-shell">
      <div class="hub-editor-top">
        <div>
          <span class="hub-section-eyebrow">EDITOR DE APRESENTAÇÃO</span>
          <h1 id="presentationEditorTitle">Nova apresentação</h1>
          <p>O conteúdo institucional permanece padronizado; personalize somente o que fizer sentido para este envio.</p>
        </div>
        <button class="ghost-button" type="button" id="closePresentationEditor">Fechar</button>
      </div>

      <form id="presentationForm">
        <input id="presentationId" type="hidden" />
        <input id="presentationVersionId" type="hidden" />

        <section class="hub-editor-card">
          <header><h2>Identificação</h2></header>
          <div class="hub-editor-body">
            <div class="form-grid">
              <div class="form-field full">
                <label for="presentationTitle">Título interno</label>
                <input id="presentationTitle" value="Apresentação Institucional ViaGate" required />
              </div>
              <div class="form-field">
                <label for="presentationClientName">Empresa do cliente</label>
                <input id="presentationClientName" placeholder="Opcional" />
              </div>
              <div class="form-field">
                <label for="presentationContactName">Contato do cliente</label>
                <input id="presentationContactName" placeholder="Opcional" />
              </div>
              <div class="form-field full">
                <label for="presentationClientLogo">URL do logo do cliente</label>
                <input id="presentationClientLogo" type="url" placeholder="Opcional" />
              </div>
            </div>
            <div class="hub-client-note">Esses dados são opcionais. A apresentação pode continuar institucional e usar apenas o contato do vendedor.</div>
          </div>
        </section>

        <section class="hub-editor-card">
          <header><h2>Personalização deste link</h2></header>
          <div class="hub-editor-body">
            <div class="hub-check-grid">
              <label class="hub-check">
                <input id="presentationShowContact" type="checkbox" checked />
                <span><strong>Slide final do comercial</strong><small>Usa foto, nome, cargo, e-mail e telefone do perfil logado.</small></span>
              </label>
              <label class="hub-check">
                <input id="presentationShowClient" type="checkbox" />
                <span><strong>Identificação do cliente</strong><small>Permite mostrar empresa/contato neste link personalizado.</small></span>
              </label>
            </div>
          </div>
        </section>

        <div class="hub-editor-footer">
          <span id="presentationEditorVersion" class="admin-user">Rascunho ainda não salvo</span>
          <div class="hub-actions">
            <button class="ghost-button" id="copyPresentationLink" type="button" hidden>Copiar link</button>
            <button class="secondary-button" type="submit">Salvar rascunho</button>
            <button class="primary-button" id="publishPresentationButton" type="button">Publicar nova versão</button>
          </div>
        </div>
        <div class="message-box" id="presentationMessage" hidden></div>
      </form>
    </div>
  `;
  adminMain.appendChild(editor);
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
  document.getElementById('presentationEditorView').hidden = false;
}

function closePresentationEditor() {
  document.getElementById('presentationEditorView').hidden = true;
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
  document.getElementById('presentationEditorView').hidden = false;
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
    throw new Error('Preencha e salve seu perfil comercial antes de gerar uma apresentação.');
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

  const { data: latest } = await supabase
    .from('presentation_versions')
    .select('version_number')
    .eq('presentation_id', presentationId)
    .order('version_number', { ascending: false })
    .limit(1)
    .maybeSingle();

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
  hubState.version = { ...version, published_at: published?.published_at, public_token: published?.public_token };
  const copyButton = document.getElementById('copyPresentationLink');
  copyButton.hidden = false;
  copyButton.dataset.token = published?.public_token ?? '';
  document.getElementById('presentationEditorVersion').textContent = `Versão ${version.version_number} · publicada`;
  showMessage('Apresentação publicada. O link já pode ser compartilhado.', 'success');
  await refreshHub();
}

async function copyPresentationLink(token) {
  if (!token) return;
  await navigator.clipboard.writeText(buildPublicPresentationUrl(token));
}

async function loadStats() {
  const { data, error } = await supabase.rpc('get_my_shared_document_stats');
  hubState.stats = error || !Array.isArray(data) ? [] : data;
  renderSummary();
  decorateProposalRows();
}

function renderSummary() {
  const element = document.getElementById('hubSummary');
  if (!element) return;

  const published = hubState.stats.length;
  const opens = hubState.stats.reduce((total, item) => total + Number(item.opens || 0), 0);
  const read = hubState.stats.filter((item) => Number(item.completions) > 0).length;
  const active = hubState.stats.filter((item) => Number(item.opens) > 0 && Number(item.completions) === 0).length;

  element.innerHTML = `
    <article class="hub-metric"><small>Links publicados</small><strong>${published}</strong></article>
    <article class="hub-metric"><small>Aberturas</small><strong>${opens}</strong></article>
    <article class="hub-metric"><small>Materiais lidos</small><strong>${read}</strong></article>
    <article class="hub-metric"><small>Em acompanhamento</small><strong>${active}</strong></article>
  `;
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
    list.innerHTML = '<div class="hub-empty">Nenhuma apresentação gerada ainda.</div>';
    return;
  }

  list.innerHTML = data.map((presentation) => {
    const stat = getStat('presentation', presentation.id);
    const status = getReadingStatus(stat);
    return `
      <article class="hub-document-row">
        <div><strong>${escapeHtml(presentation.title)}</strong><small>Apresentação institucional personalizada</small></div>
        <div><span class="hub-reading-status ${status.className}">${status.label}</span><small>${buildStatsText(stat)}</small></div>
        <div><span>${presentation.status === 'published' ? `Publicada · v${presentation.current_version}` : 'Rascunho'}</span><small>${stat?.client_name ? escapeHtml(stat.client_name) : 'Sem cliente vinculado'}</small></div>
        <div class="hub-row-actions">
          ${stat?.public_token ? `<button class="link-button" type="button" data-copy-presentation="${stat.public_token}">Copiar link</button>` : ''}
          <button class="link-button" type="button" data-open-presentation="${presentation.id}">Abrir</button>
        </div>
      </article>
    `;
  }).join('');

  list.querySelectorAll('[data-open-presentation]').forEach((button) => {
    button.addEventListener('click', () => openExistingPresentation(button.dataset.openPresentation).catch(console.error));
  });

  list.querySelectorAll('[data-copy-presentation]').forEach((button) => {
    button.addEventListener('click', async () => {
      await copyPresentationLink(button.dataset.copyPresentation);
      button.textContent = 'Copiado';
      window.setTimeout(() => { button.textContent = 'Copiar link'; }, 1200);
    });
  });
}

function decorateProposalRows() {
  const list = document.getElementById('proposalList');
  if (!list) return;

  list.querySelectorAll('[data-open-proposal]').forEach((button) => {
    const row = button.closest('.proposal-row');
    const documentId = button.dataset.openProposal;
    const firstColumn = row?.querySelector('div');
    if (!firstColumn || firstColumn.querySelector('.hub-analytics-inline')) return;

    const stat = getStat('proposal', documentId);
    const status = getReadingStatus(stat);
    const info = document.createElement('small');
    info.className = 'hub-analytics-inline';
    info.textContent = `${status.label} · ${buildStatsText(stat)}`;
    firstColumn.appendChild(info);
  });
}

async function refreshHub() {
  if (!hubState.session?.user) return;
  await loadStats();
  await loadPresentations();
}

function bindHubEvents() {
  document.getElementById('newPresentationButton')?.addEventListener('click', openPresentationEditor);
  document.getElementById('closePresentationEditor')?.addEventListener('click', closePresentationEditor);
  document.getElementById('refreshPresentationButton')?.addEventListener('click', refreshHub);
  document.getElementById('presentationForm')?.addEventListener('submit', savePresentationDraft);
  document.getElementById('publishPresentationButton')?.addEventListener('click', publishPresentation);
  document.getElementById('copyPresentationLink')?.addEventListener('click', (event) => copyPresentationLink(event.currentTarget.dataset.token));

  const proposalList = document.getElementById('proposalList');
  if (proposalList) {
    new MutationObserver(decorateProposalRows).observe(proposalList, { childList: true, subtree: true });
  }
}

async function handleAuthenticatedSession(session) {
  hubState.session = session;
  if (!session?.user) return;

  try {
    await loadSalesProfile();
    await refreshHub();
  } catch (error) {
    console.error('Não foi possível carregar o hub comercial.', error);
  }
}

async function initializeCommercialHub() {
  appendStylesheet();
  ensureHubStructure();
  bindHubEvents();

  const { data } = await supabase.auth.getSession();
  await handleAuthenticatedSession(data.session);

  supabase.auth.onAuthStateChange((_event, session) => {
    handleAuthenticatedSession(session);
  });
}

initializeCommercialHub();
