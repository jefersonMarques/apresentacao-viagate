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
  arrow: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M5 12h14M13 6l6 6-6 6"/></svg>',
  message: '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 11.5a8 8 0 0 1-8.5 8A8.5 8.5 0 1 1 20 11.5Z"/><path d="m8 19-4 1 1-4"/></svg>',
  mail: '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="5" width="18" height="14" rx="1"/><path d="m4 7 8 6 8-6"/></svg>',
};

function getIcon(name) {
  return icons[name] ?? icons['shield-check'];
}

function resolveGroupStatus(group) {
  const statuses = Array.from(group.querySelectorAll('tbody tr td:nth-child(3)'))
    .map((cell) => cell.textContent?.trim().toLocaleLowerCase('pt-BR') ?? '')
    .filter(Boolean);

  if (!statuses.length) {
    return 'included';
  }

  const optionalCount = statuses.filter((status) => status.includes('opcional')).length;

  if (optionalCount === statuses.length) {
    return 'optional';
  }

  if (optionalCount > 0) {
    return 'mixed';
  }

  return 'included';
}

function getStatusLabel(status) {
  if (status === 'optional') {
    return 'Opcional';
  }

  if (status === 'mixed') {
    return 'Incluído + opcional';
  }

  return 'Incluído';
}

function getSelectedModules() {
  return Array.from(document.querySelectorAll('.proposal-price-group')).map((group) => {
    const name = group.querySelector('.proposal-price-group-header strong')?.textContent?.trim() ?? '';
    const module = moduleDescriptions[name];

    if (!module) {
      return null;
    }

    return {
      ...module,
      status: resolveGroupStatus(group),
    };
  }).filter(Boolean);
}

function createCompanySlide(modules) {
  const selectedModules = modules.length
    ? modules
    : [{ ...moduleDescriptions['Score | Análise cadastral'], status: 'included' }];

  const slide = document.createElement('section');
  slide.className = 'proposal-slide light proposal-company-integrated';
  slide.dataset.proposalSlide = '';
  slide.innerHTML = `
    <div class="proposal-slide-inner proposal-company-integrated-inner">
      <div class="proposal-company-integrated-intro">
        <div>
          <p class="proposal-kicker">RESUMO DA SOLUÇÃO RECOMENDADA</p>
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
          <small>CONFIGURAÇÃO RECOMENDADA</small>
          <strong>O que está sendo oferecido nesta negociação.</strong>
        </div>
        <div class="proposal-company-selected-grid">
          ${selectedModules.map((module) => `
            <article>
              <span>${getIcon(module.icon)}</span>
              <div>
                <div class="proposal-company-module-heading">
                  <strong>${module.title}</strong>
                  <em class="proposal-module-status ${module.status}">${getStatusLabel(module.status)}</em>
                </div>
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

function getPricingModelLabel() {
  const modelTitles = Array.from(document.querySelectorAll('.proposal-model-card strong'))
    .map((element) => element.textContent?.trim())
    .filter(Boolean);

  if (modelTitles.length > 1) {
    return 'Item + Conjunto';
  }

  return modelTitles[0] ?? 'Condição comercial personalizada';
}

function getCommercialHighlights() {
  return Array.from(document.querySelectorAll('.proposal-highlight')).map((highlight) => ({
    label: highlight.querySelector('small')?.textContent?.trim() ?? '',
    value: highlight.querySelector('strong')?.textContent?.trim() ?? '',
  })).filter((item) => item.label && item.value);
}

function getValidityText() {
  return document.querySelector('.proposal-validity strong')?.textContent?.trim() ?? '';
}

function createCommercialSummarySlide(modules) {
  const includedModules = modules.filter((module) => module.status !== 'optional');
  const optionalModules = modules.filter((module) => module.status === 'optional' || module.status === 'mixed');
  const highlights = getCommercialHighlights();
  const validity = getValidityText();
  const modelLabel = getPricingModelLabel();

  const slide = document.createElement('section');
  slide.className = 'proposal-slide dark proposal-commercial-summary';
  slide.dataset.proposalSlide = '';
  slide.innerHTML = `
    <div class="proposal-slide-inner proposal-commercial-summary-inner">
      <div class="proposal-commercial-summary-heading">
        <p class="proposal-kicker">RESUMO COMERCIAL</p>
        <h2>Os principais pontos da <span>negociação</span>.</h2>
        <p class="proposal-commercial-summary-copy">Uma visão consolidada para facilitar a avaliação e o encaminhamento interno desta proposta.</p>
      </div>

      <div class="proposal-commercial-summary-grid">
        <article class="proposal-commercial-summary-card primary">
          <small>MODELO COMERCIAL</small>
          <strong>${modelLabel}</strong>
        </article>

        ${highlights.map((item) => `
          <article class="proposal-commercial-summary-card">
            <small>${item.label}</small>
            <strong>${item.value}</strong>
          </article>
        `).join('')}

        ${validity ? `
          <article class="proposal-commercial-summary-card">
            <small>VALIDADE</small>
            <strong>${validity}</strong>
          </article>
        ` : ''}
      </div>

      <div class="proposal-commercial-products">
        <section>
          <small>INCLUÍDO NA PROPOSTA</small>
          <div>
            ${(includedModules.length ? includedModules : modules).map((module) => `
              <span>${getIcon(module.icon)}<strong>${module.title}</strong></span>
            `).join('')}
          </div>
        </section>
        ${optionalModules.length ? `
          <section>
            <small>OPORTUNIDADES ADICIONAIS</small>
            <div>
              ${optionalModules.map((module) => `
                <span>${getIcon(module.icon)}<strong>${module.title}</strong></span>
              `).join('')}
            </div>
          </section>
        ` : ''}
      </div>
    </div>
    <div class="proposal-slide-footer"><span>Resumo comercial</span><span data-slide-number></span></div>
  `;

  return slide;
}

function normalizeWhatsAppNumber(value) {
  const digits = String(value ?? '').replace(/\D/g, '');

  if (digits.length === 10 || digits.length === 11) {
    return `55${digits}`;
  }

  return digits.length >= 12 ? digits : '';
}

function getSalesContact() {
  const values = Array.from(document.querySelectorAll('.proposal-contact-data span'))
    .map((element) => element.textContent?.trim() ?? '')
    .filter(Boolean);
  const email = values.find((value) => value.includes('@')) ?? '';
  const phone = values.find((value) => !value.includes('@') && value.replace(/\D/g, '').length >= 10) ?? '';

  return {
    email,
    phone,
    whatsapp: normalizeWhatsAppNumber(phone),
  };
}

function enhanceFinalSlide() {
  const finalContent = document.querySelector('.proposal-contact-final');
  if (!finalContent || finalContent.dataset.nextStepEnhanced === 'true') {
    return;
  }

  finalContent.dataset.nextStepEnhanced = 'true';
  const leftColumn = finalContent.firstElementChild;
  if (!leftColumn) {
    return;
  }

  const contact = getSalesContact();
  const kicker = leftColumn.querySelector('.proposal-kicker');
  const title = leftColumn.querySelector('h2');
  const lead = leftColumn.querySelector('.proposal-lead');

  if (kicker) {
    kicker.textContent = 'PRÓXIMOS PASSOS';
  }

  if (title) {
    title.textContent = 'Vamos avançar com esta proposta?';
  }

  if (lead) {
    lead.textContent = 'Se as condições estiverem de acordo, podemos seguir com a formalização comercial e preparar os próximos passos da implantação.';
  } else {
    const copy = document.createElement('p');
    copy.className = 'proposal-lead proposal-next-step-copy';
    copy.textContent = 'Se as condições estiverem de acordo, podemos seguir com a formalização comercial e preparar os próximos passos da implantação.';
    leftColumn.appendChild(copy);
  }

  const actions = document.createElement('div');
  actions.className = 'proposal-next-actions';

  if (contact.whatsapp) {
    const message = encodeURIComponent('Olá, gostaria de conversar sobre a proposta comercial ViaGate.');
    actions.innerHTML += `
      <a class="proposal-next-action primary" href="https://wa.me/${contact.whatsapp}?text=${message}" target="_blank" rel="noopener noreferrer">
        ${getIcon('message')}
        <span>Falar com o comercial</span>
      </a>
    `;
  }

  if (contact.email) {
    const subject = encodeURIComponent('Ajuste na proposta comercial ViaGate');
    actions.innerHTML += `
      <a class="proposal-next-action" href="mailto:${contact.email}?subject=${subject}">
        ${getIcon('mail')}
        <span>Solicitar ajuste</span>
      </a>
    `;
  }

  if (actions.children.length) {
    leftColumn.appendChild(actions);
  }
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

function integrateDecisionFlow() {
  const presentation = document.getElementById('proposalPresentation');
  if (!presentation || presentation.dataset.decisionFlowReady === 'true') {
    return presentation?.dataset.decisionFlowReady === 'true';
  }

  const firstSlide = presentation.querySelector('[data-proposal-slide]');
  const finalSlide = presentation.querySelector('.proposal-contact-final')?.closest('[data-proposal-slide]');

  if (!firstSlide || !finalSlide) {
    return false;
  }

  const modules = getSelectedModules();
  const companySlide = createCompanySlide(modules);
  firstSlide.insertAdjacentElement('afterend', companySlide);

  const conditionsSlide = Array.from(presentation.querySelectorAll('[data-proposal-slide]')).find((slide) => (
    slide.querySelector('.proposal-validity')
    || slide.textContent?.includes('OBSERVAÇÕES E CONDIÇÕES')
  ));
  const summarySlide = createCommercialSummarySlide(modules);

  if (conditionsSlide) {
    conditionsSlide.insertAdjacentElement('beforebegin', summarySlide);
  } else {
    finalSlide.insertAdjacentElement('beforebegin', summarySlide);
  }

  enhanceFinalSlide();
  presentation.dataset.decisionFlowReady = 'true';
  renumberSlides();
  window.dispatchEvent(new Event('scroll'));
  return true;
}

function initialize() {
  if (integrateDecisionFlow()) {
    return;
  }

  const presentation = document.getElementById('proposalPresentation');
  if (!presentation) {
    return;
  }

  const observer = new MutationObserver(() => {
    if (integrateDecisionFlow()) {
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
