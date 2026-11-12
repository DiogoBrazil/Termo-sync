# Compilação e teste Windows

## Requisitos de desenvolvimento

Go 1.26+ (Wails 3.0.0 exige 1.26), Node 22.12+ ou 24, npm e NSIS para o instalador. Versões JS fixadas em package-lock.json, Go em go.mod/go.sum. Não instalar encoding/json separadamente.

Windows PowerShell: `./scripts/build-windows.ps1`. Linux: `./scripts/build-windows.sh`. O resultado é `build/DepoimentoStudio.exe`; as ferramentas de mídia não são baixadas implicitamente pelos scripts. Nenhuma API Key é necessária para compilar.

## Pasta de teste local

Montagem usada no teste manual (a partir do WSL/Linux, após `./scripts/build-windows.sh`): copie `build/DepoimentoStudio.exe` para `build/portable/`, mantenha em `build/portable/tools/` o `ffmpeg.exe`, o `ffprobe.exe` e as DLLs do zip registrado em `docs/ffmpeg-manifest.json` (hash conferido), e copie a pasta `build/portable` para um diretório do Windows. Execute o `.exe` a partir do Windows, não de um caminho `\\wsl$`.

Coloque o executável e as ferramentas verificadas nesta estrutura:

```
DepoimentoStudio.exe
tools/
  ffmpeg.exe
  ffprobe.exe
  *.dll (quando o build for compartilhado)
licenses/
  notices e evidências correspondentes ao build
```

O app só executa ferramentas nessa pasta, nunca um FFmpeg encontrado casualmente no PATH ou no diretório de trabalho. O build deve ter libmp3lame. Antes de usar, registre `ffmpeg -version`, `ffmpeg -buildconf`, `ffmpeg -L`, o fornecedor, o hash do arquivo original e os hashes extraídos. Guarde fontes correspondentes de FFmpeg e dependências conforme exigido pelo conjunto de licenças.

Windows 11 x64 requer Microsoft Edge WebView2 Evergreen Runtime. Site oficial: https://developer.microsoft.com/microsoft-edge/webview2/. O instalador verifica as chaves oficiais do runtime; se ausente, oferece abrir a página e exige instalação antes de continuar. A verificação e execução ainda precisam de teste Windows real.

## API Keys para testar

Abra **Configurações** no app. Digite uma chave nova de cada provedor e clique em Salvar. Campos são de senha e são limpos após a tentativa de gravação; indicador mostra somente configurada/não configurada. Remova a chave pelo mesmo painel. Nenhuma chave precisa ser escrita no código, em .env ou em arquivo JSON.

Use somente mídia fictícia autorizada nos primeiros testes. O aplicativo não pede confirmação por arquivo; a autorização é dada em termo externo. Consulte docs/test-matrix.md.

## Empacotamento

O instalador atual é o MSI descrito ao final deste arquivo. O script NSIS abaixo é anterior e não foi atualizado.

A especificação NSIS é `packaging/installer.nsi`. Depois de preparar ferramentas e licenças em build/tools e build/licenses, execute `makensis packaging/installer.nsi` a partir da raiz (o NSIS resolve arquivos relativamente ao script). Resultado: build/DepoimentoStudio-Setup.exe, instalação por usuário em LocalAppData/Programs.

A redistribuição do FFmpeg permanece condicionada à verificação de fontes, configuração, hashes e notices de **todas** as bibliotecas do build. Um rótulo LGPL no nome do zip não substitui essas evidências. Não publicar o instalador enquanto esse conjunto estiver incompleto. A instalação não inclui certificado: executável/instalador não são declarados assinados.

Desinstalação remove arquivos instalados e atalhos. Credenciais genéricas não são removidas implicitamente: remova-as em Configurações antes, se desejado. Exportações escolhidas pelo usuário e registros remotos dos provedores não são removidos pela desinstalação.

A release final depende do roteiro Windows 11 e da aprovação funcional do usuário.

## Instalador MSI

    python3 scripts/build-msi.py --teste-local   # só para a própria máquina
    python3 scripts/build-msi.py                 # release; exige scripts/check-release.py aprovado

Requer `wixl` (`sudo apt install wixl`) e `build/portable` atualizado. O MSI instala para todos os usuários em `C:\Program Files\Depoimento Studio` (pede administrador) e cria atalhos no Menu Iniciar e na Área de Trabalho. Para desinstalar, use "Aplicativos instalados" do Windows.
