import { describe,it,expect,vi,afterEach } from 'vitest'
import {call} from './backend'
afterEach(()=>vi.unstubAllGlobals())
describe('backend bridge',()=>{
 it('requires the native runtime before accessing providers',async()=>{
  vi.stubGlobal('window',{})
  await expect(call('Settings')).rejects.toThrow('Abra o aplicativo Windows')
 })
 it('sends typed operations to the single native bridge',async()=>{
  const ByName=vi.fn().mockResolvedValue({AssemblyAIConfigured:true,DeepSeekConfigured:false,Model:'deepseek-flash'})
  vi.stubGlobal('window',{wails:{Call:{ByName}}})
  const settings=await call('Settings')
  expect(settings.AssemblyAIConfigured).toBe(true)
  expect(ByName).toHaveBeenCalledWith('main.Bridge.Settings')
  await call('StartTranscription',9,false,false)
  expect(ByName).toHaveBeenLastCalledWith('main.Bridge.StartTranscription',9,false,false)
 })
})
