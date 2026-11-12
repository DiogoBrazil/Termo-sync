# Decisões aprovadas

- Depoimento Studio; Windows 11 x64; Go/Wails 3 e Vue/TypeScript.
- Um arquivo por vez; diálogo padrão; narrativa apenas após identificação manual.
- Não completar respostas usando perguntas. Preservar original.
- Sem pseudonimização; autorização separada por arquivo e provedor obrigatória.
- Copiar, editar proposta e exportar TXT; sem DOCX e sem histórico automático.
- Exclusão AssemblyAI manual durante a sessão.
- Escolha de trilhas ambíguas pelo usuário.
- Visual editorial azul/grafite; aprovação visual antes do acabamento.
- Usuário validará em máquina/VM Windows 11 limpa.

## Continuação da implementação

- Usuário autorizou continuar até concluir e solicitou configurações para inserir API Keys no teste.
- Direção visual A do 12ui aprovada pelo usuário; expandir estados mantendo escopo individual e provedores AssemblyAI/DeepSeek.
- Wails 3.0.0 fixado exige Go >=1.26.0; requisito de desenvolvimento atualizado. Go 1.23 bastava somente para a base inicial.
- Corrigida suposição de fidelidade em português: disfluencies=true é documentado como inglês apenas. Exigir reconhecimento visível dessa limitação, preservar o parâmetro solicitado e recusar qualquer fallback silencioso. Não prometer hesitações completas.
- Não adotar Reka UI antes de haver necessidade concreta de um componente: controles nativos com labels e foco atendem esta implementação, reduzindo dependências.
- Snapshots identificados consultados pela interface substituem um barramento próprio de eventos para o único trabalho em sessão. A sessão rejeita callbacks antigos/cancelados no backend.
- Configuração de modelo fica em memória; chaves são persistidas apenas no Credential Manager.

## Retomada em 2026-11-12 (preparação do teste manual)

- Escopo: corrigir o que impedia o uso e liberar teste manual pelo usuário; lacunas restantes ficam para depois (arrastar-e-soltar, testar conexão, escolha do modelo AssemblyAI, persistência de modelo/tema, ícone, instalador).
- Distribuição por enquanto: pasta portátil (`DepoimentoStudio.exe` + `tools/` + `licenses/`), sem instalador.
- FFmpeg atualizado para a série estável 9.0 (build BtbN `n9.0` LGPL shared, 9.0.2), hash conferido com o `checksums.sha256` do fornecedor; registrado em `docs/ffmpeg-manifest.json`. Redistribuição continua não verificada.
- Política editorial: o prompt de sistema passa a usar o texto integral da seção 7 dos requisitos (`editorial.Policy`, versão 2), completado por `editorial.Instruction` com o rótulo do depoente confirmado pelo usuário.
- Mensagens de erro passam a incluir o diagnóstico curto do provedor (campo `error`, até 300 caracteres) em requisições recusadas 4xx, exceto 401/403. Altera a decisão anterior de omitir todo corpo remoto; chaves e transcrições continuam fora das mensagens.
- Cancelar ou falhar não descarta o que já foi obtido: transcrição volta à autorização AssemblyAI; revisão volta à autorização DeepSeek mantendo a transcrição.
- A consulta de estado (GET) tolera até 20 falhas transitórias consecutivas (rede, 429, 5xx) com espera crescente e `Retry-After`; POST continua sem repetição automática.
- Formatos AVI e ASF (WMV/WMA) aceitos; Opus já era aceito via OGG.

## Rodada de ajustes de uso (2026-11-12)

Decisões do usuário após o primeiro teste manual. Substituem as decisões anteriores sobre consentimento por arquivo, identificação do depoente e formato diálogo/narrativa.

- Autorização: passa a ser dada em um termo externo ao aplicativo. As confirmações por arquivo e por provedor foram removidas; `Session.Begin` substitui `Authorize`.
- Um depoente por arquivo: o usuário garante que só uma pessoa fala. A etapa de escolha do falante e a opção diálogo/narrativa foram removidas. Todos os segmentos são tratados como falas do depoente (`editorial.Instruction`, versão 3) e a proposta é texto corrido, sem rótulo de falante.
- A diarização continua pedida ao provedor só para segmentar o texto. Se vier mais de um rótulo, a tela de conferência mostra um aviso; nada é bloqueado. Transcrição sem segmentos vira um segmento único com o texto inteiro (antes bloqueava a revisão).
- Fluxo em um clique: `Service.Start` prepara o áudio, transcreve e revisa em sequência. As duas chaves são exigidas antes de iniciar. Estados: `awaiting_track`, `awaiting_start`, `transcribing`, `reviewing`, `ready`, `failed`.
- Cancelar ou falhar na revisão leva a `failed` com a transcrição preservada; `RetryReview` refaz somente a revisão. Cancelar ou falhar na transcrição volta a `awaiting_start`.
- Interface não cita serviços externos fora da tela de Configurações (onde as chaves precisam ser distinguidas); mensagens de erro do aplicativo também não. O diagnóstico curto devolvido pelo próprio provedor em erros 4xx continua sendo exibido como veio.
- O aviso de hesitações virou texto simples, sem caixa de confirmação. Original e proposta só aparecem juntos ao final. "Ações do documento" só aparece com a proposta pronta. Mensagens de sucesso somem em 2 segundos.
- Ícone do executável: onda de voz branca (a mesma marca da interface) sobre fundo azul, em `packaging/windows/`. É embutido pelo `resource_windows_amd64.syso` versionado na raiz, junto com o manifesto padrão do Wails (controles visuais atuais e DPI por monitor) e a versão 0.1.0. Regeneração descrita em `packaging/windows/README.md`.
- Instalador MSI: `scripts/build-msi.py` gera `build/DepoimentoStudio-<versão>.msi` a partir de `build/portable`, usando `wixl` (pacote `wixl` do Ubuntu; a variável `WIXL` aponta para outro executável). Instalação para todos os usuários (pede administrador), em `C:\Program Files\Depoimento Studio`, com atalhos no Menu Iniciar e na Área de Trabalho; sem telas próprias (usa a janela de progresso do Windows). Chaves de API e cache não são removidos na desinstalação.
- O instalador de release continua bloqueado por `scripts/check-release.py` enquanto a redistribuição do FFmpeg não for verificada. `--teste-local` pula a verificação e marca o nome do arquivo; serve só para instalar na própria máquina.
