import {
  formatCurrency,
  formatDate,
  hasSupabaseConfiguration,
  supabase,
} from './supabase.js';

const viewerState = {
  started: false,
  wheelLocked: false,
};

const loadingElement = document.getElementById('proposalLoading');
const presentationElement = document.getElementById('proposalPresentation');
const controlsElement = document.getElementById('proposalControls');
const startButton = document.getElementById('proposalStartButton');
const restartButton = document.getElementById('proposalRestartButton');
const previousButton = document.querySelector('[data-proposal-previous]');
const nextButton = document.querySelector('[data-proposal-next]');
const counterElement = document.querySelector('[data-proposal-counter]');

function escapeHtml(value) {
  const element = document.createElement('div');
  element.textContent = String(value ?? '');
  return element.innerHTML;
}

function safeImageUrl(value) {
  if (!value) {
    return '';
  }

  try {
    const url = new URL(value, window.location.origin);
    return ['http:', 'https:'].includes(url.protocol) ? url.toString() : '';
  } catch {
    return '';
  }
}

function renderSimpleCards(items) {
  return (items ?? []).map((item, index) => `
    <article class="proposal-content-card">
      <small>${String(index + 1).padStart(2, '0')}</small>
      <h3>${escapeHtml(item)}</h3>
    </article>
  `).join('');
}

function getPricingModelCards(model) {
  const perItem = {
    eyebrow: 'MODELO 1',
    title: 'Análise por item',
    copy: 'Cada cadastro ou consulta é processado de forma individual. O investimento é aplicado individualmente ao motorista e ao veículo. Indicado para operações com frota, agregados e terceiros.',
  };

  const bundle = {
    eyebrow: 'MODELO 2',
    title: 'Análise por conjunto',
    copy: 'Processamento unificado de motorista + veículos, incluindo cadastro ou consulta. O modelo simplifica o processo de análise em uma única composição.',
  };

  if (model === 'per_item') {
    return [perItem];
  }

  if (model === 'bundle') {
    return [bundle];
  }

  if (model === 'item_and_bundle') {
    return [perItem, bundle];
  }

  return [];
}

function resolveItemGroup(item) {
  const configuredGroup = String(item.metadata?.group ?? '').trim();
  if (configuredGroup) {
    return configuredGroup;
  }

  const label = String(item.label ?? '').toLocaleLowerCase('pt-BR');

  if (/autenticador|processo|vitimologia|antt/.test(label)) {
    return 'Consultas e autenticação';
  }

  if (/multa|histórico veicular/.test(label)) {
    return 'Prevenção';
  }

  if (/veículo fixo|viagem avulsa|autotrac|check list|checklist|monitoramento/.test(label)) {
    return 'Monitoramento de veículos';
  }

  if (/primeira viagem|viagens subsequentes|rastreamento via aplicativo|cargo truck/.test(label)) {
    return 'Aplicativo | Logística';
  }

  return 'Score | Análise cadastral';
}

function groupPricingItems(items) {
  return (items ?? []).reduce((groups, item) => {
    const group = resolveItemGroup(item);
    if (!groups.has(group)) {
      groups.set(group, []);
    }
    groups.get(group).push(item);
    return groups;
  }, new Map());
}

function renderPricingGroups(items) {
  const groups = groupPricingItems(items);

  return Array.from(groups.entries()).map(([group, groupItems]) => {
    const allOptional = groupItems.every((item) => item.is_optional);
    const rows = groupItems.map((item) => `
      <tr>
        <td>${escapeHtml(item.label)}</td>
        <td>${escapeHtml(item.unit || '—')}</td>
        <td>${item.is_optional ? 'Opcional' : 'Proposto'}</td>
        <td>${formatCurrency(item.price)}</td>
      </tr>
    `).join('');

    return `
      <section class="proposal-price-group">
        <header class="proposal-price-group-header">
          <strong>${escapeHtml(group)}</strong>
          ${allOptional ? '<span class="proposal-optional-label">Opcional</span>' : ''}
        </header>
        <table class="proposal-price-table">
          <thead><tr><th>Item</th><th>Unidade</th><th>Status</th><th>Valor</th></tr></thead>
          <tbody>${rows}</tbody>
        </table>
      </section>
    `;
  }).join('');
}

function createSlide(content, theme = 'light', footerLabel = 'ViaGate') {
  return `
    <section class="proposal-slide ${theme}" data-proposal-slide>
      ${content}
      <div class="proposal-slide-footer"><span>${escapeHtml(footerLabel)}</span><span data-slide-number></span></div>
    </section>
  `;
}

function renderProposal(data) {
  const proposal = data.proposal ?? {};
  const version = data.version ?? {};
  const content = version.content ?? {};
  const client = content.client ?? {};
  const contact = content.contact ?? {};
  const salesperson = content.salesperson ?? {};
  const conditions = version.conditions?.items ?? [];
  const pricingItems = version.items ?? [];
  const modelCards = getPricingModelCards(version.pricing_model);
  const clientLogoUrl = safeImageUrl(client.logo_url);
  const salespersonPhotoUrl = safeImageUrl(salesperson.photo_url);
  const slides = [];

  document.title = client.company_name
    ? `ViaGate — Proposta para ${client.company_name}`
    : `ViaGate — ${proposal.title || 'Proposta Comercial'}`;

  slides.push(createSlide(`
    <div class="proposal-slide-inner proposal-cover-grid">
      <div>
        <img class="proposal-cover-logo" src="../assets/logo-viagate-white.svg" alt="ViaGate" />
        <p class="proposal-kicker">PROPOSTA COMERCIAL · VERSÃO ${String(version.version_number ?? 1).padStart(2, '0')}</p>
        <h1>${client.company_name ? `Preparada para<br /><span>${escapeHtml(client.company_name)}</span>` : escapeHtml(proposal.title || 'Proposta Comercial ViaGate')}</h1>
        ${contact.name ? `<p class="proposal-lead">Aos cuidados de ${escapeHtml(contact.name)}${contact.role ? ` · ${escapeHtml(contact.role)}` : ''}</p>` : ''}
        ${clientLogoUrl ? `<img class="proposal-client-logo" src="${escapeHtml(clientLogoUrl)}" alt="${escapeHtml(client.company_name || 'Cliente')}" />` : ''}
      </div>
      <aside class="proposal-person-card">
        ${salespersonPhotoUrl ? `<img src="${escapeHtml(salespersonPhotoUrl)}" alt="${escapeHtml(salesperson.name || 'Comercial ViaGate')}" />` : ''}
        <small>APRESENTADO POR</small>
        ${salesperson.name ? `<strong>${escapeHtml(salesperson.name)}</strong>` : ''}
        ${salesperson.role ? `<span>${escapeHtml(salesperson.role)}</span>` : ''}
        ${salesperson.email ? `<span>${escapeHtml(salesperson.email)}</span>` : ''}
        ${salesperson.phone ? `<span>${escapeHtml(salesperson.phone)}</span>` : ''}
      </aside>
    </div>
  `, 'dark', 'Proposta comercial'));

  if (content.operation_context || content.customer_priorities?.length) {
    slides.push(createSlide(`
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">CENÁRIO CONSIDERADO</p>
          <h2>Contexto da <span>operação</span></h2>
          ${content.operation_context ? `<p class="proposal-lead">${escapeHtml(content.operation_context)}</p>` : ''}
          ${content.customer_priorities?.length ? `<div class="proposal-content-grid">${renderSimpleCards(content.customer_priorities)}</div>` : ''}
        </div>
      </div>
    `, 'light', 'Cenário considerado'));
  }

  if (content.solution_title || content.solution_scope?.length) {
    slides.push(createSlide(`
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">SOLUÇÃO PROPOSTA</p>
          ${content.solution_title ? `<h2>${escapeHtml(content.solution_title)}</h2>` : ''}
          ${content.solution_scope?.length ? `<div class="proposal-content-grid">${renderSimpleCards(content.solution_scope)}</div>` : ''}
        </div>
      </div>
    `, 'dark', 'Solução proposta'));
  }

  if (modelCards.length) {
    slides.push(createSlide(`
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">MODELO DE ANÁLISE CADASTRAL</p>
          <h2>${version.pricing_model === 'item_and_bundle' ? 'Duas formas de <span>compor a análise</span>' : escapeHtml(modelCards[0].title)}</h2>
          <div class="proposal-model-grid">
            ${modelCards.map((model) => `
              <article class="proposal-model-card">
                <small>${escapeHtml(model.eyebrow)}</small>
                <strong>${escapeHtml(model.title)}</strong>
                <span>${escapeHtml(model.copy)}</span>
              </article>
            `).join('')}
          </div>
        </div>
      </div>
    `, 'light', 'Modelo comercial'));
  }

  if (pricingItems.length || Number(version.minimum_invoice) > 0 || Number(version.setup_fee) > 0) {
    slides.push(createSlide(`
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">INVESTIMENTO</p>
          <h2>Valores da <span>proposta comercial</span></h2>
          <p class="proposal-commercial-note">Os valores são aplicados de acordo com o uso e podem variar conforme o número de consultas realizadas, respeitando os valores definidos nesta proposta.</p>
          ${pricingItems.length ? `<div class="proposal-price-groups">${renderPricingGroups(pricingItems)}</div>` : ''}
          ${(Number(version.minimum_invoice) > 0 || Number(version.setup_fee) > 0) ? `
            <div class="proposal-highlight-grid">
              ${Number(version.minimum_invoice) > 0 ? `
                <div class="proposal-highlight">
                  <small>Fatura mínima mensal</small>
                  <strong>${formatCurrency(version.minimum_invoice)}</strong>
                </div>
              ` : ''}
              ${Number(version.setup_fee) > 0 ? `
                <div class="proposal-highlight">
                  <small>Implantação</small>
                  <strong>${formatCurrency(version.setup_fee)}</strong>
                </div>
              ` : ''}
            </div>
          ` : ''}
        </div>
      </div>
    `, 'dark', 'Investimento'));
  }

  if (conditions.length || proposal.valid_until) {
    const validityText = data.is_expired
      ? `Expirada em ${formatDate(proposal.valid_until)}`
      : `Válida até ${formatDate(proposal.valid_until)}`;

    slides.push(createSlide(`
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">OBSERVAÇÕES E CONDIÇÕES</p>
          <h2>Condições <span>comerciais</span></h2>
          ${conditions.length ? `<div class="proposal-content-grid">${renderSimpleCards(conditions)}</div>` : ''}
          ${proposal.valid_until ? `<div class="proposal-validity"><strong>${escapeHtml(validityText)}</strong></div>` : ''}
        </div>
      </div>
    `, 'light', 'Condições comerciais'));
  }

  slides.push(createSlide(`
    <div class="proposal-slide-inner proposal-contact-final">
      <div>
        <p class="proposal-kicker" style="color:#fff">CONTATO COMERCIAL</p>
        <h2>${salesperson.name ? escapeHtml(salesperson.name) : 'ViaGate'}</h2>
        ${salesperson.role ? `<p class="proposal-lead" style="color:rgba(255,255,255,.84)">${escapeHtml(salesperson.role)}</p>` : ''}
      </div>
      <aside class="proposal-contact-data">
        ${salespersonPhotoUrl ? `<img src="${escapeHtml(salespersonPhotoUrl)}" alt="${escapeHtml(salesperson.name || 'ViaGate')}" />` : ''}
        <small>CONTATO RESPONSÁVEL</small>
        ${salesperson.name ? `<strong>${escapeHtml(salesperson.name)}</strong>` : ''}
        ${salesperson.phone ? `<span>${escapeHtml(salesperson.phone)}</span>` : ''}
        ${salesperson.email ? `<span>${escapeHtml(salesperson.email)}</span>` : ''}
      </aside>
    </div>
  `, 'orange', 'Contato comercial'));

  presentationElement.innerHTML = slides.join('');
  presentationElement.hidden = false;
  loadingElement.hidden = true;
  renumberSlides();
  goToSlide(0, 'auto');
  showGate('start');
}

function getSlides() {
  return Array.from(document.querySelectorAll('[data-proposal-slide]'));
}

function getCurrentSlideIndex() {
  const slides = getSlides();
  const center = window.scrollY + window.innerHeight / 2;
  let currentIndex = 0;

  slides.forEach((slide, index) => {
    if (center >= slide.offsetTop) {
      currentIndex = index;
    }
  });

  return currentIndex;
}

function renumberSlides() {
  const slides = getSlides();
  const total = slides.length;

  slides.forEach((slide, index) => {
    const element = slide.querySelector('[data-slide-number]');
    if (element) {
      element.textContent = `${String(index + 1).padStart(2, '0')} / ${String(total).padStart(2, '0')}`;
    }
  });

  updateControls();
}

function goToSlide(index, behavior = 'smooth') {
  const slides = getSlides();

  if (index < 0 || index >= slides.length) {
    return;
  }

  slides[index].scrollIntoView({ behavior, block: 'start' });
  window.setTimeout(updateControls, behavior === 'auto' ? 0 : 120);
}

function updateControls() {
  const slides = getSlides();
  const currentIndex = getCurrentSlideIndex();

  counterElement.textContent = `${String(currentIndex + 1).padStart(2, '0')} / ${String(slides.length).padStart(2, '0')}`;
  previousButton.disabled = currentIndex <= 0;
  nextButton.disabled = currentIndex >= slides.length - 1;
}

function showGate(mode) {
  document.body.classList.add('proposal-locked');
  controlsElement.hidden = true;

  const continuing = mode === 'continue';
  startButton.textContent = continuing ? 'CONTINUAR APRESENTAÇÃO' : 'INICIAR APRESENTAÇÃO';
  restartButton.hidden = !continuing;
}

function revealProposal() {
  document.body.classList.remove('proposal-locked');
  controlsElement.hidden = false;
  restartButton.hidden = true;
  updateControls();
}

async function startPresentation() {
  try {
    if (!document.fullscreenElement) {
      await document.documentElement.requestFullscreen();
    }

    viewerState.started = true;
    revealProposal();
    document.dispatchEvent(new CustomEvent('proposal:started'));
  } catch {
    showGate(viewerState.started ? 'continue' : 'start');
  }
}

function restartPresentation() {
  goToSlide(0, 'auto');
  viewerState.started = false;
  showGate('start');
  document.dispatchEvent(new CustomEvent('proposal:restarted'));
}

function handleKeyboard(event) {
  if (!viewerState.started || !document.fullscreenElement) {
    return;
  }

  if (['ArrowRight', 'ArrowDown', 'PageDown'].includes(event.key)) {
    event.preventDefault();
    goToSlide(getCurrentSlideIndex() + 1);
    return;
  }

  if (['ArrowLeft', 'ArrowUp', 'PageUp'].includes(event.key)) {
    event.preventDefault();
    goToSlide(getCurrentSlideIndex() - 1);
    return;
  }

  if (event.key === 'Home') {
    event.preventDefault();
    goToSlide(0);
    return;
  }

  if (event.key === 'End') {
    event.preventDefault();
    goToSlide(getSlides().length - 1);
  }
}

function handleWheel(event) {
  if (!viewerState.started || !document.fullscreenElement || Math.abs(event.deltaY) < 18) {
    return;
  }

  event.preventDefault();

  if (viewerState.wheelLocked) {
    return;
  }

  viewerState.wheelLocked = true;
  goToSlide(getCurrentSlideIndex() + (event.deltaY > 0 ? 1 : -1));

  window.setTimeout(() => {
    viewerState.wheelLocked = false;
  }, 720);
}

function bindViewerEvents() {
  startButton.addEventListener('click', startPresentation);
  restartButton.addEventListener('click', restartPresentation);
  previousButton.addEventListener('click', () => goToSlide(getCurrentSlideIndex() - 1));
  nextButton.addEventListener('click', () => goToSlide(getCurrentSlideIndex() + 1));
  window.addEventListener('keydown', handleKeyboard);
  window.addEventListener('wheel', handleWheel, { passive: false });
  window.addEventListener('scroll', updateControls, { passive: true });

  document.addEventListener('fullscreenchange', () => {
    if (document.fullscreenElement) {
      if (viewerState.started) {
        revealProposal();
      }
      return;
    }

    if (viewerState.started) {
      showGate('continue');
    }
  });
}

function showLoadError(message) {
  loadingElement.textContent = message;
}

async function initializeViewer() {
  bindViewerEvents();

  if (!hasSupabaseConfiguration() || !supabase) {
    showLoadError('Proposta indisponível: Supabase ainda não configurado.');
    return;
  }

  const token = new URLSearchParams(window.location.search).get('token');
  if (!token) {
    showLoadError('Link de proposta inválido.');
    return;
  }

  const { data, error } = await supabase.rpc('get_public_proposal', {
    proposal_token: token,
  });

  if (error || !data) {
    showLoadError('Proposta não encontrada ou indisponível.');
    return;
  }

  renderProposal(data);
}

initializeViewer();
