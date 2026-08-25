function initializePresentationNavigation() {
  const controls = document.querySelector('.presentation-controls');
  const previousButton = controls?.querySelector('[data-presentation-previous]');
  const nextButton = controls?.querySelector('[data-presentation-next]');
  const fullscreenButton = controls?.querySelector('[data-presentation-fullscreen]');

  if (!controls || !previousButton || !nextButton) {
    return;
  }

  controls.classList.add('presentation-navigation');

  previousButton.setAttribute('aria-label', 'Slide anterior');
  previousButton.innerHTML = '<i data-lucide="chevron-up"></i>';

  nextButton.setAttribute('aria-label', 'Próximo slide');
  nextButton.innerHTML = '<i data-lucide="chevron-down"></i>';

  fullscreenButton?.remove();

  if (window.lucide) {
    window.lucide.createIcons();
  }
}

initializePresentationNavigation();
