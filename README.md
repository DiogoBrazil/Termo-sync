# Depoimento Studio

Aplicativo desktop individual para **Windows 11 x64** construído com **Go 1.26 + Wails 3.0.0** e **Vue 3 + TypeScript + Vite**, projetado para processar **um arquivo por vez** mantendo separadas em memória a transcrição automática original (imutável), a proposta revisada conservadora e as edições humanas.

## Documentação do projeto

- [Plano técnico e funcional](docs/plan.md)
- [Andamento e entrega da implementação](docs/implementation.md)
- [Decisões aprovadas](docs/decisions.md)
- [Arquitetura](docs/architecture.md)
- [Integrações (AssemblyAI e DeepSeek)](docs/integrations.md)
- [Segurança e privacidade](docs/security-privacy.md)
- [Tokens visuais e sistema de interface](docs/tokens.md)
- [Matriz de validação e roteiro Windows 11](docs/test-matrix.md)
- [Compilação, instalação e empacotamento Windows](docs/install.md)

## Verificação e compilação reproduzíveis

Requisitos: **Go 1.26.0+** e **Node 22.12+ / 24** com `npm`.

```sh
# 1. Verificar pacote padrão e suíte Go (com detector de concorrência)
go list encoding/json
go test -race ./...
go vet ./...
GOOS=windows GOARCH=amd64 go vet ./...

# 2. Executar testes reais de mídia sintética (informando caminhos absolutos do FFmpeg/ffprobe)
DEPOIMENTO_TEST_FFMPEG=/caminho/para/ffmpeg \
DEPOIMENTO_TEST_FFPROBE=/caminho/para/ffprobe \
go test -v -race ./...

# 3. Executar testes unitários e E2E da interface (Vitest + Playwright)
cd frontend
npm ci
npm test
npm run test:ui
cd ..

# 4. Gerar o executável Windows x64 com frontend embutido e hash SHA-256
./scripts/build-windows.sh

# 5. Verificar pré-condições de redistribuição antes de empacotar com NSIS
python3 scripts/check-release.py
```

No Windows PowerShell, utilize `./scripts/build-windows.ps1`.

## Limitações e estado da release

- **Clientes HTTP validados por mocks:** Nenhuma chamada paga ou com credenciais reais foi executada durante o desenvolvimento.
- **Hesitações em português (`disfluencies`):** Documentado oficialmente pela AssemblyAI apenas para inglês; o aplicativo exige confirmação explícita dessa limitação e conferência com o áudio original antes de transcrever, sem fallback silencioso de modelo.
- **Redistribuição do FFmpeg:** `docs/ffmpeg-manifest.json` mantém `redistribution_verified: false` até que as fontes completas e notices de todas as bibliotecas do bundle sejam anexadas e verificadas; `scripts/check-release.py` e `packaging/installer.nsi` bloqueiam o empacotamento de release enquanto essa verificação estiver pendente.
- **Assinatura e validação final:** Os binários não possuem assinatura Authenticode (dependente de certificado externo) e a release permanece condicionada à execução do roteiro funcional em máquina/VM **Windows 11 x64 limpa** pelo usuário ([docs/test-matrix.md](docs/test-matrix.md)).
