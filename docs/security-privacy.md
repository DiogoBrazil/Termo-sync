# Segurança e privacidade

Cada arquivo exige confirmação aplicável separada para AssemblyAI (áudio) e DeepSeek (texto). Sem autorização, a operação não inicia. Sem pseudonimização. O aplicativo não determina se o usuário pode legalmente enviar um depoimento: não use dados sigilosos enquanto as condições aplicáveis dos provedores não estiverem verificadas.

Chaves são digitadas em campos de senha, transmitidas somente pela ponte local e salvas no Windows Credential Manager como credenciais genéricas do usuário, destinos `DepoimentoStudio/assemblyai` e `DepoimentoStudio/deepseek`. Não há leitura de segredo pela interface. A remoção deve ser feita nas configurações antes da desinstalação, se desejada. O Credential Manager protege armazenamento, não impede um processo malicioso na mesma conta de acessar segredos. Strings em memória Go não têm garantia de apagamento físico imediato.

HTTP: somente HTTPS para os dois hosts oficiais, redirecionamentos bloqueados, timeout e limite de resposta. Chaves e corpos remotos não entram nas mensagens de erro. POST de upload, criação de transcrição e revisão não é repetido automaticamente. Em falha ambígua após envio, confira o painel do provedor antes de tentar de novo: o servidor pode ter recebido a solicitação mesmo sem resposta local.

Interface: conteúdo tratado como texto, sem renderizar HTML de transcrições, CSP local e sem telemetria/logs de conteúdo. A área de transferência e o TXT passam a existir fora da proteção em memória; o usuário escolhe copiar/exportar.

Mídia: ferramentas somente na pasta `tools` ao lado do executável; argumentos separados, sem shell, protocolos file/pipe e formatos delimitados. O arquivo original nunca é convertido no lugar. MP3 é decodificado para validar e enviado diretamente quando aplicável. Outras mídias produzem MP3 privado mantendo canais/taxa compatíveis; mais de dois canais ou taxa incompatível exige decisão manual e a conversão é recusada.

Temporários: subpastas em cache de usuário/DepoimentoStudio/private-media, com ACL de usuário no Windows. Limpeza em sucesso, falha, cancelamento e encerramento normal. Recuperação remove apenas subpastas session- antigas (24h), evitando interferir em uma conversão concorrente. Queda de energia pode deixar dados durante esse período; exclusão comum não equivale a apagamento forense em SSD. Nenhum arquivo fora da pasta controlada é varrido para limpeza.

Exclusão remota é manual e separada de limpeza local. O ID retornado pela AssemblyAI fica disponível durante a sessão, inclusive após cancelamento quando já conhecido. A resposta HTTP de exclusão não certifica eliminação instantânea de backups, uploads sem ID ou de todos os artefatos contratualmente retidos. Se a criação tiver resposta ambígua, consulte o painel. Não há exclusão remota na DeepSeek implementada neste fluxo.

Políticas de referência, verificadas documentalmente em 2026-11-11:
- AssemblyAI: https://www.assemblyai.com/legal/privacy-policy e condições da conta/contrato.
- DeepSeek: https://cdn.deepseek.com/policies/en-US/deepseek-privacy-policy.html e condições específicas da API. A política pública não comprova autorização para conteúdo sigiloso.

A equivalência semântica não é certificada pelo app. Conferir áudio, atribuições, números, datas, negativas, rumores, contradições e incertezas continua obrigatório.
