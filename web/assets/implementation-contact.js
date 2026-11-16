(() => {
  const form = document.querySelector('[data-implementation-contact-form]');
  if (!(form instanceof HTMLFormElement)) {
    return;
  }

  const teamFields = form.querySelector('[data-implementation-team-fields]');
  const nameInput = form.elements.namedItem('contact_name');
  const phoneInput = form.elements.namedItem('contact_phone');
  const emailInput = form.elements.namedItem('contact_email');
  const submitButton = form.querySelector('button[type="submit"]');

  if (!(teamFields instanceof HTMLElement) ||
      !(nameInput instanceof HTMLInputElement) ||
      !(phoneInput instanceof HTMLInputElement) ||
      !(emailInput instanceof HTMLInputElement) ||
      !(submitButton instanceof HTMLButtonElement)) {
    return;
  }

  function updateFields() {
    const selectedMode = form.querySelector('input[name="contact_mode"]:checked');
    const teamSelected = selectedMode?.value === 'team';

    teamFields.hidden = !teamSelected;
    nameInput.required = teamSelected;
    phoneInput.required = teamSelected;

    nameInput.disabled = !teamSelected;
    phoneInput.disabled = !teamSelected;
    emailInput.disabled = !teamSelected;

    const nameValid = nameInput.value.trim().length > 0;
    const phoneValid = phoneInput.value.trim().length > 0;
    const emailValid = emailInput.value.trim() === '' || emailInput.validity.valid;

    submitButton.disabled = !selectedMode ||
      (teamSelected && (!nameValid || !phoneValid || !emailValid));
  }

  form.addEventListener('change', updateFields);
  form.addEventListener('input', updateFields);
  updateFields();
})();
