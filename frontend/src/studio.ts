import {computed,onMounted,onUnmounted,ref} from 'vue'
import {call,type Settings,type View} from './backend'

export function useStudio(){
 const view=ref<View|null>(null), settings=ref<Settings>({AssemblyAIConfigured:false,DeepSeekConfigured:false,Model:'deepseek-flash'})
 const error=ref(''),notice=ref(''),pending=ref(false),draft=ref(''),dirty=ref(false)
 const consentA=ref(false),consentD=ref(false),limitation=ref(false),narrative=ref(false)
 const selectedTrack=ref<number>(0),selectedSpeaker=ref('')
 const id=computed(()=>view.value?.Document.ID??0)
 const state=computed(()=>view.value?.Document.State??'empty')
 const busy=computed(()=>pending.value||!!view.value?.Busy)
 const speakers=computed(()=>[...new Set((view.value?.Document.Original.utterances??[]).map(u=>u.speaker))])
 const flags=computed(()=>(view.value?.Changes??[]).filter(c=>c.flags?.length))
 const original=computed(()=>{
  const t=view.value?.Document.Original
  if(!t)return ''
  return t.utterances?.length?t.utterances.map(u=>`[${time(u.start)}] Falante ${u.speaker}: ${u.text}`).join('\n\n'):t.text
 })
 let timer:ReturnType<typeof setTimeout>|undefined, disposed=false, reading=false
 function message(e:unknown){return e instanceof Error?e.message:String(e)}
 async function refresh(){
  if(reading)return;reading=true
  try{
   const fresh=await call('View')
   if(disposed)return
   const old=view.value?.Document,doc=fresh.Document
   if(!old||old.ID!==doc.ID){
    consentA.value=false;consentD.value=false;limitation.value=false;narrative.value=false
    dirty.value=false;selectedSpeaker.value='';selectedTrack.value=doc.Track
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
  if(draft.value && !window.confirm('Abrir outro arquivo descarta o documento desta sessão. Exporte o TXT antes, se precisar. Continuar?'))return
  await action(()=>call('SelectFile'))
 }
 function edit(value:string){draft.value=value;dirty.value=true}
 async function copy(){await action(async()=>{await syncEdit();await call('Copy')},'Proposta copiada para a área de transferência.')}
 async function exportText(){await action(async()=>{await syncEdit();await call('Export')})}
 async function saveKey(provider:string,key:string){
  await action(async()=>{await call('SaveKey',provider,key);settings.value=await call('Settings')},'API Key salva no Credential Manager.')
 }
 async function removeKey(provider:string){await action(async()=>{await call('RemoveKey',provider);settings.value=await call('Settings')},'API Key removida.')}
 async function model(value:string){await action(async()=>{await call('SetModel',value);settings.value=await call('Settings')})}
 async function deleteRemote(){
  if(!window.confirm('Solicitar exclusão desta transcrição na AssemblyAI? A exclusão local é independente e não comprova remoção de todos os artefatos remotos.'))return
  await action(()=>call('DeleteRemote',id.value,true),'Solicitação de exclusão confirmada pela AssemblyAI.')
 }
 async function keepRemote(){
  if(!window.confirm('Manter o registro remoto e liberar a troca de arquivo? Anote o ID antes: o aplicativo não mantém histórico entre sessões.'))return
  await action(()=>call('KeepRemote',id.value,true))
 }
 async function poll(){await refresh();if(!disposed)timer=setTimeout(poll,900)}
 onMounted(async()=>{try{settings.value=await call('Settings')}catch(e){error.value=message(e)};await poll()})
 onUnmounted(()=>{disposed=true;if(timer)clearTimeout(timer)})
 return {view,settings,error,notice,pending,draft,dirty,consentA,consentD,limitation,narrative,selectedTrack,selectedSpeaker,id,state,busy,speakers,flags,original,action,edit,copy,exportText,selectFile,saveKey,removeKey,model,deleteRemote,keepRemote,
  selectTrack:()=>action(()=>call('SelectTrack',id.value,selectedTrack.value)),
  identify:()=>action(()=>call('Identify',id.value,selectedSpeaker.value)),
  transcribe:()=>action(async()=>{await call('StartTranscription',id.value,consentA.value,limitation.value);consentA.value=false}),
  review:()=>action(async()=>{await call('StartReview',id.value,consentD.value,narrative.value);consentD.value=false}),
  retry:()=>action(()=>call('RetryReview',id.value)),
  cancel:()=>action(()=>call('Cancel',id.value))}
}
export function time(ms:number){const seconds=Math.floor(ms/1000);return `${Math.floor(seconds/60).toString().padStart(2,'0')}:${(seconds%60).toString().padStart(2,'0')}`}
