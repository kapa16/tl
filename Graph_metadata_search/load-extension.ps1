# Одноразовая загрузка расширения в Neo4j (profile load-extension).
# Требует: extension.env (скопировать из extension.env.example), запущенный neo4j.

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

if (-not (Test-Path "extension.env")) {
    Write-Error "Файл extension.env не найден. Выполните: copy extension.env.example extension.env"
}

Write-Host "Загрузка расширения в Neo4j (одноразовый контейнер)..." -ForegroundColor Cyan
docker compose --env-file .env --env-file extension.env --profile load-extension run --rm mcp-extension-loader

if ($LASTEXITCODE -ne 0) {
    Write-Error "Загрузка расширения завершилась с ошибкой (код $LASTEXITCODE)"
}

Write-Host "Готово. Проверка: docker logs 1c_graph_metadata | Select-Object -Last 30" -ForegroundColor Green
Write-Host "MCP: compare_base_and_extension(object_name='...', extension_name='...')" -ForegroundColor Green
