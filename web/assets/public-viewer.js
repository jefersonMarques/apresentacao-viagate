(() => {
  const root = document.querySelector('[data-secure-viewer]');
  if (!root) return;

  const gate = document.querySelector('[data-viewer-gate]');
  const start = document.querySelector('[data-viewer-start]');
  const restart = document.querySelector('[data-viewer-restart]');
  const previous = document.querySelector('[data-viewer-previous]');
  const next = document.querySelector('[data-viewer-next]');
  const counter = document.querySelector('[data-viewer-counter]');
  const inlineAction = document.querySelector('[data-proposal-accept-inline]');
  const slides = () => Array.from(root.querySelectorAll('[data-viewer-slide]'));
  const startLabel = start?.getAttribute('data-viewer-start-label') || 'INICIAR';
  const continueLabel = start?.getAttribute('data-viewer-continue-label') || 'CONTINUAR';
  let started = false;
  let locked = false;

  function createAcceptanceFlow() {
    if (!document.body.classList.contains('public-proposal')) return null;

    const action = document.querySelector('[data-proposal-accept-floating]');
    const modal = document.querySelector('[data-proposal-accept-modal]');
    if (!(action instanceof HTMLButtonElement) || !(modal instanceof HTMLElement)) return null;


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

    function updateActionButton(button) {
      if (!(button instanceof HTMLButtonElement)) return;
      button.textContent = journey.label;
      button.classList.toggle('is-success', journey.tone === 'success');
      button.setAttribute('aria-label', journey.label);
    }

    function applyJourney(nextJourney) {
      if (!nextJourney || typeof nextJourney !== 'object') return;
      journey = {
        state: nextJourney.state || 'proposal',
        label: nextJourney.label || 'ACEITAR PROPOSTA',
        url: nextJourney.url || '',
        tone: nextJourney.tone || 'primary',
      };
      updateActionButton(action);
      updateActionButton(inlineAction);
    }

    async function loadState(force = false) {
      if (loaded && !force) return journey;
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

    async function advance() {
      await loadState();
      if (journey.state !== 'proposal' && journey.url) {
        window.location.assign(journey.url);
        return;
      }
      openModal();
    }

    action.addEventListener('click', advance);
    inlineAction?.addEventListener('click', advance);

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
    return { action, modal, loadState };
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
    const marker = root.querySelector('.proposal-price-groups, .proposal-highlight-grid, [data-proposal-accept-inline]');
    const slide = marker?.closest('[data-viewer-slide]');
    return slide ? items.indexOf(slide) : -1;
  }

  function inlineActionVisible() {
    if (!(inlineAction instanceof HTMLElement)) return false;
    const rect = inlineAction.getBoundingClientRect();
    const viewportHeight = window.innerHeight || document.documentElement.clientHeight;
    return rect.bottom > 0 && rect.top < viewportHeight;
  }

  function updateAcceptAction(items, index) {
    if (!contractFlow?.action) return;
    const investment = investmentIndex(items);
    const threshold = investment >= 0 ? investment : Math.max(0, items.length - 1);
    const unavailable = !started || index < threshold || document.body.classList.contains('viewer-locked') || document.body.classList.contains('proposal-contract-open');
    if (unavailable) {
      contractFlow.action.hidden = true;
      return;
    }

    // No bloco de investimento o CTA principal aparece abaixo dos valores.
    // Quando ele sai da tela (ou o cliente avança), a versão fixa assume e
    // acompanha a navegação no centro inferior.
    contractFlow.action.hidden = index === threshold && inlineActionVisible();
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
  window.addEventListener('resize', updateControls);
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
