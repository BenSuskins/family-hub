const fs=require('fs');
const path=require('path');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const root=path.resolve(__dirname,'..');
let browser;
(async()=>{
  browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const page=await browser.newPage({viewport:{width:1000,height:1100},deviceScaleFactor:3});
  const errors=[];page.on('pageerror',error=>errors.push(String(error)));
  const url='file://'+path.join(__dirname,'mockups.html')+'?screen=calendar-week';
  await page.goto(url);
  const entries=await page.evaluate(()=>window.weekProposals.map(({render,...entry})=>entry));
  const screens=[];
  for(const entry of entries){
    await page.goto(url+'&variant='+entry.id);
    await page.evaluate(()=>document.fonts.ready);
    const phone=page.locator('.phone');
    const metrics=await phone.evaluate(node=>({width:node.offsetWidth,height:node.offsetHeight,horizontalOverflow:node.scrollWidth>node.clientWidth}));
    if(metrics.horizontalOverflow)throw new Error(entry.id+' has horizontal overflow');
    await phone.screenshot({path:path.join(root,'images',entry.id+'.png'),omitBackground:true});
    screens.push({...entry,...metrics,file:'images/'+entry.id+'.png'});
  }
  if(errors.length)throw new Error(errors.join('\n'));
  fs.writeFileSync(path.join(root,'manifest.json'),JSON.stringify({kind:'Static iPhone proposals; not simulator captures',scale:3,screens},null,2)+'\n');
  console.log('Rendered '+screens.length+' week-view proposals.');
})().catch(error=>{console.error(error);process.exitCode=1;}).finally(async()=>{await browser?.close();});
