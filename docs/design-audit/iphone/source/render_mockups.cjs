const { chromium } = require('/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('fs');
const path = require('path');
const vm = require('vm');
const directory = __dirname;
const context = {};
vm.createContext(context);
vm.runInContext(fs.readFileSync(path.join(directory, 'screens.js'),'utf8')+fs.readFileSync(path.join(directory, 'redesign.js'),'utf8')+'\nthis.screenKeys=Object.keys(screens);',context);
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const page=await browser.newPage({viewport:{width:780,height:900},deviceScaleFactor:3});
 const errors=[];page.on('pageerror',error=>errors.push(String(error)));
 const checks=[];
 for(const key of context.screenKeys){
  await page.goto('file://'+path.join(directory,'mockups.html')+'?screen='+key);
  await page.locator('.phone').screenshot({path:path.join(directory,'../mockups',key+'.png')});
  const check=await page.evaluate(()=>({width:document.querySelector('.phone').offsetWidth,height:document.querySelector('.phone').offsetHeight,overflow:document.querySelector('.content').scrollWidth>document.querySelector('.content').clientWidth,title:document.title}));
  checks.push({key,...check});
 }
 fs.writeFileSync(path.join(directory,'../mockups/manifest.json'),JSON.stringify(checks,null,2)+'\n');
 await browser.close();
 if(errors.length||checks.some(c=>c.overflow)){throw new Error(JSON.stringify({errors,overflow:checks.filter(c=>c.overflow)}));}
 console.log('Rendered '+checks.length+' static iPhone proposals. No horizontal overflow.');
})();
