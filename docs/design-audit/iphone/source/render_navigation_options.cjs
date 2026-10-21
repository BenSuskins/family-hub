const { chromium } = require('/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('fs');
const path = require('path');
(async()=>{
 const directory=path.join(__dirname,'../navigation-options');
 fs.mkdirSync(directory,{recursive:true});
 const browser=await chromium.launch({headless:true,executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const page=await browser.newPage({viewport:{width:780,height:900},deviceScaleFactor:3});
 const checks=[];
 for(const option of ['direct','menu','overview'])for(const screen of ['food','household']){
  await page.goto('file://'+path.join(__dirname,'mockups.html')+`?screen=${screen}&navigation=${option}`);
  await page.locator('.phone').screenshot({path:path.join(directory,option+'-'+screen+'.png')});
  for(const width of [402,375]){
   await page.addStyleTag({content:`.phone{width:${width}px}`});
   const result=await page.evaluate(()=>({overflow:document.querySelector('.content').scrollWidth>document.querySelector('.content').clientWidth,segments:document.querySelectorAll('.content .segments').length}));
   checks.push({option,screen,width,...result});
  }
 }
 await page.goto('file://'+path.join(__dirname,'../navigation-options.html'));
 for(const width of [1200,390]){
  await page.setViewportSize({width,height:900});
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);
  if(overflow)throw Error('Alternative gallery overflow at '+width);
 }
 const images=await page.locator('img').evaluateAll(images=>images.every(image=>image.complete&&image.naturalWidth>0));
 if(!images)throw Error('Alternative gallery has a missing image');
 for(const screen of ['calendar-day','event']){
  await page.goto('file://'+path.join(__dirname,'mockups.html')+'?screen='+screen);
  if(/arrival margin/i.test(await page.locator('.content').innerText()))throw Error('Arrival margin remains in '+screen);
 }
 await browser.close();
 fs.writeFileSync(path.join(directory,'checks.json'),JSON.stringify(checks,null,2)+'\n');
 if(checks.some(check=>check.overflow||check.segments))throw Error(JSON.stringify(checks));
 console.log('Six navigation alternatives pass at 402 and 375 points. No inner segmented controls.');
})();
