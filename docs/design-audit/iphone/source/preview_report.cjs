const { chromium } = require('/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const path = require('path');
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
 const page=await browser.newPage({viewport:{width:1320,height:1000},deviceScaleFactor:1});
 const errors=[];page.on('pageerror',error=>errors.push(String(error)));
 await page.goto('file://'+path.join(__dirname,'../index.html'));
 await page.evaluate(async()=>{for(const image of document.images)image.loading='eager';await Promise.all(Array.from(document.images).map(image=>image.decode().catch(()=>{})));});
 await page.screenshot({path:'/private/tmp/family-hub-audit-report-header.png'});
 await page.locator('#today').screenshot({path:'/private/tmp/family-hub-audit-report-comparison.png'});
 const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);
 await page.setViewportSize({width:390,height:844});
 await page.locator('#today').scrollIntoViewIfNeeded();
 await page.screenshot({path:'/private/tmp/family-hub-audit-report-mobile.png'});
 const narrowOverflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);
 await browser.close();
 if(errors.length||overflow||narrowOverflow)throw new Error(JSON.stringify({errors,overflow,narrowOverflow}));
 console.log('Report preview passes at 1320px and 390px widths.');
})();
