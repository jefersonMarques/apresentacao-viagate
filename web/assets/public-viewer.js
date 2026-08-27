(() => {
  const root = document.querySelector('[data-secure-viewer]');
  if (!root) return;

  const legacyAcceptance = root.querySelector('.proposal-acceptance-layout')?.closest('[data-viewer-slide]');
  legacyAcceptance?.remove();

  const gate = document.querySelector('[data-viewer-gate]');
  const start = document.querySelector('[data-viewer-start]');
  const restart = document.querySelector('[data-viewer-restart]');
  const previous = document.querySelector('[data-viewer-previous]');
  const next = document.querySelector('[data-viewer-next]');
  const counter = document.querySelector('[data-viewer-counter]');
  const slides = () => Array.from(root.querySelectorAll('[data-viewer-slide]'));
  const startLabel = start?.getAttribute('data-viewer-start-label') || 'INICIAR';
  const continueLabel = start?.getAttribute('data-viewer-continue-label') || 'CONTINUAR';
  let started = false;
  let locked = false;

  function createContractFlow() {
    if (!document.body.classList.contains('public-proposal')) return null;

    const action = document.createElement('button');
    action.className = 'proposal-accept-floating';
    action.type = 'button';
    action.hidden = true;
    action.textContent = 'ACEITAR PROPOSTA';
    action.setAttribute('data-proposal-accept-floating', '');

    const modal = document.createElement('div');
    modal.className = 'proposal-contract-modal';
    modal.hidden = true;
    modal.setAttribute('data-proposal-contract-modal', '');
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-label', 'Aceitar proposta e gerar contrato');
    modal.innerHTML = `
      <button class="proposal-contract-backdrop" type="button" aria-label="Fechar" data-proposal-contract-close></button>
      <section class="proposal-contract-panel">
        <header class="proposal-contract-header">
          <div>
            <small>ACEITE E CONTRATAÇÃO</small>
            <h2>Aceitar proposta</h2>
            <p>Complete os dados necessários para gerar o contrato e envie a apólice de seguros.</p>
          </div>
          <button class="proposal-contract-close" type="button" aria-label="Fechar" data-proposal-contract-close>×</button>
        </header>
        <div class="proposal-contract-body">
          <div class="proposal-contract-model" data-proposal-contract-label>Carregando modelo de contrato...</div>
          <div class="proposal-contract-error" data-proposal-contract-error hidden></div>
          <form class="proposal-contract-form" method="post" enctype="multipart/form-data" data-proposal-contract-form>
            <section class="proposal-contract-section">
              <header><small>01</small><div><strong>Empresa contratante</strong><span>O CNPJ preenche automaticamente os dados públicos disponíveis.</span></div></header>
              <div class="proposal-contract-grid">
                <label class="full"><span>CNPJ</span><input name="cnpj" data-mask="cnpj" autocomplete="off" required/><small data-contract-cnpj-status></small></label>
                <label><span>Razão social</span><input name="legal_name" required/></label>
                <label><span>Nome fantasia</span><input name="trade_name"/></label>
                <label><span>CEP</span><input name="postal_code" data-mask="postal-code" autocomplete="postal-code" required/></label>
                <label><span>Logradouro</span><input name="street" required/></label>
                <label><span>Número</span><input name="street_number" required/></label>
                <label><span>Complemento</span><input name="complement"/></label>
                <label><span>Bairro</span><input name="district"/></label>
                <label><span>Cidade</span><input name="city" required/></label>
                <label><span>UF</span><input name="state" maxlength="2" required/></label>
              </div>
            </section>

            <section class="proposal-contract-section">
              <header><small>02</small><div><strong>Operação e seguro</strong><span>Estes dados serão incorporados à contratação.</span></div></header>
              <div class="proposal-contract-grid">
                <label><span>Tipo de operação</span><select name="operation_type" required><option value="normal">Normal</option><option value="avulsa">Avulsa</option></select></label>
                <label><span>Seguradora</span><input name="insurer" required/></label>
                <label><span>Início da vigência</span><input name="policy_start_date" type="date" required/></label>
                <label><span>Fim da vigência</span><input name="policy_end_date" type="date" required/></label>
                <label><span>Corretor parceiro</span><input name="broker_company"/></label>
                <label><span>Produtor do corretor</span><input name="broker_producer"/></label>
                <label class="full proposal-policy-upload"><span>Apólice de seguros</span><input name="insurance_policy" type="file" accept="application/pdf,image/jpeg,image/png" required/><small>PDF, JPG ou PNG · máximo 15 MB.</small></label>
              </div>
            </section>

            <section class="proposal-contract-section">
              <header><small>03</small><div><strong>Responsável pela contratação</strong><span>Este responsável também será o primeiro signatário do contrato.</span></div></header>
              <div class="proposal-contract-grid">
                <label><span>Nome completo</span><input name="responsible_name" autocomplete="name" required/></label>
                <label><span>CPF</span><input name="responsible_cpf" data-mask="cpf" autocomplete="off" required/></label>
                <label><span>Cargo / função</span><input name="responsible_role"/></label>
                <label><span>E-mail</span><input name="responsible_email" type="email" autocomplete="email" required/></label>
                <label><span>Telefone</span><input name="responsible_phone" data-mask="phone" autocomplete="tel" required/></label>
              </div>
              <label class="proposal-contract-authority"><input type="checkbox" name="authority" value="1" required/><span data-proposal-acceptance-text></span></label>
            </section>

            <footer class="proposal-contract-footer">
              <p>Ao concluir, o contrato atribuído a esta versão será gerado e apresentado para assinatura eletrônica por código enviado ao e-mail informado.</p>
              <button type="submit" data-proposal-contract-submit>GERAR CONTRATO PARA ASSINATURA</button>
            </footer>
          </form>
        </div>
      </section>`;

    document.body.append(action, modal);

    const form = modal.querySelector('[data-proposal-contract-form]');
    const errorBox = modal.querySelector('[data-proposal-contract-error]');
    const label = modal.querySelector('[data-proposal-contract-label]');
    const acceptanceText = modal.querySelector('[data-proposal-acceptance-text]');
    const submit = modal.querySelector('[data-proposal-contract-submit]');
    const cnpjStatus = modal.querySelector('[data-contract-cnpj-status]');
    let loaded = false;
    let contractAssigned = true;
    let cnpjTimer = 0;
    let lastLookup = '';

    function setError(message) {
      if (!errorBox) return;
      errorBox.textContent = message || '';
      errorBox.hidden = !message;
    }

    function rawCNPJ(value) {
      return String(value || '').toUpperCase().replace(/[^0-9A-Z]/g, '').slice(0, 14);
    }

    function field(name) {
      return form?.elements.namedItem(name);
    }

    function setField(name, value, overwrite = true) {
      const input = field(name);
      if (!(input instanceof HTMLInputElement) && !(input instanceof HTMLTextAreaElement) && !(input instanceof HTMLSelectElement)) return;
      if (!overwrite && input.value.trim()) return;
      if (value == null || String(value).trim() === '') return;
      input.value = String(value);
      input.dispatchEvent(new Event('input', { bubbles: true }));
      input.dispatchEvent(new Event('change', { bubbles: true }));
    }

    async function loadInitialData() {
      if (loaded) return;
      loaded = true;
      if (label) label.textContent = 'Carregando modelo de contrato...';
      try {
        const response = await fetch(window.location.pathname, {
          credentials: 'same-origin',
          headers: { Accept: 'application/json' },
        });
        if (!response.ok) throw new Error('Não foi possível carregar os dados da contratação.');
        const data = await response.json();
        contractAssigned = Boolean(data.contract_assigned);
        if (label) label.textContent = contractAssigned
          ? `Contrato desta proposta: ${data.contract_label || 'modelo atribuído'}`
          : 'Esta proposta ainda não possui modelo de contrato atribuído.';
        if (acceptanceText) acceptanceText.textContent = data.acceptance_text || '';
        const company = data.company || {};
        Object.entries(company).forEach(([name, value]) => setField(name, value, false));
        const responsible = data.responsible || {};
        setField('responsible_name', responsible.name, false);
        setField('responsible_role', responsible.role, false);
        setField('responsible_email', responsible.email, false);
        setField('responsible_phone', responsible.phone, false);
        if (!contractAssigned) setError('O comercial precisa atribuir um modelo de contrato e publicar uma nova versão antes do aceite.');
      } catch (error) {
        loaded = false;
        setError(error?.message || 'Não foi possível carregar os dados da contratação.');
      }
    }

    async function lookupCNPJ() {
      const input = field('cnpj');
      if (!(input instanceof HTMLInputElement)) return;
      const cnpj = rawCNPJ(input.value);
      if (cnpj.length !== 14 || cnpj === lastLookup) return;
      lastLookup = cnpj;
      if (cnpjStatus) cnpjStatus.textContent = 'Consultando dados públicos...';
      try {
        const response = await fetch(`${window.location.pathname}?cnpj=${encodeURIComponent(cnpj)}`, {
          credentials: 'same-origin',
          headers: { Accept: 'application/json' },
        });
        const data = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(data.error || 'CNPJ não encontrado.');
        ['cnpj','legal_name','trade_name','postal_code','street','street_number','complement','district','city','state'].forEach((name) => setField(name, data[name], true));
        if (cnpjStatus) cnpjStatus.textContent = 'Dados encontrados. Revise antes de continuar.';
      } catch (error) {
        lastLookup = '';
        if (cnpjStatus) cnpjStatus.textContent = error?.message || 'Não foi possível consultar o CNPJ.';
      }
    }

    function open() {
      modal.hidden = false;
      document.body.classList.add('proposal-contract-open');
      action.hidden = true;
      setError('');
      loadInitialData();
      window.setTimeout(() => modal.querySelector('input,select,button')?.focus(), 30);
    }

    function close() {
      modal.hidden = true;
      document.body.classList.remove('proposal-contract-open');
      updateControls();
    }

    action.addEventListener('click', open);
    modal.querySelectorAll('[data-proposal-contract-close]').forEach((button) => button.addEventListener('click', close));
    document.addEventListener('keydown', (event) => {
      if (event.key === 'Escape' && !modal.hidden) {
        event.preventDefault();
        close();
      }
    });

    const cnpj = field('cnpj');
    cnpj?.addEventListener('input', () => {
      window.clearTimeout(cnpjTimer);
      if (rawCNPJ(cnpj.value).length === 14) cnpjTimer = window.setTimeout(lookupCNPJ, 450);
    });
    cnpj?.addEventListener('blur', lookupCNPJ);

    form?.addEventListener('submit', async (event) => {
      event.preventDefault();
      setError('');
      if (!contractAssigned) {
        setError('Esta proposta não possui um modelo de contrato atribuído.');
        return;
      }
      if (!form.reportValidity()) return;
      const fileInput = field('insurance_policy');
      const file = fileInput instanceof HTMLInputElement ? fileInput.files?.[0] : null;
      if (file && file.size > 15 * 1024 * 1024) {
        setError('A apólice deve ter no máximo 15 MB.');
        return;
      }
      if (submit instanceof HTMLButtonElement) {
        submit.disabled = true;
        submit.textContent = 'GERANDO CONTRATO...';
      }
      try {
        const response = await fetch(`${window.location.pathname}/accept`, {
          method: 'POST',
          body: new FormData(form),
          credentials: 'same-origin',
          headers: { Accept: 'application/json' },
        });
        const data = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(data.error || 'Não foi possível concluir a contratação.');
        if (!data.sign_url) throw new Error('Contrato gerado sem link de assinatura.');
        window.location.assign(data.sign_url);
      } catch (error) {
        setError(error?.message || 'Não foi possível concluir a contratação.');
        if (submit instanceof HTMLButtonElement) {
          submit.disabled = false;
          submit.textContent = 'GERAR CONTRATO PARA ASSINATURA';
        }
      }
    });

    return { action, modal };
  }

  const contractFlow = createContractFlow();

  function currentSlide() {
    const items = slides();
    if (!items.length) return null;
    const center = window.innerHeight / 2;
    let candidate = items[0];
    let distance = Number.POSITIVE_INFINITY;
    for (const slide of items) {
      const rect = slide.getBoundingClientRect();
      if (rect.top <= center && rect.bottom >= center) return slide;
      const current = Math.min(Math.abs(rect.top - center), Math.abs(rect.bottom - center));
      if (current < distance) {
        candidate = slide;
        distance = current;
      }
    }
    return candidate;
  }

  function currentIndex() {
    return Math.max(0, slides().indexOf(currentSlide()));
  }

  function investmentIndex(items) {
    const marker = root.querySelector('.proposal-price-groups, .proposal-highlight-grid');
    const slide = marker?.closest('[data-viewer-slide]');
    return slide ? items.indexOf(slide) : -1;
  }

  function updateAcceptAction(items, index) {
    if (!contractFlow?.action) return;
    const threshold = investmentIndex(items);
    const unavailable = !started || threshold < 0 || index < threshold || document.body.classList.contains('viewer-locked') || document.body.classList.contains('proposal-contract-open');
    contractFlow.action.hidden = unavailable;
  }

  function updateControls() {
    const items = slides();
    if (!items.length) return;
    const index = currentIndex();
    if (counter) counter.textContent = `${String(index + 1).padStart(2, '0')} / ${String(items.length).padStart(2, '0')}`;
    items.forEach((slide, slideIndex) => {
      const number = slide.querySelector('[data-slide-number]');
      if (number) number.textContent = `${String(slideIndex + 1).padStart(2, '0')} / ${String(items.length).padStart(2, '0')}`;
    });
    if (previous) previous.disabled = index <= 0;
    if (next) next.disabled = index >= items.length - 1;
    updateAcceptAction(items, index);
  }

  function showGate(continuing) {
    document.body.classList.add('viewer-locked');
    if (document.body.classList.contains('public-proposal')) document.body.classList.add('proposal-locked');
    if (gate) gate.hidden = false;
    if (start) start.textContent = continuing ? continueLabel : startLabel;
    if (restart) restart.hidden = !continuing;
    const controls = counter?.closest('[data-viewer-controls]') || counter?.parentElement;
    if (controls instanceof HTMLElement) controls.hidden = true;
    if (contractFlow?.action) contractFlow.action.hidden = true;
  }

  function reveal() {
    document.body.classList.remove('viewer-locked', 'proposal-locked');
    if (gate) gate.hidden = true;
    if (restart) restart.hidden = true;
    const controls = counter?.closest('[data-viewer-controls]') || counter?.parentElement;
    if (controls instanceof HTMLElement) controls.hidden = false;
    updateControls();
  }

  async function enter() {
    try {
      if (!document.fullscreenElement) await document.documentElement.requestFullscreen();
      started = true;
      reveal();
    } catch (_) {
      showGate(started);
    }
  }

  function go(index) {
    const items = slides();
    if (index < 0 || index >= items.length || locked || document.body.classList.contains('proposal-contract-open')) return;
    locked = true;
    items[index].scrollIntoView({ behavior: 'smooth', block: 'start' });
    window.setTimeout(() => {
      locked = false;
      updateControls();
    }, 560);
  }

  function canScrollInside(slide, direction) {
    const rect = slide.getBoundingClientRect();
    if (direction > 0) return rect.bottom > window.innerHeight + 3;
    return rect.top < -3;
  }

  function wheel(event) {
    if (document.body.classList.contains('proposal-contract-open')) return;
    if (!started || !document.fullscreenElement || Math.abs(event.deltaY) < 18) return;
    const slide = currentSlide();
    if (!slide) return;
    const direction = event.deltaY > 0 ? 1 : -1;
    if (canScrollInside(slide, direction)) return;
    event.preventDefault();
    go(currentIndex() + direction);
  }

  function keyboard(event) {
    if (document.body.classList.contains('proposal-contract-open')) return;
    if (!started || !document.fullscreenElement) return;
    const directions = { ArrowDown: 1, PageDown: 1, ArrowRight: 1, ArrowUp: -1, PageUp: -1, ArrowLeft: -1 };
    const direction = directions[event.key];
    if (!direction) return;
    const slide = currentSlide();
    if (!slide) return;
    if (canScrollInside(slide, direction)) return;
    event.preventDefault();
    go(currentIndex() + direction);
  }

  start?.addEventListener('click', enter);
  restart?.addEventListener('click', () => {
    slides()[0]?.scrollIntoView({ behavior: 'auto', block: 'start' });
    started = false;
    showGate(false);
  });
  previous?.addEventListener('click', () => go(currentIndex() - 1));
  next?.addEventListener('click', () => go(currentIndex() + 1));
  window.addEventListener('wheel', wheel, { passive: false });
  window.addEventListener('keydown', keyboard);
  window.addEventListener('scroll', updateControls, { passive: true });
  document.addEventListener('fullscreenchange', () => {
    if (document.fullscreenElement) {
      if (started) reveal();
      return;
    }
    if (started) showGate(true);
  });

  showGate(false);
  updateControls();
})();
