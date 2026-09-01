(() => {
  if (window.__VIAGATE_PDF_MODE__ !== true) return;

  const MIN_SECTION_HEIGHT = 675;
  const STABILITY_WINDOW_MS = 1200;
  const STABILITY_POLL_MS = 160;
  const MAX_STABILITY_WAIT_MS = 8000;
  const WARM_SCROLL_DELAY_MS = 180;
  const slideSelector = '.proposal-slide, #presentation > .slide';
  const root = document.documentElement;
  let preparing = false;

  root.classList.add('commercial-pdf-mode');
  document.body?.classList.add('commercial-pdf-mode');

  const delay = (milliseconds) => new Promise((resolve) => window.setTimeout(resolve, milliseconds));

  function renumberProposal() {
    const slides = Array.from(document.querySelectorAll('[data-viewer-slide]'));
    slides.forEach((slide, index) => {
      const counter = slide.querySelector('[data-slide-number]');
      if (counter) {
        counter.textContent = `${String(index + 1).padStart(2, '0')} / ${String(slides.length).padStart(2, '0')}`;
      }
    });
  }

  function visibleSlides() {
    return Array.from(document.querySelectorAll(slideSelector))
      .filter((slide) => getComputedStyle(slide).display !== 'none');
  }

  function forceAssetsEager(scope = document) {
    scope.querySelectorAll('img').forEach((image) => {
      image.loading = 'eager';
      image.decoding = 'auto';
    });
  }

  function resetSlideHeights() {
    visibleSlides().forEach((slide) => {
      if (!(slide instanceof HTMLElement)) return;
      slide.style.removeProperty('height');
      slide.style.removeProperty('min-height');
    });
  }

  function expandSlideToContents(slide) {
    if (!(slide instanceof HTMLElement)) return;

    const rect = slide.getBoundingClientRect();
    let bottom = rect.top + Math.max(MIN_SECTION_HEIGHT, rect.height);
    slide.querySelectorAll('*').forEach((element) => {
      if (!(element instanceof HTMLElement) && !(element instanceof SVGElement)) return;
      const elementRect = element.getBoundingClientRect();
      if (!Number.isFinite(elementRect.bottom)) return;
      bottom = Math.max(bottom, elementRect.bottom);
    });

    const requiredHeight = Math.ceil(Math.max(MIN_SECTION_HEIGHT, bottom - rect.top));
    slide.style.minHeight = `${requiredHeight}px`;
  }

  function prepareLayout() {
    root.classList.add('commercial-pdf-mode');
    document.body?.classList.add('commercial-pdf-mode');
    root.dataset.captureReady = '0';
    renumberProposal();
    forceAssetsEager();
    resetSlideHeights();
    visibleSlides().forEach(expandSlideToContents);
  }

  async function waitForAssets() {
    try {
      await document.fonts?.ready;
    } catch (_) {}

    forceAssetsEager();
    const images = Array.from(document.images);
    await Promise.all(images.map(async (image) => {
      if (!image.complete) {
        await new Promise((resolve) => {
          image.addEventListener('load', resolve, { once: true });
          image.addEventListener('error', resolve, { once: true });
          window.setTimeout(resolve, 3200);
        });
      }
      if (typeof image.decode === 'function') {
        try {
          await Promise.race([image.decode(), delay(1200)]);
        } catch (_) {}
      }
    }));
  }

  async function warmDynamicSections() {
    const slides = visibleSlides();
    for (const slide of slides) {
      forceAssetsEager(slide);
      slide.scrollIntoView({ block: 'start', inline: 'nearest' });
      window.dispatchEvent(new Event('scroll'));
      await delay(WARM_SCROLL_DELAY_MS);
    }
    window.scrollTo(0, 0);
    window.dispatchEvent(new Event('scroll'));
    await delay(300);
  }

  function layoutSignature() {
    const slides = visibleSlides().map((slide) => {
      const rect = slide.getBoundingClientRect();
      return [
        Math.round(rect.width),
        Math.round(rect.height),
        slide.querySelectorAll('*').length,
      ].join(':');
    });
    const images = Array.from(document.images);
    const loadedImages = images.filter((image) => image.complete && image.naturalWidth > 0).length;
    return [
      Math.round(document.documentElement.scrollWidth),
      Math.round(document.documentElement.scrollHeight),
      `${loadedImages}/${images.length}`,
      slides.join('|'),
    ].join('::');
  }

  async function waitForStableLayout() {
    const startedAt = performance.now();
    let stableSince = performance.now();
    let previous = layoutSignature();

    while (performance.now() - startedAt < MAX_STABILITY_WAIT_MS) {
      await delay(STABILITY_POLL_MS);
      prepareLayout();
      const current = layoutSignature();
      if (current === previous) {
        if (performance.now() - stableSince >= STABILITY_WINDOW_MS) return;
        continue;
      }
      previous = current;
      stableSince = performance.now();
    }
  }

  async function finalizeCaptureLayout() {
    if (preparing) return;
    preparing = true;
    root.dataset.captureReady = '0';

    try {
      forceAssetsEager();
      await waitForAssets();
      await warmDynamicSections();
      await waitForAssets();
      prepareLayout();
      await delay(450);
      prepareLayout();
      await waitForStableLayout();
      await waitForAssets();
      prepareLayout();
      await delay(350);
      window.scrollTo(0, 0);
      root.dataset.captureReady = '1';
    } finally {
      preparing = false;
    }
  }

  window.addEventListener('load', () => {
    if (document.getElementById('proposalPresentation')) {
      finalizeCaptureLayout();
      return;
    }

    // Presentation V1 performs canonical DOM enhancements after load. This
    // fallback only runs when the authoritative commercial-pdf-ready signal
    // does not arrive for some reason.
    window.setTimeout(finalizeCaptureLayout, 7000);
  }, { once: true });

  window.addEventListener('commercial-pdf-ready', () => {
    finalizeCaptureLayout();
  });
})();
