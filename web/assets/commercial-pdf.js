(() => {
  if (window.__VIAGATE_PDF_MODE__ !== true) return;

  const MIN_SECTION_HEIGHT = 675;
  const slideSelector = '.proposal-slide, #presentation > .slide';
  const root = document.documentElement;

  root.classList.add('commercial-pdf-mode');
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

  function visibleSlides() {
    return Array.from(document.querySelectorAll(slideSelector))
      .filter((slide) => getComputedStyle(slide).display !== 'none');
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
    resetSlideHeights();

    visibleSlides().forEach(expandSlideToContents);

    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        root.dataset.captureReady = '1';
      });
    });
  }

  async function waitForAssets() {
    try {
      await document.fonts?.ready;
    } catch (_) {}

    const images = Array.from(document.images).filter((image) => !image.complete);
    await Promise.all(images.map((image) => new Promise((resolve) => {
      image.addEventListener('load', resolve, { once: true });
      image.addEventListener('error', resolve, { once: true });
      window.setTimeout(resolve, 2200);
    })));
  }

  async function finalizeCaptureLayout() {
    await waitForAssets();
    prepareLayout();
    window.setTimeout(prepareLayout, 180);
  }

  window.addEventListener('load', () => {
    if (document.getElementById('proposalPresentation')) {
      finalizeCaptureLayout();
      return;
    }

    // Presentation V1 still performs its canonical DOM enhancements after the
    // initial load. This is only a safety fallback; commercial-pdf-ready is the
    // authoritative signal emitted after those modules finish.
    window.setTimeout(finalizeCaptureLayout, 7000);
  }, { once: true });

  window.addEventListener('commercial-pdf-ready', () => {
    finalizeCaptureLayout();
  });
})();
