# Arquitetura

O executável Windows usa Go 1.26+ e Wails 3.0.0 fixado. A biblioteca padrão encoding/json não é dependência externa. O frontend Vue/TypeScript/Vite é compilado e embutido no executável; não precisa de Node no computador do usuário.

- `internal/domain`: transcrição e segmentos, resposta bruta preservada somente em memória.
- `internal/editorial`: prompt versionado, segmentação em unidades completas, cobertura e alertas heurísticos.
- `internal/application`: sessão de um arquivo e serviço de orquestração. Resultados identificados pela sessão são rejeitados após cancelamento ou substituição.
- `internal/providers`: REST AssemblyAI e DeepSeek, sem SDK nem repetição automática de POST.
- `internal/media`: inspeção, validação e conversão via processos com argumentos separados, sem shell.
- `internal/credentials`: Credential Manager nativo no Windows; outros sistemas não persistem chaves.
- `main_windows.go`: janela, ponte com métodos específicos, diálogos nativos, clipboard e exportação TXT. Não expõe operação para ler chaves ou abrir caminhos arbitrários enviados pela interface.

A apresentação consulta snapshots identificados em vez de receber conteúdo por logs. A autorização de envio é dada em termo externo ao aplicativo; a ponte não registra consentimento. Cada arquivo contém um único depoente, garantido pelo usuário: a revisão recebe todos os segmentos como falas dele e a proposta conserva os segmentos em texto corrido, sem inventar transições.

Estados: aguardando trilha → aguardando início → transcrição → revisão → proposta. Um único comando inicia transcrição e revisão em sequência. Falha ou cancelamento da revisão preserva o original; tentar novamente refaz só a revisão. A edição humana tem campo próprio e prevalece na cópia/exportação.

Conversão e chamadas remotas rodam em goroutines com contexto cancelável. Desligamento aguarda limpeza dos temporários. Uma resposta truncada não é publicada como proposta completa. O app não tem armazenamento automático de documentos ou histórico.
