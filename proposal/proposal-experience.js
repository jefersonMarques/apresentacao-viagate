const moduleDescriptions = {
  'Score | Análise cadastral': {
    icon: 'shield-check',
    title: 'Cargo Score',
    description: 'Pesquisa cadastral com biometria facial, validações oficiais e análise de risco para motorista e veículo.',
  },
  'Consultas e autenticação': {
    icon: 'scan-face',
    title: 'Consultas e autenticação',
    description: 'Consultas pontuais, autenticação de formulários e validações complementares ao processo cadastral.',
  },
  'Aplicativo | Logística': {
    icon: 'route',
    title: 'Cargo Truck',
    description: 'Aplicativo para cadastro, coletas, entregas, eventos de parada e rastreamento por GPS do smartphone do motorista.',
  },
  'Prevenção': {
    icon: 'shield-alert',
    title: 'Prevenção',
    description: 'Gestão preventiva com recursos para multas e histórico veicular completo.',
  },
  'Monitoramento de veículos': {
    icon: 'satellite',
    title: 'Monitoramento de veículos',
    description: 'Monitoramento satelital por integração com gerenciadora, com opções por veículo, viagem e checklist.',
  },
};

const state = {
  started: false,
  gateEnhanced: false,
};

function getGate() {
  return document.getElementById('proposalGate');
}

function getControls() {
  return document.getElementById('proposalControls');
}

function replaceStartButton() {
  const current = document.getElementById('proposalStartButton');
  if (!current || current.dataset.experienceButton === 'true') {
    return current;
  }

  const replacement = current.cloneNode(true);
  replacement.dataset.experienceButton = 'true';
  replacement.textContent = 'VER PROPOSTA';
  current.replaceWith(replacement);
  return replacement;
}

function revealProposal() {
  state.started = true;
  document.body.classList.remove('proposal-locked');
  document.body.classList.add('proposal-reading');

  const gate = getGate();
  const controls = getControls();
  if (gate) gate.hidden = true;
  if (controls) controls.hidden = false;

  document.dispatchEvent(new CustomEvent('proposal:started'));
  window.setTimeout(() => window.scrollTo({ top: 0, behavior: 'smooth' }), 0);
}

async function toggleFullscreen() {
  try {
    if (document.fullscreenElement) {
      await document.exitFullscreen();
      return;
    }

    await document.documentElement.requestFullscreen();
  } catch {
    return;
  }
}

function getSelectedModules() {
  const names = Array.from(document.querySelectorAll('.proposal-price-group-header strong'))
    .map((element) => element.textContent?.trim())
    .filter(Boolean);

  return Array.from(new Set(names))
    .map((name) => moduleDescriptions[name])
    .filter(Boolean);
}

function createCompanyLayer() {
  let layer = document.getElementById('proposalCompanyLayer');
  if (layer) {
    return layer;
  }

  layer = document.createElement('section');
  layer.className = 'proposal-company-layer';
  layer.id = 'proposalCompanyLayer';
  layer.hidden = true;
  document.body.appendChild(layer);
  return layer;
}

function renderCompanyLayer() {
  const layer = createCompanyLayer();
  const modules = getSelectedModules();
  const cards = (modules.length ? modules : Object.values(moduleDescriptions).slice(0, 3)).map((module) => `
    <article class="proposal-company-module">
      <span><i data-lucide="${module.icon}"></i></span>
      <div>
        <strong>${module.title}</strong>
        <p>${module.description}</p>
      </div>
    </article>
  `).join('');

  layer.innerHTML = `
    <header class="proposal-company-header">
      <img src="../assets/logo-viagate-white.svg" alt="ViaGate" />
      <button type="button" data-company-close>VOLTAR À PROPOSTA</button>
    </header>
    <main class="proposal-company-content">
      <section class="proposal-company-hero">
        <small>CONHEÇA A VIAGATE</small>
        <h1>Tecnologia para gestão de risco e operação logística.</h1>
        <p>A ViaGate desenvolve soluções para validação cadastral, biometria, consultas, rastreabilidade, prevenção e integração de operações logísticas.</p>
      </section>
      <section class="proposal-company-proof">
        <article><strong>Biometria facial</strong><span>Validação de identidade com prova de vida e integração ao processo cadastral.</span></article>
        <article><strong>Segurança e LGPD</strong><span>Fluxos voltados à proteção de dados, rastreabilidade e controle de acesso às informações.</span></article>
        <article><strong>Integração</strong><span>APIs e soluções configuráveis para conectar a ViaGate aos processos existentes da operação.</span></article>
      </section>
      <section class="proposal-company-related">
        <div class="proposal-company-section-title">
          <small>SOLUÇÕES RELACIONADAS</small>
          <h2>O que está conectado a esta proposta</h2>
        </div>
        <div class="proposal-company-modules">${cards}</div>
      </section>
      <section class="proposal-company-footer">
        <div>
          <small>PROPOSTA COMERCIAL</small>
          <strong>Continue de onde parou quando quiser.</strong>
        </div>
        <button type="button" data-company-close>VOLTAR À PROPOSTA</button>
      </section>
    </main>
  `;

  layer.querySelectorAll('[data-company-close]').forEach((button) => {
    button.addEventListener('click', () => {
      layer.hidden = true;
      document.body.classList.remove('proposal-company-open');
    });
  });

  window.lucide?.createIcons();
}

function showCompanyLayer() {
  renderCompanyLayer();
  const layer = createCompanyLayer();
  layer.hidden = false;
  document.body.classList.add('proposal-company-open');
  layer.scrollTop = 0;
}

function addAboutButton() {
  const actionRow = getGate()?.querySelector('.proposal-view-action-row');
  if (!actionRow || actionRow.querySelector('[data-proposal-about]')) {
    return;
  }

  const button = document.createElement('button');
  button.className = 'proposal-view-button secondary proposal-about-button';
  button.type = 'button';
  button.dataset.proposalAbout = 'true';
  button.textContent = 'CONHECER A VIAGATE';
  button.addEventListener('click', showCompanyLayer);
  actionRow.appendChild(button);
}

function addReadingActions() {
  if (document.getElementById('proposalReadingActions')) {
    return;
  }

  const actions = document.createElement('div');
  actions.className = 'proposal-reading-actions';
  actions.id = 'proposalReadingActions';
  actions.innerHTML = `
    <button type="button" data-reading-about>CONHECER A VIAGATE</button>
    <button type="button" data-reading-fullscreen>APRESENTAR EM TELA CHEIA</button>
  `;

  actions.querySelector('[data-reading-about]')?.addEventListener('click', showCompanyLayer);
  actions.querySelector('[data-reading-fullscreen]')?.addEventListener('click', toggleFullscreen);
  document.body.appendChild(actions);
}

function enhanceGate() {
  if (state.gateEnhanced) {
    return;
  }

  const startButton = replaceStartButton();
  const gate = getGate();
  if (!startButton || !gate) {
    return;
  }

  state.gateEnhanced = true;
  addAboutButton();
  addReadingActions();

  startButton.addEventListener('click', revealProposal);

  const restartButton = document.getElementById('proposalRestartButton');
  if (restartButton) {
    restartButton.hidden = true;
  }
}

function keepNormalReadingMode() {
  if (!state.started) {
    return;
  }

  document.body.classList.remove('proposal-locked');
  const gate = getGate();
  if (gate) gate.hidden = true;
}

function initialize() {
  enhanceGate();

  const observer = new MutationObserver(() => {
    enhanceGate();
    keepNormalReadingMode();
  });

  observer.observe(document.body, { childList: true, subtree: true, attributes: true, attributeFilter: ['class', 'hidden'] });
  window.setTimeout(() => observer.disconnect(), 15000);

  document.addEventListener('fullscreenchange', keepNormalReadingMode);
}

initialize();
