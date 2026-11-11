export type Utterance = { speaker:string; start:number; end:number; text:string }
export type Transcript = { id:string; status:string; text:string; speech_model_used:string; utterances:Utterance[]|null }
export type Document = { ID:number; File:string; State:string; Track:number; Speaker:string; Original:Transcript; Proposal:string; Edited:string; HasEdits:boolean }
export type Settings = { AssemblyAIConfigured:boolean; DeepSeekConfigured:boolean; Model:string }
export type View = {Document:Document; Tracks: {index:number; codec_name:string; channels:number; sample_rate:string; tags:{language:string;title:string}}[]|null; Busy:boolean; Error:string; RemoteID:string; RemoteDeleted:boolean; Changes:{id:string;original:string;revised:string;flags:string[]|null}[]|null}
interface Methods {
 Settings: {args:[];result:Settings}
 SaveKey:{args:[string,string];result:void}
 RemoveKey:{args:[string];result:void}
 SetModel:{args:[string];result:void}
 View:{args:[];result:View}
 SelectFile:{args:[];result:void}
 SelectTrack:{args:[number,number];result:void}
 Identify:{args:[number,string];result:void}
 StartTranscription:{args:[number,boolean,boolean];result:void}
 StartReview:{args:[number,boolean,boolean];result:void}
 RetryReview:{args:[number];result:void}
 Cancel:{args:[number];result:void}
 Edit:{args:[number,string];result:void}
 DeleteRemote:{args:[number,boolean];result:void}
 KeepRemote:{args:[number,boolean];result:void}
 Copy:{args:[];result:void}
 Export:{args:[];result:void}
}
declare global {interface Window{wails?:{Call:{ByName:(name:string,...args:unknown[])=>Promise<unknown>}}}}
// Go bridge is the sole transport. No keys or content are stored in localStorage.
export async function call<K extends keyof Methods>(method:K,...args:Methods[K]['args']):Promise<Methods[K]['result']>{
 if (!window.wails?.Call) throw new Error('Abra o aplicativo Windows para usar arquivos, credenciais e provedores.')
 return await window.wails.Call.ByName(`main.Bridge.${method}`,...args) as Methods[K]['result']
}
