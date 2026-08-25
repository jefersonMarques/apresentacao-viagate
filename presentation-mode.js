const presentationState = {
  initialized: false,
  started: false,
  paused: true,
  wheelLocked: false,
  controlsTimer: null,
  cursorTimer: null,
};

function getPresentationSlides() {
  return Array.from(document.querySelectorAll('#presentation > [data-slide]'));
}

function getPresentationCurrentIndex() {
  const slides = getPresentationSlides();
  const viewportCenter = window.scrollY + window.innerHeight / 2;
  let currentIndex = 0;

  slides.forEach((slide, index) => {
    if (viewportCenter >= slide.offsetTop) {
      currentIndex = index;
    }
  });

  return currentIndex;
}

function goToPresentationSlide(index, behavior = 'smooth') {
  const slides = getPresentationSlides();

  if (index < 0 || index >= slides.length) {
    return;
  }

  slides[index].scrollIntoView({ behavior, block: 'start' });
}

function renumberPresentationSlides() {
  const slides = getPresentationSlides();
  const total = slides.length;

  slides.forEach((slide, index) => {
    const number = index + 1;
    slide.dataset.slide = String(number);
    slide.id = `slide-${String(number).padStart(2, '0')}`;

    const counter = slide.querySelector('.slide-footer span:last-child');
    if (counter) {
      counter.textContent = `${String(number).padStart(2, '0')} / ${String(total).padStart(2, '0')}`;
    }
  });
}

function escapePresentationText(value) {
  const element = document.createElement('span');
  element.textContent = String(value ?? '');
  return element.innerHTML;
}

function getPresentationContact() {
  return window.presentationContact ?? {
    name: 'ViaGate',
    role: 'Comercial',
    email: 'contato@viagate.com.br',
    phone: '',
    whatsapp: '',
    photoUrl: './assets/logo-viagate-color.svg',
  };
}

function createPresentationGate() {
  if (document.querySelector('.presentation-gate')) {
    return;
  }

  const gate = document.createElement('div');
  gate.className = 'presentation-gate';
  gate.setAttribute('role', 'dialog');
  gate.setAttribute('aria-modal', 'true');
  gate.setAttribute('aria-label', 'Iniciar apresentação ViaGate');
  gate.innerHTML = `
    <div class="presentation-gate-panel">
      <img class="presentation-gate-logo" src="./assets/logo-viagate-white.svg" alt="ViaGate" />
      <span class="presentation-gate-eyebrow" data-presentation-gate-eyebrow>APRESENTAÇÃO INSTITUCIONAL · 2026</span>
      <h1 data-presentation-gate-title>ViaGate</h1>
      <p data-presentation-gate-copy>Validação, análise de risco e operação logística.</p>
      <button class="presentation-start-button" type="button" data-presentation-start>
        <span data-presentation-start-label>Iniciar apresentação</span>
        <i data-lucide="maximize-2"></i>
      </button>
      <small class="presentation-gate-hint" data-presentation-gate-hint>A apresentação será aberta em tela cheia.</small>
    </div>
  `;

  document.body.appendChild(gate);
}

function setPresentationGateMode(mode) {
  const eyebrow = document.querySelector('[data-presentation-gate-eyebrow]');
  const title = document.querySelector('[data-presentation-gate-title]');
  const copy = document.querySelector('[data-presentation-gate-copy]');
  const label = document.querySelector('[data-presentation-start-label]');
  const hint = document.querySelector('[data-presentation-gate-hint]');

  if (mode === 'continue') {
    if (eyebrow) eyebrow.textContent = 'APRESENTAÇÃO PAUSADA';
    if (title) title.textContent = 'Continuar apresentação';
    if (copy) copy.textContent = 'Você saiu da tela cheia. A apresentação permanece exatamente no ponto em que parou.';
    if (label) label.textContent = 'Continuar apresentação';
    if (hint) hint.textContent = 'Clique para retornar à tela cheia.';
    return;
  }

  if (eyebrow) eyebrow.textContent = 'APRESENTAÇÃO INSTITUCIONAL · 2026';
  if (title) title.textContent = 'ViaGate';
  if (copy) copy.textContent = 'Validação, análise de risco e operação logística.';
  if (label) label.textContent = 'Iniciar apresentação';
  if (hint) hint.textContent = 'A apresentação será aberta em tela cheia.';
}

function showPresentationGate(mode) {
  presentationState.paused = true;
  setPresentationGateMode(mode);
  document.body.classList.add('presentation-paused', 'presentation-gate-visible');
  document.body.classList.remove('presentation-controls-visible', 'presentation-cursor-hidden');
}

function hidePresentationGate() {
  presentationState.paused = false;
  document.body.classList.remove('presentation-paused', 'presentation-gate-visible');
}

function createPresentationControls() {
  if (document.querySelector('.presentation-controls')) {
    return;
  }

  const controls = document.createElement('div');
  controls.className = 'presentation-controls';
  controls.setAttribute('aria-label', 'Controles da apresentação');
  controls.innerHTML = `
    <button class="presentation-control-button" type="button" data-presentation-previous aria-label="Slide anterior">
      <i data-lucide="chevron-left"></i>
    </button>
    <span class="presentation-control-counter" data-presentation-counter>01 / 01</span>
    <button class="presentation-control-button" type="button" data-presentation-next aria-label="Próximo slide">
      <i data-lucide="chevron-right"></i>
    </button>
    <button class="presentation-control-button" type="button" data-presentation-fullscreen aria-label="Alternar tela cheia">
      <i data-lucide="maximize-2"></i>
    </button>
  `;

  document.body.appendChild(controls);
}

function freezeAnalysisMetrics() {
  const staticValues = new Map([
    ['[data-analysis-total]', '237.500+'],
    ['[data-analysis-vehicles]', '154.150'],
    ['[data-analysis-people]', '83.350'],
  ]);

  staticValues.forEach((value, selector) => {
    const element = document.querySelector(selector);
    if (!element) {
      return;
    }

    const clone = element.cloneNode(true);
    clone.textContent = value;
    clone.removeAttribute(selector.slice(1, -1));
    element.replaceWith(clone);
  });

  const footnote = document.querySelector('.exec-analysis-footnote');
  if (footnote) {
    footnote.innerHTML = `
      <span>Base consolidada até <strong>agosto de 2026.</strong></span>
      <span>Volumes apresentados como referência operacional da ViaGate.</span>
    `;
  }
}

function prepareInstitutionalClosingSlide() {
  const presentation = document.getElementById('presentation');
  const closingLayout = presentation?.querySelector('.closing-layout');

  if (!closingLayout) {
    return;
  }

  closingLayout.querySelector('.contact-line')?.remove();
  closingLayout.querySelector('.presentation-sales-contact')?.remove();
}

function createSalesContactSlide() {
  const presentation = document.getElementById('presentation');

  if (!presentation || presentation.querySelector('.presentation-contact-slide')) {
    return;
  }

  const contact = getPresentationContact();
  const name = escapePresentationText(contact.name);
  const role = escapePresentationText(contact.role);
  const email = escapePresentationText(contact.email);
  const phone = escapePresentationText(contact.phone);
  const photoUrl = escapePresentationText(contact.photoUrl);
  const whatsapp = String(contact.whatsapp ?? '').replace(/\D/g, '');
  const whatsappUrl = whatsapp
    ? `https://wa.me/${whatsapp}?text=${encodeURIComponent(`Olá ${contact.name}, gostaria de falar sobre as soluções da ViaGate.`)}`
    : '';

  const slide = document.createElement('section');
  slide.className = 'slide slide-deep presentation-contact-slide';
  slide.dataset.slide = String(getPresentationSlides().length + 1);
  slide.innerHTML = `
    <div class="slide-inner presentation-contact-layout">
      <div class="presentation-contact-heading">
        <p class="kicker">CONTATO COMERCIAL</p>
        <h2>Vamos conversar sobre <span>a sua operação?</span></h2>
        <p class="lead">Este é o contato responsável por apresentar a ViaGate e entender como as soluções podem ser aplicadas à sua operação.</p>
      </div>

      <div class="presentation-contact-card">
        <div class="presentation-contact-photo-wrap">
          <img class="presentation-contact-photo" src="${photoUrl}" alt="${name}" />
        </div>
        <div class="presentation-contact-person">
          <small>SEU CONTATO NA VIAGATE</small>
          <h3>${name}</h3>
          <p>${role}</p>
          <div class="presentation-contact-details">
            ${phone ? `<a href="tel:${escapePresentationText(String(contact.phone).replace(/\D/g, ''))}"><i data-lucide="phone"></i><span>${phone}</span></a>` : ''}
            ${email ? `<a href="mailto:${email}"><i data-lucide="mail"></i><span>${email}</span></a>` : ''}
          </div>
          ${whatsappUrl ? `<a class="presentation-contact-cta" href="${whatsappUrl}" target="_blank" rel="noopener noreferrer"><span>Falar no WhatsApp</span><i data-lucide="arrow-up-right"></i></a>` : ''}
        </div>
      </div>
    </div>
    <div class="slide-footer"><span>Contato comercial</span><span></span></div>
  `;

  presentation.appendChild(slide);
}

async function requestPresentationFullscreen() {
  if (document.fullscreenElement) {
    return true;
  }

  if (!document.documentElement.requestFullscreen) {
    return false;
  }

  try {
    await document.documentElement.requestFullscreen();
    return Boolean(document.fullscreenElement);
  } catch (error) {
    console.warn('Não foi possível ativar tela cheia automaticamente.', error);
    return false;
  }
}

async function togglePresentationFullscreen() {
  if (document.fullscreenElement) {
    await document.exitFullscreen();
    return;
  }

  await requestPresentationFullscreen();
}

async function startOrResumePresentation() {
  const isFirstStart = !presentationState.started;

  if (isFirstStart) {
    presentationState.started = true;
    document.body.classList.add('presentation-started');
  }

  const fullscreenStarted = await requestPresentationFullscreen();

  if (!fullscreenStarted) {
    const hint = document.querySelector('[data-presentation-gate-hint]');
    if (hint) {
      hint.textContent = 'O navegador bloqueou a tela cheia. Clique novamente ou permita fullscreen para continuar.';
    }
    showPresentationGate(isFirstStart ? 'start' : 'continue');
    return;
  }

  if (isFirstStart) {
    goToPresentationSlide(0, 'auto');
  }

  hidePresentationGate();
  showPresentationControls();
  updatePresentationControls();
}

function updatePresentationControls() {
  const slides = getPresentationSlides();
  const currentIndex = getPresentationCurrentIndex();
  const counter = document.querySelector('[data-presentation-counter]');
  const previousButton = document.querySelector('[data-presentation-previous]');
  const nextButton = document.querySelector('[data-presentation-next]');
  const fullscreenButton = document.querySelector('[data-presentation-fullscreen]');

  if (counter) {
    counter.textContent = `${String(currentIndex + 1).padStart(2, '0')} / ${String(slides.length).padStart(2, '0')}`;
  }

  if (previousButton) {
    previousButton.disabled = currentIndex === 0;
  }

  if (nextButton) {
    nextButton.disabled = currentIndex === slides.length - 1;
  }

  if (fullscreenButton) {
    fullscreenButton.setAttribute('aria-label', document.fullscreenElement ? 'Sair da tela cheia' : 'Entrar em tela cheia');
  }
}

function showPresentationControls() {
  if (!presentationState.started || presentationState.paused) {
    return;
  }

  document.body.classList.add('presentation-controls-visible');
  window.clearTimeout(presentationState.controlsTimer);
  presentationState.controlsTimer = window.setTimeout(() => {
    document.body.classList.remove('presentation-controls-visible');
  }, 2200);
}

function showPresentationCursor() {
  document.body.classList.remove('presentation-cursor-hidden');
  window.clearTimeout(presentationState.cursorTimer);

  if (!presentationState.started || presentationState.paused) {
    return;
  }

  presentationState.cursorTimer = window.setTimeout(() => {
    document.body.classList.add('presentation-cursor-hidden');
  }, 2600);
}

function handlePresentationWheel(event) {
  if (!presentationState.started || presentationState.paused || Math.abs(event.deltaY) < 18) {
    return;
  }

  event.preventDefault();

  if (presentationState.wheelLocked) {
    return;
  }

  presentationState.wheelLocked = true;
  const direction = event.deltaY > 0 ? 1 : -1;
  goToPresentationSlide(getPresentationCurrentIndex() + direction);

  window.setTimeout(() => {
    presentationState.wheelLocked = false;
  }, 720);
}

function handlePresentationKeyboard(event) {
  if (presentationState.paused) {
    return;
  }

  if (event.key.toLowerCase() === 'f') {
    event.preventDefault();
    togglePresentationFullscreen().catch(() => {});
    return;
  }

  if (event.key === 'Home') {
    event.preventDefault();
    goToPresentationSlide(0);
    return;
  }

  if (event.key === 'End') {
    event.preventDefault();
    goToPresentationSlide(getPresentationSlides().length - 1);
  }
}

function handleFullscreenChange() {
  updatePresentationControls();

  if (document.fullscreenElement) {
    hidePresentationGate();
    return;
  }

  if (presentationState.started) {
    showPresentationGate('continue');
  }
}

function bindPresentationEvents() {
  document.querySelector('[data-presentation-start]')?.addEventListener('click', startOrResumePresentation);

  document.querySelector('[data-presentation-previous]')?.addEventListener('click', () => {
    goToPresentationSlide(getPresentationCurrentIndex() - 1);
    showPresentationControls();
  });

  document.querySelector('[data-presentation-next]')?.addEventListener('click', () => {
    goToPresentationSlide(getPresentationCurrentIndex() + 1);
    showPresentationControls();
  });

  document.querySelector('[data-presentation-fullscreen]')?.addEventListener('click', () => {
    togglePresentationFullscreen().catch(() => {});
    showPresentationControls();
  });

  window.addEventListener('wheel', handlePresentationWheel, { passive: false });
  window.addEventListener('keydown', handlePresentationKeyboard);
  window.addEventListener('scroll', updatePresentationControls, { passive: true });
  window.addEventListener('mousemove', () => {
    showPresentationControls();
    showPresentationCursor();
  }, { passive: true });
  window.addEventListener('touchstart', showPresentationControls, { passive: true });
  document.addEventListener('fullscreenchange', handleFullscreenChange);
}

function disableWebsiteBehaviors() {
  const brand = document.querySelector('.presentation-header .brand');

  if (brand) {
    brand.removeAttribute('href');
    brand.setAttribute('aria-disabled', 'true');
  }
}

function initializePresentationMode() {
  if (presentationState.initialized) {
    return;
  }

  presentationState.initialized = true;
  document.body.classList.add('presentation-shell', 'presentation-paused', 'presentation-gate-visible');
  freezeAnalysisMetrics();
  prepareInstitutionalClosingSlide();
  createSalesContactSlide();
  renumberPresentationSlides();
  createPresentationControls();
  createPresentationGate();
  disableWebsiteBehaviors();
  bindPresentationEvents();
  updatePresentationControls();
  setPresentationGateMode('start');

  if (window.lucide) {
    window.lucide.createIcons();
  }
}

window.initializePresentationMode = initializePresentationMode;
