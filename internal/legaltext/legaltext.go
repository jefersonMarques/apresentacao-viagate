package legaltext

import "crypto/sha256"

const ProposalAcceptanceVersion = "v1"
const ProposalAcceptanceText = "Declaro possuir poderes ou autorização para prosseguir com esta contratação em nome da empresa. Ao aceitar, confirmo a versão apresentada da proposta e autorizo o prosseguimento para a etapa de contratação."

const SignatureConsentVersion = "v1"
const SignatureConsentText = "Declaro que li o contrato apresentado acima e manifesto minha vontade de assiná-lo eletronicamente."

func SHA256(text string) []byte {
	hash:=sha256.Sum256([]byte(text))
	return hash[:]
}
