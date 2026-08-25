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
};

window.hostPausePresentation = function hostPausePresentation() {
  if (!presentationState.started) {
    return;
  }

  presentationState.paused = true;
  document.body.classList.add('presentation-paused');
  document.body.classList.remove('presentation-controls-visible', 'presentation-cursor-hidden');
};
