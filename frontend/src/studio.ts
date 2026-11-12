import {computed,onMounted,onUnmounted,ref,watch} from 'vue'
import {call,type Settings,type View} from './backend'

// alt is an optional second way forward; ask() resolves to the chosen button or false when cancelled.
export type Question={title:string;message:string;confirm:string;danger?:boolean;alt?:string}
type Answer=false|'confirm'|'alt'
export function useStudio(){
 // One in-app confirmation at a time; answer() resolves the pending ask().
 const question=ref<Question|null>(null)
 let settle:((choice:Answer)=>void)|undefined
 function ask(q:Question){settle?.(false);question.value=q;return new Promise<Answer>(resolve=>{settle=resolve})}
 function answer(choice:Answer){question.value=null;const done=settle;settle=undefined;done?.(choice)}
 const view=ref<View|null>(null), settings=ref<Settings>({AssemblyAIConfigured:false,DeepSeekConfigured:false,Model:'deepseek-flash'})
 const error=ref(''),notice=ref(''),pending=ref(false),draft=ref(''),dirty=ref(false)
 const selectedTrack=ref<number>(0)
 const id=computed(()=>view.value?.Document.ID??0)
 const state=computed(()=>view.value?.Document.State||'empty')
 const busy=computed(()=>pending.value||!!view.value?.Busy)
 // More than one detected voice only raises a notice: the whole recording is one deponent's speech.
 const voices=computed(()=>new Set((view.value?.Document.Original.utterances??[]).map(u=>u.speaker)).size)
 const keysReady=computed(()=>settings.value.AssemblyAIConfigured&&settings.value.DeepSeekConfigured)
 const flags=computed(()=>(view.value?.Changes??[]).filter(c=>c.flags?.length))
 const original=computed(()=>{
  const t=view.value?.Document.Original
  if(!t)return ''
  return t.utterances?.length?t.utterances.map(u=>`[${time(u.start)}] ${u.text}`).join('\n\n'):t.text
 })
 let timer:ReturnType<typeof setTimeout>|undefined, disposed=false, reading=false
 // Success messages leave on their own; errors stay until the next action.
 let fading:ReturnType<typeof setTimeout>|undefined
 watch(notice,value=>{if(fading)clearTimeout(fading);if(value)fading=setTimeout(()=>{notice.value=''},2000)})
 // Errors of an action also leave on their own, with more time to be read.
 let failing:ReturnType<typeof setTimeout>|undefined
 watch(error,value=>{if(failing)clearTimeout(failing);if(value)failing=setTimeout(()=>{error.value=''},6000)})
 // A processing failure stays until the user closes it: it may happen while they are away.
 const closed=ref('')
 const failure=computed(()=>{const text=view.value?.Error??'';return text&&text!==closed.value?text:''})
 watch(()=>view.value?.Error,value=>{if(!value)closed.value=''})
 function closeFailure(){closed.value=view.value?.Error??''}
 function message(e:unknown){return e instanceof Error?e.message:String(e)}
 async function refresh(){
  if(reading)return;reading=true
  try{
   const fresh=await call('View')
   if(disposed)return
   const old=view.value?.Document,doc=fresh.Document
   if(!old||old.ID!==doc.ID){
    dirty.value=false;selectedTrack.value=doc.Track
    draft.value=doc.HasEdits?doc.Edited:doc.Proposal
   }else if(!dirty.value && doc.Proposal!==old.Proposal){draft.value=doc.HasEdits?doc.Edited:doc.Proposal}
   view.value=fresh
  }catch(e){error.value=message(e)}finally{reading=false}
 }
 async function action(fn:()=>Promise<unknown>,success=''){
  if(pending.value)return
  pending.value=true;error.value='';notice.value=''
  try{await fn();notice.value=success;await refresh()}catch(e){error.value=message(e)}finally{pending.value=false}
 }
 async function syncEdit(){if(dirty.value){await call('Edit',id.value,draft.value);dirty.value=false}}
 async function selectFile(){
  if(draft.value && !await ask({title:'Abrir outro arquivo?',message:'O documento desta sessão será descartado. Exporte o TXT antes, se precisar.',confirm:'Abrir outro arquivo'}))return
  // The record of the current file must be settled before another file is picked.
  let record:Answer=false
  if(view.value?.RemoteID && !view.value.RemoteDeleted){
   record=await ask({title:'O que fazer com o registro deste arquivo?',message:'O texto extraído do arquivo atual continua guardado no serviço de transcrição. Escolha se ele deve ser excluído ou mantido antes de abrir outro arquivo. Se for manter, anote o código do registro: '+view.value.RemoteID+'.',confirm:'Excluir registro',danger:true,alt:'Manter registro'})
   if(!record)return
  }
  await action(async()=>{
   if(record==='confirm')await call('DeleteRemote',id.value,true)
   if(record==='alt')await call('KeepRemote',id.value,true)
   await call('SelectFile')
  })
 }
 function edit(value:string){draft.value=value;dirty.value=true}
 async function copy(){await action(async()=>{await syncEdit();await call('Copy')},'Proposta copiada para a área de transferência.')}
 async function copyOriginal(){await action(()=>call('CopyOriginal'),'Transcrição original copiada para a área de transferência.')}
 async function exportText(){await action(async()=>{await syncEdit();await call('Export')})}
 async function saveKey(provider:string,key:string){
  await action(async()=>{await call('SaveKey',provider,key);settings.value=await call('Settings')},'Chave de API salva.')
 }
 async function removeKey(provider:string){await action(async()=>{await call('RemoveKey',provider);settings.value=await call('Settings')},'Chave de API removida.')}
 async function model(value:string){await action(async()=>{await call('SetModel',value);settings.value=await call('Settings')})}
 async function deleteRemote(){
  if(!await ask({title:'Excluir o registro deste arquivo?',message:'O texto extraído será apagado do serviço de transcrição. Isso não comprova a remoção de todas as cópias guardadas pelo serviço.',confirm:'Excluir registro',danger:true}))return
  await action(()=>call('DeleteRemote',id.value,true),'Registro do arquivo excluído.')
 }
 async function keepRemote(){
  if(!await ask({title:'Manter o registro e trocar de arquivo?',message:'Anote o código do registro antes: o Depoimento Studio não guarda histórico entre sessões.',confirm:'Manter e trocar'}))return
  await action(()=>call('KeepRemote',id.value,true))
 }
 async function poll(){await refresh();if(!disposed)timer=setTimeout(poll,900)}
 onMounted(async()=>{try{settings.value=await call('Settings')}catch(e){error.value=message(e)};await poll()})
 onUnmounted(()=>{disposed=true;if(timer)clearTimeout(timer);if(fading)clearTimeout(fading);if(failing)clearTimeout(failing)})
 return {question,answer,failure,closeFailure,view,settings,error,notice,pending,draft,dirty,selectedTrack,id,state,busy,voices,keysReady,flags,original,action,edit,copy,copyOriginal,exportText,selectFile,saveKey,removeKey,model,deleteRemote,keepRemote,
  selectTrack:()=>action(()=>call('SelectTrack',id.value,selectedTrack.value)),
  start:()=>action(()=>call('Start',id.value)),
  retry:()=>action(()=>call('RetryReview',id.value)),
  cancel:()=>action(()=>call('Cancel',id.value))}
}
export function time(ms:number){const seconds=Math.floor(ms/1000);return `${Math.floor(seconds/60).toString().padStart(2,'0')}:${(seconds%60).toString().padStart(2,'0')}`}
