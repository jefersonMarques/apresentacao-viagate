(() => {
  function rawCNPJ(value) {
    return String(value || '').toUpperCase().replace(/[^0-9A-Z]/g, '').slice(0, 14);
  }

  async function uploadImage(wrapper, file) {
    const category = wrapper.dataset.imageCategory || '';
    const body = new FormData();
    body.append('image', file);
    const response = await fetch(`/admin/assets?category=${encodeURIComponent(category)}`, {
      method: 'POST',
      body,
      credentials: 'same-origin',
      headers: { Accept: 'application/json' },
    });
    if (!response.ok) {
      const message = (await response.text()).trim();
      throw new Error(message || 'Não foi possível enviar a imagem.');
    }
    return response.json();
  }

  function renderImageUpload(wrapper) {
    const value = wrapper.querySelector('[data-image-value]');
    const preview = wrapper.querySelector('[data-image-preview]');
    const image = wrapper.querySelector('[data-image-preview-image]');
    const remove = wrapper.querySelector('[data-image-remove]');
    if (!(value instanceof HTMLInputElement) || !preview || !(image instanceof HTMLImageElement)) return;
    const url = value.value.trim();
    preview.hidden = !url;
    if (url) image.src = url;
    else image.removeAttribute('src');
    if (remove) remove.hidden = !url;
  }

  function setImageStatus(wrapper, text, error = false) {
    const status = wrapper.querySelector('[data-image-status]');
    if (!status) return;
    status.textContent = text;
    status.classList.toggle('error', error);
  }

  function initImageUploads() {
    document.querySelectorAll('[data-image-upload]').forEach((wrapper) => {
      if (wrapper.dataset.imageBound === '1') return;
      wrapper.dataset.imageBound = '1';
      const input = wrapper.querySelector('[data-image-file]');
      const value = wrapper.querySelector('[data-image-value]');
      const remove = wrapper.querySelector('[data-image-remove]');
      if (!(input instanceof HTMLInputElement) || !(value instanceof HTMLInputElement)) return;

      input.addEventListener('change', async () => {
        const file = input.files?.[0];
        if (!file) return;
        if (file.size > 2 * 1024 * 1024) {
          setImageStatus(wrapper, 'A imagem deve ter no máximo 2 MB.', true);
          input.value = '';
          return;
        }
        setImageStatus(wrapper, 'Enviando...');
        input.disabled = true;
        try {
          const uploaded = await uploadImage(wrapper, file);
          value.value = uploaded.url || '';
          renderImageUpload(wrapper);
          value.dispatchEvent(new Event('change', { bubbles: true }));
          setImageStatus(wrapper, 'Imagem enviada.');
        } catch (error) {
          setImageStatus(wrapper, error?.message || 'Não foi possível enviar a imagem.', true);
        } finally {
          input.disabled = false;
          input.value = '';
        }
      });

      remove?.addEventListener('click', () => {
        value.value = '';
        renderImageUpload(wrapper);
        value.dispatchEvent(new Event('change', { bubbles: true }));
        setImageStatus(wrapper, 'Imagem removida deste material.');
      });
      renderImageUpload(wrapper);
    });
  }

  const companyFields = {
    legal_name: 'legal_name',
    trade_name: 'trade_name',
    email: 'email',
    phone: 'phone',
    street: 'street',
    number: 'number',
    complement: 'complement',
    district: 'district',
    city: 'city',
    state: 'state',
    postal_code: 'postal_code',
  };

  function setCompanyData(form, company) {
    Object.entries(companyFields).forEach(([key, responseKey]) => {
      const field = form.querySelector(`[data-company-field="${key}"]`);
      if (!(field instanceof HTMLInputElement) && !(field instanceof HTMLTextAreaElement)) return;
      const value = String(company?.[responseKey] || '').trim();
      if (!value) return;
      field.value = value;
      field.dispatchEvent(new Event('input', { bubbles: true }));
      field.dispatchEvent(new Event('change', { bubbles: true }));
    });
  }

  async function lookupCNPJ(wrapper, force = false) {
    const form = wrapper.closest('form');
    const input = wrapper.querySelector('[data-cnpj-input]');
    const status = wrapper.querySelector('[data-cnpj-status]');
    const button = wrapper.querySelector('[data-cnpj-button]');
    if (!(form instanceof HTMLFormElement) || !(input instanceof HTMLInputElement)) return;
    const cnpj = rawCNPJ(input.value);
    if (cnpj.length !== 14) {
      if (force && status) status.textContent = 'Informe um CNPJ completo.';
      return;
    }
    if (!force && wrapper.dataset.lastCnpj === cnpj) return;
    wrapper.dataset.lastCnpj = cnpj;
    if (status) {
      status.textContent = 'Consultando dados públicos...';
      status.classList.remove('error', 'success');
    }
    if (button instanceof HTMLButtonElement) button.disabled = true;
    try {
      const response = await fetch(`/admin/api/cnpj/${encodeURIComponent(cnpj)}`, {
        credentials: 'same-origin',
        headers: { Accept: 'application/json' },
      });
      if (!response.ok) throw new Error((await response.text()).trim() || 'CNPJ não encontrado.');
      const company = await response.json();
      setCompanyData(form, company);
      if (status) {
        status.textContent = 'Dados cadastrais preenchidos. Revise antes de salvar.';
        status.classList.add('success');
      }
    } catch (error) {
      wrapper.dataset.lastCnpj = '';
      if (status) {
        status.textContent = error?.message || 'Não foi possível consultar o CNPJ.';
        status.classList.add('error');
      }
    } finally {
      if (button instanceof HTMLButtonElement) button.disabled = false;
    }
  }

  function initCNPJLookup() {
    document.querySelectorAll('[data-cnpj-lookup]').forEach((wrapper) => {
      if (wrapper.dataset.cnpjBound === '1') return;
      wrapper.dataset.cnpjBound = '1';
      const input = wrapper.querySelector('[data-cnpj-input]');
      const button = wrapper.querySelector('[data-cnpj-button]');
      let timer = 0;
      button?.addEventListener('click', () => lookupCNPJ(wrapper, true));
      input?.addEventListener('input', () => {
        window.clearTimeout(timer);
        const cnpj = rawCNPJ(input.value);
        if (cnpj.length === 14) timer = window.setTimeout(() => lookupCNPJ(wrapper, false), 450);
      });
      input?.addEventListener('blur', () => lookupCNPJ(wrapper, false));
    });
  }

  function initPipelineFilters() {
    document.querySelectorAll('[data-pipeline-table]').forEach((root) => {
      const rows = Array.from(root.querySelectorAll('[data-pipeline-row]'));
      const search = root.querySelector('[data-pipeline-search]');
      const stage = root.querySelector('[data-pipeline-stage]');
      const commercial = root.querySelector('[data-pipeline-commercial]');
      const count = root.querySelector('[data-pipeline-count]');
      const empty = root.querySelector('[data-pipeline-empty]');
      const normalize = (value) => String(value || '').normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLowerCase().trim();
      const apply = () => {
        const q = normalize(search?.value);
        const selectedStage = stage?.value || '';
        const selectedCommercial = commercial?.value || '';
        let visible = 0;
        rows.forEach((row) => {
          const show = (!q || normalize(row.dataset.search).includes(q))
            && (!selectedStage || row.dataset.stage === selectedStage)
            && (!selectedCommercial || row.dataset.commercial === selectedCommercial);
          row.hidden = !show;
          if (show) visible += 1;
        });
        if (count) count.textContent = String(visible);
        if (empty) empty.hidden = visible !== 0;
      };
      search?.addEventListener('input', apply);
      stage?.addEventListener('change', apply);
      commercial?.addEventListener('change', apply);
      apply();
    });
  }

  document.addEventListener('DOMContentLoaded', () => {
    initImageUploads();
    initCNPJLookup();
    initPipelineFilters();
  });
})();
