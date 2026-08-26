const EDGE_TOLERANCE = 3;
const NAVIGATION_LOCK_MS = 560;

const state = {
  locked: false,
};

function getSlides() {
  return Array.from(document.querySelectorAll('[data-proposal-slide]'));
}

function getCurrentSlide() {
  const slides = getSlides();
  if (!slides.length) {
    return null;
  }

  const viewportCenter = window.innerHeight / 2;
  let closest = slides[0];
  let closestDistance = Number.POSITIVE_INFINITY;

  for (const slide of slides) {
    const rect = slide.getBoundingClientRect();

    if (rect.top <= viewportCenter && rect.bottom >= viewportCenter) {
      return slide;
    }

    const distance = Math.min(
      Math.abs(rect.top - viewportCenter),
      Math.abs(rect.bottom - viewportCenter),
    );

    if (distance < closestDistance) {
      closest = slide;
      closestDistance = distance;
    }
  }

  return closest;
}

function canScrollInside(slide, direction) {
  const rect = slide.getBoundingClientRect();

  if (direction > 0) {
    return rect.bottom > window.innerHeight + EDGE_TOLERANCE;
  }

  return rect.top < -EDGE_TOLERANCE;
}

function navigateFrom(slide, direction) {
  if (state.locked) {
    return;
  }

  const slides = getSlides();
  const currentIndex = slides.indexOf(slide);
  const nextIndex = currentIndex + direction;

  if (currentIndex < 0 || nextIndex < 0 || nextIndex >= slides.length) {
    return;
  }

  state.locked = true;
  slides[nextIndex].scrollIntoView({ behavior: 'smooth', block: 'start' });

  window.setTimeout(() => {
    state.locked = false;
  }, NAVIGATION_LOCK_MS);
}

function isViewerActive() {
  return document.body.classList.contains('public-proposal')
    && Boolean(document.fullscreenElement)
    && !document.body.classList.contains('proposal-locked');
}

function handleWheel(event) {
  if (!isViewerActive() || Math.abs(event.deltaY) < 18) {
    return;
  }

  const slide = getCurrentSlide();
  if (!slide) {
    return;
  }

  const direction = event.deltaY > 0 ? 1 : -1;

  // Impede o listener legado de trocar de slide enquanto ainda existe
  // conteúdo da seção para rolar normalmente dentro da viewport.
  event.stopImmediatePropagation();

  if (canScrollInside(slide, direction)) {
    return;
  }

  event.preventDefault();
  navigateFrom(slide, direction);
}

function handleKeyboard(event) {
  if (!isViewerActive()) {
    return;
  }

  const directionByKey = {
    ArrowDown: 1,
    PageDown: 1,
    ArrowUp: -1,
    PageUp: -1,
  };
  const direction = directionByKey[event.key];

  if (!direction) {
    return;
  }

  const slide = getCurrentSlide();
  if (!slide) {
    return;
  }

  event.stopImmediatePropagation();

  if (canScrollInside(slide, direction)) {
    return;
  }

  event.preventDefault();
  navigateFrom(slide, direction);
}

function initialize() {
  // O snap obrigatório do CSS base pode antecipar a próxima seção.
  // A navegação passa a ser controlada apenas quando a borda da viewport
  // realmente alcança o início ou o fim da seção atual.
  document.body.style.scrollSnapType = 'none';

  window.addEventListener('wheel', handleWheel, {
    capture: true,
    passive: false,
  });
  window.addEventListener('keydown', handleKeyboard, true);
}

initialize();
