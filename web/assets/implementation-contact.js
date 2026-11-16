(() => {
  const form = document.querySelector('[data-implementation-contact-form]');
  if (!(form instanceof HTMLFormElement)) {
    return;
  }

  const teamFields = form.querySelector('[data-implementation-team-fields]');
  const nameInput = form.elements.namedItem('contact_name');
  const phoneInput = form.elements.namedItem('contact_phone');
  const submitButton = form.querySelector('button[type="submit"]');

  function updateFields() {
    const teamSelected = form.elements.namedItem('contact_mode').value === 'team';

    teamFields.hidden = !teamSelected;
    nameInput.required = teamSelected;
    phoneInput.required = teamSelected;

    const nameValid = nameInput.value.trim().length > 0;
    const phoneValid = phoneInput.value.trim().length > 0;
    submitButton.disabled = teamSelected && (!nameValid || !phoneValid);
  }

  form.addEventListener('change', updateFields);
  form.addEventListener('input', updateFields);
  updateFields();
})();
