import {
  formatCurrency,
  formatDate,
  hasSupabaseConfiguration,
  supabase,
} from './supabase.js';

const viewerState = {
  started: false,
  wheelLocked: false,
  controlsTimer: null,
};

const loadingElement = document.getElementById('proposalLoading');
const presentationElement = document.getElementById('proposalPresentation');
const controlsElement = document.getElementById('proposalControls');

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

function renderList(items, emptyText) {
  if (!items?.length) {
    return `<article class="proposal-content-card"><small>CONTEXTO</small><h3>${escapeHtml(emptyText)}</h3><p>Detalhamento a confirmar com o cliente.</p></article>`;
  }

  return items.map((item, index) => `
    <article class="proposal-content-card">
      <small>${String(index + 1).padStart(2, '0')}</small>
      <h3>${escapeHtml(item)}</h3>
      <p>Item considerado na construção desta proposta comercial.</p>
    </article>
  `).join('');
}

function renderPricingRows(items) {
  if (!items?.length) {
    return '<tr><td colspan="4">Investimento a definir.</td></tr>';
  }

  return items.map((item) => `
    <tr>
      <td>${escapeHtml(item.label)}</td>
      <td>${item.is_optional ? 'Opcional' : 'Proposto'}</td>
      <td>${escapeHtml(item.unit || '—')}</td>
      <td>${formatCurrency(item.price)}</td>
    </tr>
  `).join('');
}

function renderProposal(data) {
  const proposal = data.proposal ?? {};
  const version = data.version ?? {};
  const content = version.content ?? {};
  const client = content.client ?? {};
  const contact = content.contact ?? {};
  const salesperson = content.salesperson ?? {};
  const clientLogoUrl = safeImageUrl(client.logo_url);
  const salespersonPhotoUrl = safeImageUrl(salesperson.photo_url);
  const conditions = version.conditions?.items ?? [];
  const pricingItems = version.items ?? [];
  const requiredItems = pricingItems.filter((item) => !item.is_optional);
  const optionalItems = pricingItems.filter((item) => item.is_optional);
  const expiredText = data.is_expired ? 'Proposta expirada' : `Válida até ${formatDate(proposal.valid_until)}`;

  document.title = `ViaGate — Proposta para ${client.company_name || 'cliente'}`;

  presentationElement.innerHTML = `
    <section class="proposal-slide dark" data-proposal-slide>
      <div class="proposal-slide-inner proposal-cover-grid">
        <div>
          <img class="proposal-cover-logo" src="../assets/logo-viagate-white.svg" alt="ViaGate" />
          <p class="proposal-kicker">PROPOSTA COMERCIAL · VERSÃO ${String(version.version_number ?? 1).padStart(2, '0')}</p>
          <h1>Preparada para<br /><span>${escapeHtml(client.company_name || 'sua operação')}</span></h1>
          ${contact.name ? `<p class="proposal-lead">Aos cuidados de ${escapeHtml(contact.name)}${contact.role ? ` · ${escapeHtml(contact.role)}` : ''}.</p>` : '<p class="proposal-lead">Uma proposta construída para o contexto desta operação.</p>'}
          ${clientLogoUrl ? `<img class="proposal-client-logo" src="${escapeHtml(clientLogoUrl)}" alt="${escapeHtml(client.company_name || 'Cliente')}" />` : ''}
          <button class="proposal-start-button" type="button" data-proposal-start>Iniciar apresentação</button>
        </div>
        <aside class="proposal-person-card">
          ${salespersonPhotoUrl ? `<img src="${escapeHtml(salespersonPhotoUrl)}" alt="${escapeHtml(salesperson.name || 'Comercial ViaGate')}" />` : ''}
          <small>Apresentado por</small>
          <strong>${escapeHtml(salesperson.name || 'ViaGate Comercial')}</strong>
          <span>${escapeHtml(salesperson.role || 'Equipe Comercial')}</span>
          ${salesperson.email ? `<span>${escapeHtml(salesperson.email)}</span>` : ''}
          ${salesperson.phone ? `<span>${escapeHtml(salesperson.phone)}</span>` : ''}
        </aside>
      </div>
      <div class="proposal-slide-footer"><span>ViaGate</span><span>01</span></div>
    </section>

    <section class="proposal-slide light" data-proposal-slide>
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">CENÁRIO CONSIDERADO</p>
          <h2>Partimos do contexto <span>da sua operação.</span></h2>
          <p class="proposal-lead">${escapeHtml(content.operation_context || 'A proposta foi estruturada a partir das necessidades apresentadas durante a negociação.')}</p>
          <div class="proposal-content-grid">
            ${renderList(content.customer_priorities ?? [], 'Prioridades a confirmar')}
          </div>
        </div>
      </div>
      <div class="proposal-slide-footer"><span>Contexto</span><span>02</span></div>
    </section>

    <section class="proposal-slide dark" data-proposal-slide>
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">SOLUÇÃO PROPOSTA</p>
          <h2>${escapeHtml(content.solution_title || 'Solução ViaGate')} <span>para esta operação.</span></h2>
          <p class="proposal-lead">O escopo abaixo representa os componentes previstos para atender o cenário apresentado.</p>
          <div class="proposal-content-grid">
            ${renderList(content.solution_scope ?? [], 'Escopo em definição')}
          </div>
        </div>
      </div>
      <div class="proposal-slide-footer"><span>Solução</span><span>03</span></div>
    </section>

    <section class="proposal-slide light" data-proposal-slide>
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">ESCOPO COMERCIAL</p>
          <h2>O que está <span>incluído e opcional.</span></h2>
          <div class="proposal-content-grid">
            <article class="proposal-content-card">
              <small>ITENS PROPOSTOS</small>
              <h3>${requiredItems.length || 0} itens</h3>
              <p>${escapeHtml(requiredItems.map((item) => item.label).join(' · ') || 'A definir')}</p>
            </article>
            <article class="proposal-content-card">
              <small>ITENS OPCIONAIS</small>
              <h3>${optionalItems.length || 0} itens</h3>
              <p>${escapeHtml(optionalItems.map((item) => item.label).join(' · ') || 'Nenhum item opcional nesta versão')}</p>
            </article>
            <article class="proposal-content-card">
              <small>MODELO COMERCIAL</small>
              <h3>${escapeHtml(version.pricing_model_label || version.pricing_model || 'Personalizado')}</h3>
              <p>Modelo utilizado como base para os valores desta versão.</p>
            </article>
          </div>
        </div>
      </div>
      <div class="proposal-slide-footer"><span>Escopo</span><span>04</span></div>
    </section>

    <section class="proposal-slide dark" data-proposal-slide>
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">INVESTIMENTO</p>
          <h2>Valores desta <span>versão comercial.</span></h2>
          <table class="proposal-price-table">
            <thead><tr><th>Solução</th><th>Tipo</th><th>Unidade</th><th>Valor</th></tr></thead>
            <tbody>${renderPricingRows(pricingItems)}</tbody>
          </table>
          <div class="proposal-highlight-grid">
            <div class="proposal-highlight">
              <small>Fatura mínima mensal</small>
              <strong>${formatCurrency(version.minimum_invoice)}</strong>
              <span>Quando aplicável à contratação.</span>
            </div>
            <div class="proposal-highlight">
              <small>Implantação</small>
              <strong>${formatCurrency(version.setup_fee)}</strong>
              <span>Valor previsto nesta versão.</span>
            </div>
          </div>
        </div>
      </div>
      <div class="proposal-slide-footer"><span>Investimento</span><span>05</span></div>
    </section>

    <section class="proposal-slide light" data-proposal-slide>
      <div class="proposal-slide-inner">
        <div>
          <p class="proposal-kicker">CONDIÇÕES COMERCIAIS</p>
          <h2>Condições desta <span>proposta.</span></h2>
          <div class="proposal-content-grid">
            ${renderList(conditions, 'Condições padrão')}
            <article class="proposal-content-card">
              <small>VALIDADE</small>
              <h3>${escapeHtml(expiredText)}</h3>
              <p>Versão ${escapeHtml(version.version_number ?? 1)} publicada em ${formatDate((version.published_at || '').slice(0, 10))}.</p>
            </article>
          </div>
        </div>
      </div>
      <div class="proposal-slide-footer"><span>Condições</span><span>06</span></div>
    </section>

    <section class="proposal-slide orange" data-proposal-slide>
      <div class="proposal-slide-inner proposal-cover-grid">
        <div>
          <p class="proposal-kicker" style="color:#fff">PRÓXIMOS PASSOS</p>
          <h2>Vamos avançar com <span style="color:#071827">esta operação?</span></h2>
          <p class="proposal-lead" style="color:rgba(255,255,255,.86)">O contato comercial responsável está disponível para revisar escopo, volumes, integrações e próximos passos.</p>
        </div>
        <aside class="proposal-person-card">
          ${salespersonPhotoUrl ? `<img src="${escapeHtml(salespersonPhotoUrl)}" alt="${escapeHtml(salesperson.name || 'Comercial ViaGate')}" />` : ''}
          <small>Contato comercial</small>
          <strong>${escapeHtml(salesperson.name || 'ViaGate Comercial')}</strong>
          <span>${escapeHtml(salesperson.role || 'Equipe Comercial')}</span>
          ${salesperson.email ? `<span>${escapeHtml(salesperson.email)}</span>` : ''}
          ${salesperson.phone ? `<span>${escapeHtml(salesperson.phone)}</span>` : ''}
        </aside>
      </div>
      <div class="proposal-slide-footer"><span>ViaGate</span><span>07</span></div>
    </section>
  `;

  loadingElement.hidden = true;
  presentationElement.hidden = false;
  controlsElement.hidden = false;
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

function goToSlide(index) {
  const slides = getSlides();
  if (index < 0 || index >= slides.length) {
    return;
  }

  slides[index].scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function updateControls() {
  const slides = getSlides();
  const currentIndex = getCurrentSlideIndex();
  const counter = document.querySelector('[data-proposal-counter]');

  if (counter) {
    counter.textContent = `${String(currentIndex + 1).padStart(2, '0')} / ${String(slides.length).padStart(2, '0')}`;
  }
}

function showControls() {
  if (!viewerState.started) {
    return;
  }

  document.body.classList.add('controls-visible');
  window.clearTimeout(viewerState.controlsTimer);
  viewerState.controlsTimer = window.setTimeout(() => {
    document.body.classList.remove('controls-visible');
  }, 2200);
}

async function startPresentation() {
  viewerState.started = true;
  document.body.classList.add('started');
  showControls();

  try {
    if (!document.fullscreenElement) {
      await document.documentElement.requestFullscreen();
    }
  } catch {
  }
}

async function toggleFullscreen() {
  if (document.fullscreenElement) {
    await document.exitFullscreen();
    return;
  }

  await document.documentElement.requestFullscreen();
}

function handleKeyboard(event) {
  if (['ArrowRight', 'ArrowDown', 'PageDown'].includes(event.key)) {
    event.preventDefault();
    goToSlide(getCurrentSlideIndex() + 1);
  }

  if (['ArrowLeft', 'ArrowUp', 'PageUp'].includes(event.key)) {
    event.preventDefault();
    goToSlide(getCurrentSlideIndex() - 1);
  }

  if (event.key === 'Home') {
    event.preventDefault();
    goToSlide(0);
  }

  if (event.key === 'End') {
    event.preventDefault();
    goToSlide(getSlides().length - 1);
  }

  if (event.key.toLowerCase() === 'f') {
    event.preventDefault();
    toggleFullscreen().catch(() => {});
  }
}

function handleWheel(event) {
  if (!viewerState.started || Math.abs(event.deltaY) < 18) {
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
  document.querySelector('[data-proposal-start]')?.addEventListener('click', startPresentation);
  document.querySelector('[data-proposal-previous]')?.addEventListener('click', () => goToSlide(getCurrentSlideIndex() - 1));
  document.querySelector('[data-proposal-next]')?.addEventListener('click', () => goToSlide(getCurrentSlideIndex() + 1));
  document.querySelector('[data-proposal-fullscreen]')?.addEventListener('click', () => toggleFullscreen().catch(() => {}));
  window.addEventListener('keydown', handleKeyboard);
  window.addEventListener('wheel', handleWheel, { passive: false });
  window.addEventListener('scroll', updateControls, { passive: true });
  window.addEventListener('mousemove', showControls, { passive: true });
}

function showLoadError(message) {
  loadingElement.textContent = message;
}

async function initializeViewer() {
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
  bindViewerEvents();
  updateControls();
}

initializeViewer();
