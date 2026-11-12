<script setup lang="ts">
import {computed,nextTick,ref,watch} from 'vue'
import {useStudio,time} from './studio'
const studio=useStudio()
const {question,view,settings,error,notice,draft,dirty,selectedTrack,id,state,busy,voices,keysReady,flags,original}=studio
const page=ref(location.hash==='#settings'?'settings':'document')
const theme=ref<'light'|'dark'>(matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light')
watch(theme,value=>document.documentElement.dataset.theme=value,{immediate:true})
const tab=ref('original'),keyA=ref(''),keyD=ref(''),revealA=ref(false),revealD=ref(false)
watch(page,value=>{location.hash=value==='settings'?'settings':'';keyA.value='';keyD.value='';revealA.value=false;revealD.value=false})
const file=computed(()=>view.value?.Document.File.split(/[\\/]/).pop()??'Nenhum arquivo selecionado')
const stage=computed(()=>({empty:0,awaiting_track:0,awaiting_start:0,transcribing:1,reviewing:2,failed:2,ready:3}[state.value]??0))
const title=computed(()=>({empty:'Cada palavra, com o devido cuidado.',awaiting_track:'Escolha a trilha de áudio',awaiting_start:'Antes de iniciar',transcribing:'Preparando o seu depoimento',reviewing:'Preparando o seu depoimento',ready:'Compare e aperfeiçoe o depoimento',failed:'Falta ajustar o texto'}[state.value]??'Seu depoimento'))
const details=computed(()=>{
 const v=view.value;if(!v)return ''
 const parts:string[]=[]
 const ext=v.Document.File.match(/\.([a-z0-9]+)$/i)?.[1]
 if(ext)parts.push(ext.toUpperCase())
 if(v.Duration>0){const s=Math.round(v.Duration),h=Math.floor(s/3600),m=Math.floor(s%3600/60),p=(n:number)=>n.toString().padStart(2,'0');parts.push(`duração ${h?h+':':''}${p(m)}:${p(s%60)}`)}
 if(v.Size>0)parts.push(v.Size>=1048576?`${(v.Size/1048576).toFixed(1).replace('.',',')} MB`:`${Math.max(1,Math.round(v.Size/1024))} KB`)
 parts.push(`${v.Tracks?.length??0} trilha(s) de áudio`)
 return parts.join(' · ')
})
const stageInfo=computed(()=>({validating:['Preparando o arquivo','Verificação feita neste computador.'],converting:['Convertendo o áudio','O arquivo original não é alterado.'],uploading:['Extraindo o texto do áudio','Enviando o áudio para a extração.'],transcribing:['Extraindo o texto do áudio','O tempo depende da duração da gravação. Cancelar agora não desfaz a extração já iniciada, que pode ser cobrada.'],reviewing:['Ajustando o texto','O texto extraído é mantido se você cancelar.']}[view.value?.Stage??'']??['Processando','Aguarde.']))
const percent=computed(()=>{const p=view.value?.Progress??-1;return p>=0?Math.min(100,Math.round(p*100)):null})
const preview=import.meta.env.DEV && new URLSearchParams(location.search).has('preview')
async function saveA(){const key=keyA.value;keyA.value='';revealA.value=false;await studio.saveKey('assemblyai',key)}
async function saveD(){const key=keyD.value;keyD.value='';revealD.value=false;await studio.saveKey('deepseek',key)}
function navigate(value:string){page.value=value}
const modal=ref<HTMLDialogElement|null>(null)
// The native dialog element traps focus, closes on Esc and restores focus to the button that opened it.
watch(question,async value=>{await nextTick();const el=modal.value;if(!el)return;if(value&&!el.open)el.showModal();else if(!value&&el.open)el.close()})
</script>

<template>
 <a class="skip-link" href="#main">Ir ao documento</a>
 <dialog ref="modal" class="confirm" aria-labelledby="confirm-title" aria-describedby="confirm-text" @cancel.prevent="studio.answer(false)" @click.self="studio.answer(false)"><div class="confirm-body"><h2 id="confirm-title">{{question?.title}}</h2><p id="confirm-text">{{question?.message}}</p><div class="confirm-actions"><button class="button" autofocus @click="studio.answer(false)">Cancelar</button><button v-if="question?.alt" class="button" @click="studio.answer('alt')">{{question.alt}}</button><button class="button" :class="question?.danger?'danger-solid':'primary'" @click="studio.answer('confirm')">{{question?.confirm}}</button></div></div></dialog>
 <div v-if="preview" class="preview-banner">Prévia de desenvolvimento, com dados fictícios e sem envio de conteúdo</div>
 <div class="shell">
  <div class="brand"><svg viewBox="0 0 28 32" aria-hidden="true"><path d="M3 14v4m5-9v14m6-21v28m6-20v12m5-8v4"/></svg><span>Depoimento Studio</span></div>
  <header class="header"><div class="breadcrumbs"><span>Espaço de trabalho</span><span aria-hidden="true">›</span><strong>{{page==='settings'?'Configurações':'Revisão de depoimento'}}</strong></div><div class="top-actions"><button class="pill" @click="theme=theme==='light'?'dark':'light'">{{theme==='light'?'Tema escuro':'Tema claro'}}</button><button class="pill dark" @click="navigate(page==='settings'?'document':'settings')">{{page==='settings'?'Voltar ao documento':'Configurações'}}</button></div></header>
  <aside class="sidebar"><nav class="navigation" aria-label="Navegação principal"><button :class="{selected:page==='document'}" @click="navigate('document')"><span aria-hidden="true">▤</span> Depoimento</button><button :class="{selected:page==='settings'}" @click="navigate('settings')"><span aria-hidden="true">⚙</span> Configurações</button></nav><div class="profile"><div><p>Uma sessão. Um arquivo.</p><small>Conteúdo mantido em memória.<br>Exporte antes de encerrar.</small></div></div></aside>
  <main id="main" class="workspace" :class="{'settings-workspace':page==='settings','ready-workspace':page==='document'&&state==='ready'}" tabindex="-1">
   <div class="main">
    <div v-if="studio.failure.value" class="feedback error failure" role="alert"><span>{{studio.failure.value}}</span><button class="button small" @click="studio.closeFailure">Fechar</button></div>
    <div v-if="error" class="feedback error toast" role="alert">{{error}}</div>
    <div v-if="notice" class="feedback success" role="status">{{notice}}</div>
    <template v-if="page==='document'">
     <p class="status"><span class="dot"/>{{busy?'Processamento em andamento':state==='ready'?'Proposta disponível para conferência':'Seu espaço de revisão'}}</p>
     <h1>{{title}}</h1><p v-if="state!=='ready'" class="intro">O original permanece intacto. Revise a proposta com apoio do áudio e preserve o que foi efetivamente dito.</p>
     <ol class="steps" aria-label="Etapas do processamento"><template v-for="(label,index) in ['Arquivo','Extração do texto','Ajuste do texto','Conferência']" :key="label"><li class="step" :class="{active:stage===index}" :aria-current="stage===index?'step':undefined"><i>{{index+1}}</i>{{label}}</li><span v-if="index<3" class="connector" aria-hidden="true"/></template></ol>
     <section v-if="state==='empty'" class="upload-panel">
      <img src="/design/audio-folder.png" alt="" width="160" height="120"><h2>Selecione um áudio ou vídeo</h2><p>Envie a gravação de um único depoente. O Depoimento Studio extrai o texto e prepara uma proposta de redação.</p><button class="button primary" :disabled="busy" @click="studio.selectFile">Escolher arquivo</button><p class="muted">Áudio: MP3, WAV, M4A, AAC, OGG, OPUS, FLAC e WMA. Vídeo: MP4, MKV, MOV, WEBM, AVI e WMV.</p>
     </section>
     <section v-if="id" class="file-strip"><div><strong>{{file}}</strong><small>{{details}}</small></div><button class="button" :disabled="busy" @click="studio.selectFile">Trocar arquivo</button></section>
     <section v-if="state==='awaiting_track'" class="panel"><h2>Há mais de uma trilha de áudio</h2><p>Escolha qual trilha contém o depoimento. Nenhuma será escolhida automaticamente.</p><label for="track">Trilha</label><select id="track" v-model.number="selectedTrack"><option v-for="track in view?.Tracks" :key="track.index" :value="track.index">{{track.index}} · {{track.tags.language||'idioma não informado'}} · {{track.channels}} canal(is) · {{track.codec_name}} {{track.tags.title}}</option></select><button class="button primary" :disabled="busy" @click="studio.selectTrack">Confirmar trilha</button></section>
     <section v-if="state==='awaiting_start'" class="panel start"><h2>Confira antes de iniciar</h2><ul class="cautions"><li><strong>Uma pessoa por gravação</strong><p>O Depoimento Studio trata todo o áudio como a fala de um único depoente. Envie somente gravações em que apenas ele fala.</p></li><li><strong>Pausas e repetições podem não aparecer</strong><p>Hesitações, palavras repetidas ou ditas pela metade às vezes ficam de fora do texto. Ao final, ouça a gravação e confira se nada importante se perdeu.</p></li></ul><p v-if="!keysReady" class="warning">Salve as chaves de API em Configurações antes de iniciar.</p><button class="button primary" :disabled="busy||!keysReady" @click="studio.start">Iniciar transcrição</button></section>
     <section v-if="view?.Busy" class="panel processing" aria-live="polite"><div class="pulse" aria-hidden="true"/><h2>{{stageInfo[0]}}</h2><p>{{stageInfo[1]}}</p><div v-if="percent!==null" class="progress-row"><progress :value="percent" max="100" :aria-label="stageInfo[0]"/><span>{{percent}}%</span></div><button class="button" @click="studio.cancel" :disabled="studio.pending.value">Cancelar</button></section>
     <section v-if="state==='failed'" class="panel"><h2>O texto foi extraído, mas o ajuste não terminou</h2><p>O Depoimento Studio aproveita o texto já extraído e refaz somente o ajuste. Uma nova tentativa pode ser cobrada.</p><button class="button primary" :disabled="busy||!keysReady" @click="studio.retry">Tentar o ajuste novamente</button></section>
     <template v-if="state==='ready'">
      <p v-if="voices>1" class="warning">Foi percebida mais de uma voz nesta gravação. O texto inteiro foi tratado como fala do depoente; confira a proposta com atenção redobrada.</p>
      <div class="comparison-wrap"><div class="document-tabs" role="tablist" aria-label="Documentos"><button id="tab-original" role="tab" :aria-selected="tab==='original'" :tabindex="tab==='original'?0:-1" @keydown.right="tab='proposal'" @click="tab='original'">Original</button><button id="tab-proposal" role="tab" :aria-selected="tab==='proposal'" :tabindex="tab==='proposal'?0:-1" @keydown.left="tab='original'" @click="tab='proposal'">Proposta</button></div><div class="comparison" :data-tab="tab"><section class="recording original" aria-label="Transcrição original"><div class="card-head"><span class="badge">Original</span><span class="immutable">Não pode ser editado</span><button class="button small" :disabled="busy" @click="studio.copyOriginal">Copiar original</button></div><pre class="transcript" tabindex="0">{{original}}</pre><div class="card-foot">Texto extraído automaticamente. Confira com o áudio.</div></section><section class="recording proposal" aria-label="Proposta revisada"><div class="card-head"><span class="badge">Proposta</span><span class="editable">{{dirty||view?.Document.HasEdits?'Editada por você':'Pode ser editada'}}</span><button class="button small" :disabled="busy" @click="studio.copy">Copiar proposta</button><button class="button small" :disabled="busy" @click="studio.exportText">Exportar TXT</button></div><label class="sr-only" for="proposal">Proposta editável</label><textarea id="proposal" :value="draft" placeholder="Proposta vazia" spellcheck="true" lang="pt-BR" @input="studio.edit(($event.target as HTMLTextAreaElement).value)"/><div class="card-foot">Suas edições valem ao copiar e exportar.</div></section></div></div>
      <section v-if="flags.length" class="panel flags"><h2>{{flags.length}} trecho(s) exigem atenção</h2><p>Estes alertas são automáticos e não garantem que o sentido foi mantido. Confira o áudio e corrija a proposta quando necessário.</p><details v-for="change in flags" :key="change.id"><summary>{{change.flags?.join('; ')}}</summary><p><strong>Original:</strong> {{change.original}}</p><p><strong>Proposta:</strong> {{change.revised}}</p></details></section>
      <div class="notice">A proposta é uma sugestão. Confira números, datas, negativas e incertezas antes de utilizar o texto.</div>
     </template>
    </template>
    <section v-else class="settings-page">
     <p class="status"><span class="dot"/>Configurações</p><h1>Credenciais e preferências</h1><p class="intro">As chaves ficam salvas no Gerenciador de Credenciais do Windows, neste computador.</p>
     <div class="section-head"><h2>Chaves de API</h2></div>
     <div class="credentials">
      <form class="credential" @submit.prevent="saveA"><div class="service-copy"><h3>AssemblyAI</h3><p>Extração do texto · {{settings.AssemblyAIConfigured?'Chave configurada':'Sem chave salva'}}</p></div><label class="sr-only" for="key-a">API Key AssemblyAI</label><div class="secret"><input id="key-a" v-model="keyA" :type="revealA?'text':'password'" autocomplete="off" spellcheck="false" placeholder="Insira uma nova API Key" :disabled="busy"><button type="button" :aria-label="revealA?'Ocultar API Key AssemblyAI':'Mostrar API Key AssemblyAI'" @click="revealA=!revealA">{{revealA?'Ocultar':'Ver'}}</button></div><div class="credential-actions"><button class="button" type="button" :disabled="busy||!settings.AssemblyAIConfigured" @click="studio.removeKey('assemblyai')">Remover</button><button class="button primary" type="submit" :disabled="busy||!keyA.trim()">Salvar</button></div></form>
      <form class="credential" @submit.prevent="saveD"><div class="service-copy"><h3>DeepSeek</h3><p>Ajuste do texto · {{settings.DeepSeekConfigured?'Chave configurada':'Sem chave salva'}}</p></div><label class="sr-only" for="key-d">API Key DeepSeek</label><div class="secret"><input id="key-d" v-model="keyD" :type="revealD?'text':'password'" autocomplete="off" spellcheck="false" placeholder="Insira uma nova API Key" :disabled="busy"><button type="button" :aria-label="revealD?'Ocultar API Key DeepSeek':'Mostrar API Key DeepSeek'" @click="revealD=!revealD">{{revealD?'Ocultar':'Ver'}}</button></div><div class="credential-actions"><button class="button" type="button" :disabled="busy||!settings.DeepSeekConfigured" @click="studio.removeKey('deepseek')">Remover</button><button class="button primary" type="submit" :disabled="busy||!keyD.trim()">Salvar</button></div></form>
     </div>
     <section class="section"><h2>Modelo de ajuste do texto</h2><div class="choices"><label v-for="model in ['deepseek-flash','deepseek-v4-pro']" :key="model" class="choice"><input type="radio" name="model" :value="model" :checked="settings.Model===model" :disabled="busy" @change="studio.model(model)"><span><strong>{{model}}</strong><p>{{model==='deepseek-flash'?'Modelo padrão do Depoimento Studio.':'Modelo alternativo; consulte preços da sua conta.'}}</p></span></label></div></section>
     <section class="section"><h2>Tema da interface</h2><div class="choices"><label v-for="value in (['light','dark'] as const)" :key="value" class="choice"><input v-model="theme" type="radio" name="theme" :value="value"><img :src="`/design/${value}-preview.png`" alt="" width="90" height="60"><strong>{{value==='light'?'Claro':'Escuro'}}</strong></label></div></section>
     <div class="notice">As chaves salvas nunca são exibidas novamente. Remova-as aqui antes de desinstalar, se desejar. Modelo e tema valem para a sessão atual.</div>
    </section>
   </div>
   <aside class="copilot session-panel" aria-label="Informações da sessão">
    <template v-if="page==='settings'"><img class="settings-art" src="/design/settings-illustration.png" alt=""><h2>Seu documento continua aqui</h2><p>Volte ao depoimento para continuar de onde parou.</p><button class="button" @click="navigate('document')">Voltar ao documento</button></template>
    <template v-else><h2>Seu espaço de trabalho</h2><p>Um arquivo por vez, com o texto original e a proposta lado a lado.</p><div class="tip"><div><h3>Preserve o sentido</h3><p>Incertezas, negações e relatos de terceiros precisam continuar explícitos.</p></div></div><section v-if="view?.RemoteID" class="remote"><h3>Registro do arquivo</h3><code>{{view.RemoteID}}</code><p>{{view.RemoteDeleted?'A exclusão do registro foi confirmada.':'O texto extraído deste arquivo continua guardado no serviço de transcrição, mesmo depois de cancelar ou fechar o Depoimento Studio.'}}</p><button v-if="!view.RemoteDeleted" class="button danger" :disabled="busy" @click="studio.deleteRemote">Excluir registro</button><button v-if="!view.RemoteDeleted" class="button" :disabled="busy" @click="studio.keepRemote">Manter registro e trocar de arquivo</button></section></template>
   </aside>
  </main>
 </div>
</template>
