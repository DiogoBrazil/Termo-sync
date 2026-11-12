# Recursos do executável Windows

- `icon.svg` e `icon-small.svg`: desenho do ícone (a variante pequena, com três barras, é usada até 24 px).
- `icon.ico`: ícone com 16, 20, 24, 32, 40, 48, 64, 128 e 256 px, gerado a partir dos SVG.
- `info.json` e `app.manifest`: informações de versão e manifesto (controles visuais atuais e DPI por monitor).

O arquivo `resource_windows_amd64.syso`, na raiz do repositório, embute esses recursos no executável durante o `go build`. Para regenerá-lo depois de alterar algo aqui:

    wails3 generate syso -arch amd64 -icon packaging/windows/icon.ico -manifest packaging/windows/app.manifest -info packaging/windows/info.json -out resource_windows_amd64.syso
