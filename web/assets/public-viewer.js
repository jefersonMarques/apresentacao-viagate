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

  function createAcceptanceFlow() {
    if (!document.body.classList.contains('public-proposal')) return null;

    const action = document.createElement('button');
    action.className = 'proposal-accept-floating';
    action.type = 'button';
    action.hidden = true;
    action.textContent = 'ACEITAR PROPOSTA';
    action.setAttribute('data-proposal-accept-floating', '');

    const modal = document.createElement('div');
    modal.className = 'proposal-contract-modal proposal-accept-modal';
    modal.hidden = true;
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-label', 'Aceitar proposta');
    modal.innerHTML = `
      <button class="proposal-contract-backdrop" type="button" aria-label="Fechar" data-proposal-contract-close></button>
      <section class="proposal-contract-panel proposal-accept-panel">
        <header class="proposal-contract-header">
          <div>
            <small>ACEITE DA PROPOSTA</small>
            <h2>Confirmar aceite</h2>
            <p>O aceite registra exatamente esta versão. Os dados necessários para gerar o contrato serão solicitados no próximo passo.</p>
          </div>
          <button class="proposal-contract-close" type="button" aria-label="Fechar" data-proposal-contract-close>×</button>
        </header>
        <div class="proposal-contract-body">
          <div class="proposal-contract-error" data-proposal-contract-error hidden></div>
          <form class="proposal-contract-form proposal-accept-form" method="post" data-proposal-accept-form>
            <div class="proposal-contract-grid">
              <label class="full"><span>Nome completo do responsável</span><input name="name" autocomplete="name" required/></label>
              <label><span>CPF</span><input name="cpf" inputmode="numeric" autocomplete="off" maxlength="14" required/></label>
              <label><span>Cargo / função</span><input name="role" autocomplete="organization-title"/></label>
              <label><span>E-mail</span><input name="email" type="email" autocomplete="email" required/></label>
              <label><span>Telefone</span><input name="phone" inputmode="tel" autocomplete="tel" required/></label>
            </div>
            <label class="proposal-contract-authority"><input type="checkbox" name="authority" value="1" required/><span data-proposal-acceptance-text>Confirmo que possuo poderes para representar a empresa nesta contratação.</span></label>
            <footer class="proposal-contract-footer proposal-accept-footer">
              <p>Depois do aceite, você poderá continuar agora ou retomar a contratação pelo link seguro enviado ao seu e-mail.</p>
              <button type="submit" data-proposal-accept-submit>ACEITAR PROPOSTA</button>
            </footer>
          </form>
        </div>
      </section>`;

    document.body.append(action, modal);

    const form = modal.querySelector('[data-proposal-accept-form]');
    const errorBox = modal.querySelector('[data-proposal-contract-error]');
    const acceptanceText = modal.querySelector('[data-proposal-acceptance-text]');
    const submit = modal.querySelector('[data-proposal-accept-submit]');
    let loaded = false;
    let journey = { state: 'proposal', label: 'ACEITAR PROPOSTA', url: '', tone: 'primary' };

    function field(name) {
      return form?.elements.namedItem(name);
    }

    function setField(name, value) {
      const input = field(name);
      if (!(input instanceof HTMLInputElement) || input.value.trim() || !value) return;
      input.value = String(value);
    }

    function setError(message) {
      if (!errorBox) return;
      errorBox.textContent = message || '';
      errorBox.hidden = !message;
    }

    function applyJourney(nextJourney) {
      if (!nextJourney || typeof nextJourney !== 'object') return;
      journey = {
        state: nextJourney.state || 'proposal',
        label: nextJourney.label || 'ACEITAR PROPOSTA',
        url: nextJourney.url || '',
        tone: nextJourney.tone || 'primary',
      };
      action.textContent = journey.label;
      action.classList.toggle('is-success', journey.tone === 'success');
    }

    async function loadState() {
      if (loaded) return journey;
      loaded = true;
      try {
        const response = await fetch(window.location.pathname, {
          credentials: 'same-origin',
          headers: { Accept: 'application/json' },
        });
        if (!response.ok) throw new Error('Não foi possível carregar o estado da proposta.');
        const data = await response.json();
        applyJourney(data.journey);
        if (acceptanceText && data.acceptance_text) acceptanceText.textContent = data.acceptance_text;
        const responsible = data.responsible || {};
        setField('name', responsible.name);
        setField('role', responsible.role);
        setField('email', responsible.email);
        setField('phone', responsible.phone);
      } catch (_) {
        loaded = false;
      }
      return journey;
    }

    function openModal() {
      modal.hidden = false;
      document.body.classList.add('proposal-contract-open');
      action.hidden = true;
      setError('');
      window.setTimeout(() => form?.querySelector('input')?.focus(), 30);
    }

    function closeModal() {
      modal.hidden = true;
      document.body.classList.remove('proposal-contract-open');
      updateControls();
    }

    action.addEventListener('click', async () => {
      await loadState();
      if (journey.state !== 'proposal' && journey.url) {
        window.location.assign(journey.url);
        return;
      }
      openModal();
    });

    modal.querySelectorAll('[data-proposal-contract-close]').forEach((button) => button.addEventListener('click', closeModal));
    document.addEventListener('keydown', (event) => {
      if (event.key === 'Escape' && !modal.hidden) {
        event.preventDefault();
        closeModal();
      }
    });

    form?.addEventListener('submit', async (event) => {
      event.preventDefault();
      setError('');
      if (!form.reportValidity()) return;

      const body = new URLSearchParams();
      new FormData(form).forEach((value, key) => body.append(key, String(value)));
      if (submit instanceof HTMLButtonElement) {
        submit.disabled = true;
        submit.textContent = 'REGISTRANDO ACEITE...';
      }
      try {
        const response = await fetch(`${window.location.pathname}/accept`, {
          method: 'POST',
          body,
          credentials: 'same-origin',
          headers: {
            Accept: 'application/json',
            'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8',
          },
        });
        if (!response.ok) {
          const contentType = response.headers.get('content-type') || '';
          const message = contentType.includes('application/json')
            ? (await response.json().catch(() => ({}))).error
            : '';
          throw new Error(message || 'Não foi possível registrar o aceite. Revise os dados e tente novamente.');
        }
        const data = await response.json();
        if (!data.next_url) throw new Error('O aceite foi registrado, mas o próximo passo não foi localizado.');
        window.location.assign(data.next_url);
      } catch (error) {
        setError(error?.message || 'Não foi possível registrar o aceite.');
        if (submit instanceof HTMLButtonElement) {
          submit.disabled = false;
          submit.textContent = 'ACEITAR PROPOSTA';
        }
      }
    });

    loadState().then(updateControls);
    return { action, modal };
  }

  const contractFlow = createAcceptanceFlow();

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
    const investment = investmentIndex(items);
    const threshold = investment >= 0 ? investment : Math.max(0, items.length - 1);
    const unavailable = !started || index < threshold || document.body.classList.contains('viewer-locked') || document.body.classList.contains('proposal-contract-open');
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
    if (!slide || canScrollInside(slide, direction)) return;
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
