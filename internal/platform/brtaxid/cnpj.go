package brtaxid

import (
	"fmt"
	"strings"
	"unicode"
)

var cnpjFirstWeights=[]int{5,4,3,2,9,8,7,6,5,4,3,2}
var cnpjSecondWeights=[]int{6,5,4,3,2,9,8,7,6,5,4,3,2}

func CleanCNPJ(value string)(string,error){
	normalized,err:=NormalizeCNPJ(value)
	if err!=nil{return "",err}
	if !ValidCNPJ(normalized){return "",fmt.Errorf("invalid CNPJ")}
	return normalized,nil
}

func NormalizeCNPJ(value string)(string,error){
	var builder strings.Builder
	builder.Grow(14)
	for _,r:=range strings.ToUpper(strings.TrimSpace(value)){
		switch{
		case r>='0'&&r<='9':
			builder.WriteRune(r)
		case r>='A'&&r<='Z':
			builder.WriteRune(r)
		case r=='.'||r=='/'||r=='-'||unicode.IsSpace(r):
			continue
		default:
			return "",fmt.Errorf("CNPJ contains invalid character")
		}
	}
	normalized:=builder.String()
	if len(normalized)!=14{return "",fmt.Errorf("CNPJ must contain 14 positions")}
	for index:=0;index<12;index++{
		if !isCNPJBaseCharacter(normalized[index]){return "",fmt.Errorf("invalid CNPJ base")}
	}
	if normalized[12]<'0'||normalized[12]>'9'||normalized[13]<'0'||normalized[13]>'9'{return "",fmt.Errorf("CNPJ check digits must be numeric")}
	return normalized,nil
}

func ValidCNPJ(value string)bool{
	if len(value)!=14{return false}
	for index:=0;index<12;index++{if !isCNPJBaseCharacter(value[index]){return false}}
	if value[12]<'0'||value[12]>'9'||value[13]<'0'||value[13]>'9'{return false}
	first:=cnpjCheckDigit(value[:12],cnpjFirstWeights)
	if first!=int(value[12]-'0'){return false}
	second:=cnpjCheckDigit(value[:13],cnpjSecondWeights)
	return second==int(value[13]-'0')
}

func cnpjCheckDigit(value string,weights []int)int{
	if len(value)!=len(weights){return -1}
	sum:=0
	for index:=0;index<len(value);index++{
		characterValue:=int(value[index])-48
		sum+=characterValue*weights[index]
	}
	remainder:=sum%11
	if remainder==0||remainder==1{return 0}
	return 11-remainder
}

func isCNPJBaseCharacter(value byte)bool{
	return value>='0'&&value<='9'||value>='A'&&value<='Z'
}
