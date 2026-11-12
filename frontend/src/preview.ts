// Development-only visual fixtures. Vite removes this import from production builds.
import type {View,Settings} from './backend'
export function installPreview(){
 const stage=new URLSearchParams(location.search).get('preview')??'empty'
 const states:Record<string,string>={empty:'',track:'awaiting_track',start:'awaiting_start',converting:'transcribing',processing:'transcribing',adjusting:'reviewing',review:'ready',voices:'ready',failed:'failed',settings:'ready'}
 const state=states[stage]??''
 const done=['reviewing','ready','failed'].includes(state)
 const settings:Settings={AssemblyAIConfigured:true,DeepSeekConfigured:true,Model:'deepseek-flash'}
 const view:View={Format:state?'wav':'',Duration:state?22:0,Size:state?1940000:0,Stage:stage==='converting'?'converting':state==='transcribing'?'transcribing':state==='reviewing'?'reviewing':'',Progress:stage==='converting'?0.42:state==='reviewing'?0.5:-1,Document:{ID:state?1:0,File:state?'C:\\Ficticios\\relato-feira.wav':'',State:state,Track:0,Original:{id:'ficticio-001',status:done?'completed':'queued',text:done?'no dia 3 eu tava em casa né quando ele me ligou eu não vi o veículo.':'',speech_model_used:'universal-3-5-pro',utterances:done?[{speaker:'A',start:6000,end:10000,text:'no dia 3 eu tava em casa né quando ele me ligou'},{speaker:stage==='voices'?'B':'A',start:14000,end:22000,text:'eu não vi o veículo. ouvi dizer que tinha 12 pessoas mas não tenho certeza'}]:[]},Proposal:state==='ready'?'No dia 3, eu estava em casa quando ele me ligou.\n\nEu não vi o veículo. Ouvi dizer que havia 12 pessoas, mas não tenho certeza.':'',Edited:'',HasEdits:false},Tracks:state?[{index:0,codec_name:'pcm_s16le',channels:1,sample_rate:'44100',tags:{language:'pt',title:'Depoimento'}},{index:1,codec_name:'aac',channels:2,sample_rate:'48000',tags:{language:'pt',title:'Ambiente'}}]:[],Busy:state==='transcribing'||state==='reviewing',Error:'',RemoteID:done?'ficticio-001':'',RemoteDeleted:false,Changes:[]}
 window.wails={Call:{ByName:async(name,...args)=>{
  switch(name.split('.').pop()){
   case 'View':return structuredClone(view)
   case 'Settings':return structuredClone(settings)
   case 'SaveKey':if(args[0]==='assemblyai')settings.AssemblyAIConfigured=true;else settings.DeepSeekConfigured=true;return
   case 'RemoveKey':if(args[0]==='assemblyai')settings.AssemblyAIConfigured=false;else settings.DeepSeekConfigured=false;return
   case 'SetModel':settings.Model=String(args[0]);return
   case 'Edit':view.Document.Edited=String(args[1]);view.Document.HasEdits=true;return
   case 'Copy':return
   case 'CopyOriginal':return
   case 'DeleteRemote':view.RemoteDeleted=true;return
   case 'KeepRemote':view.RemoteID='';return
   case 'SelectFile':if(view.RemoteID&&!view.RemoteDeleted)throw new Error('registro pendente');Object.assign(view.Document,{ID:2,File:'C:\\Ficticios\\outro.wav',State:'awaiting_start',Proposal:''});view.RemoteID='';return
   default:throw new Error('Prévia visual: esta ação requer o aplicativo Windows.')
  }
 }}}
 if(stage==='settings')location.hash='settings'
}
