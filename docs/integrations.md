# Integrações

Verificação documental: 2026-11-11. Testes locais com transporte HTTP simulado; nenhuma credencial real foi utilizada.

## AssemblyAI

Upload binário em `https://api.assemblyai.com/v2/upload`; criação em `/v2/transcript`; polling de três segundos; exclusão DELETE por ID por ação manual. Autorização sem prefixo Bearer. Modelo explícito único `universal-3-5-pro`, idioma `pt`, diarização e `disfluencies: true`. Sem fallback automático para outro modelo.

**Limitação corrigida em relação ao plano inicial:** o changelog oficial documenta disfluencies como inglês apenas. Portanto o parâmetro não comprova preservação de hesitações em português. A interface exige reconhecer isso e a necessidade de ouvir o áudio; não declara transcrição literal garantida. Não se troca silenciosamente de modelo e não se adiciona um recurso pago de prompting sem decisão explícita.

Resposta bruta, ID, texto, segmentos, tempos e modelo efetivo retornado são mantidos em memória. O JSON bruto não é exposto em snapshots da interface nem persistido. Credenciais e conteúdo não entram em logs. Se o provedor não fornecer falantes, a revisão fica bloqueada: falta implementar identificação manual por segmentos para esse caso específico.

Fontes oficiais:
- https://www.assemblyai.com/docs/api-reference/transcripts/submit
- https://www.assemblyai.com/docs/pre-recorded-audio/include-filler-words
- https://www.assemblyai.com/changelog (27 de agosto de 2026: suporte a hesitações apenas em inglês)
- https://www.assemblyai.com/docs/api-reference/transcripts/delete

## DeepSeek

POST `https://api.deepseek.com/chat/completions`, Bearer, `thinking.type: disabled`, `stream: false`, JSON output, limite de 16384 tokens por lote. Padrão `deepseek-flash`; opção `deepseek-v4-pro`. Valida quantidade de choices, finish_reason=stop, papel assistant, conteúdo, uso e presença de modelo efetivo. Erro ou length não são publicados como revisão completa.

Lotes de até 12000 caracteres Unicode incluindo margem por segmento; uma fala não é dividida automaticamente. Falas que excedem o limite são recusadas sem omissão. IDs estáveis e falante devem coincidir exatamente, na mesma ordem. A cobertura estrutural não prova fidelidade factual. Números/datas, palavras de negação/atribuição/incerteza e reduções substanciais geram alertas.

Fonte: https://api-docs.deepseek.com/api/create-chat-completion/

Preços variam por modelo, conta, recursos e data. Consulte os painéis antes do teste; o app não apresenta custo estimado fixo:
- https://www.assemblyai.com/pricing
- https://api-docs.deepseek.com/quick_start/pricing
