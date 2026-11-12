# Estado atual e como continuar

Atualizado em 2026-11-12. Leia este arquivo primeiro ao retomar o trabalho em outra máquina. O histórico das decisões está em [decisions.md](decisions.md).

## O que o aplicativo faz hoje

1. O usuário escolhe um áudio ou vídeo de **um único depoente**.
2. A tela "Antes de iniciar" mostra dois avisos (uma pessoa por gravação; pausas e repetições podem não aparecer) e o botão **Iniciar transcrição**. Não há confirmações por arquivo: a autorização é dada em termo externo ao aplicativo.
3. Um único comando faz, em sequência: conversão local para MP3 quando necessário (FFmpeg), extração do texto (AssemblyAI) e ajuste do texto (DeepSeek).
4. Ao final, original (não editável) e proposta (editável) aparecem lado a lado, com **Copiar original**, **Copiar proposta** e **Exportar TXT**.
5. O painel lateral mostra o "Registro do arquivo" (a transcrição guardada no serviço), que pode ser excluído ou mantido. Trocar de arquivo exige essa escolha, feita em um modal.

Regras de interface em vigor: nomes de serviços externos só aparecem na tela de Configurações; confirmações usam o modal do aplicativo; mensagens de sucesso somem em 2 s e erros de ação em 6 s; a falha do processamento fica na tela até o usuário fechar.

## O que foi verificado

- `go test -race ./...`, `go vet ./...` e `GOOS=windows GOARCH=amd64 go vet ./...` passam.
- Vitest (2 testes) e Playwright (11 cenários, sobre dados fictícios de `frontend/src/preview.ts`) passam.
- Os testes de mídia e o fluxo completo com provedores simulados foram executados no Windows com o FFmpeg 9.0.2 real.
- O usuário testou manualmente o aplicativo no Windows 11 com chaves reais: salvar chaves, transcrever, ajustar, comparar, trocar de arquivo e os modais.
- O instalador MSI por usuário foi instalado e desinstalado com sucesso. A versão atual (todos os usuários, `C:\Program Files`) teve o conteúdo conferido e extraído, mas **a instalação real ainda não foi testada**.

## Pendências

- **Redistribuição do FFmpeg:** `docs/ffmpeg-manifest.json` segue com `redistribution_verified: false`. Faltam o código-fonte correspondente ao build, os avisos de licença das bibliotecas incluídas e o registro da configuração. Até lá, `scripts/check-release.py` bloqueia o instalador de release; só existe o MSI `--teste-local`, que não deve ser distribuído.
- **Instalador:** testar o MSI para todos os usuários por duplo clique e em outro computador. Ele não verifica Windows 11 nem a presença do WebView2 (o script NSIS antigo verificava). O script NSIS (`packaging/installer.nsi`) continua por usuário e não foi atualizado.
- **Informações de versão do executável:** a versão 0.1.0 está gravada, mas nome e descrição do produto saíram vazios na leitura pelo PowerShell (aba Detalhes das propriedades). Não investigado.
- **Aviso "Encerrar a sessão?":** continua sendo a caixa nativa do Windows, não o modal do aplicativo.
- **Assinatura de código:** não há certificado; executável e instalador não são assinados.
- **Fora do escopo até agora:** arrastar-e-soltar, botão de testar conexão, escolha do modelo de transcrição, persistência de modelo e tema entre sessões.

## Continuar em outra máquina

O repositório não contém a pasta `build/` (ignorada pelo Git). Isso inclui o executável, o FFmpeg e o instalador, que precisam ser refeitos.

1. **Ferramentas:** Go 1.26+, Node 22.12+ ou 24 com npm, Python 3. Para o instalador, `wixl` (`sudo apt install wixl`). O desenvolvimento foi feito no WSL (Ubuntu) com o aplicativo executado no Windows 11.
2. **Dependências e testes:**

       cd frontend && npm ci && npx playwright install chromium && npm test && npm run test:ui && cd ..
       go test -race ./...

3. **FFmpeg:** baixe o arquivo indicado em `docs/ffmpeg-manifest.json` (campo `url`), confira o SHA-256 com `archive_sha256` e copie o conteúdo da pasta `bin` do zip (`ffmpeg.exe`, `ffprobe.exe` e as DLLs) para `build/portable/tools/`. Os hashes de cada arquivo estão no campo `files`. O link aponta para a versão "latest" do fornecedor: se o hash não bater, o arquivo mudou e o manifesto precisa ser atualizado junto. Copie também a licença do zip para `build/portable/licenses/FFmpeg-LICENSE.txt`.
4. **Executável:** `./scripts/build-windows.sh` gera `build/DepoimentoStudio.exe`, já com ícone, manifesto e versão (vêm de `resource_windows_amd64.syso`; ver `packaging/windows/README.md`). Copie-o para `build/portable/`.
5. **Rodar:** copie `build/portable` para uma pasta do Windows e abra o `DepoimentoStudio.exe` de lá (não de um caminho `\\wsl$`). Salve as duas chaves em Configurações; elas ficam no Gerenciador de Credenciais do Windows de cada máquina e de cada usuário, e não acompanham o repositório.
6. **Instalador para a própria máquina:** `python3 scripts/build-msi.py --teste-local`.
7. **Testes com FFmpeg real (opcional):** compile os testes para Windows (`GOOS=windows go test -c ./internal/application`) e execute-os no Windows com `DEPOIMENTO_TEST_FFMPEG` e `DEPOIMENTO_TEST_FFPROBE` apontando para os executáveis.

## Onde está cada coisa

- `internal/application`: sessão (estados) e serviço que encadeia preparação, transcrição e ajuste.
- `internal/editorial`: política editorial enviada ao modelo (versão 3, depoente único) e alertas de alteração.
- `internal/providers`: clientes HTTP dos dois serviços.
- `internal/media`: inspeção e conversão com FFmpeg, temporários privados.
- `main_windows.go`: janela Wails e ponte com a interface.
- `frontend/src`: `App.vue` (telas), `studio.ts` (estado e ações), `preview.ts` (dados fictícios; abra `npm run dev` e use `?preview=start`, `converting`, `processing`, `adjusting`, `review`, `voices`, `failed` ou `settings`).
- `packaging/windows`: ícone, manifesto e versão do executável. `scripts/build-msi.py`: instalador.
