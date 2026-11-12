# Plano técnico e funcional — Depoimento Studio

## 1. Objetivo e decisões aprovadas

Construir aplicativo desktop individual para **Windows 11 x64**, operando sobre **um arquivo por vez** em memória, mantendo estritamente separados:
1. **Transcrição automática original** (imutável durante toda a sessão);
2. **Proposta revisada** (gerada sob regras editoriais conservadoras);
3. **Edições humanas** (preservadas separadamente e prioritárias ao copiar/exportar).

Regras de negócio e segurança confirmadas (os itens "Formato" e "Consentimento" foram substituídos em 2026-11-12; ver `docs/decisions.md`):
- **Formato:** Diálogo (`Falante X: ...`) como padrão; narrativa em primeira pessoa somente após identificação manual do depoente e enviando à revisão exclusivamente os segmentos do depoente confirmado, sem jamais usar perguntas de terceiros para completar respostas.
- **Consentimento e privacidade:** Sem pseudonimização automática. Autorização explícita, inicialmente desmarcada, por arquivo e por provedor (AssemblyAI para áudio; DeepSeek para texto), além de ciência explícita da limitação de hesitações em português antes da transcrição.
- **Saída e retenção local:** Edição da proposta na interface, cópia para área de transferência e exportação em `.txt` UTF-8 (CRLF no Windows), sem sobrescrever o arquivo de mídia original, sem geração de `.docx` e sem histórico automático em disco.
- **Trilhas ambíguas:** Arquivos com mais de uma trilha de áudio pausam em `awaiting_track` e exigem escolha manual.
- **Exclusão remota:** Ação manual durante a sessão para solicitar `DELETE /v2/transcript/{id}` na AssemblyAI, ou confirmação explícita de retenção antes de trocar de arquivo.

## 2. Comparação tecnológica e stack fixada

| Aspecto | Go + Wails 3 | Rust + Tauri 2 |
|---|---|---|
| Maturidade consultada em novembro de 2026 | Repositório oficial identifica `v3.0.0` como estável (requer Go `>= 1.26.0`) | Linha 2 consolidada, com modelo de capabilities e distribuição NSIS/MSI |
| Simplicidade neste projeto | Go favorece clientes HTTP/REST, controle de contexto/cancelamento, ACL de diretórios e subprocessos `ffprobe`/`ffmpeg` com pouca infraestrutura | Controle rigoroso, porém maior complexidade para orquestração de subprocessos e integração Win32 |
| Recursos e Windows | WebView2 do sistema e empacotamento NSIS por usuário | Também utiliza WebView2 do sistema |

**Stack adotada e fixada:**
- Backend: **Go 1.26.0** + **Wails v3.0.0** (`go.mod` / `go.sum`).
- Frontend: **Vue 3.5.45**, **TypeScript 5.9.3**, **Vite 7.3.8**, **Vitest 4.1.12**, **Playwright 1.65.0** (`frontend/package.json` / `frontend/package-lock.json`).
- Estilização: Tokens CSS próprios (`docs/tokens.md`) derivados da Direção A aprovada no `12ui`. Controles nativos acessíveis substituem a necessidade de Reka UI nesta versão.

## 3. Integrações externas (verificação documental em 2026-11-11)

- **AssemblyAI (`internal/providers/assemblyai.go`):**
  - Upload binário (`POST /v2/upload`, `application/octet-stream`), criação (`POST /v2/transcript`) com `speech_models: ["universal-3-5-pro"]`, `language_code: "pt"`, `speaker_labels: true`, `disfluencies: true`, polling a cada 3s (`GET /v2/transcript/{id}`) e exclusão remota manual (`DELETE /v2/transcript/{id}`).
  - **Limitação documentada:** `disfluencies: true` é documentado oficialmente apenas para inglês. O aplicativo não faz fallback silencioso de modelo e exige reconhecimento explícito dessa limitação pelo usuário antes do envio.
  - Preserva em memória `Raw` (JSON bruto), `ID`, `Status`, `Text`, `Model` (`speech_model_used`) e `Utterances` com `Speaker`, `Start`, `End` e `Text`.
- **DeepSeek (`internal/providers/deepseek.go`):**
  - `POST https://api.deepseek.com/chat/completions`, autenticação `Bearer`, `thinking.type: "disabled"`, `stream: false`, `max_tokens: 16384`, `response_format: {"type": "json_object"}`.
  - Modelos permitidos: `deepseek-flash` (padrão) e `deepseek-v4-pro`.
  - Validação estrita: 1 `choice`, `finish_reason == "stop"`, `role == "assistant"`, `usage` consistente (`Prompt >= 0`, `Completion > 0`, `Total == Prompt + Completion`), `model` preenchido e cobertura exata de IDs e falantes (`editorial.Compare`). Respostas truncadas (`length`) falham de forma segura sem sobrescrever o original.

## 4. Entregas e estado de conclusão

1. **Domínio, Editorial, Aplicação e Infraestrutura Go:** Concluídos e testados com `-race`.
2. **Interface Desktop e Fluxo de Estados:** Concluída em Vue 3 + TypeScript, validada com Vitest e Playwright (incluindo temas claro/escuro, contraste WCAG AA, `1366×768`, `1920×1080`, `900×768`, DPI 200%, teclado e `prefers-reduced-motion`).
3. **Empacotamento e Governança de Licenças:** Scripts `scripts/build-windows.sh`, `scripts/build-windows.ps1`, verificador fail-closed `scripts/check-release.py` e instalador `packaging/installer.nsi` prontos.
4. **Pendências condicionadas ao ambiente externo:**
   - Execução do roteiro funcional em máquina/VM **Windows 11 x64 limpa** pelo usuário (`docs/test-matrix.md`);
   - Testes com credenciais reais e mídia fictícia autorizada (até lá, os clientes HTTP estão validados exclusivamente por mocks);
   - Verificação completa de código-fonte correspondente e notices de todas as bibliotecas do bundle FFmpeg antes de habilitar `redistribution_verified: true` para distribuição do instalador;
   - Assinatura de código (depende de certificado Authenticode não fornecido; binários não são declarados assinados).
