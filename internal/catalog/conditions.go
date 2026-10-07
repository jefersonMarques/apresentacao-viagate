package catalog

type Condition struct {
	ID        string
	Text      string
	Groups    []string
	IsActive  bool
	SortOrder int
}

var StandardConditions = []Condition{
	{ID: "score-turnaround", Text: "O retorno da pesquisa cadastral ocorre em até 10 minutos após a conclusão da biometria, salvo indisponibilidade de fontes externas.", Groups: []string{"score"}, IsActive: true, SortOrder: 10},
	{ID: "score-biometry", Text: "A autorização biométrica está incluída no fluxo da análise cadastral.", Groups: []string{"score"}, IsActive: true, SortOrder: 20},
	{ID: "score-channels", Text: "A operação pode utilizar link web e aplicativo conforme a configuração comercial contratada.", Groups: []string{"score", "logistics"}, IsActive: true, SortOrder: 30},
	{ID: "support-group", Text: "Após a formalização comercial poderá ser criado grupo de atendimento para implantação e acompanhamento da operação.", IsActive: true, SortOrder: 40},
	{ID: "victimology", Text: "Vitimologia por estado possui precificação conforme abrangência solicitada e prazo de retorno estimado de até 3 horas úteis.", Groups: []string{"authentication"}, IsActive: true, SortOrder: 50},
	{ID: "logistics-scope", Text: "Os recursos logísticos complementam a gestão de risco, mas não substituem integralmente um processo completo de gerenciamento de risco.", Groups: []string{"logistics", "monitoring"}, IsActive: true, SortOrder: 60},
	{ID: "integration", Text: "Integrações dependem de documentação técnica, disponibilidade e homologação dos sistemas envolvidos.", Groups: []string{"monitoring", "logistics", "authentication"}, IsActive: true, SortOrder: 70},
	{ID: "customization", Text: "Customizações fora do escopo contratado serão previamente analisadas e, quando aplicável, orçadas por hora técnica.", IsActive: true, SortOrder: 80},
	{ID: "expenses", Text: "Despesas extraordinárias de deslocamento, alimentação e hospedagem não estão incluídas, quando aplicáveis.", IsActive: true, SortOrder: 90},
	{ID: "score-biometry-proof-life-geolocation", Text: "Biometria Facial com Prova de Vida e Geolocalização Inclusa no Cadastro e na Consulta", Groups: []string{"score"}, IsActive: true, SortOrder: 100},
	{ID: "victimology-integrated-register", Text: "Vitimologia Integrada Inclusa no Cadastro", Groups: []string{"score", "authentication"}, IsActive: true, SortOrder: 110},
}
