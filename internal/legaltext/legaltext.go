package legaltext

import "crypto/sha256"

const ProposalAcceptanceVersion = "v1"
const ProposalAcceptanceText = "O responsável abaixo confirma que está prosseguindo em nome da empresa. A assinatura do contrato ocorrerá em etapa posterior com verificação por código de uso único; a biometria facial já está prevista para a versão final.\nDeclaro possuir poderes ou autorização para prosseguir com esta contratação em nome da empresa.\nAção: Aceitar proposta e continuar"

const SignatureConsentVersion = "v1"
const SignatureConsentText = "Li o contrato e concordo em assiná-lo eletronicamente."

func SHA256(text string) []byte {
	hash:=sha256.Sum256([]byte(text))
	return hash[:]
}
