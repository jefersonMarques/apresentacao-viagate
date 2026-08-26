const pricingCatalog = [
  {
    id: 'score',
    title: 'Cargo Score | Análise cadastral',
    shortTitle: 'Cargo Score',
    icon: 'shield-check',
    summary: 'Pesquisa cadastral com autorização biométrica, validações oficiais e análise de risco para motorista e veículo.',
    items: [
      { id: 'score-item-driver-register', label: 'Cadastro | Motorista — Frota, agregado e terceiro', unit: 'cadastro', models: ['per_item', 'item_and_bundle'] },
      { id: 'score-item-driver-other', label: 'Cadastro | Motorista — Outras funções', unit: 'cadastro', models: ['per_item', 'item_and_bundle'] },
      { id: 'score-item-vehicle-register', label: 'Cadastro | Veículos', unit: 'cadastro', models: ['per_item', 'item_and_bundle'] },
      { id: 'score-item-driver-query', label: 'Consulta | Motorista — Frota, agregado e terceiro', unit: 'consulta', models: ['per_item', 'item_and_bundle'] },
      { id: 'score-item-vehicle-query', label: 'Consulta | Veículos', unit: 'consulta', models: ['per_item', 'item_and_bundle'] },
      { id: 'score-item-reanalysis', label: 'Reanálise de campos preenchidos incorretamente', unit: 'reanálise', models: ['per_item', 'item_and_bundle'] },
      { id: 'score-bundle-register', label: 'Cadastro | Motorista + veículos + colaboradores', unit: 'conjunto', models: ['bundle', 'item_and_bundle'] },
      { id: 'score-bundle-query', label: 'Consulta | Motorista + veículos + colaboradores', unit: 'conjunto', models: ['bundle', 'item_and_bundle'] },
      { id: 'score-bundle-reanalysis', label: 'Reanálise | Conjunto', unit: 'reanálise', models: ['bundle', 'item_and_bundle'] },
    ],
  },
  {
    id: 'authentication',
    title: 'Consultas e autenticação',
    shortTitle: 'Consultas e autenticação',
    icon: 'scan-face',
    summary: 'Consultas pontuais, autenticação de formulários e validações complementares ao processo cadastral.',
    items: [
      { id: 'auth-cargo', label: 'Cargo Autenticador', unit: 'consulta', defaultOptional: true },
      { id: 'auth-lawsuits', label: 'Pesquisa processo criminal, trabalhista, cível e familiar', unit: 'consulta', defaultOptional: true },
      { id: 'auth-victimology-state', label: 'Vitimologia por estado', unit: 'estado', defaultOptional: true },
      { id: 'auth-victimology-integrated', label: 'Vitimologia integrada', unit: 'consulta', defaultOptional: true },
      { id: 'auth-antt', label: 'Consulta veículos | ANTT', unit: 'consulta', defaultOptional: true },
      { id: 'auth-on-demand', label: 'Avulso | Consultas e autenticação de formulários', unit: 'consulta', defaultOptional: true },
    ],
  },
  {
    id: 'logistics',
    title: 'Cargo Truck | Aplicativo e logística',
    shortTitle: 'Cargo Truck',
    icon: 'route',
    summary: 'Aplicativo para cadastro, coletas, entregas, eventos de parada e rastreamento por GPS do smartphone do motorista.',
    items: [
      { id: 'truck-first-without-score', label: 'Cargo Truck | Primeira viagem (sem Score)', unit: 'viagem', defaultOptional: true },
      { id: 'truck-next-without-score', label: 'Cargo Truck | Viagens subsequentes (sem Score)', unit: 'viagem', defaultOptional: true },
      { id: 'truck-with-score', label: 'Cargo Truck | Primeira viagem e subsequentes (com Score)', unit: 'viagem', defaultOptional: true },
    ],
  },
  {
    id: 'prevention',
    title: 'Prevenção',
    shortTitle: 'Prevenção',
    icon: 'shield-alert',
    summary: 'Recursos complementares para gestão preventiva, incluindo multas e histórico veicular completo.',
    items: [
      { id: 'prevention-fines', label: 'Sistema Gestor de Multas', unit: 'veículo', defaultOptional: true },
      { id: 'prevention-history', label: 'Histórico Veicular Completo', unit: 'consulta', defaultOptional: true },
    ],
  },
  {
    id: 'monitoring',
    title: 'Monitoramento de veículos | Integração com gerenciadora',
    shortTitle: 'Monitoramento de veículos',
    icon: 'satellite',
    summary: 'Monitoramento satelital via integração com gerenciadora, com opções por veículo, viagem e checklist.',
    items: [
      { id: 'monitoring-fixed', label: 'Veículo Fixo', unit: 'veículo', defaultOptional: true },
      { id: 'monitoring-trip', label: 'Viagem Avulsa', unit: 'viagem', defaultOptional: true },
      { id: 'monitoring-autotrac', label: 'ADE Autotrac', unit: 'veículo', defaultOptional: true },
      { id: 'monitoring-checklist', label: 'Check List', unit: 'veículo', defaultOptional: true },
    ],
  },
];

const pricingModels = [
  {
    id: 'per_item',
    title: 'Análise por item',
    icon: 'list-checks',
    summary: 'Cada cadastro ou consulta é processado individualmente, com valor separado para motorista e veículo.',
  },
  {
    id: 'bundle',
    title: 'Análise por conjunto',
    icon: 'combine',
    summary: 'Motorista e veículos são processados de forma unificada, simplificando cadastro e consulta.',
  },
  {
    id: 'item_and_bundle',
    title: 'Item + conjunto',
    icon: 'columns-3',
    summary: 'Apresente as duas modalidades na mesma proposta e informe os valores de cada alternativa.',
  },
  {
    id: 'custom',
    title: 'Condições específicas',
    icon: 'settings-2',
    summary: 'Use somente quando a negociação não se enquadrar nos modelos comerciais padronizados.',
  },
];

const state = {
  initialized: false,
  syncing: false,
  priceCache: new Map(),
};

function appendStylesheet() {
  if (document.querySelector('link[data-proposal-configurator]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './proposal-configurator.css?v=20260825-1';
  link.dataset.proposalConfigurator = 'true';
  document.head.appendChild(link);
}

function ensurePricingModels() {
  const select = document.getElementById('pricingModel');
  if (!select) {
    return;
  }

  if (!select.querySelector('option[value="item_and_bundle"]')) {
    const option = document.createElement('option');
    option.value = 'item_and_bundle';
    option.textContent = 'Análise por item + conjunto';
    select.insertBefore(option, select.querySelector('option[value="custom"]') ?? null);
  }
}

function getPricingModel() {
  return document.getElementById('pricingModel')?.value || 'per_item';
}

function getSourceRows() {
  return Array.from(document.querySelectorAll('#pricingItemsBody tr'));
}

function readSourceRow(row) {
  return {
    label: row.querySelector('[data-pricing-label]')?.value?.trim() ?? '',
    unit: row.querySelector('[data-pricing-unit]')?.value?.trim() ?? '',
    price: row.querySelector('[data-pricing-price]')?.value ?? '',
    isOptional: row.querySelector('[data-pricing-optional]')?.value === 'true',
  };
}

function getCatalogItemByLabel(label) {
  for (const group of pricingCatalog) {
    const item = group.items.find((candidate) => candidate.label === label);
    if (item) {
      return { group, item };
    }
  }

  return null;
}

function findSourceRow(label) {
  return getSourceRows().find((row) => readSourceRow(row).label === label) ?? null;
}

function removeBlankSourceRows() {
  getSourceRows().forEach((row) => {
    if (!readSourceRow(row).label) {
      row.remove();
    }
  });
}

function escapeAttribute(value) {
  return String(value ?? '')
    .replaceAll('&', '&amp;')
    .replaceAll('"', '&quot;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;');
}

function createSourceRow(item, values = {}) {
  const row = document.createElement('tr');
  row.dataset.configuratorItem = item.id;
  row.innerHTML = `
    <td><input data-pricing-label value="${escapeAttribute(item.label)}" required /></td>
    <td><input data-pricing-unit value="${escapeAttribute(values.unit ?? item.unit)}" /></td>
    <td><input data-pricing-price type="number" min="0" step="0.01" value="${escapeAttribute(values.price ?? '')}" /></td>
    <td>
      <select data-pricing-optional>
        <option value="false" ${values.isOptional ? '' : 'selected'}>Não</option>
        <option value="true" ${values.isOptional ? 'selected' : ''}>Sim</option>
      </select>
    </td>
    <td><button class="link-button" type="button" data-remove-pricing-item>Remover</button></td>
  `;

  row.querySelector('[data-remove-pricing-item]')?.addEventListener('click', () => {
    row.remove();
    window.setTimeout(syncFromSource, 0);
  });

  document.getElementById('pricingItemsBody')?.appendChild(row);
  return row;
}

function updateSourceRow(item, values) {
  let row = findSourceRow(item.label);

  if (!row) {
    row = createSourceRow(item, values);
  }

  const unitInput = row.querySelector('[data-pricing-unit]');
  const priceInput = row.querySelector('[data-pricing-price]');
  const optionalSelect = row.querySelector('[data-pricing-optional]');

  if (unitInput) unitInput.value = values.unit ?? item.unit;
  if (priceInput) priceInput.value = values.price ?? '';
  if (optionalSelect) optionalSelect.value = values.isOptional ? 'true' : 'false';
}

function removeSourceRow(item) {
  const row = findSourceRow(item.label);
  if (!row) {
    return;
  }

  const values = readSourceRow(row);
  state.priceCache.set(item.id, values);
  row.remove();
}

function getVisibleItems(group) {
  const model = getPricingModel();

  return group.items.filter((item) => {
    if (!item.models?.length) {
      return true;
    }

    if (model === 'custom') {
      return true;
    }

    return item.models.includes(model);
  });
}

function renderModelSelector() {
  const container = document.getElementById('proposalModelSelector');
  if (!container) {
    return;
  }

  const selectedModel = getPricingModel();
  container.innerHTML = pricingModels.map((model) => `
    <button class="proposal-model-option ${model.id === selectedModel ? 'selected' : ''}" type="button" data-pricing-model="${model.id}">
      <span class="proposal-model-icon"><i data-lucide="${model.icon}"></i></span>
      <strong>${model.title}</strong>
      <small>${model.summary}</small>
    </button>
  `).join('');

  container.querySelectorAll('[data-pricing-model]').forEach((button) => {
    button.addEventListener('click', () => selectPricingModel(button.dataset.pricingModel));
  });

  window.lucide?.createIcons();
}

function renderCatalog() {
  const container = document.getElementById('proposalProductCatalog');
  if (!container) {
    return;
  }

  container.innerHTML = pricingCatalog.map((group) => {
    const visibleItems = getVisibleItems(group);

    if (!visibleItems.length) {
      return '';
    }

    return `
      <section class="proposal-product-group" data-product-group="${group.id}">
        <header class="proposal-product-header">
          <span class="proposal-product-icon"><i data-lucide="${group.icon}"></i></span>
          <div>
            <h3>${group.title}</h3>
            <p>${group.summary}</p>
          </div>
        </header>
        <div class="proposal-service-list">
          ${visibleItems.map((item) => renderCatalogItem(item)).join('')}
        </div>
      </section>
    `;
  }).join('');

  bindCatalogEvents();
  updateProposalSummary();
  window.lucide?.createIcons();
}

function renderCatalogItem(item) {
  const row = findSourceRow(item.label);
  const source = row ? readSourceRow(row) : state.priceCache.get(item.id);
  const selected = Boolean(row);
  const unit = source?.unit || item.unit;
  const price = source?.price ?? '';
  const isOptional = source?.isOptional ?? item.defaultOptional ?? false;

  return `
    <article class="proposal-service ${selected ? 'selected' : ''}" data-catalog-item="${item.id}" data-catalog-label="${escapeAttribute(item.label)}">
      <label class="proposal-service-toggle">
        <input type="checkbox" data-service-enabled ${selected ? 'checked' : ''} />
        <span aria-hidden="true"></span>
      </label>
      <div class="proposal-service-copy">
        <strong>${item.label}</strong>
        <small>${unit}</small>
      </div>
      <div class="proposal-service-value">
        <label>Valor</label>
        <div class="proposal-currency-input">
          <span>R$</span>
          <input type="number" min="0" step="0.01" data-service-price value="${escapeAttribute(price)}" ${selected ? '' : 'disabled'} />
        </div>
      </div>
      <label class="proposal-service-optional">
        <input type="checkbox" data-service-optional ${isOptional ? 'checked' : ''} ${selected ? '' : 'disabled'} />
        <span>Opcional</span>
      </label>
    </article>
  `;
}

function bindCatalogEvents() {
  document.querySelectorAll('[data-catalog-item]').forEach((card) => {
    const item = pricingCatalog.flatMap((group) => group.items).find((candidate) => candidate.id === card.dataset.catalogItem);
    if (!item) {
      return;
    }

    const enabled = card.querySelector('[data-service-enabled]');
    const price = card.querySelector('[data-service-price]');
    const optional = card.querySelector('[data-service-optional]');

    enabled?.addEventListener('change', () => {
      if (enabled.checked) {
        const cached = state.priceCache.get(item.id) ?? {};
        updateSourceRow(item, {
          unit: cached.unit || item.unit,
          price: cached.price ?? price?.value ?? '',
          isOptional: cached.isOptional ?? item.defaultOptional ?? optional?.checked ?? false,
        });
      } else {
        removeSourceRow(item);
      }

      syncFromSource();
    });

    price?.addEventListener('input', () => {
      if (!enabled?.checked) {
        return;
      }

      updateSourceRow(item, {
        unit: item.unit,
        price: price.value,
        isOptional: Boolean(optional?.checked),
      });
      updateGeneratedScope();
      updateProposalSummary();
    });

    optional?.addEventListener('change', () => {
      if (!enabled?.checked) {
        return;
      }

      updateSourceRow(item, {
        unit: item.unit,
        price: price?.value ?? '',
        isOptional: optional.checked,
      });
      updateProposalSummary();
    });
  });
}

function selectPricingModel(model) {
  const select = document.getElementById('pricingModel');
  if (!select || select.value === model) {
    return;
  }

  pricingCatalog[0].items.forEach((item) => {
    const row = findSourceRow(item.label);
    if (!row) {
      return;
    }

    const isVisibleInNextModel = model === 'custom' || !item.models?.length || item.models.includes(model);
    if (!isVisibleInNextModel) {
      removeSourceRow(item);
    }
  });

  select.value = model;
  select.dispatchEvent(new Event('change', { bubbles: true }));
  renderModelSelector();
  renderCatalog();
  updateGeneratedScope();
}

function getSelectedGroups() {
  const selectedLabels = new Set(
    getSourceRows()
      .map((row) => readSourceRow(row).label)
      .filter(Boolean),
  );

  return pricingCatalog.filter((group) => group.items.some((item) => selectedLabels.has(item.label)));
}

function updateGeneratedScope() {
  const selectedGroups = getSelectedGroups();
  const titleInput = document.getElementById('solutionTitle');
  const scopeInput = document.getElementById('solutionScope');

  if (!titleInput || !scopeInput) {
    return;
  }

  titleInput.value = selectedGroups.map((group) => group.shortTitle).join(' + ');
  scopeInput.value = selectedGroups.map((group) => `${group.shortTitle} | ${group.summary}`).join('\n');
}

function formatCurrency(value) {
  const number = Number(value || 0);
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
  }).format(Number.isFinite(number) ? number : 0);
}

function updateProposalSummary() {
  const container = document.getElementById('proposalCompositionSummary');
  if (!container) {
    return;
  }

  const selectedRows = getSourceRows().filter((row) => readSourceRow(row).label);
  const selectedGroups = getSelectedGroups();
  const minimumInvoice = document.getElementById('minimumInvoice')?.value ?? '';
  const setupFee = document.getElementById('setupFee')?.value ?? '';

  container.innerHTML = `
    <div class="proposal-summary-heading">
      <span>${selectedRows.length}</span>
      <div><strong>serviço${selectedRows.length === 1 ? '' : 's'} selecionado${selectedRows.length === 1 ? '' : 's'}</strong><small>Composição atual da proposta</small></div>
    </div>
    ${selectedGroups.length ? `
      <div class="proposal-summary-products">
        ${selectedGroups.map((group) => `<span><i data-lucide="${group.icon}"></i>${group.shortTitle}</span>`).join('')}
      </div>
    ` : '<p class="proposal-summary-empty">Selecione os produtos e serviços que farão parte desta proposta.</p>'}
    ${(Number(minimumInvoice) > 0 || Number(setupFee) > 0) ? `
      <div class="proposal-summary-financial">
        ${Number(minimumInvoice) > 0 ? `<div><small>Fatura mínima</small><strong>${formatCurrency(minimumInvoice)}</strong></div>` : ''}
        ${Number(setupFee) > 0 ? `<div><small>Implantação</small><strong>${formatCurrency(setupFee)}</strong></div>` : ''}
      </div>
    ` : ''}
  `;

  window.lucide?.createIcons();
}

function syncFromSource() {
  if (state.syncing) {
    return;
  }

  state.syncing = true;
  removeBlankSourceRows();
  renderModelSelector();
  renderCatalog();
  updateGeneratedScope();
  updateProposalSummary();
  state.syncing = false;
}

function moveAdvancedFields(contextSection) {
  if (contextSection.querySelector('.proposal-advanced-fields')) {
    return;
  }

  const body = contextSection.querySelector('.editor-section-body');
  const grid = body?.querySelector('.form-grid');
  if (!body || !grid) {
    return;
  }

  const details = document.createElement('details');
  details.className = 'proposal-advanced-fields';
  details.innerHTML = `
    <summary><i data-lucide="sliders-horizontal"></i><span><strong>Personalização avançada</strong><small>Contexto, prioridades e textos adicionais para negociações específicas.</small></span></summary>
    <div class="proposal-advanced-fields-body"></div>
  `;

  const detailsBody = details.querySelector('.proposal-advanced-fields-body');
  ['operationContext', 'customerPriorities', 'solutionTitle', 'solutionScope'].forEach((id) => {
    const field = document.getElementById(id)?.closest('.form-field');
    if (field) {
      detailsBody.appendChild(field);
    }
  });

  const modelField = document.getElementById('pricingModel')?.closest('.form-field');
  if (modelField) {
    modelField.classList.add('proposal-configurator-source-field');
  }

  const modelBlock = document.createElement('section');
  modelBlock.className = 'proposal-model-block';
  modelBlock.innerHTML = `
    <div class="proposal-configurator-heading">
      <div><span>1</span><h3>Como será feita a análise cadastral?</h3></div>
      <p>Escolha o modelo comercial. Os itens do Cargo Score se ajustam automaticamente à opção selecionada.</p>
    </div>
    <div class="proposal-model-selector" id="proposalModelSelector"></div>
  `;

  body.append(modelBlock, details);
  window.lucide?.createIcons();
}

function createProductConfigurator(investmentSection) {
  if (document.getElementById('proposalProductConfigurator')) {
    return;
  }

  const section = document.createElement('section');
  section.className = 'editor-section proposal-configurator';
  section.id = 'proposalProductConfigurator';
  section.innerHTML = `
    <header class="editor-section-header proposal-configurator-main-header">
      <div>
        <span class="proposal-configurator-step">2</span>
        <div><h2>O que será oferecido?</h2><p>Selecione somente os módulos e serviços desta negociação. O resumo da solução é gerado automaticamente.</p></div>
      </div>
    </header>
    <div class="editor-section-body proposal-configurator-layout">
      <div class="proposal-product-catalog" id="proposalProductCatalog"></div>
      <aside class="proposal-composition-summary" id="proposalCompositionSummary"></aside>
    </div>
  `;

  investmentSection.insertAdjacentElement('beforebegin', section);
}

function simplifyInvestmentSection(investmentSection) {
  investmentSection.classList.add('proposal-investment-guided');

  const title = investmentSection.querySelector('.editor-section-header h2');
  if (title) {
    title.textContent = 'Valores e condições de cobrança';
  }

  const addButton = document.getElementById('addPricingItemButton');
  const table = investmentSection.querySelector('.table-scroll');

  if (addButton && table && !investmentSection.querySelector('.proposal-advanced-pricing')) {
    const details = document.createElement('details');
    details.className = 'proposal-advanced-pricing';
    details.innerHTML = `
      <summary><i data-lucide="table-properties"></i><span><strong>Ajustes avançados dos itens</strong><small>Use apenas para itens fora do catálogo ou ajustes manuais.</small></span></summary>
      <div class="proposal-advanced-pricing-body"></div>
    `;

    const body = details.querySelector('.proposal-advanced-pricing-body');
    body.append(table);
    body.append(addButton);
    investmentSection.querySelector('.editor-section-body')?.appendChild(details);
  }

  investmentSection.querySelector('.pricing-actions')?.classList.add('proposal-billing-fields');
  ['minimumInvoice', 'setupFee'].forEach((id) => {
    document.getElementById(id)?.addEventListener('input', updateProposalSummary);
  });

  window.lucide?.createIcons();
}

function enhanceEditorCopy() {
  const header = document.querySelector('#editorView .editor-header');
  const description = header?.querySelector('p');
  if (description) {
    description.textContent = 'Selecione o modelo de análise, marque os módulos desejados e informe os valores. O escopo da proposta é montado automaticamente.';
  }

  const contextSection = document.getElementById('pricingModel')?.closest('.editor-section');
  const contextTitle = contextSection?.querySelector('.editor-section-header h2');
  if (contextTitle) {
    contextTitle.textContent = 'Dados e modelo da proposta';
  }
}

function bindSourceObserver() {
  const body = document.getElementById('pricingItemsBody');
  if (!body) {
    return;
  }

  const observer = new MutationObserver(() => {
    if (!state.syncing) {
      window.setTimeout(syncFromSource, 0);
    }
  });

  observer.observe(body, { childList: true, subtree: false });

  document.getElementById('pricingModel')?.addEventListener('change', () => {
    if (!state.syncing) {
      renderModelSelector();
      renderCatalog();
      updateGeneratedScope();
    }
  });

  document.addEventListener('click', (event) => {
    if (event.target.closest('[data-open-proposal], #newProposalButton')) {
      window.setTimeout(syncFromSource, 180);
      window.setTimeout(syncFromSource, 650);
    }
  }, true);
}

async function waitForEditor(attempts = 80) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const contextSection = document.getElementById('pricingModel')?.closest('.editor-section');
    const investmentSection = document.getElementById('pricingItemsBody')?.closest('.editor-section');

    if (contextSection && investmentSection) {
      return { contextSection, investmentSection };
    }

    await new Promise((resolve) => window.setTimeout(resolve, 100));
  }

  return null;
}

async function initialize() {
  if (state.initialized) {
    return;
  }

  const editor = await waitForEditor();
  if (!editor) {
    return;
  }

  state.initialized = true;
  appendStylesheet();
  ensurePricingModels();
  enhanceEditorCopy();
  moveAdvancedFields(editor.contextSection);
  createProductConfigurator(editor.investmentSection);
  simplifyInvestmentSection(editor.investmentSection);
  bindSourceObserver();
  syncFromSource();
}

initialize().catch((error) => {
  console.error('Não foi possível inicializar o configurador da proposta.', error);
});
