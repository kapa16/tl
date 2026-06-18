# Первая загрузка базовой конфигурации в пустую Neo4j.
# В .env установите RESET_DATABASE=true, затем запустите этот скрипт.
# После успешной индексации верните RESET_DATABASE=false в .env.

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$envContent = Get-Content ".env" -Raw
if ($envContent -notmatch "RESET_DATABASE=true") {
    Write-Warning "В .env не задано RESET_DATABASE=true. Для первой загрузки раскомментируйте/установите RESET_DATABASE=true"
}

Write-Host "Запуск Neo4j + MCP (первая индексация базы)..." -ForegroundColor Cyan
docker compose up -d

Write-Host @"

Сервисы запущены. Следите за индексацией:
  docker logs -f 1c_graph_metadata
  curl http://localhost:8006/status

Neo4j Browser: http://localhost:7474  (neo4j / пароль из NEO4J_PASSWORD)

После завершения metadata_report_load и vector_indexing установите в .env:
  RESET_DATABASE=false

"@ -ForegroundColor Green
