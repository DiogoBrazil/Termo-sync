# Matriz de validação

## Automatizados locais

| Área | Evidência | Estado |
|---|---|---|
| Go | `go test -race ./...`, `go vet ./...` e `GOOS=windows GOARCH=amd64 go vet ./...` | Passaram integralmente |
| Biblioteca padrão | `go list encoding/json` | Localizado; sem dependência externa |
| Estados | Trilhas, início único, transcrição sem segmentos, falha, retry, cancelamento, resultados antigos, edição vazia (`internal/application/session_test.go`, `settings_test.go`) | Passaram |
| HTTP | 401, 402, 429, 503, redirecionamento, destinos, JSON inválido/trailing/null, cancelamento, POST único (`internal/providers/http_test.go`) | Passaram (mocks) |
| DeepSeek | Bearer, `thinking.type: disabled`, `stream: false`, truncamento (`length`), metadados (`usage`/`model`), cobertura e rótulos (`internal/providers/deepseek_test.go`) | Passaram (mocks) |
| AssemblyAI | Upload binário, criação, original bruto e modelo, polling e cancelamento, exclusão manual, ID inválido (`internal/providers/assemblyai_test.go`) | Passaram (mocks; chamada remota real pendente) |
| Fidelidade editorial | Regra de depoente único no prompt, números/datas, negações/rumores/incertezas/contradições, segmentação sem perdas (`internal/editorial/policy_test.go`, `revision_test.go`) | Passaram |
| Mídia e temporários | MP3 direto, vídeo convertido, duas trilhas, corrupção, sem áudio, nomes especiais, cancelamento, limpeza após falha e recuperação de resíduos >24h (`internal/media/media_test.go`) | Passaram com FFmpeg Linux 7.0.2 local e dados sintéticos |
| Integração | WAV sintético → FFmpeg → AssemblyAI mock → DeepSeek mock (sequência automática) → edição humana → exclusão remota mock (`internal/application/workflow_test.go`) | Passou |
| Frontend (unidade) | Ponte nativa obrigatória e argumentos tipados (`frontend/src/backend.test.ts`) | Passou (Vitest) |
| Frontend (E2E/UI) | Início sem confirmações e sem nomes de serviços externos, progresso por etapa, original e proposta só ao final, sucesso que some em 2 s, limpeza de API Keys, preservação e cópia de edição humana, contraste WCAG AA (`>= 4.5:1`), temas claro/escuro, `1366×768`, `1920×1080`, `900×768`, `700×768` (abas), modal de confirmação centralizado, troca de arquivo com decisão sobre o registro, erro de ação que some em 6 s, DPI 200%, teclado e `prefers-reduced-motion` (`frontend/e2e/studio.spec.ts`) | Passou (Playwright) |
| Dependências JS | `npm audit` | Zero vulnerabilidades |
| Windows x64 | Compilação cruzada do executável Wails (`./scripts/build-windows.sh` → `build/DepoimentoStudio.exe`) | Passou com frontend final embutido |
| Governança de release | `python3 scripts/check-release.py` e guarda `!ifndef REDISTRIBUTION_VERIFIED` em `packaging/installer.nsi` | Bloqueio fail-closed ativo até verificação completa das fontes/licenças do bundle FFmpeg |

Os testes que requerem mídia são pulados quando `DEPOIMENTO_TEST_FFMPEG` e `DEPOIMENTO_TEST_FFPROBE` não estão definidos. Para evidência completa, configure ambos com caminhos absolutos e execute `go test -race ./...`.

## Roteiro obrigatório Windows 11 limpo (usuário)

1. Registrar versão/build do Windows, resolução, DPI, WebView2 e hash SHA-256 do executável (`build/DepoimentoStudio.exe.sha256`). Não usar depoimento real no primeiro teste.
2. Instalar por usuário sem administrador (após conclusão do pacote de licenças FFmpeg) ou executar na pasta de teste local conforme `docs/install.md`. Verificar abertura e desinstalação. Testar o bloqueio/encaminhamento quando falta WebView2.
3. Abrir configurações; inserir chaves novas; salvar; confirmar indicador. Fechar/reabrir e confirmar persistência no Credential Manager (`DepoimentoStudio/assemblyai` e `DepoimentoStudio/deepseek`). Remover e confirmar bloqueio de envio sem chave. Nunca enviar prints com segredo visível.
4. Selecionar MP3 fictício de um único depoente; ler os avisos e iniciar a transcrição.
5. Repetir com vídeo fictício e múltiplas trilhas: exigir escolha manual. Conferir que original não foi sobrescrito e temporário tem ACL privada e é limpo.
6. Cancelar validação, conversão, upload, polling e revisão. Conferir UI responsiva, liberação do arquivo e ausência de publicação tardia. Cancelamento local não deve ser apresentado como exclusão remota.
7. Conferir que a revisão começa sozinha após a transcrição e que original e proposta aparecem juntos só ao final.
8. Conferir negativas, números, datas, rumores, contradições e incertezas. Edição humana deve sobreviver a consultas de estado e prevalecer no copiar/exportar. Verificar TXT UTF-8 com acentos e quebras de linha CRLF.
9. Induzir saldo insuficiente/credencial inválida sem compartilhar chaves; conferir mensagem sem conteúdo remoto. Não repetir POST automaticamente após falha ambígua.
10. Solicitar exclusão remota manual; verificar no painel do provedor as condições/artefatos aplicáveis. Repetir revisão sem transcrever novamente.
11. Verificar claro/escuro, Tab/Shift+Tab, foco, labels, contraste, Ctrl+A/C, movimento reduzido; `1366×768`, `1920×1080` e DPI 150/200%. Janela estreita deve permitir alternar original/proposta.
12. Encerrar durante trabalho e reabrir; verificar limpeza normal e recuperação de resíduos antigos. Remover chaves nas configurações antes de desinstalar, se desejado.

Registrar resultado, data, captura sem dados sensíveis e hash do artefato. A release permanece pendente dessa aprovação funcional; compilação cruzada não equivale a execução Windows.
