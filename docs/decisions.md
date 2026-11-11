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
