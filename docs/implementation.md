# Andamento e entrega da implementação

O resumo do estado atual, das pendências e dos passos para continuar em outra máquina está em [status.md](status.md).

## Implementado e verificado localmente

- **Domínio (`internal/domain`):** `Transcript` e `Utterance` com `Speaker`, `Start`, `End`, `Text`, `Model` (`speech_model_used`) e retenção em memória do JSON bruto (`Raw`).
- **Política editorial e revisão (`internal/editorial`):** Prompt versionado (`Version = "3"`), depoente único por arquivo (todos os segmentos são falas dele), lotes (`Batches`) de até `12000` caracteres Unicode por unidades completas com IDs estáveis (`s000000`), validação de cobertura 1:1 e alertas heurísticos (`Compare`) para alterações de números/datas, negações/atribuições/incertezas/contradições e reduções substanciais.
- **Aplicação e sessão (`internal/application`):**
  - Máquina de estados (`awaiting_track`, `awaiting_start`, `transcribing`, `reviewing`, `ready`, `failed`) com identificadores monotônicos (`ID uint64`) que rejeitam resultados tardios de arquivos substituídos ou cancelados.
  - Cópias defensivas garantem que o original permaneça imutável; proposta revisada e edição humana (`Edited` / `HasEdits`, inclusive edição vazia) ficam separadas.
  - Falha na revisão (`failed`) preserva a transcrição original e permite `RetryReview` sem repetir a transcrição na AssemblyAI.
  - Controle de retenção/exclusão remota (`RemoteID`, `DeleteRemote`, `KeepRemote`) impedindo troca acidental de arquivo sem decisão explícita sobre o registro remoto.
- **Provedores HTTP (`internal/providers`):**
  - Transporte seguro (`http.go`): somente HTTPS para `api.assemblyai.com` e `api.deepseek.com`, sem query/fragment/userinfo, redirecionamentos bloqueados mesmo com cliente injetado, limite de leitura de 64 MiB, rejeição de JSON inválido/trailing/`null`, omissão de corpos remotos e segredos em mensagens de erro, preservação de `context.Canceled` e zero repetição automática de `POST`.
  - Cliente AssemblyAI (`assemblyai.go`): upload binário, criação com `universal-3-5-pro`, `language_code: "pt"`, `speaker_labels: true`, `disfluencies: true`, polling cancelável e `DeleteTranscript` manual com validação de ID.
  - Cliente DeepSeek (`deepseek.go`): `/chat/completions`, `deepseek-flash` e `deepseek-v4-pro`, `thinking.type: "disabled"`, `stream: false`, validação de `finish_reason == "stop"`, `usage` e cobertura de segmentos.
- **Mídia e temporários (`internal/media`):**
  - Localização restrita a `tools/ffmpeg.exe` e `tools/ffprobe.exe` ao lado do executável (`Locate()`).
  - Inspeção com `ffprobe` (`-protocol_whitelist file,pipe`, `-format_whitelist ...`), validação de decodificação de MP3 sem regravar o original, conversão de outros formatos para MP3 privado (`libmp3lame`, preservando 1–2 canais e taxas suportadas, com `-n` para nunca sobrescrever).
  - Diretórios temporários em cache do usuário (`DepoimentoStudio/private-media/session-*`) com ACL privada no Windows (`icacls.exe` por SID), limpeza imediata via `defer` e recuperação segura na inicialização (`Recover()` remove apenas diretórios `session-*` com mais de 24h).
- **Credenciais (`internal/credentials`):** Integração nativa com Windows Credential Manager (`CredWriteW`, `CredReadW`, `CredFree`, `CredDeleteW` em `advapi32.dll`) para `DepoimentoStudio/assemblyai` e `DepoimentoStudio/deepseek`, zerando o buffer temporário após escrita e sem método de leitura exposto à interface.
- **Apresentação Desktop (`main_windows.go` e `frontend/`):**
  - Ponte Wails v3.0.0 com cabeçalhos `Content-Security-Policy` e `X-Content-Type-Options: nosniff`, diálogos nativos de abertura/exportação `.txt` (bloqueando sobrescrita do arquivo de mídia original), área de transferência nativa e confirmação ao fechar janela com sessão ativa (`ShouldQuit`) seguida de `Shutdown()` que cancela e aguarda limpeza de temporários.
  - Interface Vue 3 + TypeScript implementando todos os estados aprovados a partir da Direção A do `12ui`, fluxo em um clique (transcrição seguida do ajuste), modal próprio para confirmações, painel de Configurações para API Keys (`type="password"`, limpeza imediata do campo após salvar) e escolha de modelo (`deepseek-flash` / `deepseek-v4-pro`), temas claro e escuro, comparação lado a lado em telas amplas e por abas em janelas estreitas.
- **Empacotamento e distribuição (`packaging/installer.nsi`, `scripts/`):**
  - Scripts reproduzíveis `scripts/build-windows.sh` e `scripts/build-windows.ps1` gerando `build/DepoimentoStudio.exe` e `build/DepoimentoStudio.exe.sha256`.
  - Ícone, manifesto e versão embutidos por `resource_windows_amd64.syso`; instalador MSI para todos os usuários por `scripts/build-msi.py`. Verificador fail-closed `scripts/check-release.py` e instalador NSIS por usuário (`RequestExecutionLevel user`, validação de Windows 11 build `>= 22000` x64 e presença do WebView2 Evergreen Runtime).

## Limitações e etapas dependentes do usuário

1. **Validação em Windows 11 x64 real:** O executável `build/DepoimentoStudio.exe` foi compilado cruzadamente a partir de Linux amd64; a execução real da janela WebView2, do Windows Credential Manager, da ACL `icacls.exe` e do instalador NSIS requer execução do roteiro de `docs/test-matrix.md` na máquina/VM Windows 11 do usuário.
2. **Provedores:** os testes automatizados usam servidores/transportes HTTP simulados. O fluxo com chaves reais foi testado manualmente pelo usuário em 2026-11-12.
3. **Identificação manual quando a diarização remota vier vazia:** Resolvido de outra forma: com depoente único por arquivo, transcrição sem `utterances` vira um segmento único com o texto inteiro.
4. **Redistribuição do FFmpeg bloqueada por padrão (`redistribution_verified: false`):** Os binários em `build/tools/` destinam-se apenas a teste local até que o pacote completo de fontes correspondentes e notices de todas as bibliotecas vinculadas seja anexado e verificado por `scripts/check-release.py`.
5. **Assinatura de código:** Nenhum certificado Authenticode foi configurado; o executável e o instalador não são declarados assinados.
