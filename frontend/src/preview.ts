// Development-only visual fixtures. Vite removes this import from production builds.
import type {View,Settings} from './backend'
export function installPreview(){
 const stage=new URLSearchParams(location.search).get('preview')??'empty'
 const states:Record<string,string>={empty:'',track:'awaiting_track',consent:'awaiting_consent',processing:'transcribing',speaker:'awaiting_speaker',review:'ready',prepare:'awaiting_review_consent',failed:'failed',settings:'ready'}
 const state=states[stage]??''
 const settings:Settings={AssemblyAIConfigured:true,DeepSeekConfigured:true,Model:'deepseek-flash'}
 const view:View={Document:{ID:state?1:0,File:state?'C:\\Ficticios\\relato-feira.wav':'',State:state,Track:0,Speaker:state==='awaiting_speaker'?'':'B',Original:{id:'ficticio-001',status:state==='transcribing'?'queued':'completed',text:state?'Viu o veículo? Não, ouvi dizer.':'',speech_model_used:'universal-3-5-pro',utterances:state && !['transcribing','awaiting_consent','awaiting_track'].includes(state)?[{speaker:'A',start:6000,end:10000,text:'Viu o veículo naquela noite?'},{speaker:'B',start:14000,end:22000,text:'Não. Eu ouvi dizer que havia 12 pessoas, mas não tenho certeza.'}]:[]},Proposal:state==='ready'?'Falante A: Viu o veículo naquela noite?\n\nFalante B: Não. Eu ouvi dizer que havia 12 pessoas, mas não tenho certeza.':'',Edited:'',HasEdits:false},Tracks:state?[{index:0,codec_name:'pcm_s16le',channels:1,sample_rate:'44100',tags:{language:'pt',title:'Depoimento'}},{index:1,codec_name:'aac',channels:2,sample_rate:'48000',tags:{language:'pt',title:'Ambiente'}}]:[],Busy:state==='transcribing',Error:'',RemoteID:state && !['transcribing','awaiting_consent','awaiting_track'].includes(state)?'ficticio-001':'',RemoteDeleted:false,Changes:[]}
 window.wails={Call:{ByName:async(name,...args)=>{
  switch(name.split('.').pop()){
   case 'View':return structuredClone(view)
   case 'Settings':return structuredClone(settings)
   case 'SaveKey':if(args[0]==='assemblyai')settings.AssemblyAIConfigured=true;else settings.DeepSeekConfigured=true;return
   case 'RemoveKey':if(args[0]==='assemblyai')settings.AssemblyAIConfigured=false;else settings.DeepSeekConfigured=false;return
   case 'SetModel':settings.Model=String(args[0]);return
   case 'Edit':view.Document.Edited=String(args[1]);view.Document.HasEdits=true;return
   case 'Copy':return
   default:throw new Error('Prévia visual: esta ação requer o aplicativo Windows.')
  }
 }}}
 if(stage==='settings')location.hash='settings'
}
