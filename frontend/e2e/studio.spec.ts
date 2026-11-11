import {test,expect} from '@playwright/test'
test('consentimentos começam desmarcados e bloqueiam o envio',async({page})=>{
 await page.goto('/?preview=consent')
 const consent=page.getByRole('checkbox',{name:'Confirmo que tenho autorização aplicável para enviar este arquivo à AssemblyAI.'})
 const limitation=page.getByRole('checkbox',{name:'Entendo essa limitação e vou conferir a transcrição ouvindo o áudio original.'})
 const start=page.getByRole('button',{name:'Autorizar e transcrever'})
 await expect(consent).not.toBeChecked();await expect(limitation).not.toBeChecked();await expect(start).toBeDisabled()
 await consent.check();await expect(start).toBeDisabled();await limitation.check();await expect(start).toBeEnabled()
})
test('configurações limpam chaves digitadas e preservam edição humana',async({page})=>{
 await page.goto('/?preview=review')
 await page.getByLabel('Proposta editável').fill('Edição humana fictícia: não vi o veículo.')
 await page.getByRole('button',{name:'Configurações',exact:true}).first().click()
 const field=page.getByLabel('API Key AssemblyAI',{exact:true})
 await expect(field).toHaveAttribute('type','password')
 await field.fill('chave-ficticia-apenas-teste')
 await field.press('Enter')
 await expect(field).toHaveValue('')
 await expect(page.getByRole('status')).toContainText('salva')
 await page.getByRole('button',{name:'Voltar ao documento',exact:true}).first().click()
 await expect(page.getByLabel('Proposta editável')).toHaveValue('Edição humana fictícia: não vi o veículo.')
 await page.getByRole('button',{name:/Copiar proposta/}).click()
 await expect(page.getByRole('status')).toContainText('Proposta copiada para a área de transferência.')
})
test('layout, temas, contraste, DPI alto e navegação estreita',async({page,browser})=>{
 const checkContrast=async()=>{
  const ratio=await page.evaluate(()=>{
   const parse=(c:string)=>{const m=c.match(/\d+(\.\d+)?/g)?.map(Number)??[0,0,0];return m.slice(0,3)}
   const lum=([r=0,g=0,b=0]:number[])=>{const f=(v:number)=>{v/=255;return v<=0.03928?v/12.92:((v+0.055)/1.055)**2.4};return 0.2126*f(r)+0.7152*f(g)+0.0722*f(b)}
   const fg=lum(parse(getComputedStyle(document.body).color))
   const bg=lum(parse(getComputedStyle(document.body).backgroundColor))
   return (Math.max(fg,bg)+0.05)/(Math.min(fg,bg)+0.05)
  })
  expect(ratio).toBeGreaterThanOrEqual(4.5)
 }
 for(const [width,height] of [[1366,768],[1920,1080],[900,768]]){
  await page.setViewportSize({width,height});await page.goto('/?preview=review')
  await expect(page.getByRole('heading',{name:'Compare e aperfeiçoe o depoimento'})).toBeVisible()
  expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true)
  await checkContrast()
  if(width===900){await page.getByRole('tab',{name:'Proposta',exact:true}).click();await expect(page.getByLabel('Proposta editável')).toBeVisible()}
  await page.screenshot({path:`../build/screenshots/review-${width}.png`,fullPage:true})
 }
 await page.setViewportSize({width:1366,height:768});await page.goto('/?preview=settings#settings')
 await page.screenshot({path:'../build/screenshots/settings-light.png',fullPage:true})
 await page.getByLabel('Escuro',{exact:true}).check()
 await expect(page.locator('html')).toHaveAttribute('data-theme','dark')
 await checkContrast()
 await page.screenshot({path:'../build/screenshots/settings-dark.png',fullPage:true})
 const hidpi=await browser.newContext({viewport:{width:1366,height:768},deviceScaleFactor:2})
 const hidpiPage=await hidpi.newPage()
 await hidpiPage.goto('http://127.0.0.1:5173/?preview=review')
 await expect(hidpiPage.getByRole('heading',{name:'Compare e aperfeiçoe o depoimento'})).toBeVisible()
 expect(await hidpiPage.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true)
 await hidpi.close()
})
test('teclado e movimento reduzido',async({page})=>{
 await page.emulateMedia({reducedMotion:'reduce'})
 await page.goto('/?preview=processing')
 expect(await page.locator('.pulse').evaluate(el=>getComputedStyle(el).animationName)).toBe('none')
 await page.keyboard.press('Tab');await expect(page.getByRole('link',{name:'Ir ao documento'})).toBeFocused()
})
