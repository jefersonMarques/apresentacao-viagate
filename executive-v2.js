function loadExecutiveStyles() {
  if (document.querySelector('link[data-viagate-executive-v2]')) {
    return;
  }

  const link = document.createElement('link');
  link.rel = 'stylesheet';
  link.href = './executive-v2.css';
  link.dataset.viagateExecutiveV2 = 'true';
  document.head.appendChild(link);
}

function replaceSlideContent(slideId, layoutClass, content) {
  const slide = document.querySelector(slideId);
  const inner = slide?.querySelector(':scope > .slide-inner');

  if (!slide || !inner) {
    return null;
  }

  inner.className = `slide-inner ${layoutClass}`;
  inner.innerHTML = content;
  return slide;
}

function updateMarketSlide() {
  replaceSlideContent('#slide-02', 'exec-market-layout', `
    <div class="section-copy exec-market-copy exec-animate">
      <p class="kicker dark-kicker">CENÁRIO DO MERCADO</p>
      <h2>O risco começa quando a decisão depende de <span>informação fragmentada.</span></h2>
      <p class="lead dark-lead">Bases desatualizadas, conferência manual e validações sem biometria tornam a análise mais lenta, menos rastreável e mais exposta.</p>
      <div class="exec-market-outcome">
        <small>IMPACTO OPERACIONAL</small>
        <strong>Mais tempo · menos rastreabilidade · maior exposição</strong>
      </div>
    </div>

    <div class="exec-risk-register" aria-label="Principais riscos do modelo tradicional">
      <article class="exec-risk-row exec-animate">
        <span class="exec-index">01</span>
        <i data-lucide="database-zap"></i>
        <div><small>DADOS</small><h3>Bases desatualizadas + análise manual</h3><p>Decisões podem partir de informações incompletas e de conferências operacionais sujeitas a erro.</p></div>
      </article>
      <article class="exec-risk-row exec-animate">
        <span class="exec-index">02</span>
        <i data-lucide="scan-face"></i>
        <div><small>IDENTIDADE</small><h3>Sem biometria e prova de vida</h3><p>Sem confirmação facial, a identidade de quem realiza a validação pode não ser comprovada com segurança.</p></div>
      </article>
      <article class="exec-risk-row exec-animate">
        <span class="exec-index">03</span>
        <i data-lucide="route-off"></i>
        <div><small>PROCESSO</small><h3>Exposição, demora e baixa rastreabilidade</h3><p>Documentos circulam entre canais, o tempo de análise cresce e a auditoria do processo se torna mais difícil.</p></div>
      </article>
    </div>
  `);
}

function updateDecisionSlide() {
  replaceSlideContent('#slide-04', 'exec-decision-layout', `
    <div class="section-copy exec-decision-heading exec-animate">
      <p class="kicker dark-kicker">CARGO SCORE · RESULTADO DA ANÁLISE</p>
      <h2>Uma decisão consolidada, <span>não apenas uma lista de consultas.</span></h2>
      <p class="lead dark-lead">O Cargo Score reúne identidade, fontes consultadas, veículo, proprietário e regras parametrizadas em um processo rastreável.</p>
    </div>

    <div class="exec-decision-stage">
      <article class="exec-decision-input exec-animate">
        <span class="exec-stage-label">ENTRADA</span>
        <i data-lucide="user-round"></i>
        <h3>Pessoa + operação</h3>
        <p>Dados necessários para iniciar a pesquisa e a validação.</p>
      </article>

      <div class="exec-decision-arrow exec-animate"><i data-lucide="arrow-right"></i></div>

      <article class="exec-decision-engine exec-animate">
        <header><span>CARGO SCORE</span><small>MOTOR DE ANÁLISE</small></header>
        <div><i data-lucide="scan-face"></i><span>Biometria e identidade</span><strong>01</strong></div>
        <div><i data-lucide="database"></i><span>CNH · ANTT/RNTRC · processos</span><strong>02</strong></div>
        <div><i data-lucide="car-front"></i><span>Veículo e proprietário</span><strong>03</strong></div>
        <div><i data-lucide="settings-2"></i><span>Regras parametrizadas</span><strong>04</strong></div>
      </article>

      <div class="exec-decision-arrow exec-animate"><i data-lucide="arrow-right"></i></div>

      <article class="exec-decision-output exec-animate">
        <span class="exec-stage-label">SAÍDA</span>
        <i data-lucide="file-check-2"></i>
        <small>DOCUMENTO DE LIBERAÇÃO</small>
        <h3>Resultado consolidado</h3>
        <p>Informação organizada para apoiar a decisão operacional.</p>
        <div><i data-lucide="shield-check"></i><span>PROCESSO RASTREÁVEL</span></div>
      </article>
    </div>
  `);
}

function updateScoreFlowSlide() {
  replaceSlideContent('#slide-05', 'exec-score-flow-layout', `
    <div class="section-copy centered-copy exec-score-flow-heading exec-animate">
      <p class="kicker">COMO O CARGO SCORE FUNCIONA</p>
      <h2>Da pesquisa ao resultado em <span>seis etapas.</span></h2>
      <p class="lead">A empresa inicia a análise, o pesquisado valida a identidade e o sistema consolida consultas e regras até o resultado operacional.</p>
    </div>

    <div class="exec-timeline" aria-label="Etapas do Cargo Score">
      <article class="exec-timeline-step exec-animate"><span>01</span><i data-lucide="send"></i><h3>Pesquisa</h3><p>Início da análise e envio do link seguro.</p></article>
      <article class="exec-timeline-step exec-animate"><span>02</span><i data-lucide="scan-face"></i><h3>Biometria</h3><p>Validação facial com prova de vida.</p></article>
      <article class="exec-timeline-step exec-animate"><span>03</span><i data-lucide="badge-check"></i><h3>Cadastro</h3><p>Identidade e CNH entram no processo.</p></article>
      <article class="exec-timeline-step exec-animate"><span>04</span><i data-lucide="database"></i><h3>Consultas</h3><p>ANTT/RNTRC, processos, veículo e proprietário.</p></article>
      <article class="exec-timeline-step exec-animate"><span>05</span><i data-lucide="settings-2"></i><h3>Regras</h3><p>Critérios conforme o perfil da operação.</p></article>
      <article class="exec-timeline-step is-final exec-animate"><span>06</span><i data-lucide="shield-check"></i><h3>Resultado</h3><p>Decisão e documentação de liberação.</p></article>
    </div>

    <div class="exec-time-note exec-animate"><i data-lucide="timer"></i><div><small>PRAZO INFORMADO</small><strong>1 a 10 minutos após a validação facial</strong></div><span>Varia conforme a parametrização da operação.</span></div>
  `);
}

function updateSourceMapSlide() {
  replaceSlideContent('#slide-06', 'exec-source-layout', `
    <div class="section-copy exec-source-heading exec-animate">
      <p class="kicker dark-kicker">CONTEXTO DA DECISÃO</p>
      <h2>O Cargo Score conecta diferentes fontes <span>em uma única análise.</span></h2>
      <p class="lead dark-lead">Os dados não são apresentados de forma isolada: eles compõem o contexto utilizado pelas regras da operação.</p>
    </div>

    <div class="exec-source-map">
      <div class="exec-source-column">
        <article class="exec-source-node exec-animate"><i data-lucide="scan-face"></i><div><small>PESSOA</small><h3>Identidade</h3><p>Biometria e prova de vida.</p></div></article>
        <article class="exec-source-node exec-animate"><i data-lucide="badge-check"></i><div><small>DOCUMENTO</small><h3>CNH</h3><p>Dados da habilitação.</p></div></article>
        <article class="exec-source-node exec-animate"><i data-lucide="scale"></i><div><small>CONTEXTO</small><h3>Processos</h3><p>Cíveis, trabalhistas e criminais.</p></div></article>
      </div>

      <div class="exec-source-engine exec-animate">
        <i data-lucide="shield-check"></i>
        <small>CARGO SCORE</small>
        <strong>Contexto + regras</strong>
        <span>Resultado operacional</span>
      </div>

      <div class="exec-source-column is-right">
        <article class="exec-source-node exec-animate"><i data-lucide="truck"></i><div><small>TRANSPORTE</small><h3>ANTT / RNTRC</h3><p>Regularidade do transporte.</p></div></article>
        <article class="exec-source-node exec-animate"><i data-lucide="car-front"></i><div><small>VEÍCULO</small><h3>Veículo + proprietário</h3><p>Validação conjunta da operação.</p></div></article>
        <article class="exec-source-node exec-animate"><i data-lucide="shield-alert"></i><div><small>RISCO</small><h3>Sinais de fraude</h3><p>Indícios avaliados no contexto.</p></div></article>
      </div>
    </div>
  `);
}

function updateConsultationsSlide() {
  replaceSlideContent('#slide-07', 'exec-consultations-layout', `
    <div class="section-copy exec-consultations-copy exec-animate">
      <p class="kicker">CONSULTAS AVULSAS</p>
      <h2>Use apenas a informação <span>que a operação precisa.</span></h2>
      <p class="lead">Além do Cargo Score, consultas e validações podem ser contratadas separadamente conforme a necessidade.</p>
      <div class="exec-consultations-note"><i data-lucide="layers-3"></i><span>Uso pontual ou integrado ao fluxo do cliente.</span></div>
    </div>

    <div class="exec-consultation-table" aria-label="Consultas avulsas ViaGate">
      <article class="exec-consultation-row exec-animate"><span>01</span><i data-lucide="truck"></i><h3>ANTT / RNTRC</h3><p>Regularidade e informações do transporte rodoviário.</p></article>
      <article class="exec-consultation-row exec-animate"><span>02</span><i data-lucide="badge-check"></i><h3>CNH</h3><p>Consulta e validação dos dados da habilitação.</p></article>
      <article class="exec-consultation-row exec-animate"><span>03</span><i data-lucide="car-front"></i><h3>Histórico veicular</h3><p>Histórico, restrições e contexto do veículo.</p></article>
      <article class="exec-consultation-row exec-animate"><span>04</span><i data-lucide="scale"></i><h3>Processos</h3><p>Informações processuais para ampliar o contexto da análise.</p></article>
      <article class="exec-consultation-row exec-animate"><span>05</span><i data-lucide="search"></i><h3>Consultas especiais</h3><p>Pesquisas adicionais conforme a necessidade da operação.</p></article>
      <article class="exec-consultation-row exec-animate"><span>06</span><i data-lucide="scan-face"></i><h3>Biometria</h3><p>Validação facial com prova de vida e confirmação de identidade.</p></article>
    </div>
  `);
}

function updatePortfolioSlide() {
  replaceSlideContent('#slide-09', 'exec-portfolio-layout', `
    <div class="section-copy exec-portfolio-heading exec-animate">
      <p class="kicker">PORTFÓLIO VIAGATE</p>
      <h2>Quatro linhas de solução. <span>Um único ecossistema.</span></h2>
      <p class="lead">Validação, consultas, execução logística e integração organizadas conforme o momento da operação.</p>
    </div>

    <div class="exec-portfolio-list">
      <article class="exec-portfolio-row exec-animate"><span class="exec-portfolio-index">01</span><div class="exec-portfolio-title"><small>ANÁLISE E VALIDAÇÃO</small><strong>Identidade e gerenciamento de risco</strong></div><ul><li>Cargo Score</li><li>Biometria Facial</li><li>Autenticador</li><li>Pesquisa Cadastral</li></ul></article>
      <article class="exec-portfolio-row exec-animate"><span class="exec-portfolio-index">02</span><div class="exec-portfolio-title"><small>CONSULTAS</small><strong>Dados para análise operacional</strong></div><ul><li>CNH</li><li>ANTT</li><li>Histórico Veicular</li><li>Processos</li><li>Antecedentes</li><li>Vitimologia</li></ul></article>
      <article class="exec-portfolio-row exec-animate"><span class="exec-portfolio-index">03</span><div class="exec-portfolio-title"><small>OPERAÇÃO</small><strong>Execução e acompanhamento logístico</strong></div><ul><li>Cargo Truck</li><li>Plataforma Cargo</li><li>Gestão Logística</li><li>Gestão Securitária</li></ul></article>
      <article class="exec-portfolio-row exec-animate"><span class="exec-portfolio-index">04</span><div class="exec-portfolio-title"><small>INTEGRAÇÃO</small><strong>Tecnologia dentro do ambiente do cliente</strong></div><ul><li>API</li><li>White Label</li></ul></article>
    </div>
  `);
}

function updateEcosystemSlide() {
  replaceSlideContent('#slide-10', 'exec-ecosystem-layout', `
    <div class="section-copy centered-copy exec-ecosystem-heading exec-animate">
      <p class="kicker dark-kicker">DO RISCO À EXECUÇÃO</p>
      <h2>A análise termina. <span>A operação continua.</span></h2>
      <p class="lead dark-lead">Cargo Score, Cargo Truck e Plataforma Cargo formam uma sequência da liberação ao acompanhamento da viagem.</p>
    </div>

    <div class="exec-ecosystem-chain">
      <article class="exec-ecosystem-stage exec-animate"><small>ANTES DA VIAGEM</small><span>01</span><i data-lucide="shield-check"></i><h3>Cargo Score</h3><p>Analisa o motorista e retorna a decisão operacional.</p></article>
      <div class="exec-chain-connector exec-animate"><i data-lucide="arrow-right"></i></div>
      <article class="exec-ecosystem-stage exec-animate"><small>EXECUÇÃO</small><span>02</span><i data-lucide="smartphone"></i><h3>Cargo Truck</h3><p>Entrega atividades ao motorista e registra a execução da viagem.</p></article>
      <div class="exec-chain-connector exec-animate"><i data-lucide="arrow-right"></i></div>
      <article class="exec-ecosystem-stage exec-animate"><small>GESTÃO</small><span>03</span><i data-lucide="monitor-dot"></i><h3>Plataforma Cargo</h3><p>Centraliza rastreamento, eventos e timeline da operação.</p></article>
    </div>
  `);
}

function updateOperationSlide() {
  replaceSlideContent('#slide-11', 'exec-operation-layout', `
    <figure class="exec-operation-visual exec-animate">
      <img src="./assets/cargo-truck-slide-11.webp" alt="Operação logística com Cargo Truck e Plataforma Cargo" loading="eager" decoding="async" />
      <figcaption><span>OPERAÇÃO EM CAMPO</span><strong>Aplicativo + execução + rastreabilidade</strong></figcaption>
    </figure>

    <div class="exec-operation-copy exec-animate">
      <p class="kicker">CARGO TRUCK + PLATAFORMA CARGO</p>
      <h2>Da atividade recebida <span>à entrega concluída.</span></h2>
      <p class="lead">O aplicativo acompanha a execução da viagem enquanto a plataforma organiza os eventos e o histórico da operação.</p>
      <div class="operation-steps exec-operation-steps">
        <div><span>01</span><i data-lucide="package-open"></i><div><strong>Atividade</strong><small>Coletas e entregas são enviadas ao motorista.</small></div></div>
        <div><span>02</span><i data-lucide="route"></i><div><strong>Viagem</strong><small>O início da atividade passa a registrar rota e execução.</small></div></div>
        <div><span>03</span><i data-lucide="circle-dot-dashed"></i><div><strong>Eventos</strong><small>Paradas e ocorrências ficam registradas no fluxo.</small></div></div>
        <div><span>04</span><i data-lucide="file-check-2"></i><div><strong>Entrega</strong><small>A conclusão pode registrar o canhoto assinado.</small></div></div>
      </div>
    </div>
  `);
}

function updateIntegrationSlide() {
  replaceSlideContent('#slide-12', 'exec-integration-layout', `
    <div class="section-copy exec-integration-copy exec-animate">
      <p class="kicker dark-kicker">API + WHITE LABEL</p>
      <h2>A tecnologia pode entrar <span>no fluxo e na marca do cliente.</span></h2>
      <p class="lead dark-lead">Biometria, CNH, ANTT, veículo, antecedentes e outras consultas podem ser integradas ao software do cliente ou entregues com identidade visual própria.</p>
    </div>

    <div class="exec-integration-diagram">
      <div class="exec-integration-core exec-animate"><img src="./assets/logo-viagate-color.svg" alt="ViaGate" /><small>TECNOLOGIA VIAGATE</small><strong>Serviços de validação e consulta</strong></div>
      <div class="exec-integration-branch exec-animate">
        <span class="exec-branch-line"></span>
        <article><i data-lucide="plug"></i><small>INTEGRAÇÃO</small><h3>API</h3><p>Recursos ViaGate conectados diretamente ao software do cliente.</p><strong>SEU SOFTWARE</strong></article>
        <article><i data-lucide="panels-top-left"></i><small>PERSONALIZAÇÃO</small><h3>White label</h3><p>Logo, cores e URL próprias para uma experiência integrada à marca.</p><strong>SUA MARCA</strong></article>
      </div>
    </div>
  `);
}

function updateProofSlide() {
  replaceSlideContent('#slide-13', 'exec-proof-layout', `
    <div class="section-copy exec-proof-copy exec-animate">
      <p class="kicker">VALIDAÇÃO DE MERCADO</p>
      <h2>Experiência operacional <span>com reconhecimento do mercado.</span></h2>
      <p class="lead">Números e referências presentes no material institucional da ViaGate.</p>
    </div>

    <div class="exec-proof-metrics">
      <article class="exec-proof-primary exec-animate"><small>CLIENTES DIRETOS</small><strong>300+</strong><span>empresas atendidas diretamente</span></article>
      <article class="exec-proof-metric exec-animate"><small>EXPERIÊNCIA</small><strong>30+ anos</strong><span>de experiência acumulada no mercado securitário, gerenciamento de riscos e rastreamento</span></article>
      <article class="exec-proof-metric exec-animate"><small>HISTÓRICO DE SEGURANÇA</small><strong>5 anos</strong><span>de operação com biometria e validação de CNH sem evento adverso registrado</span></article>
      <article class="exec-proof-metric exec-animate"><small>TECNOLOGIA</small><strong>100%</strong><span>tecnologia própria, permitindo flexibilidade para atender diferentes operações</span></article>
      <article class="exec-proof-certification exec-animate"><i data-lucide="badge-check"></i><div><small>RECONHECIMENTO</small><strong>Auditada pela FENSEG</strong><span>e reconhecida pelas principais seguradoras.</span></div></article>
    </div>
  `);
}

function refineSupportingSlides() {
  document.querySelector('#slide-03')?.classList.add('exec-biometric-slide');
  document.querySelector('#slide-08')?.classList.add('exec-authenticator-slide');
  document.querySelector('#slide-14')?.classList.add('exec-insurers-slide');
  document.querySelector('#slide-15')?.classList.add('exec-team-slide');
}

function applyExecutivePresentation() {
  loadExecutiveStyles();
  updateMarketSlide();
  updateDecisionSlide();
  updateScoreFlowSlide();
  updateSourceMapSlide();
  updateConsultationsSlide();
  updatePortfolioSlide();
  updateEcosystemSlide();
  updateOperationSlide();
  updateIntegrationSlide();
  updateProofSlide();
  refineSupportingSlides();

  if (window.lucide) {
    window.lucide.createIcons();
  }
}

applyExecutivePresentation();