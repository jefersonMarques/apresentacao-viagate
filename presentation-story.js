function createOperationalUseCaseSlide() {
  if (document.querySelector('.presentation-use-case')) {
    return;
  }

  const integrationSlide = document.querySelector('.exec-integration-layout')?.closest('.slide');
  if (!integrationSlide) {
    return;
  }

  const section = document.createElement('section');
  section.className = 'slide presentation-use-case';
  section.dataset.slide = '0';
  section.innerHTML = `
    <div class="presentation-use-case-layout">
      <div class="presentation-use-case-copy">
        <p class="kicker dark-kicker">EXEMPLO DE OPERAÇÃO</p>
        <h2>Do pedido de liberação <span>à viagem acompanhada.</span></h2>
        <p class="lead">Um fluxo típico mostra como as soluções ViaGate podem se conectar antes e durante a viagem.</p>
        <div class="presentation-use-case-context">
          <small>CENÁRIO</small>
          <strong>Transportadora precisa liberar um motorista antes do carregamento.</strong>
          <span>A análise precisa confirmar identidade, consultar informações relevantes e manter a decisão rastreável.</span>
        </div>
      </div>

      <div>
        <div class="presentation-use-case-flow">
          <article class="presentation-use-case-step">
            <span>01</span><i data-lucide="send"></i><div><strong>Início da pesquisa</strong><small>A operação informa os dados necessários para iniciar a análise.</small></div><strong>EMPRESA</strong>
          </article>
          <article class="presentation-use-case-step">
            <span>02</span><i data-lucide="scan-face"></i><div><strong>Biometria e identidade</strong><small>O motorista confirma a presença e realiza a validação facial.</small></div><strong>MOTORISTA</strong>
          </article>
          <article class="presentation-use-case-step">
            <span>03</span><i data-lucide="database"></i><div><strong>Consultas + regras</strong><small>CNH, ANTT/RNTRC, processos, veículo e regras formam o contexto.</small></div><strong>CARGO SCORE</strong>
          </article>
          <article class="presentation-use-case-step">
            <span>04</span><i data-lucide="shield-check"></i><div><strong>Resultado de liberação</strong><small>A decisão operacional é consolidada em um processo rastreável.</small></div><strong>DECISÃO</strong>
          </article>
          <article class="presentation-use-case-step">
            <span>05</span><i data-lucide="route"></i><div><strong>Execução da viagem</strong><small>Cargo Truck e Plataforma Cargo acompanham atividades, eventos e entrega.</small></div><strong>OPERAÇÃO</strong>
          </article>
        </div>

        <div class="presentation-use-case-outcomes">
          <div><small>PROCESSO</small><strong>Menos etapas manuais</strong></div>
          <div><small>IDENTIDADE</small><strong>Validação antes da viagem</strong></div>
          <div><small>GOVERNANÇA</small><strong>Fluxo rastreável</strong></div>
        </div>
      </div>
    </div>
    <div class="slide-footer dark-footer"><span>Exemplo de operação</span><span></span></div>
  `;

  integrationSlide.after(section);

  if (window.lucide) {
    window.lucide.createIcons();
  }
}

createOperationalUseCaseSlide();
