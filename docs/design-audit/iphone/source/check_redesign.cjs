const { chromium } = require('/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('fs');
const path = require('path');
const keys = JSON.parse(fs.readFileSync(path.join(__dirname,'../mockups/manifest.json'))).map(item=>item.key);
const luminance = color => {
 const channels=color.match(/[\d.]+/g).slice(0,3).map(Number).map(value=>value/255).map(value=>value<=0.04045?value/12.92:((value+0.055)/1.055)**2.4);
 return channels[0]*.2126+channels[1]*.7152+channels[2]*.0722;
};
const contrast = (first,second) => {
 const values=[luminance(first),luminance(second)].sort((a,b)=>b-a);
 return Number(((values[0]+.05)/(values[1]+.05)).toFixed(2));
};
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const page=await browser.newPage({viewport:{width:780,height:900},deviceScaleFactor:1});
 const failures=[];
 const measurements=[];
 for (const key of keys) {
  await page.goto('file://'+path.join(__dirname,'mockups.html')+'?screen='+key);
  await page.addStyleTag({content:'.phone{width:375px;height:812px}'});
  const result=await page.evaluate(()=>{
   const content=document.querySelector('.content');
   const navigation=Array.from(document.querySelector('.navbar').children).filter(element=>element.textContent.trim()||element.querySelector('svg'));
   const overlaps=navigation.slice(1).some((element,index)=>element.getBoundingClientRect().left<navigation[index].getBoundingClientRect().right-1);
   return {overflow:content.scrollWidth>content.clientWidth,overlaps,contentWidth:content.clientWidth,cosy:document.querySelector('.phone').classList.contains('style-cosy'),titleSize:parseFloat(getComputedStyle(document.querySelector('h1')||content).fontSize)};
  });
  measurements.push({key,...result});
  if(result.cosy===key.startsWith('calendar-')||(key.endsWith('-large')&&result.titleSize<40)||result.overflow||result.overlaps||result.contentWidth!==375)failures.push({key,...result});
  if(['today','today-large','calendar-month','area'].includes(key))await page.locator('.phone').screenshot({path:'/private/tmp/family-hub-revised-small-'+key+'.png'});
 }
 const pairs=[];
 for (const key of ['today','today-dark']) {
  await page.goto('file://'+path.join(__dirname,'mockups.html')+'?screen='+key);
  const colors=await page.evaluate(()=>{
   const phone=document.querySelector('.phone');
   return ['--ink','--secondary','--blue','--surface','--background'].reduce((result,name)=>{
    const sample=document.createElement('span');sample.style.color=`var(${name})`;phone.append(sample);result[name]=getComputedStyle(sample).color;sample.remove();return result;
   },{});
  });
  for (const text of ['--ink','--secondary','--blue'])for (const surface of ['--surface','--background']){
   const ratio=contrast(colors[text],colors[surface]);pairs.push({key,text,surface,ratio});if(ratio<4.5)failures.push({key,text,surface,ratio});
  }
 }
 const preferenceChecks=[];
 const session=await page.context().newCDPSession(page);
 for(const preference of ['prefers-reduced-transparency','prefers-contrast']){
  await session.send('Emulation.setEmulatedMedia',{features:[{name:'prefers-reduced-motion',value:'reduce'},{name:'prefers-reduced-transparency',value:preference==='prefers-reduced-transparency'?'reduce':'no-preference'},{name:'prefers-contrast',value:preference==='prefers-contrast'?'more':'no-preference'}]});
  const result=await page.evaluate(()=>({reducedTransparency:matchMedia('(prefers-reduced-transparency:reduce)').matches,increasedContrast:matchMedia('(prefers-contrast:more)').matches,motion:matchMedia('(prefers-reduced-motion:reduce)').matches,glass:getComputedStyle(document.querySelector('.phone')).getPropertyValue('--glass').trim(),border:getComputedStyle(document.querySelector('.tabs')).borderColor}));
  preferenceChecks.push({preference,...result});
  if(!result.motion || (preference==='prefers-reduced-transparency' && (!result.reducedTransparency || result.glass.includes('rgba'))) || (preference==='prefers-contrast' && (!result.increasedContrast || result.glass.includes('rgba')))) failures.push({preference,...result});
 }
 await browser.close();
 fs.writeFileSync(path.join(__dirname,'../evidence/redesign-checks.json'),JSON.stringify({scope:'Static HTML source; native behaviour requires implementation checks',smallPhone:measurements,contrastPairs:pairs,preferences:preferenceChecks,failures},null,2)+'\n');
 if(failures.length)throw new Error(JSON.stringify(failures));
 console.log(keys.length+' small-phone layouts pass. 12 text contrast pairs pass. Accessibility preference styles recorded.');
})();
