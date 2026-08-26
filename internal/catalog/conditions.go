package catalog

type Condition struct {
	ID     string
	Text   string
	Groups []string
}

var StandardConditions = []Condition{
	{ID:"score-turnaround",Text:"O retorno da pesquisa cadastral ocorre em até 10 minutos após a conclusão da biometria, salvo indisponibilidade de fontes externas.",Groups:[]string{"score"}},
	{ID:"score-biometry",Text:"A autorização biométrica está incluída no fluxo da análise cadastral.",Groups:[]string{"score"}},
	{ID:"score-channels",Text:"A operação pode utilizar link web e aplicativo conforme a configuração comercial contratada.",Groups:[]string{"score","logistics"}},
	{ID:"support-group",Text:"Após a formalização comercial poderá ser criado grupo de atendimento para implantação e acompanhamento da operação."},
	{ID:"victimology",Text:"Vitimologia por estado possui precificação conforme abrangência solicitada e prazo de retorno estimado de até 3 horas úteis.",Groups:[]string{"authentication"}},
	{ID:"logistics-scope",Text:"Os recursos logísticos complementam a gestão de risco, mas não substituem integralmente um processo completo de gerenciamento de risco.",Groups:[]string{"logistics","monitoring"}},
	{ID:"integration",Text:"Integrações dependem de documentação técnica, disponibilidade e homologação dos sistemas envolvidos.",Groups:[]string{"monitoring","logistics","authentication"}},
	{ID:"customization",Text:"Customizações fora do escopo contratado serão previamente analisadas e, quando aplicável, orçadas por hora técnica."},
	{ID:"expenses",Text:"Despesas extraordinárias de deslocamento, alimentação e hospedagem não estão incluídas, quando aplicáveis."},
}
