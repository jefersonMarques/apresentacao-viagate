(() => {
  if (window.__VIAGATE_PDF_MODE__ !== true) return;

  document.documentElement.classList.add('commercial-pdf-mode');
  document.body?.classList.add('commercial-pdf-mode');

  function contentFor(slide) {
    return slide.querySelector(':scope > .proposal-slide-inner')
      || slide.querySelector(':scope > .slide-inner')
      || slide.querySelector(':scope > [class$="-layout"]')
      || slide.firstElementChild;
  }

  function resetScale(element) {
    if (!(element instanceof HTMLElement)) return;
    element.style.removeProperty('transform');
    element.style.removeProperty('transform-origin');
  }

  function fitSlide(slide) {
    const content = contentFor(slide);
    if (!(slide instanceof HTMLElement) || !(content instanceof HTMLElement)) return;
    resetScale(content);

    const pageWidth = Math.max(1, slide.clientWidth);
    const pageHeight = Math.max(1, slide.clientHeight);
    const contentWidth = Math.max(1, content.scrollWidth);
    const contentHeight = Math.max(1, content.scrollHeight);
    const scale = Math.min(1, pageWidth / contentWidth, pageHeight / contentHeight);

    if (scale < 0.999) {
      content.style.transformOrigin = 'top center';
      content.style.transform = `scale(${Math.max(0.1, scale)})`;
    }
  }

  function renumberProposal() {
    const slides = Array.from(document.querySelectorAll('[data-viewer-slide]'));
    slides.forEach((slide, index) => {
      const counter = slide.querySelector('[data-slide-number]');
      if (counter) counter.textContent = `${String(index + 1).padStart(2, '0')} / ${String(slides.length).padStart(2, '0')}`;
    });
  }

  function prepare() {
    document.documentElement.classList.add('commercial-pdf-mode');
    document.body?.classList.add('commercial-pdf-mode');
    renumberProposal();
    const slides = document.querySelectorAll('.proposal-slide, #presentation > .slide');
    slides.forEach(fitSlide);
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
    window.setTimeout(prepare, 500);
    window.setTimeout(prepare, 1800);
    window.setTimeout(prepare, 4200);
  }, { once: true });
  window.addEventListener('commercial-pdf-ready', () => {
    window.setTimeout(prepare, 80);
    window.setTimeout(prepare, 650);
  });
  window.addEventListener('beforeprint', prepare);
})();
