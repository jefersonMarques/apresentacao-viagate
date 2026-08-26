import { supabase } from './supabase.js';

const priceStoragePrefix = 'viagate.proposal.prices.v1';

const proposalTemplates = [
  {
    id: 'score-item',
    title: 'Score por Item',
    description: 'Análise cadastral com motorista e veículo cobrados separadamente.',
    model: 'per_item',
    groups: ['score'],
    optionalGroups: [],
  },
  {
    id: 'score-bundle',
    title: 'Score por Conjunto',
    description: 'Motorista, veículos e colaboradores em uma única composição.',
    model: 'bundle',
    groups: ['score'],
    optionalGroups: [],
  },
  {
    id: 'score-both',
    title: 'Item + Conjunto',
    description: 'Apresenta as duas modalidades para o cliente comparar.',
    model: 'item_and_bundle',
    groups: ['score'],
    optionalGroups: [],
  },
  {
    id: 'score-truck',
    title: 'Score + Cargo Truck',
    description: 'Análise cadastral combinada com aplicativo e logística.',
    model: 'per_item',
    groups: ['score', 'logistics'],
    optionalGroups: [],
  },
  {
    id: 'complete',
    title: 'Operação completa',
    description: 'Score como solução principal e demais módulos como oportunidades adicionais.',
    model: 'item_and_bundle',
    groups: ['score'],
    optionalGroups: ['authentication', 'logistics', 'prevention', 'monitoring'],
  },
];

const standardConditions = [
  {
    id: 'score-turnaround',
    text: 'Tempo médio de retorno das pesquisas cadastrais em até 10 minutos após a realização da biometria facial pelo pesquisado.',
    groups: ['score'],
  },
  {
    id: 'score-biometry',
    text: 'Biometria facial inclusa nos custos de cadastro e consulta.',
    groups: ['score'],
  },
  {
    id: 'score-channels',
    text: 'Disponibilização da biometria via link, ambiente web e aplicativo.',
    groups: ['score'],
  },
  {
    id: 'support',
    text: 'Após o fechamento do contrato, poderá ser criado um grupo em aplicativo de mensagens para atendimento operacional ao parceiro ViaGate.',
    groups: ['all'],
  },
  {
    id: 'victimology-pricing',
    text: 'Consultas de vitimologia possuem precificação de acordo com o estado consultado.',
    groups: ['authentication'],
  },
  {
    id: 'victimology-turnaround',
    text: 'Prazo de retorno das solicitações de vitimologia em até 3 horas, dentro do horário operacional informado pela ViaGate.',
    groups: ['authentication'],
  },
  {
    id: 'logistics-scope',
    text: 'Os serviços de logística contemplam a gestão da viagem dentro das funcionalidades disponibilizadas pela Plataforma Cargo e não substituem, isoladamente, todas as regras de Gerenciamento de Riscos.',
    groups: ['logistics'],
  },
  {
    id: 'integration-analysis',
    text: 'Integrações entre sistemas estão sujeitas à análise técnica prévia da documentação e dos requisitos da operação.',
    groups: ['authentication', 'logistics', 'monitoring'],
  },
  {
    id: 'customization',
    text: 'Customizações não previstas na proposta dependem de levantamento de requisitos e aprovação comercial específica.',
    groups: ['all'],
  },
  {
    id: 'travel-expenses',
    text: 'Despesas eventuais com deslocamento, alimentação e hospedagem não estão contempladas, salvo quando expressamente indicadas nesta proposta.',
    groups: ['all'],
  },
];

const state = {
  initialized: false,
  userId: 'anonymous',
  priceDefaults: {},
  observer: null,
  conditionSelection: new Set(),
  customConditions: [],
};

function getPriceStorageKey() {
  return `${priceStoragePrefix}.${state.userId}`;
}

function loadPriceDefaults() {
  try {
    const value = window.localStorage.getItem(getPriceStorageKey());
    state.priceDefaults = value ? JSON.parse(value) : {};
  } catch {
    state.priceDefaults = {};
  }
}

function savePriceDefaults() {
  try {
    window.localStorage.setItem(getPriceStorageKey(), JSON.stringify(state.priceDefaults));
  } catch {
    return;
  }
}

function getCatalogCards() {
  return Array.from(document.querySelectorAll('[data-catalog-item]'));
}

function getCardGroup(card) {
  return card.closest('[data-product-group]')?.dataset.productGroup ?? '';
}

function getCardStatus(card) {
  const enabledInput = card.querySelector('[data-service-enabled]');
  const optionalInput = card.querySelector('[data-service-optional]');

  if (!enabledInput?.checked) {
    return 'off';
  }

  return optionalInput?.checked ? 'optional' : 'included';
}

function setCardStatus(card, status) {
  const enabledInput = card.querySelector('[data-service-enabled]');
  const optionalInput = card.querySelector('[data-service-optional]');

  if (!enabledInput || !optionalInput) {
    return;
  }

  const shouldEnable = status !== 'off';
  const shouldBeOptional = status === 'optional';

  if (enabledInput.checked !== shouldEnable) {
    enabledInput.checked = shouldEnable;
    enabledInput.dispatchEvent(new Event('change', { bubbles: true }));
  }

  window.setTimeout(() => {
    const currentCard = document.querySelector(`[data-catalog-item="${CSS.escape(card.dataset.catalogItem)}"]`);
    const currentOptional = currentCard?.querySelector('[data-service-optional]');

    if (currentOptional && currentOptional.checked !== shouldBeOptional) {
      currentOptional.checked = shouldBeOptional;
      currentOptional.dispatchEvent(new Event('change', { bubbles: true }));
    }

    refreshEnhancements();
  }, 0);
}

function createStatusControl(card) {
  if (card.querySelector('[data-generator-status]')) {
    updateStatusControl(card);
    return;
  }

  const control = document.createElement('div');
  control.className = 'proposal-generator-status';
  control.dataset.generatorStatus = 'true';
  control.setAttribute('role', 'group');
  control.setAttribute('aria-label', 'Status do serviço na proposta');
  control.innerHTML = `
    <button type="button" data-status="off">Não oferecer</button>
    <button type="button" data-status="included">Incluído</button>
    <button type="button" data-status="optional">Opcional</button>
  `;

  control.querySelectorAll('[data-status]').forEach((button) => {
    button.addEventListener('click', () => setCardStatus(card, button.dataset.status));
  });

  const copy = card.querySelector('.proposal-service-copy');
  copy?.insertAdjacentElement('afterend', control);
  card.classList.add('proposal-generator-service');
  updateStatusControl(card);
}

function updateStatusControl(card) {
  const status = getCardStatus(card);
  card.dataset.generatorServiceStatus = status;

  card.querySelectorAll('[data-generator-status] [data-status]').forEach((button) => {
    const active = button.dataset.status === status;
    button.classList.toggle('active', active);
    button.setAttribute('aria-pressed', active ? 'true' : 'false');
  });
}

function applyRememberedPrice(card) {
  const input = card.querySelector('[data-service-price]');
  const itemId = card.dataset.catalogItem;

  if (!input || !itemId || input.value || !state.priceDefaults[itemId]) {
    return;
  }

  input.value = state.priceDefaults[itemId];

  if (getCardStatus(card) !== 'off') {
    input.dispatchEvent(new Event('input', { bubbles: true }));
  }
}

function bindPriceMemory(card) {
  const input = card.querySelector('[data-service-price]');
  if (!input || input.dataset.priceMemoryBound === 'true') {
    return;
  }

  input.dataset.priceMemoryBound = 'true';
  input.addEventListener('change', () => {
    const itemId = card.dataset.catalogItem;
    const value = input.value.trim();

    if (!itemId || !value || Number(value) < 0) {
      return;
    }

    state.priceDefaults[itemId] = value;
    savePriceDefaults();
  });
}

function getSelectedGroups() {
  const groups = new Set();

  getCatalogCards().forEach((card) => {
    if (getCardStatus(card) !== 'off') {
      const group = getCardGroup(card);
      if (group) {
        groups.add(group);
      }
    }
  });

  return groups;
}

function getApplicableConditions() {
  const groups = getSelectedGroups();

  return standardConditions.filter((condition) => (
    condition.groups.includes('all') || condition.groups.some((group) => groups.has(group))
  ));
}

function splitConditionLines(value) {
  return String(value ?? '')
    .split('\n')
    .map((item) => item.trim())
    .filter(Boolean);
}

function initializeConditionState() {
  const textarea = document.getElementById('commercialConditions');
  if (!textarea) {
    return;
  }

  const existingLines = splitConditionLines(textarea.value);
  const standardTexts = new Set(standardConditions.map((condition) => condition.text));

  state.customConditions = existingLines.filter((line) => !standardTexts.has(line));
  existingLines.forEach((line) => {
    const condition = standardConditions.find((candidate) => candidate.text === line);
    if (condition) {
      state.conditionSelection.add(condition.id);
    }
  });
}

function syncConditionTextarea() {
  const textarea = document.getElementById('commercialConditions');
  if (!textarea) {
    return;
  }

  const selectedStandard = standardConditions
    .filter((condition) => state.conditionSelection.has(condition.id))
    .map((condition) => condition.text);

  textarea.value = [...selectedStandard, ...state.customConditions].join('\n');
  textarea.dispatchEvent(new Event('input', { bubbles: true }));
}

function renderConditionLibrary() {
  const container = document.getElementById('proposalConditionLibrary');
  if (!container) {
    return;
  }

  const applicable = getApplicableConditions();
  const applicableIds = new Set(applicable.map((condition) => condition.id));

  standardConditions.forEach((condition) => {
    if (!applicableIds.has(condition.id)) {
      state.conditionSelection.delete(condition.id);
    } else if (!state.conditionSelection.has(condition.id)) {
      state.conditionSelection.add(condition.id);
    }
  });

  container.innerHTML = applicable.map((condition) => `
    <label class="proposal-condition-option">
      <input type="checkbox" value="${condition.id}" ${state.conditionSelection.has(condition.id) ? 'checked' : ''} />
      <span>${condition.text}</span>
    </label>
  `).join('') || '<p class="proposal-condition-empty">Selecione ao menos um produto para sugerir condições comerciais relacionadas.</p>';

  container.querySelectorAll('input[type="checkbox"]').forEach((input) => {
    input.addEventListener('change', () => {
      if (input.checked) {
        state.conditionSelection.add(input.value);
      } else {
        state.conditionSelection.delete(input.value);
      }
      syncConditionTextarea();
    });
  });

  syncConditionTextarea();
}

function bindCustomConditions() {
  const input = document.getElementById('proposalCustomConditions');
  if (!input || input.dataset.bound === 'true') {
    return;
  }

  input.dataset.bound = 'true';
  input.value = state.customConditions.join('\n');
  input.addEventListener('input', () => {
    state.customConditions = splitConditionLines(input.value);
    syncConditionTextarea();
  });
}

function enhanceConditionsSection() {
  const textarea = document.getElementById('commercialConditions');
  const section = textarea?.closest('.editor-section');
  const body = section?.querySelector('.editor-section-body');

  if (!textarea || !section || !body || section.dataset.generatorConditions === 'true') {
    return;
  }

  section.dataset.generatorConditions = 'true';
  section.querySelector('.editor-section-header h2').textContent = 'Condições comerciais';

  const originalField = textarea.closest('.form-field');
  originalField?.classList.add('proposal-generator-source-condition');

  const wrapper = document.createElement('div');
  wrapper.className = 'proposal-condition-builder';
  wrapper.innerHTML = `
    <div class="proposal-generator-step-heading">
      <span>4</span>
      <div>
        <h3>Condições aplicadas automaticamente</h3>
        <p>As condições abaixo são sugeridas conforme os produtos escolhidos. Desmarque somente o que não fizer sentido nesta negociação.</p>
      </div>
    </div>
    <div class="proposal-condition-library" id="proposalConditionLibrary"></div>
    <details class="proposal-generator-custom-conditions">
      <summary>Adicionar observações específicas</summary>
      <textarea id="proposalCustomConditions" rows="5" placeholder="Uma condição específica por linha."></textarea>
    </details>
  `;

  body.insertBefore(wrapper, body.firstChild);
  initializeConditionState();
  bindCustomConditions();
  renderConditionLibrary();
}

function setPricingModel(model) {
  const button = document.querySelector(`[data-pricing-model="${CSS.escape(model)}"]`);
  if (button) {
    button.click();
    return;
  }

  const select = document.getElementById('pricingModel');
  if (select) {
    select.value = model;
    select.dispatchEvent(new Event('change', { bubbles: true }));
  }
}

function applyTemplate(template) {
  setPricingModel(template.model);

  window.setTimeout(() => {
    getCatalogCards().forEach((card) => {
      const group = getCardGroup(card);
      const status = template.groups.includes(group)
        ? 'included'
        : template.optionalGroups.includes(group)
          ? 'optional'
          : 'off';

      setCardStatus(card, status);
    });

    window.setTimeout(() => {
      refreshEnhancements();
      renderConditionLibrary();
    }, 80);
  }, 80);
}

function createTemplateSelector() {
  if (document.getElementById('proposalTemplateSelector')) {
    return;
  }

  const modelBlock = document.querySelector('.proposal-model-block');
  if (!modelBlock) {
    return;
  }

  const section = document.createElement('section');
  section.className = 'proposal-template-block';
  section.id = 'proposalTemplateSelector';
  section.innerHTML = `
    <div class="proposal-generator-step-heading">
      <span>1</span>
      <div>
        <h3>Comece por um modelo pronto</h3>
        <p>O modelo apenas pré-seleciona a composição. Tudo pode ser ajustado antes de publicar.</p>
      </div>
    </div>
    <div class="proposal-template-grid">
      ${proposalTemplates.map((template) => `
        <button type="button" class="proposal-template-option" data-proposal-template="${template.id}">
          <strong>${template.title}</strong>
          <small>${template.description}</small>
        </button>
      `).join('')}
    </div>
    <p class="proposal-price-memory-note">Os últimos valores usados por este usuário neste navegador são reaproveitados automaticamente quando o campo estiver vazio.</p>
  `;

  modelBlock.insertAdjacentElement('beforebegin', section);

  section.querySelectorAll('[data-proposal-template]').forEach((button) => {
    button.addEventListener('click', () => {
      const template = proposalTemplates.find((candidate) => candidate.id === button.dataset.proposalTemplate);
      if (template) {
        applyTemplate(template);
      }
    });
  });

  const originalHeading = modelBlock.querySelector('.proposal-configurator-heading > div > span');
  if (originalHeading) {
    originalHeading.textContent = '2';
  }

  const productStep = document.querySelector('.proposal-configurator-step');
  if (productStep) {
    productStep.textContent = '3';
  }
}

function refreshEnhancements() {
  getCatalogCards().forEach((card) => {
    createStatusControl(card);
    applyRememberedPrice(card);
    bindPriceMemory(card);
  });

  renderConditionLibrary();
}

function bindCatalogObserver() {
  const catalog = document.getElementById('proposalProductCatalog');
  if (!catalog || state.observer) {
    return;
  }

  let scheduled = false;
  state.observer = new MutationObserver(() => {
    if (scheduled) {
      return;
    }

    scheduled = true;
    window.requestAnimationFrame(() => {
      scheduled = false;
      refreshEnhancements();
    });
  });

  state.observer.observe(catalog, { childList: true, subtree: true });

  catalog.addEventListener('change', () => {
    window.setTimeout(() => {
      refreshEnhancements();
      renderConditionLibrary();
    }, 0);
  });
}

function appendStylesheet() {
  if (document.querySelector('link[data-proposal-generator-flow]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './proposal-generator-flow.css?v=20260826-1';
  link.dataset.proposalGeneratorFlow = 'true';
  document.head.appendChild(link);
}

async function waitForConfigurator(attempts = 100) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    if (document.querySelector('.proposal-model-block') && document.getElementById('proposalProductCatalog')) {
      return true;
    }
    await new Promise((resolve) => window.setTimeout(resolve, 100));
  }

  return false;
}

async function initializeUser() {
  try {
    const { data } = await supabase?.auth.getUser();
    state.userId = data?.user?.id || 'anonymous';
  } catch {
    state.userId = 'anonymous';
  }

  loadPriceDefaults();
}

async function initialize() {
  if (state.initialized) {
    return;
  }

  const ready = await waitForConfigurator();
  if (!ready) {
    return;
  }

  state.initialized = true;
  appendStylesheet();
  await initializeUser();
  createTemplateSelector();
  enhanceConditionsSection();
  bindCatalogObserver();
  refreshEnhancements();
}

initialize().catch((error) => {
  console.error('Não foi possível inicializar o fluxo simplificado de propostas.', error);
});
