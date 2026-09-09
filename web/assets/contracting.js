(() => {
  const root = document.querySelector('[data-contracting-wizard]');
  if (!root) return;

  const steps = ['company', 'insurance', 'review'];
  const panels = new Map(steps.map((step) => [step, root.querySelector(`[data-contracting-step="${step}"]`)]));
  const links = new Map(steps.map((step) => [step, root.querySelector(`[data-contracting-step-link="${step}"]`)]));
  const params = new URL(window.location.href).searchParams;
  const policyPresent = root.getAttribute('data-policy-present') === 'true';
  const insurancePersistedReady = root.getAttribute('data-insurance-ready') === 'true';
  const insuranceForm = panels.get('insurance')?.querySelector('form.grid');
  let insuranceDirty = false;

  function fieldValue(name) {
    const input = root.querySelector(`[name="${name}"]`);
    return input instanceof HTMLInputElement || input instanceof HTMLSelectElement ? input.value.trim() : '';
  }

  function companyComplete() {
    return ['cnpj', 'legal_name', 'postal_code', 'street', 'street_number', 'city', 'state'].every((name) => fieldValue(name));
  }

  function insuranceComplete() {
    return insurancePersistedReady &&
      !insuranceDirty &&
      policyPresent &&
      ['operation_type', 'insurer', 'policy_start_date', 'policy_end_date'].every((name) => fieldValue(name));
  }

  function resolveStep(step) {
    if (step === 'insurance' && !companyComplete()) return 'company';
    if (step === 'review' && !insuranceComplete()) return companyComplete() ? 'insurance' : 'company';
    return step;
  }

  function initialStep() {
    const requested = params.get('step');
    if (steps.includes(requested)) return resolveStep(requested);
    const saved = params.get('saved');
    if (saved === 'company') return 'insurance';
    if (saved === 'document' && insuranceComplete()) return 'review';
    if (saved === 'insurance') return insuranceComplete() ? 'review' : 'insurance';
    if (!companyComplete()) return 'company';
    if (!insuranceComplete()) return 'insurance';
    return 'review';
  }

  function setStep(step, push = false) {
    if (!steps.includes(step)) return;
    step = resolveStep(step);
    for (const current of steps) {
      const panel = panels.get(current);
      const link = links.get(current);
      if (panel) panel.hidden = current !== step;
      if (link) {
        link.classList.toggle('is-active', current === step);
        const complete = current === 'company' ? companyComplete() : current === 'insurance' ? insuranceComplete() : false;
        link.classList.toggle('is-complete', complete && current !== step);
      }
    }
    if (push) {
      const url = new URL(window.location.href);
      url.searchParams.set('step', step);
      url.searchParams.delete('saved');
      window.history.replaceState({}, '', url);
    }
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  root.querySelectorAll('[data-contracting-step-link]').forEach((button) => {
    button.addEventListener('click', () => setStep(button.getAttribute('data-contracting-step-link'), true));
  });

  if (insuranceForm instanceof HTMLFormElement) {
    ['input', 'change'].forEach((eventName) => {
      insuranceForm.addEventListener(eventName, () => {
        insuranceDirty = true;
      });
    });
  }

  root.querySelectorAll('[data-contracting-previous]').forEach((button) => {
    button.addEventListener('click', () => setStep(button.getAttribute('data-contracting-previous'), true));
  });
  root.querySelectorAll('[data-contracting-next]').forEach((button) => {
    button.addEventListener('click', () => setStep(button.getAttribute('data-contracting-next'), true));
  });

  setStep(initialStep());
})();
