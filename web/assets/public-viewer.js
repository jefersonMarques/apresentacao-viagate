(() => {
  const root = document.querySelector('[data-secure-viewer]');
  if (!root) return;

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
  }

  function showGate(continuing) {
    document.body.classList.add('viewer-locked');
    if (document.body.classList.contains('public-proposal')) document.body.classList.add('proposal-locked');
    if (gate) gate.hidden = false;
    if (start) start.textContent = continuing ? continueLabel : startLabel;
    if (restart) restart.hidden = !continuing;
    const controls = counter?.closest('[data-viewer-controls]') || counter?.parentElement;
    if (controls instanceof HTMLElement) controls.hidden = true;
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
    if (index < 0 || index >= items.length || locked) return;
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
    if (!started || !document.fullscreenElement || Math.abs(event.deltaY) < 18) return;
    const slide = currentSlide();
    if (!slide) return;
    const direction = event.deltaY > 0 ? 1 : -1;
    if (canScrollInside(slide, direction)) return;
    event.preventDefault();
    go(currentIndex() + direction);
  }

  function keyboard(event) {
    if (!started || !document.fullscreenElement) return;
    const directions = { ArrowDown: 1, PageDown: 1, ArrowRight: 1, ArrowUp: -1, PageUp: -1, ArrowLeft: -1 };
    const direction = directions[event.key];
    if (!direction) return;
    const slide = currentSlide();
    if (!slide) return;
    if (canScrollInside(slide, direction)) return;
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
