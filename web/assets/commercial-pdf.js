(() => {
  if (window.__VIAGATE_PDF_MODE__ !== true) return;

  const A4_CONTENT_HEIGHT_PX = 1030;
  const MIN_COMPACT_SCALE = 0.86;
  const protectedSelector = [
    'article',
    'figure',
    '.proposal-price-group',
    '.proposal-highlight',
    '.proposal-person-card',
    '.presentation-contact-card',
    '.presentation-use-case-context',
    '.presentation-use-case-outcomes > div',
    'tr',
  ].join(',');

  document.documentElement.classList.add('commercial-pdf-mode');
  document.body?.classList.add('commercial-pdf-mode');

  function renumberProposal() {
    const slides = Array.from(document.querySelectorAll('[data-viewer-slide]'));
    slides.forEach((slide, index) => {
      const counter = slide.querySelector('[data-slide-number]');
      if (counter) {
        counter.textContent = `${String(index + 1).padStart(2, '0')} / ${String(slides.length).padStart(2, '0')}`;
      }
    });
  }

  function resetUnit(unit) {
    if (!(unit instanceof HTMLElement)) return;
    unit.classList.remove('pdf-allow-break');
    unit.style.removeProperty('zoom');
  }

  function classifyUnit(unit) {
    if (!(unit instanceof HTMLElement)) return;
    resetUnit(unit);

    const height = Math.max(unit.scrollHeight, unit.getBoundingClientRect().height);
    if (height <= A4_CONTENT_HEIGHT_PX) return;

    const scale = A4_CONTENT_HEIGHT_PX / Math.max(1, height);
    if (scale >= MIN_COMPACT_SCALE) {
      // Chrome supports zoom in print layout and, unlike transform, it also
      // changes the element's layout box. Small overflows therefore stay on
      // one sheet without leaving invisible space or clipping descendants.
      unit.style.zoom = String(Math.min(1, scale));
      return;
    }

    // A unit that cannot fit legibly on one sheet is explicitly allowed to
    // fragment. The print CSS protects its smaller descendants independently.
    unit.classList.add('pdf-allow-break');
  }

  function prepareProtectedUnits() {
    const units = Array.from(document.querySelectorAll(protectedSelector));
    // Work from the smallest/deepest units first. That lets a large container
    // observe the final size of cards and rows already compacted inside it.
    units.sort((a, b) => {
      const depth = (element) => {
        let value = 0;
        for (let node = element; node?.parentElement; node = node.parentElement) value += 1;
        return value;
      };
      return depth(b) - depth(a);
    });
    units.forEach(classifyUnit);
  }

  function prepare() {
    document.documentElement.classList.add('commercial-pdf-mode');
    document.body?.classList.add('commercial-pdf-mode');
    renumberProposal();
    prepareProtectedUnits();
  }

  async function waitForAssets() {
    try {
      await document.fonts?.ready;
    } catch (_) {}

    const images = Array.from(document.images).filter((image) => !image.complete);
    await Promise.all(images.map((image) => new Promise((resolve) => {
      image.addEventListener('load', resolve, { once: true });
      image.addEventListener('error', resolve, { once: true });
      window.setTimeout(resolve, 1800);
    })));
  }

  window.addEventListener('load', async () => {
    await waitForAssets();
    prepare();
    // Dynamic V1 presentation modules finish their DOM adjustments shortly
    // after load. One delayed pass is enough; repeated scaling passes made the
    // previous slide PDF unnecessarily expensive to render.
    window.setTimeout(prepare, 1200);
  }, { once: true });

  window.addEventListener('commercial-pdf-ready', () => {
    window.setTimeout(prepare, 120);
  });
  window.addEventListener('beforeprint', prepare);
})();
