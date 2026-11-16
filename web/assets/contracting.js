(() => {
  const root = document.querySelector('[data-contracting-wizard]');
  if (!root) {
    return;
  }

  const steps = ['company', 'review'];
  const panels = new Map(steps.map((step) => [
    step, root.querySelector(`[data-contracting-step="${step}"]`),
  ]));
  const links = new Map(steps.map((step) => [
    step, root.querySelector(`[data-contracting-step-link="${step}"]`),
  ]));
  const params = new URL(window.location.href).searchParams;

  function companyComplete() {
    const required = ['cnpj', 'legal_name', 'postal_code', 'street', 'street_number', 'city', 'state'];
    return required.every((name) => {
      const input = root.querySelector(`[name="${name}"]`);
      return input instanceof HTMLInputElement && input.value.trim() !== '';
    });
  }

  function setStep(requested, changeURL = false) {
    const step = requested === 'review' && companyComplete() ? 'review' : 'company';
    for (const current of steps) {
      const panel = panels.get(current);
      const link = links.get(current);
      if (panel) {
        panel.hidden = current !== step;
      }
      if (link) {
        link.classList.toggle('is-active', current === step);
        link.classList.toggle('is-complete', current === 'company' && step === 'review');
      }
    }

    if (changeURL) {
      const url = new URL(window.location.href);
      url.searchParams.set('step', step);
      url.searchParams.delete('saved');
      window.history.replaceState({}, '', url);
    }
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function initialStep() {
    if (params.get('step') === 'company') {
      return 'company';
    }
    if (params.get('step') === 'review' || params.get('saved') === 'company') {
      return 'review';
    }
    return companyComplete() ? 'review' : 'company';
  }

  root.querySelectorAll('[data-contracting-step-link]').forEach((button) => {
    button.addEventListener('click', () => {
      setStep(button.getAttribute('data-contracting-step-link'), true);
    });
  });

  root.querySelectorAll('[data-contracting-previous]').forEach((button) => {
    button.addEventListener('click', () => {
      setStep(button.getAttribute('data-contracting-previous'), true);
    });
  });

  setStep(initialStep());
})();
