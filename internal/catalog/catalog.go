package catalog

type PricingModel struct {
	ID      string
	Title   string
	Summary string
}

type Group struct {
	ID         string
	Title      string
	ShortTitle string
	Summary    string
	Items      []Item
}

type Item struct {
	ID              string
	Label           string
	Unit            string
	Models          []string
	DefaultOptional bool
}

var PricingModels = []PricingModel{
	{ID:"per_item",Title:"Análise por item",Summary:"Cada cadastro ou consulta é processado individualmente, com valor separado para motorista e veículo."},
	{ID:"bundle",Title:"Análise por conjunto",Summary:"Motorista e veículos são processados de forma unificada, simplificando cadastro e consulta."},
	{ID:"item_and_bundle",Title:"Item + conjunto",Summary:"Apresente as duas modalidades na mesma proposta e informe os valores de cada alternativa."},
	{ID:"custom",Title:"Condições específicas",Summary:"Use somente quando a negociação não se enquadrar nos modelos comerciais padronizados."},
}

var Groups = []Group{
	{ID:"score",Title:"Cargo Score | Análise cadastral",ShortTitle:"Cargo Score",Summary:"Pesquisa cadastral com autorização biométrica, validações oficiais e análise de risco para motorista e veículo.",Items:[]Item{
		{ID:"score-item-driver-register",Label:"Cadastro | Motorista — Frota, agregado e terceiro",Unit:"cadastro",Models:[]string{"per_item","item_and_bundle"}},
		{ID:"score-item-driver-other",Label:"Cadastro | Motorista — Outras funções",Unit:"cadastro",Models:[]string{"per_item","item_and_bundle"}},
		{ID:"score-item-vehicle-register",Label:"Cadastro | Veículos",Unit:"cadastro",Models:[]string{"per_item","item_and_bundle"}},
		{ID:"score-item-driver-query",Label:"Consulta | Motorista — Frota, agregado e terceiro",Unit:"consulta",Models:[]string{"per_item","item_and_bundle"}},
		{ID:"score-item-vehicle-query",Label:"Consulta | Veículos",Unit:"consulta",Models:[]string{"per_item","item_and_bundle"}},
		{ID:"score-item-reanalysis",Label:"Reanálise de campos preenchidos incorretamente",Unit:"reanálise",Models:[]string{"per_item","item_and_bundle"}},
		{ID:"score-bundle-register",Label:"Cadastro | Motorista + veículos + colaboradores",Unit:"conjunto",Models:[]string{"bundle","item_and_bundle"}},
		{ID:"score-bundle-query",Label:"Consulta | Motorista + veículos + colaboradores",Unit:"conjunto",Models:[]string{"bundle","item_and_bundle"}},
		{ID:"score-bundle-reanalysis",Label:"Reanálise | Conjunto",Unit:"reanálise",Models:[]string{"bundle","item_and_bundle"}},
	}},
	{ID:"authentication",Title:"Consultas e autenticação",ShortTitle:"Consultas e autenticação",Summary:"Consultas pontuais, autenticação de formulários e validações complementares ao processo cadastral.",Items:[]Item{
		{ID:"auth-cargo",Label:"Cargo Autenticador",Unit:"consulta",DefaultOptional:true},
		{ID:"auth-lawsuits",Label:"Pesquisa processo criminal, trabalhista, cível e familiar",Unit:"consulta",DefaultOptional:true},
		{ID:"auth-victimology-state",Label:"Vitimologia por estado",Unit:"estado",DefaultOptional:true},
		{ID:"auth-victimology-integrated",Label:"Vitimologia integrada",Unit:"consulta",DefaultOptional:true},
		{ID:"auth-antt",Label:"Consulta veículos | ANTT",Unit:"consulta",DefaultOptional:true},
		{ID:"auth-on-demand",Label:"Avulso | Consultas e autenticação de formulários",Unit:"consulta",DefaultOptional:true},
	}},
	{ID:"logistics",Title:"Cargo Truck | Aplicativo e logística",ShortTitle:"Cargo Truck",Summary:"Aplicativo para cadastro, coletas, entregas, eventos de parada e rastreamento por GPS do smartphone do motorista.",Items:[]Item{
		{ID:"truck-first-without-score",Label:"Cargo Truck | Primeira viagem (sem Score)",Unit:"viagem",DefaultOptional:true},
		{ID:"truck-next-without-score",Label:"Cargo Truck | Viagens subsequentes (sem Score)",Unit:"viagem",DefaultOptional:true},
		{ID:"truck-with-score",Label:"Cargo Truck | Primeira viagem e subsequentes (com Score)",Unit:"viagem",DefaultOptional:true},
	}},
	{ID:"prevention",Title:"Prevenção",ShortTitle:"Prevenção",Summary:"Recursos complementares para gestão preventiva, incluindo multas e histórico veicular completo.",Items:[]Item{
		{ID:"prevention-fines",Label:"Sistema Gestor de Multas",Unit:"veículo",DefaultOptional:true},
		{ID:"prevention-history",Label:"Histórico Veicular Completo",Unit:"consulta",DefaultOptional:true},
	}},
	{ID:"monitoring",Title:"Monitoramento de veículos | Integração com gerenciadora",ShortTitle:"Monitoramento de veículos",Summary:"Monitoramento satelital via integração com gerenciadora, com opções por veículo, viagem e checklist.",Items:[]Item{
		{ID:"monitoring-fixed",Label:"Veículo Fixo",Unit:"veículo",DefaultOptional:true},
		{ID:"monitoring-trip",Label:"Viagem Avulsa",Unit:"viagem",DefaultOptional:true},
		{ID:"monitoring-autotrac",Label:"ADE Autotrac",Unit:"veículo",DefaultOptional:true},
		{ID:"monitoring-checklist",Label:"Check List",Unit:"veículo",DefaultOptional:true},
	}},
}

func ItemByID(id string)(Group,Item,bool){for _,group:=range Groups{for _,item:=range group.Items{if item.ID==id{return group,item,true}}};return Group{},Item{},false}

func ModelAllows(item Item,model string)bool{if len(item.Models)==0{return true};for _,candidate:=range item.Models{if candidate==model{return true}};return false}
