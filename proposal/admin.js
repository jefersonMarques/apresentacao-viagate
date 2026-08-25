import {
  buildPublicProposalUrl,
  formatDate,
  hasSupabaseConfiguration,
  supabase,
} from './supabase.js';

const adminState = {
  session: null,
  salesProfile: null,
  loadedProposal: null,
  loadedVersion: null,
  publicToken: null,
};

const elements = {
  authView: document.getElementById('authView'),
  adminView: document.getElementById('adminView'),
  loginCard: document.getElementById('loginCard'),
  recoveryCard: document.getElementById('recoveryCard'),
  passwordCard: document.getElementById('passwordCard'),
  loginForm: document.getElementById('loginForm'),
  recoveryForm: document.getElementById('recoveryForm'),
  passwordForm: document.getElementById('passwordForm'),
  loginMessage: document.getElementById('loginMessage'),
  recoveryMessage: document.getElementById('recoveryMessage'),
  passwordMessage: document.getElementById('passwordMessage'),
  adminUserEmail: document.getElementById('adminUserEmail'),
  logoutButton: document.getElementById('logoutButton'),
  dashboardView: document.getElementById('dashboardView'),
  editorView: document.getElementById('editorView'),
  proposalList: document.getElementById('proposalList'),
  refreshButton: document.getElementById('refreshButton'),
  newProposalButton: document.getElementById('newProposalButton'),
  backToDashboardButton: document.getElementById('backToDashboardButton'),
  salesProfileForm: document.getElementById('salesProfileForm'),
  salesProfileMessage: document.getElementById('salesProfileMessage'),
  proposalForm: document.getElementById('proposalForm'),
  proposalMessage: document.getElementById('proposalMessage'),
  editorTitle: document.getElementById('editorTitle'),
  editorVersionLabel: document.getElementById('editorVersionLabel'),
  pricingItemsBody: document.getElementById('pricingItemsBody'),
  addPricingItemButton: document.getElementById('addPricingItemButton'),
  publishProposalButton: document.getElementById('publishProposalButton'),
  copyProposalLinkButton: document.getElementById('copyProposalLinkButton'),
};

function showMessage(element, message, type = '') {
  if (!element) {
    return;
  }

  element.textContent = message;
  element.className = `message-box${type ? ` ${type}` : ''}`;
  element.hidden = false;
}

function hideMessage(element) {
  if (element) {
    element.hidden = true;
  }
}

function showAuthCard(card) {
  [elements.loginCard, elements.recoveryCard, elements.passwordCard].forEach((item) => {
    item.hidden = item !== card;
  });
}

function setAuthenticatedView(session) {
  adminState.session = session;
  const authenticated = Boolean(session?.user);
  elements.authView.hidden = authenticated;
  elements.adminView.hidden = !authenticated;

  if (authenticated) {
    elements.adminUserEmail.textContent = session.user.email ?? '';
  }
}

function splitLines(value) {
  return String(value ?? '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);
}

function setInputValue(id, value) {
  const input = document.getElementById(id);
  if (input) {
    input.value = value ?? '';
  }
}

function getInputValue(id) {
  return document.getElementById(id)?.value?.trim() ?? '';
}

function createPricingRow(item = {}) {
  const row = document.createElement('tr');
  row.innerHTML = `
    <td><input data-pricing-label value="${escapeAttribute(item.label ?? '')}" placeholder="Cargo Score" required /></td>
    <td><input data-pricing-unit value="${escapeAttribute(item.unit ?? '')}" placeholder="análise" /></td>
    <td><input data-pricing-price type="number" min="0" step="0.01" value="${escapeAttribute(item.price ?? '')}" /></td>
    <td>
      <select data-pricing-optional>
        <option value="false" ${item.is_optional ? '' : 'selected'}>Não</option>
        <option value="true" ${item.is_optional ? 'selected' : ''}>Sim</option>
      </select>
    </td>
    <td><button class="link-button" type="button" data-remove-pricing-item>Remover</button></td>
  `;

  row.querySelector('[data-remove-pricing-item]')?.addEventListener('click', () => row.remove());
  elements.pricingItemsBody.appendChild(row);
}

function escapeAttribute(value) {
  return String(value)
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function collectPricingItems() {
  return Array.from(elements.pricingItemsBody.querySelectorAll('tr')).map((row, index) => ({
    label: row.querySelector('[data-pricing-label]')?.value?.trim() ?? '',
    unit: row.querySelector('[data-pricing-unit]')?.value?.trim() ?? '',
    price: Number(row.querySelector('[data-pricing-price]')?.value ?? 0),
    is_optional: row.querySelector('[data-pricing-optional]')?.value === 'true',
    sort_order: index,
  })).filter((item) => item.label);
}

function collectProposalContent() {
  return {
    client: {
      company_name: getInputValue('companyName'),
      logo_url: getInputValue('clientLogoUrl'),
    },
    contact: {
      name: getInputValue('contactName'),
      role: getInputValue('contactRole'),
      email: getInputValue('contactEmail'),
      phone: getInputValue('contactPhone'),
    },
    salesperson: {
      name: adminState.salesProfile?.name ?? '',
      role: adminState.salesProfile?.role ?? '',
      email: adminState.salesProfile?.email ?? '',
      phone: adminState.salesProfile?.phone ?? '',
      photo_url: adminState.salesProfile?.photo_url ?? '',
    },
    operation_context: getInputValue('operationContext'),
    customer_priorities: splitLines(getInputValue('customerPriorities')),
    solution_title: getInputValue('solutionTitle'),
    solution_scope: splitLines(getInputValue('solutionScope')),
  };
}

function resetProposalEditor() {
  adminState.loadedProposal = null;
  adminState.loadedVersion = null;
  adminState.publicToken = null;
  elements.proposalForm.reset();
  elements.pricingItemsBody.innerHTML = '';
  createPricingRow();
  setInputValue('proposalTitle', 'Proposta Comercial ViaGate');

  const validUntil = new Date();
  validUntil.setDate(validUntil.getDate() + 15);
  setInputValue('validUntil', validUntil.toISOString().slice(0, 10));

  setInputValue('proposalId', '');
  setInputValue('proposalVersionId', '');
  elements.editorTitle.textContent = 'Nova proposta';
  elements.editorVersionLabel.textContent = 'Rascunho ainda não salvo';
  elements.copyProposalLinkButton.hidden = true;
  hideMessage(elements.proposalMessage);
}

function showDashboard() {
  elements.dashboardView.hidden = false;
  elements.editorView.hidden = true;
  loadProposals();
}

function showEditor() {
  elements.dashboardView.hidden = true;
  elements.editorView.hidden = false;
  window.scrollTo({ top: 0, behavior: 'auto' });
}

async function handleLogin(event) {
  event.preventDefault();
  hideMessage(elements.loginMessage);

  if (!supabase) {
    showMessage(elements.loginMessage, 'Configure o Supabase em proposal/config.js antes de usar esta área.', 'error');
    return;
  }

  const formData = new FormData(elements.loginForm);
  const { error } = await supabase.auth.signInWithPassword({
    email: String(formData.get('email') ?? '').trim(),
    password: String(formData.get('password') ?? ''),
  });

  if (error) {
    showMessage(elements.loginMessage, 'Não foi possível entrar. Verifique o e-mail e a senha.', 'error');
  }
}

async function handleRecovery(event) {
  event.preventDefault();
  hideMessage(elements.recoveryMessage);

  if (!supabase) {
    showMessage(elements.recoveryMessage, 'Configure o Supabase antes de solicitar a recuperação.', 'error');
    return;
  }

  const formData = new FormData(elements.recoveryForm);
  const email = String(formData.get('email') ?? '').trim();
  const redirectTo = `${window.location.origin}${window.location.pathname}`;
  const { error } = await supabase.auth.resetPasswordForEmail(email, { redirectTo });

  if (error) {
    showMessage(elements.recoveryMessage, 'Não foi possível enviar o link de recuperação.', 'error');
    return;
  }

  showMessage(elements.recoveryMessage, 'Link enviado. Verifique o e-mail informado.', 'success');
}

async function handlePasswordUpdate(event) {
  event.preventDefault();
  hideMessage(elements.passwordMessage);

  const password = getInputValue('newPassword');
  const confirmPassword = getInputValue('confirmPassword');

  if (password.length < 8) {
    showMessage(elements.passwordMessage, 'A senha deve ter pelo menos 8 caracteres.', 'error');
    return;
  }

  if (password !== confirmPassword) {
    showMessage(elements.passwordMessage, 'As senhas informadas são diferentes.', 'error');
    return;
  }

  const { error } = await supabase.auth.updateUser({ password });

  if (error) {
    showMessage(elements.passwordMessage, 'Não foi possível alterar a senha.', 'error');
    return;
  }

  showMessage(elements.passwordMessage, 'Senha alterada com sucesso.', 'success');
  window.setTimeout(() => showAuthCard(elements.loginCard), 1200);
}

async function loadSalesProfile() {
  if (!adminState.session?.user || !supabase) {
    return;
  }

  const { data, error } = await supabase
    .from('salespeople')
    .select('*')
    .eq('auth_user_id', adminState.session.user.id)
    .maybeSingle();

  if (error) {
    showMessage(elements.salesProfileMessage, 'Não foi possível carregar o perfil comercial.', 'error');
    return;
  }

  adminState.salesProfile = data;
  setInputValue('salesName', data?.name ?? adminState.session.user.user_metadata?.name ?? '');
  setInputValue('salesRole', data?.role ?? '');
  setInputValue('salesEmail', data?.email ?? adminState.session.user.email ?? '');
  setInputValue('salesPhone', data?.phone ?? '');
  setInputValue('salesPhotoUrl', data?.photo_url ?? '');
}

async function saveSalesProfile(event) {
  event.preventDefault();
  hideMessage(elements.salesProfileMessage);

  const payload = {
    auth_user_id: adminState.session.user.id,
    name: getInputValue('salesName'),
    role: getInputValue('salesRole'),
    email: getInputValue('salesEmail'),
    phone: getInputValue('salesPhone'),
    photo_url: getInputValue('salesPhotoUrl'),
    is_active: true,
  };

  const { data, error } = await supabase
    .from('salespeople')
    .upsert(payload, { onConflict: 'auth_user_id' })
    .select()
    .single();

  if (error) {
    showMessage(elements.salesProfileMessage, 'Não foi possível salvar o perfil comercial.', 'error');
    return;
  }

  adminState.salesProfile = data;
  showMessage(elements.salesProfileMessage, 'Perfil comercial salvo.', 'success');
}

async function loadProposals() {
  if (!supabase) {
    return;
  }

  elements.proposalList.innerHTML = '<div class="empty-state">Carregando propostas...</div>';

  const { data, error } = await supabase
    .from('proposals')
    .select('id,title,status,valid_until,current_version,updated_at,clients(company_name),salespeople(name)')
    .order('updated_at', { ascending: false });

  if (error) {
    elements.proposalList.innerHTML = '<div class="empty-state">Não foi possível carregar as propostas.</div>';
    return;
  }

  if (!data?.length) {
    elements.proposalList.innerHTML = '<div class="empty-state">Nenhuma proposta criada ainda.</div>';
    return;
  }

  elements.proposalList.innerHTML = '';

  data.forEach((proposal) => {
    const row = document.createElement('article');
    row.className = 'proposal-row';
    row.innerHTML = `
      <div>
        <strong>${escapeHtml(proposal.clients?.company_name ?? 'Cliente')}</strong>
        <small>${escapeHtml(proposal.title)}</small>
      </div>
      <div><span>${escapeHtml(proposal.salespeople?.name ?? '—')}</span><small>Comercial</small></div>
      <div><span class="status-badge ${proposal.status === 'published' ? 'published' : ''}">${escapeHtml(proposal.status)}</span><small>v${proposal.current_version || 0} · ${formatDate(proposal.valid_until)}</small></div>
      <div><button class="link-button" type="button" data-open-proposal="${proposal.id}">Abrir</button></div>
    `;

    row.querySelector('[data-open-proposal]')?.addEventListener('click', () => openProposal(proposal.id));
    elements.proposalList.appendChild(row);
  });
}

function escapeHtml(value) {
  const element = document.createElement('div');
  element.textContent = String(value ?? '');
  return element.innerHTML;
}

async function openProposal(proposalId) {
  resetProposalEditor();
  showEditor();
  elements.editorTitle.textContent = 'Carregando proposta...';

  const { data: proposal, error: proposalError } = await supabase
    .from('proposals')
    .select('*,clients(*),client_contacts(*),salespeople(*)')
    .eq('id', proposalId)
    .single();

  if (proposalError) {
    showMessage(elements.proposalMessage, 'Não foi possível carregar a proposta.', 'error');
    return;
  }

  const { data: version, error: versionError } = await supabase
    .from('proposal_versions')
    .select('*')
    .eq('proposal_id', proposalId)
    .order('version_number', { ascending: false })
    .limit(1)
    .maybeSingle();

  if (versionError) {
    showMessage(elements.proposalMessage, 'Não foi possível carregar a versão da proposta.', 'error');
    return;
  }

  adminState.loadedProposal = proposal;
  adminState.loadedVersion = version;
  adminState.publicToken = version?.published_at ? version.public_token : null;

  setInputValue('proposalId', proposal.id);
  setInputValue('proposalVersionId', version?.id ?? '');
  setInputValue('companyName', proposal.clients?.company_name ?? version?.content?.client?.company_name ?? '');
  setInputValue('clientLogoUrl', proposal.clients?.logo_url ?? version?.content?.client?.logo_url ?? '');
  setInputValue('contactName', proposal.client_contacts?.name ?? version?.content?.contact?.name ?? '');
  setInputValue('contactRole', proposal.client_contacts?.role ?? version?.content?.contact?.role ?? '');
  setInputValue('contactEmail', proposal.client_contacts?.email ?? version?.content?.contact?.email ?? '');
  setInputValue('contactPhone', proposal.client_contacts?.phone ?? version?.content?.contact?.phone ?? '');
  setInputValue('proposalTitle', proposal.title);
  setInputValue('validUntil', proposal.valid_until ?? '');
  setInputValue('operationContext', version?.content?.operation_context ?? '');
  setInputValue('customerPriorities', (version?.content?.customer_priorities ?? []).join('\n'));
  setInputValue('solutionTitle', version?.content?.solution_title ?? '');
  setInputValue('solutionScope', (version?.content?.solution_scope ?? []).join('\n'));
  setInputValue('pricingModel', version?.pricing_model ?? 'per_item');
  setInputValue('minimumInvoice', version?.minimum_invoice ?? '');
  setInputValue('setupFee', version?.setup_fee ?? '');
  setInputValue('commercialConditions', (version?.conditions?.items ?? []).join('\n'));

  elements.pricingItemsBody.innerHTML = '';
  const { data: items } = version
    ? await supabase
        .from('proposal_version_items')
        .select('*')
        .eq('proposal_version_id', version.id)
        .order('sort_order')
    : { data: [] };

  (items?.length ? items : [{}]).forEach(createPricingRow);

  elements.editorTitle.textContent = proposal.clients?.company_name ?? 'Proposta';
  elements.editorVersionLabel.textContent = version
    ? `Versão ${version.version_number} · ${version.published_at ? 'publicada' : 'rascunho'}`
    : 'Sem versão';
  elements.copyProposalLinkButton.hidden = !adminState.publicToken;
}

async function ensureClientAndContact() {
  const companyName = getInputValue('companyName');
  const clientPayload = {
    company_name: companyName,
    logo_url: getInputValue('clientLogoUrl') || null,
    updated_at: new Date().toISOString(),
  };

  let clientId = adminState.loadedProposal?.client_id ?? null;

  if (clientId) {
    const { error } = await supabase.from('clients').update(clientPayload).eq('id', clientId);
    if (error) {
      throw error;
    }
  } else {
    const { data, error } = await supabase
      .from('clients')
      .insert({ ...clientPayload, created_by: adminState.session.user.id })
      .select('id')
      .single();

    if (error) {
      throw error;
    }

    clientId = data.id;
  }

  const contactPayload = {
    client_id: clientId,
    name: getInputValue('contactName'),
    role: getInputValue('contactRole') || null,
    email: getInputValue('contactEmail') || null,
    phone: getInputValue('contactPhone') || null,
  };

  let contactId = adminState.loadedProposal?.contact_id ?? null;

  if (contactPayload.name) {
    if (contactId) {
      const { error } = await supabase.from('client_contacts').update(contactPayload).eq('id', contactId);
      if (error) {
        throw error;
      }
    } else {
      const { data, error } = await supabase
        .from('client_contacts')
        .insert(contactPayload)
        .select('id')
        .single();

      if (error) {
        throw error;
      }

      contactId = data.id;
    }
  }

  return { clientId, contactId };
}

async function ensureProposal(clientId, contactId) {
  const payload = {
    client_id: clientId,
    contact_id: contactId,
    salesperson_id: adminState.salesProfile.id,
    title: getInputValue('proposalTitle'),
    valid_until: getInputValue('validUntil') || null,
    updated_at: new Date().toISOString(),
  };

  if (adminState.loadedProposal?.id) {
    const { data, error } = await supabase
      .from('proposals')
      .update(payload)
      .eq('id', adminState.loadedProposal.id)
      .select()
      .single();

    if (error) {
      throw error;
    }

    adminState.loadedProposal = data;
    return data;
  }

  const { data, error } = await supabase
    .from('proposals')
    .insert({
      ...payload,
      status: 'draft',
      current_version: 0,
      created_by: adminState.session.user.id,
    })
    .select()
    .single();

  if (error) {
    throw error;
  }

  adminState.loadedProposal = data;
  setInputValue('proposalId', data.id);
  return data;
}

async function ensureEditableVersion(proposal) {
  const currentVersion = adminState.loadedVersion;

  if (currentVersion && !currentVersion.published_at) {
    return currentVersion;
  }

  const nextVersionNumber = (currentVersion?.version_number ?? proposal.current_version ?? 0) + 1;
  const { data, error } = await supabase
    .from('proposal_versions')
    .insert({
      proposal_id: proposal.id,
      version_number: nextVersionNumber,
      created_by: adminState.session.user.id,
    })
    .select()
    .single();

  if (error) {
    throw error;
  }

  adminState.loadedVersion = data;
  adminState.publicToken = null;
  setInputValue('proposalVersionId', data.id);
  return data;
}

async function persistVersion(version) {
  const content = collectProposalContent();
  const conditions = { items: splitLines(getInputValue('commercialConditions')) };
  const versionPayload = {
    content,
    pricing_model: getInputValue('pricingModel') || 'custom',
    minimum_invoice: Number(getInputValue('minimumInvoice') || 0),
    setup_fee: Number(getInputValue('setupFee') || 0),
    conditions,
  };

  const { data, error } = await supabase
    .from('proposal_versions')
    .update(versionPayload)
    .eq('id', version.id)
    .select()
    .single();

  if (error) {
    throw error;
  }

  const { error: deleteError } = await supabase
    .from('proposal_version_items')
    .delete()
    .eq('proposal_version_id', version.id);

  if (deleteError) {
    throw deleteError;
  }

  const items = collectPricingItems().map((item) => ({
    ...item,
    proposal_version_id: version.id,
  }));

  if (items.length) {
    const { error: itemsError } = await supabase.from('proposal_version_items').insert(items);
    if (itemsError) {
      throw itemsError;
    }
  }

  adminState.loadedVersion = data;
  elements.editorVersionLabel.textContent = `Versão ${data.version_number} · rascunho`;
  return data;
}

async function saveProposalDraft(event) {
  event?.preventDefault();
  hideMessage(elements.proposalMessage);

  if (!adminState.salesProfile?.id) {
    showMessage(elements.proposalMessage, 'Salve seu perfil comercial antes de criar uma proposta.', 'error');
    return null;
  }

  if (!elements.proposalForm.reportValidity()) {
    return null;
  }

  try {
    const { clientId, contactId } = await ensureClientAndContact();
    const proposal = await ensureProposal(clientId, contactId);
    const version = await ensureEditableVersion(proposal);
    const savedVersion = await persistVersion(version);

    showMessage(elements.proposalMessage, `Rascunho da versão ${savedVersion.version_number} salvo.`, 'success');
    return savedVersion;
  } catch (error) {
    console.error(error);
    showMessage(elements.proposalMessage, 'Não foi possível salvar a proposta. Revise os dados e as permissões do Supabase.', 'error');
    return null;
  }
}

async function publishProposal() {
  const version = await saveProposalDraft();

  if (!version) {
    return;
  }

  try {
    const { data, error } = await supabase.rpc('publish_proposal_version', {
      target_version_id: version.id,
    });

    if (error) {
      throw error;
    }

    const result = Array.isArray(data) ? data[0] : data;
    adminState.publicToken = result?.public_token ?? version.public_token;
    adminState.loadedVersion = {
      ...version,
      published_at: result?.published_at ?? new Date().toISOString(),
      public_token: adminState.publicToken,
    };

    elements.editorVersionLabel.textContent = `Versão ${version.version_number} · publicada`;
    elements.copyProposalLinkButton.hidden = !adminState.publicToken;
    showMessage(elements.proposalMessage, `Versão ${version.version_number} publicada. Novas alterações criarão uma nova versão.`, 'success');
  } catch (error) {
    console.error(error);
    showMessage(elements.proposalMessage, 'Não foi possível publicar a proposta.', 'error');
  }
}

async function copyProposalLink() {
  if (!adminState.publicToken) {
    return;
  }

  const url = buildPublicProposalUrl(adminState.publicToken);

  try {
    await navigator.clipboard.writeText(url);
    showMessage(elements.proposalMessage, 'Link público copiado para a área de transferência.', 'success');
  } catch {
    window.prompt('Copie o link da proposta:', url);
  }
}

function bindEvents() {
  elements.loginForm.addEventListener('submit', handleLogin);
  elements.recoveryForm.addEventListener('submit', handleRecovery);
  elements.passwordForm.addEventListener('submit', handlePasswordUpdate);
  document.querySelector('[data-show-recovery]')?.addEventListener('click', () => showAuthCard(elements.recoveryCard));
  document.querySelector('[data-show-login]')?.addEventListener('click', () => showAuthCard(elements.loginCard));
  elements.logoutButton.addEventListener('click', () => supabase?.auth.signOut());
  elements.refreshButton.addEventListener('click', loadProposals);
  elements.newProposalButton.addEventListener('click', () => {
    resetProposalEditor();
    showEditor();
  });
  elements.backToDashboardButton.addEventListener('click', showDashboard);
  elements.salesProfileForm.addEventListener('submit', saveSalesProfile);
  elements.proposalForm.addEventListener('submit', saveProposalDraft);
  elements.addPricingItemButton.addEventListener('click', () => createPricingRow());
  elements.publishProposalButton.addEventListener('click', publishProposal);
  elements.copyProposalLinkButton.addEventListener('click', copyProposalLink);
}

async function handleAuthenticatedSession(session) {
  setAuthenticatedView(session);

  if (!session?.user) {
    return;
  }

  await loadSalesProfile();
  await loadProposals();
}

async function initializeAdmin() {
  bindEvents();

  if (window.lucide) {
    window.lucide.createIcons();
  }

  if (!hasSupabaseConfiguration() || !supabase) {
    showMessage(elements.loginMessage, 'Supabase ainda não configurado. Preencha proposal/config.js com a URL do projeto e a anon key.', 'error');
    return;
  }

  const { data } = await supabase.auth.getSession();
  await handleAuthenticatedSession(data.session);

  supabase.auth.onAuthStateChange((event, session) => {
    if (event === 'PASSWORD_RECOVERY') {
      setAuthenticatedView(null);
      showAuthCard(elements.passwordCard);
      return;
    }

    handleAuthenticatedSession(session);
  });
}

initializeAdmin();
