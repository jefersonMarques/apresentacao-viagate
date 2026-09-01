(() => {
  const feedbackStyles = document.createElement('link');
  feedbackStyles.rel = 'stylesheet';
  feedbackStyles.href = '/assets/action-feedback.css';
  document.head.appendChild(feedbackStyles);

  function fallbackCopy(value) {
    const textarea = document.createElement('textarea');
    textarea.value = value;
    textarea.setAttribute('readonly', '');
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    const copied = document.execCommand('copy');
    textarea.remove();
    return copied;
  }

  async function copyText(value) {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(value);
      return true;
    }
    return fallbackCopy(value);
  }

  function ensureActionFeedback() {
    let root = document.querySelector('[data-action-feedback]');
    if (root) return root;

    root = document.createElement('div');
    root.className = 'action-feedback-shell';
    root.setAttribute('data-action-feedback', '');
    root.setAttribute('data-state', 'working');
    root.setAttribute('role', 'status');
    root.setAttribute('aria-live', 'polite');
    root.innerHTML = `
      <div class="action-feedback-card">
        <div class="action-feedback-icon" aria-hidden="true"></div>
        <div class="action-feedback-copy">
          <strong data-action-feedback-title>Processando</strong>
          <span data-action-feedback-message>Aguarde alguns instantes.</span>
        </div>
        <div class="action-feedback-progress" aria-hidden="true"><span data-action-feedback-progress></span></div>
      </div>`;
    document.body.appendChild(root);
    return root;
  }

  const actionFeedback = (() => {
    let timer = null;
    let progress = 0;

    function root() {
      return ensureActionFeedback();
    }

    function setProgress(value) {
      progress = Math.max(0, Math.min(100, Number(value) || 0));
      const bar = root().querySelector('[data-action-feedback-progress]');
      if (bar) bar.style.width = `${progress}%`;
    }

    function stopTimer() {
      if (!timer) return;
      window.clearInterval(timer);
      timer = null;
    }

    function show({ title = 'Processando', message = 'Aguarde alguns instantes.' } = {}) {
      stopTimer();
      const element = root();
      element.dataset.state = 'working';
      element.querySelector('[data-action-feedback-title]').textContent = title;
      element.querySelector('[data-action-feedback-message]').textContent = message;
      setProgress(8);
      window.requestAnimationFrame(() => element.classList.add('is-visible'));

      timer = window.setInterval(() => {
        if (progress >= 82) return;
        const increment = Math.max(0.8, (82 - progress) * 0.07);
        setProgress(progress + increment);
      }, 420);
    }

    function update({ title, message, progress: nextProgress } = {}) {
      const element = root();
      if (title) element.querySelector('[data-action-feedback-title]').textContent = title;
      if (message) element.querySelector('[data-action-feedback-message]').textContent = message;
      if (nextProgress !== undefined) setProgress(nextProgress);
    }

    function succeed(message = 'Concluído.') {
      stopTimer();
      const element = root();
      element.dataset.state = 'success';
      element.querySelector('[data-action-feedback-title]').textContent = 'Documento pronto';
      element.querySelector('[data-action-feedback-message]').textContent = message;
      setProgress(100);
      window.setTimeout(hide, 1100);
    }

    function fail(message = 'Não foi possível concluir a ação.') {
      stopTimer();
      const element = root();
      element.dataset.state = 'error';
      element.querySelector('[data-action-feedback-title]').textContent = 'Não foi possível concluir';
      element.querySelector('[data-action-feedback-message]').textContent = message;
      setProgress(100);
      window.setTimeout(hide, 4200);
    }

    function hide() {
      stopTimer();
      const element = root();
      element.classList.remove('is-visible');
      window.setTimeout(() => {
        element.dataset.state = 'working';
        setProgress(0);
      }, 260);
    }

    return { show, update, succeed, fail, hide, setProgress };
  })();

  function filenameFromResponse(response, fallback = 'documento.pdf') {
    const disposition = response.headers.get('Content-Disposition') || '';
    const encoded = disposition.match(/filename\*=UTF-8''([^;]+)/i);
    if (encoded?.[1]) {
      try {
        return decodeURIComponent(encoded[1].trim().replace(/^"|"$/g, ''));
      } catch (_) {}
    }
    const plain = disposition.match(/filename="?([^";]+)"?/i);
    return plain?.[1]?.trim() || fallback;
  }

  async function responseBlobWithProgress(response) {
    const total = Number(response.headers.get('Content-Length')) || 0;
    if (!response.body || !response.body.getReader) {
      actionFeedback.setProgress(96);
      return response.blob();
    }

    const reader = response.body.getReader();
    const chunks = [];
    let received = 0;

    while (true) {
      const { value, done } = await reader.read();
      if (done) break;
      chunks.push(value);
      received += value.byteLength;
      if (total > 0) {
        const ratio = Math.min(1, received / total);
        actionFeedback.setProgress(86 + ratio * 12);
      } else {
        actionFeedback.setProgress(Math.min(98, 88 + chunks.length * 0.7));
      }
    }
    return new Blob(chunks, { type: response.headers.get('Content-Type') || 'application/octet-stream' });
  }

  function triggerBlobDownload(blob, filename) {
    const objectURL = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = objectURL;
    anchor.download = filename;
    anchor.style.display = 'none';
    document.body.appendChild(anchor);
    anchor.click();
    anchor.remove();
    window.setTimeout(() => URL.revokeObjectURL(objectURL), 3000);
  }

  async function downloadWithFeedback(anchor) {
    if (!(anchor instanceof HTMLAnchorElement) || anchor.dataset.actionRunning === 'true') return;
    anchor.dataset.actionRunning = 'true';
    anchor.setAttribute('aria-disabled', 'true');

    const title = anchor.dataset.actionTitle || 'Preparando documento';
    const message = anchor.dataset.actionMessage || 'Estamos montando o arquivo para download.';
    actionFeedback.show({ title, message });

    try {
      const response = await fetch(anchor.href, {
        method: 'GET',
        credentials: 'same-origin',
        headers: { 'X-Requested-With': 'ViaGate-Long-Action' },
      });
      if (!response.ok) {
        const detail = (await response.text()).trim();
        throw new Error(detail || `Falha HTTP ${response.status}`);
      }
      const contentType = response.headers.get('Content-Type') || '';
      if (!contentType.toLowerCase().includes('application/pdf')) {
        throw new Error('O servidor não retornou um PDF válido.');
      }

      actionFeedback.update({
        title: 'Finalizando documento',
        message: 'O PDF está pronto. Preparando o download…',
        progress: 86,
      });
      const blob = await responseBlobWithProgress(response);
      triggerBlobDownload(blob, filenameFromResponse(response));
      actionFeedback.succeed(anchor.dataset.actionSuccess || 'O download foi iniciado.');
    } catch (error) {
      console.error('long action download failed', error);
      actionFeedback.fail(error?.message || 'Tente novamente em alguns instantes.');
    } finally {
      delete anchor.dataset.actionRunning;
      anchor.removeAttribute('aria-disabled');
    }
  }

  window.ViaGate = Object.assign(window.ViaGate || {}, {
    copyText,
    actionFeedback,
    downloadWithFeedback,
  });

  document.addEventListener('click', async (event) => {
    const longAction = event.target.closest('a[data-long-action="download"]');
    if (longAction) {
      event.preventDefault();
      await downloadWithFeedback(longAction);
      return;
    }

    const button = event.target.closest('[data-copy-path]');
    if (!button) return;
    const path = button.getAttribute('data-copy-path');
    if (!path) return;

    const value = new URL(path, window.location.origin).toString();
    const original = button.innerHTML;
    try {
      const copied = await copyText(value);
      if (!copied) throw new Error('clipboard unavailable');
      button.textContent = 'Copiado';
    } catch (_) {
      window.prompt('Copie o link:', value);
    } finally {
      window.setTimeout(() => { button.innerHTML = original; }, 1400);
    }
  });

  document.addEventListener('submit', (event) => {
    const form = event.target;
    if (!(form instanceof HTMLFormElement)) return;
    const submitter = event.submitter;
    if (!(submitter instanceof HTMLButtonElement)) return;

    const label = submitter.getAttribute('data-processing');
    if (!label) return;

    window.setTimeout(() => {
      if (!document.contains(form)) return;
      submitter.dataset.originalText = submitter.textContent || '';
      submitter.textContent = label;
      submitter.classList.add('is-processing');
      form.querySelectorAll('button[type="submit"]').forEach((button) => {
        button.disabled = true;
        button.setAttribute('aria-disabled', 'true');
      });
    }, 0);
  });
})();
