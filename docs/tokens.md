# Tokens visuais e sistema de interface

## Direção aprovada

- **Nome:** Depoimento Studio
- **Estética:** Editorial sóbria azul/grafite (Direção A gerada com `12ui`, aprovada em `docs/design/state-winner.png` e expandida para os estados `docs/design/state-a.png` a `docs/design/state-g.png`).
- **Escopo estrito:** Aplicativo individual de sessão única (um arquivo por vez), sem avatares, equipes, projetos, histórico persistente, reprodução embutida ou provedores além de AssemblyAI e DeepSeek.
- **Privacidade tipográfica:** Nenhum carregamento de fontes remotas (Google Fonts ou CDNs externos foi removido), preservando a Content Security Policy (`default-src 'self'`).

## Tokens semânticos CSS (`frontend/src/style.css`)

| Token | Tema claro (`:root`) | Tema escuro (`[data-theme=dark]`) | Função semântica |
|---|---|---|---|
| `--paper` | `#ffffff` | `#131c29` | Superfície principal de leitura, painéis e cartões documentais |
| `--ink` | `#172033` | `#e6edf7` | Texto principal de alto contraste (relação >= 4.5:1 sobre `--paper` e `--soft`) |
| `--muted` | `#526684` | `#adbed5` | Metadados, subtítulos, tempos e textos auxiliares |
| `--line` | `#dbe5f1` | `#33445b` | Divisores estruturais, bordas de painéis, campos e cartões |
| `--soft` | `#f0f6fd` | `#1c2b40` | Superfície secundária (cartão da proposta, avisos informativos, item ativo) |
| `--blue` | `#2358d7` | `#4276e8` | Ação primária, seleção ativa, indicadores de etapa e destaque editorial |
| `--warn` | `#fff6e3` | `#392d1d` | Fundo de alerta para limitações de fidelidade e requisitos pendentes |
| `--danger` | `#a82732` | `#ffb5bc` | Ação destrutiva manual (solicitação de exclusão remota na AssemblyAI) |

## Tipografia e hierarquia documental

- **Interface e controles (`--font-sans` implícito):** `'Segoe UI', Arial, sans-serif`, base `14px`, `line-height: 1.65` em parágrafos operacionais.
- **Título de etapa (`h1`):** `Georgia, 'Times New Roman', serif`, `38px/1.19` (`34px` abaixo de `1200px`, `32px` abaixo de `640px`), `letter-spacing: -1.4px`.
- **Corpo documental (`.transcript` e `#proposal`):** `Georgia, 'Times New Roman', serif`, `15px/1.9`, facilitando leitura prolongada e comparação entre transcrição automática e proposta revisada.
- **Identificadores técnicos (`code`):** `overflow-wrap: anywhere` para IDs remotos e parâmetros sem estouro horizontal.

## Layout, responsividade e acessibilidade

- **Grade principal (`.shell` e `.workspace`):**
  - `>= 1201px`: barra lateral de `200px`, área documental central flexível (`minmax(0, 1fr)`) e painel lateral da sessão de `300px`. Comparação lado a lado (`.comparison` com duas colunas `1fr 1fr`).
  - `<= 1200px`: barra lateral compacta (`170px`) e painel de sessão (`245px`).
  - `<= 1100px`: ativação automática das abas `.document-tabs` (`Original` / `Proposta`) com `role="tablist"`, `role="tab"`, `aria-selected` e navegação por setas (`Left`/`Right`), evitando compressão das colunas de leitura.
  - `<= 1023px` e `<= 640px`: empilhamento vertical sem barra de rolagem horizontal (`scrollWidth <= innerWidth`), validado em `1920×1080`, `1366×768`, `900×768` e DPI `200%`.
- **Foco e teclado:**
  - Link de salto inicial `.skip-link` (`Ir ao documento`) visível ao receber foco.
  - Indicador universal `:focus-visible` com `outline: 3px solid #2d65e7; outline-offset: 3px`.
  - Altura mínima de botões interativos: `38px`–`40px`.
- **Movimento reduzido:**
  - `@media (prefers-reduced-motion: reduce)` desativa todas as animações (incluindo `.pulse`), transições e rolagem suave.
- **Componentes nativos vs. Reka UI:**
  - Conforme registrado em `docs/decisions.md`, os estados da interface utilizam elementos nativos padronizados (`<fieldset>`, `<legend>`, `<label>`, `<select>`, `<details>/<summary>`, `<textarea>`, `role="alert"`, `role="status"`) com semântica e foco nativos, dispensando a adição da dependência Reka UI nesta versão.
