$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
Set-Location $Root

if (-not (Test-Path ".env")) {
    throw "Arquivo .env não encontrado. Copie .env.example para .env e ajuste as configurações."
}

Get-Content ".env" | ForEach-Object {
    $line = $_.Trim()
    if (-not $line -or $line.StartsWith("#")) { return }
    $parts = $line.Split("=", 2)
    if ($parts.Count -ne 2) { return }
    $name = $parts[0].Trim()
    $value = $parts[1].Trim()
    if (($value.StartsWith('"') -and $value.EndsWith('"')) -or ($value.StartsWith("'") -and $value.EndsWith("'"))) {
        $value = $value.Substring(1, $value.Length - 2)
    }
    [Environment]::SetEnvironmentVariable($name, $value, "Process")
}

Write-Host "[1/4] Baixando dependências Go..."
go mod download

Write-Host "[2/4] Gerando componentes templ..."
go run github.com/a-h/templ/cmd/templ@v0.3.943 generate

Write-Host "[3/4] Aplicando migrations..."
go run ./cmd/migrate up

Write-Host "[4/4] Iniciando ViaGate Commercial Platform..."
go run ./cmd/server
