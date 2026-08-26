(() => {
  const frame = document.querySelector('[data-v1-presentation-frame]');
  const startButton = document.querySelector('[data-v1-presentation-start]');
  const restartButton = document.querySelector('[data-v1-presentation-restart]');
  const navigation = document.querySelector('[data-v1-presentation-navigation]');
  const previousButton = document.querySelector('[data-v1-presentation-previous]');
  const nextButton = document.querySelector('[data-v1-presentation-next]');
  const counter = document.querySelector('[data-v1-presentation-counter]');
  const loading = document.querySelector('[data-v1-presentation-loading]');
  if (!(frame instanceof HTMLIFrameElement) || !startButton) return;

  const state = { started: false, bridgeReady: false, slideNumber: 1, slideTotal: 1 };

  function updateNavigation(slideNumber = state.slideNumber, slideTotal = state.slideTotal) {
    state.slideNumber = Math.max(1, Number(slideNumber) || 1);
    state.slideTotal = Math.max(1, Number(slideTotal) || 1);
    if (counter) counter.textContent = `${String(state.slideNumber).padStart(2, '0')} / ${String(state.slideTotal).padStart(2, '0')}`;
    if (previousButton) previousButton.disabled = state.slideNumber <= 1;
    if (nextButton) nextButton.disabled = state.slideNumber >= state.slideTotal;
  }

  function syncNavigation() {
    const current = frame.contentWindow?.hostGetPresentationState?.();
    if (current) updateNavigation(current.slideNumber, current.slideTotal);
  }

  function contactData() {
    return Object.freeze({
      name: frame.dataset.salesName || 'ViaGate Comercial',
      role: frame.dataset.salesRole || '',
      email: frame.dataset.salesEmail || '',
      phone: frame.dataset.salesPhone || '',
      whatsapp: (frame.dataset.salesPhone || '').replace(/\D/g, ''),
      linkedin: frame.dataset.salesLinkedin || '',
      instagram: frame.dataset.salesInstagram || '',
      photoUrl: frame.dataset.salesPhoto || '/v1/assets/logo-viagate-color.svg',
    });
  }

  function presentationSettings() {
    return Object.freeze({
      showContactSlide: frame.dataset.showContact !== 'false',
      showClientIdentity: frame.dataset.showClient === 'true',
      client: {
        company_name: frame.dataset.clientName || '',
        contact_name: frame.dataset.contactName || '',
      },
    });
  }

  function installV1Runtime() {
    const doc = frame.contentDocument;
    const win = frame.contentWindow;
    if (!doc || !win) return;
    win.presentationContact = contactData();
    win.presentationSettings = presentationSettings();
    if (!doc.querySelector('script[data-presentation-bootstrap]')) {
      const script = doc.createElement('script');
      script.src = '/v1/presentation-bootstrap.js';
      script.dataset.presentationBootstrap = 'true';
      doc.body.appendChild(script);
    }
  }

  async function waitForBridge() {
    for (let attempt = 0; attempt < 160; attempt += 1) {
      if (typeof frame.contentWindow?.hostStartPresentation === 'function') {
        state.bridgeReady = true;
        startButton.disabled = false;
        loading?.setAttribute('hidden', '');
        syncNavigation();
        return;
      }
      await new Promise((resolve) => window.setTimeout(resolve, 50));
    }
    if (loading) loading.textContent = 'Não foi possível inicializar a apresentação.';
  }

  function showGate(continuing) {
    document.body.classList.add('is-paused');
    if (navigation) navigation.hidden = true;
    startButton.textContent = continuing ? 'CONTINUAR APRESENTAÇÃO' : 'INICIAR APRESENTAÇÃO';
    if (restartButton) restartButton.hidden = !continuing;
    frame.contentWindow?.hostPausePresentation?.();
  }

  function reveal() {
    document.body.classList.remove('is-paused');
    if (navigation) navigation.hidden = false;
    if (restartButton) restartButton.hidden = true;
    syncNavigation();
  }

  async function start() {
    if (!state.bridgeReady) return;
    try {
      if (!document.fullscreenElement) await document.documentElement.requestFullscreen();
      const firstStart = !state.started;
      state.started = true;
      frame.contentWindow?.hostStartPresentation?.(firstStart);
      reveal();
    } catch (_) {
      showGate(state.started);
    }
  }

  function restart() {
    frame.contentWindow?.hostRestartPresentation?.();
    state.started = false;
    updateNavigation(1, state.slideTotal);
    showGate(false);
  }

  function navigate(direction) {
    if (!state.started || !document.fullscreenElement) return;
    if (direction < 0) frame.contentWindow?.hostPreviousSlide?.();
    else frame.contentWindow?.hostNextSlide?.();
    window.setTimeout(syncNavigation, 80);
  }

  frame.addEventListener('load', () => {
    installV1Runtime();
    waitForBridge();
  }, { once: true });
  startButton.addEventListener('click', start);
  restartButton?.addEventListener('click', restart);
  previousButton?.addEventListener('click', () => navigate(-1));
  nextButton?.addEventListener('click', () => navigate(1));

  window.addEventListener('message', (event) => {
    if (event.source !== frame.contentWindow) return;
    const data = event.data || {};
    if (data.type === 'viagate:presentation:state') updateNavigation(data.slideNumber, data.slideTotal);
  });

  document.addEventListener('fullscreenchange', () => {
    if (document.fullscreenElement) {
      if (state.started) reveal();
      return;
    }
    if (state.started) showGate(true);
  });

  startButton.disabled = true;
  frame.src = '/v1/presentation-content.html';
  showGate(false);
})();
