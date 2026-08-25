const presentationState = {
  started: false,
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

function createPresentationCover() {
  const presentation = document.getElementById('presentation');

  if (!presentation || presentation.querySelector('.presentation-cover')) {
    return;
  }

  const cover = document.createElement('section');
  cover.className = 'slide slide-dark presentation-cover';
  cover.dataset.slide = '1';
  cover.innerHTML = `
    <div class="presentation-cover-inner">
      <div class="presentation-cover-copy">
        <img class="presentation-cover-brand" src="./assets/logo-viagate-white.svg" alt="ViaGate" />
        <p class="presentation-cover-kicker">APRESENTAÇÃO INSTITUCIONAL · 2026</p>
        <h1>Validação, análise de risco<br /><span>e operação logística.</span></h1>
        <p class="presentation-cover-lead">Tecnologia própria para identificar quem está na operação, apoiar decisões de risco e acompanhar a execução da viagem.</p>
      </div>
      <div class="presentation-cover-action">
        <button class="presentation-start-button" type="button" data-presentation-start>
          <span>Iniciar apresentação</span>
          <i data-lucide="maximize-2"></i>
        </button>
        <span class="presentation-cover-hint">Abre em tela cheia. Use as setas, Page Up/Page Down ou a roda do mouse para navegar.</span>
        <span class="presentation-cover-status">Apresentação iniciada · pressione F para alternar tela cheia.</span>
      </div>
    </div>
    <div class="slide-footer"><span>ViaGate</span><span></span></div>
  `;

  presentation.prepend(cover);
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

function enhanceClosingSlide() {
  const slides = getPresentationSlides();
  const closingSlide = slides.at(-1);
  const closingLayout = closingSlide?.querySelector('.closing-layout');

  if (!closingLayout || closingLayout.querySelector('.presentation-sales-contact')) {
    return;
  }

  const existingContactLine = closingLayout.querySelector('.contact-line');
  if (existingContactLine) {
    existingContactLine.remove();
  }

  const salesContact = document.createElement('div');
  salesContact.className = 'presentation-sales-contact';
  salesContact.innerHTML = `
    <img src="./assets/antonio-photo.svg" alt="Antônio Santos" />
    <div>
      <small>Contato comercial</small>
      <strong>Antônio Santos</strong>
      <span>Sócio-Diretor Comercial</span>
      <span>antonio.santos@viagate.com.br · (41) 99962-3600</span>
    </div>
  `;

  closingLayout.appendChild(salesContact);
}

async function togglePresentationFullscreen() {
  if (document.fullscreenElement) {
    await document.exitFullscreen();
    return;
  }

  await document.documentElement.requestFullscreen();
}

async function startPresentation() {
  presentationState.started = true;
  document.body.classList.add('presentation-started');
  showPresentationControls();

  try {
    if (!document.fullscreenElement) {
      await document.documentElement.requestFullscreen();
    }
  } catch (error) {
    console.warn('Não foi possível ativar tela cheia automaticamente.', error);
  }

  goToPresentationSlide(0, 'auto');
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

  const fullscreenIcon = fullscreenButton?.querySelector('svg');
  if (fullscreenButton && fullscreenIcon) {
    fullscreenButton.setAttribute('aria-label', document.fullscreenElement ? 'Sair da tela cheia' : 'Entrar em tela cheia');
  }
}

function showPresentationControls() {
  if (!presentationState.started) {
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

  if (!presentationState.started) {
    return;
  }

  presentationState.cursorTimer = window.setTimeout(() => {
    document.body.classList.add('presentation-cursor-hidden');
  }, 2600);
}

function handlePresentationWheel(event) {
  if (!presentationState.started || Math.abs(event.deltaY) < 18) {
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

function bindPresentationEvents() {
  document.querySelector('[data-presentation-start]')?.addEventListener('click', startPresentation);

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
  document.addEventListener('fullscreenchange', updatePresentationControls);
}

function disableWebsiteBehaviors() {
  const brand = document.querySelector('.presentation-header .brand');

  if (brand) {
    brand.removeAttribute('href');
    brand.setAttribute('aria-disabled', 'true');
  }
}

function initializePresentationMode() {
  document.body.classList.add('presentation-shell');
  createPresentationCover();
  freezeAnalysisMetrics();
  enhanceClosingSlide();
  renumberPresentationSlides();
  createPresentationControls();
  disableWebsiteBehaviors();
  bindPresentationEvents();
  updatePresentationControls();

  if (window.lucide) {
    window.lucide.createIcons();
  }
}

window.addEventListener('load', initializePresentationMode, { once: true });
