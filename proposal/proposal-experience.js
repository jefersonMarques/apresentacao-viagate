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
  Prevenção: {
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

const icons = {
  'shield-check': '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.6 2.8 8 7 10 4.2-2 7-5.4 7-10V6l-7-3Z"/><path d="m9 12 2 2 4-4"/></svg>',
  'scan-face': '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 3H4a1 1 0 0 0-1 1v3M17 3h3a1 1 0 0 1 1 1v3M7 21H4a1 1 0 0 1-1-1v-3M17 21h3a1 1 0 0 0 1-1v-3"/><circle cx="12" cy="10" r="3"/><path d="M7.5 18c1-2 2.5-3 4.5-3s3.5 1 4.5 3"/></svg>',
  route: '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="6" cy="18" r="2"/><circle cx="18" cy="6" r="2"/><path d="M8 18h3a3 3 0 0 0 3-3V9a3 3 0 0 1 3-3"/></svg>',
  'shield-alert': '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3 5 6v5c0 4.6 2.8 8 7 10 4.2-2 7-5.4 7-10V6l-7-3Z"/><path d="M12 8v5M12 16h.01"/></svg>',
  satellite: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m7 7 10 10M9 5l3 3-4 4-3-3 4-4ZM15 12l4 4-3 3-4-4 3-3ZM5 19c0-3.3 2.7-6 6-6M3 19c0-4.4 3.6-8 8-8"/></svg>',
  lock: '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="5" y="10" width="14" height="10" rx="1"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>',
  plug: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 12 3 7l4-4 5 5M16 12l5 5-4 4-5-5M9 15l6-6"/></svg>',
};

function getIcon(name) {
  return icons[name] ?? icons['shield-check'];
}

function getSelectedModules() {
  const names = Array.from(document.querySelectorAll('.proposal-price-group-header strong'))
    .map((element) => element.textContent?.trim())
    .filter(Boolean);

  return Array.from(new Set(names))
    .map((name) => moduleDescriptions[name])
    .filter(Boolean);
}

function createCompanySlide(modules) {
  const selectedModules = modules.length
    ? modules
    : [moduleDescriptions['Score | Análise cadastral']];

  const slide = document.createElement('section');
  slide.className = 'proposal-slide light proposal-company-integrated';
  slide.dataset.proposalSlide = '';
  slide.innerHTML = `
    <div class="proposal-slide-inner proposal-company-integrated-inner">
      <div class="proposal-company-integrated-intro">
        <div>
          <p class="proposal-kicker">VIA GATE · TECNOLOGIA E GESTÃO DE RISCO</p>
          <h2>Uma proposta conectada à <span>operação real</span>.</h2>
          <p class="proposal-lead">A ViaGate reúne validação cadastral, biometria, consultas, rastreabilidade, prevenção e integração para reduzir atrito operacional e aumentar a segurança das decisões.</p>
        </div>
        <div class="proposal-company-proof-grid">
          <article>
            <span>${getIcon('scan-face')}</span>
            <strong>Biometria facial</strong>
            <p>Validação de identidade com prova de vida integrada ao processo cadastral.</p>
          </article>
          <article>
            <span>${getIcon('lock')}</span>
            <strong>Segurança e LGPD</strong>
            <p>Fluxos com rastreabilidade, proteção de dados e controle de acesso às informações.</p>
          </article>
          <article>
            <span>${getIcon('plug')}</span>
            <strong>Integração</strong>
            <p>APIs e soluções configuráveis para conectar a ViaGate aos processos da operação.</p>
          </article>
        </div>
      </div>

      <div class="proposal-company-selected">
        <div class="proposal-company-selected-title">
          <small>SOLUÇÕES NESTA PROPOSTA</small>
          <strong>O conteúdo abaixo foi selecionado para esta negociação.</strong>
        </div>
        <div class="proposal-company-selected-grid">
          ${selectedModules.map((module) => `
            <article>
              <span>${getIcon(module.icon)}</span>
              <div>
                <strong>${module.title}</strong>
                <p>${module.description}</p>
              </div>
            </article>
          `).join('')}
        </div>
      </div>
    </div>
    <div class="proposal-slide-footer"><span>ViaGate</span><span data-slide-number></span></div>
  `;

  return slide;
}

function renumberSlides() {
  const slides = Array.from(document.querySelectorAll('[data-proposal-slide]'));
  const total = slides.length;

  slides.forEach((slide, index) => {
    const number = slide.querySelector('[data-slide-number]');
    if (number) {
      number.textContent = `${String(index + 1).padStart(2, '0')} / ${String(total).padStart(2, '0')}`;
    }
  });
}

function integrateCompanyContent() {
  const presentation = document.getElementById('proposalPresentation');
  if (!presentation || presentation.querySelector('.proposal-company-integrated')) {
    return Boolean(presentation?.querySelector('.proposal-company-integrated'));
  }

  const firstSlide = presentation.querySelector('[data-proposal-slide]');
  if (!firstSlide) {
    return false;
  }

  const companySlide = createCompanySlide(getSelectedModules());
  firstSlide.insertAdjacentElement('afterend', companySlide);
  renumberSlides();
  window.dispatchEvent(new Event('scroll'));
  return true;
}

function initialize() {
  if (integrateCompanyContent()) {
    return;
  }

  const presentation = document.getElementById('proposalPresentation');
  if (!presentation) {
    return;
  }

  const observer = new MutationObserver(() => {
    if (integrateCompanyContent()) {
      observer.disconnect();
    }
  });

  observer.observe(presentation, {
    childList: true,
    subtree: true,
  });

  window.setTimeout(() => observer.disconnect(), 8000);
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initialize, { once: true });
} else {
  initialize();
}
