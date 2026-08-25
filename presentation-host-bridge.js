let lastReportedSlide = 0;

function reportPresentationState(type) {
  if (window.parent === window) {
    return;
  }

  const slides = getPresentationSlides();
  const currentIndex = getPresentationCurrentIndex();
  const slideNumber = currentIndex + 1;
  const slideTotal = slides.length;

  window.parent.postMessage({
    source: 'viagate-presentation',
    type,
    slideNumber,
    slideTotal,
  }, window.location.origin);
}

function reportCurrentSlide() {
  const slideNumber = getPresentationCurrentIndex() + 1;

  if (slideNumber === lastReportedSlide) {
    return;
  }

  lastReportedSlide = slideNumber;
  reportPresentationState('slide_view');

  if (slideNumber === getPresentationSlides().length) {
    reportPresentationState('complete');
  }
}

window.hostStartPresentation = function hostStartPresentation(firstStart = false) {
  presentationState.started = true;
  presentationState.paused = false;

  document.body.classList.add('presentation-started');
  document.body.classList.remove('presentation-paused', 'presentation-gate-visible');
  document.querySelector('.presentation-gate')?.remove();

  if (firstStart) {
    goToPresentationSlide(0, 'auto');
  }

  updatePresentationControls();
  showPresentationControls();
  reportCurrentSlide();
};

window.hostPausePresentation = function hostPausePresentation() {
  if (!presentationState.started) {
    return;
  }

  presentationState.paused = true;
  document.body.classList.add('presentation-paused');
  document.body.classList.remove('presentation-controls-visible', 'presentation-cursor-hidden');
};

window.addEventListener('scroll', reportCurrentSlide, { passive: true });
