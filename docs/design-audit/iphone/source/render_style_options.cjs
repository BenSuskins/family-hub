const { chromium } = require('/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const fs = require('fs');
const path = require('path');
const refinement=process.argv.includes('--cosy');
const styles=refinement?['cosy']:['neutral','sage','indigo'];
const screens=['today','food','household','shopping-list','recipe-detail','today-dark'];
const luminance=color=>color.match(/[\d.]+/g).slice(0,3).map(Number).map(value=>value/255).map(value=>value<=.04045?value/12.92:((value+.055)/1.055)**2.4).reduce((total,value,index)=>total+value*[.2126,.7152,.0722][index],0);
const contrast=(first,second)=>{const values=[luminance(first),luminance(second)].sort((a,b)=>b-a);return +((values[0]+.05)/(values[1]+.05)).toFixed(2);};
(async()=>{
 const directory=path.join(__dirname,refinement?'../cosy-sage':'../style-options');fs.mkdirSync(directory,{recursive:true});
 const browser=await chromium.launch({headless:true,executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const page=await browser.newPage({viewport:{width:780,height:900},deviceScaleFactor:3});
 const checks=[];const failures=[];const errors=[];page.on('pageerror',error=>errors.push(String(error)));
 for(const style of styles)for(const screen of screens){
  await page.goto('file://'+path.join(__dirname,'mockups.html')+`?screen=${screen}&palette=${style}`);
  await page.locator('.phone').screenshot({path:path.join(directory,style+'-'+screen+'.png')});
  for(const width of [402,375]){
   await page.addStyleTag({content:`.phone{width:${width}px}`});
   const result=await page.evaluate(()=>({overflow:document.querySelector('.content').scrollWidth>document.querySelector('.content').clientWidth}));
   checks.push({style,screen,width,...result});if(result.overflow)failures.push({style,screen,width,...result});
  }
  const pairs=await page.evaluate(()=>{
   const phone=document.querySelector('.phone');const color=name=>{const sample=document.createElement('span');sample.style.color=`var(${name})`;phone.append(sample);const value=getComputedStyle(sample).color;sample.remove();return value;};
   const pairs=[];
   for(const text of ['--ink','--secondary','--blue'])for(const surface of ['--surface','--background'])pairs.push({text,surface,first:color(text),second:color(surface)});
   for(const selector of ['.departure h2','.departure .sub','.departure-label','.tag','.button:not(.secondary)']){
    const element=phone.querySelector(selector);if(!element)continue;
    let background=element;while(background&&getComputedStyle(background).backgroundColor==='rgba(0, 0, 0, 0)')background=background.parentElement;
    pairs.push({selector,first:getComputedStyle(element).color,second:getComputedStyle(background).backgroundColor});
   }
   return pairs;
  });
  for(const pair of pairs){const ratio=contrast(pair.first,pair.second);checks.push({style,screen,...pair,ratio});if(ratio<4.5)failures.push({style,screen,...pair,ratio});}
 }
 await page.goto('file://'+path.join(__dirname,refinement?'../cosy-sage.html':'../style-options.html'));
 await page.evaluate(async()=>{for(const image of document.images)image.loading='eager';await Promise.all(Array.from(document.images).map(image=>image.decode()));});
 for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});
  if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth))failures.push({galleryWidth:width});
 }
 await page.setViewportSize({width:1440,height:1000});
 await page.locator('#today').screenshot({path:path.join(directory,'comparison-today.png')});
 for(const style of styles){await page.selectOption('#direction',style);if(await page.locator('#today figure:visible').count()!==2)failures.push({filter:style});}
 await browser.close();
 fs.writeFileSync(path.join(directory,'checks.json'),JSON.stringify({scope:'Static source, 402 and 375 point layouts, selected composed text contrast, gallery and filters',checks,errors,failures},null,2)+'\n');
 if(failures.length||errors.length)throw Error(JSON.stringify({failures,errors}));
 console.log(styles.length*screens.length+' visual variants pass width, selected contrast, gallery, and filter checks.');
})();
