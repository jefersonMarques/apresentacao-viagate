let lastReportedSlide = 0;
let hostPresentationActive = false;

function getPresentationHostState() {
  const slides = getPresentationSlides();
  const currentIndex = getPresentationCurrentIndex();

  return {
    slideNumber: currentIndex + 1,
    slideTotal: slides.length,
    currentIndex,
  };
}

function reportPresentationState(type) {
  if (window.parent === window) {
    return;
  }

  const state = getPresentationHostState();

  window.parent.postMessage({
    source: 'viagate-presentation',
    type,
    slideNumber: state.slideNumber,
    slideTotal: state.slideTotal,
  }, window.location.origin);
}

function reportCurrentSlide(force = false) {
  const state = getPresentationHostState();

  if (!force && state.slideNumber === lastReportedSlide) {
    return;
  }

  lastReportedSlide = state.slideNumber;
  reportPresentationState('slide_view');

  if (state.slideNumber === state.slideTotal) {
    reportPresentationState('complete');
  }
}

function goToHostSlide(index) {
  goToPresentationSlide(index);
  window.setTimeout(() => reportCurrentSlide(true), 120);
}

function clearEmbeddedPauseState() {
  if (!hostPresentationActive) {
    return;
  }

  presentationState.paused = false;
  document.body.classList.remove(
    'presentation-paused',
    'presentation-gate-visible',
    'presentation-controls-visible',
    'presentation-cursor-hidden',
  );
}

const pauseStateObserver = new MutationObserver(() => {
  if (!hostPresentationActive) {
    return;
  }

  if (
    document.body.classList.contains('presentation-paused')
    || document.body.classList.contains('presentation-gate-visible')
  ) {
    clearEmbeddedPauseState();
  }
});

pauseStateObserver.observe(document.body, {
  attributes: true,
  attributeFilter: ['class'],
});

window.hostStartPresentation = function hostStartPresentation(firstStart = false) {
  hostPresentationActive = true;
  presentationState.started = true;
  presentationState.paused = false;

  document.body.classList.add('presentation-started');
  clearEmbeddedPauseState();
  document.querySelector('.presentation-gate')?.remove();
  document.querySelector('.presentation-controls')?.remove();

  if (firstStart) {
    goToPresentationSlide(0, 'auto');
    lastReportedSlide = 0;
  }

  window.requestAnimationFrame(clearEmbeddedPauseState);
  window.setTimeout(clearEmbeddedPauseState, 80);
  reportCurrentSlide(true);
};

window.hostPausePresentation = function hostPausePresentation() {
  hostPresentationActive = false;
  presentationState.paused = true;
  document.body.classList.add('presentation-paused');
  document.body.classList.remove('presentation-controls-visible', 'presentation-cursor-hidden');
};

window.hostRestartPresentation = function hostRestartPresentation() {
  hostPresentationActive = false;
  presentationState.started = false;
  presentationState.paused = true;
  lastReportedSlide = 0;

  goToPresentationSlide(0, 'auto');
  document.body.classList.remove('presentation-started', 'presentation-controls-visible', 'presentation-cursor-hidden');
  document.body.classList.add('presentation-paused');
  reportCurrentSlide(true);
};

window.hostPreviousSlide = function hostPreviousSlide() {
  const state = getPresentationHostState();
  goToHostSlide(state.currentIndex - 1);
};

window.hostNextSlide = function hostNextSlide() {
  const state = getPresentationHostState();
  goToHostSlide(state.currentIndex + 1);
};

window.hostGoToSlide = function hostGoToSlide(index) {
  goToHostSlide(Number(index));
};

window.hostGetPresentationState = function hostGetPresentationState() {
  return getPresentationHostState();
};

window.addEventListener('scroll', () => reportCurrentSlide(false), { passive: true });

window.parent?.postMessage({
  source: 'viagate-presentation',
  type: 'bridge_ready',
  ...getPresentationHostState(),
}, window.location.origin);
