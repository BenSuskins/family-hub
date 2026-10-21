const fs = require('fs');
const path = require('path');
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || '/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const root = path.resolve(__dirname, '..');
let activeBrowser;
(async () => {
  const browser = activeBrowser = await chromium.launch({headless:true, executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const page = await browser.newPage({viewport:{width:1000,height:1100},deviceScaleFactor:3});
  const errors=[];
  page.on('pageerror',error=>errors.push(String(error)));
  const url = 'file://' + path.join(__dirname,'mockups.html');
  await page.goto(url);
  const proposals = await page.evaluate(()=>window.proposals.map(({render,...metadata})=>metadata));
  const records=[];
  for(const proposal of proposals) {
    await page.goto(url+'?screen='+proposal.id);
    await page.evaluate(()=>document.fonts.ready);
    const element=page.locator('#artboard > :first-child');
    const metrics=await element.evaluate(node=>({
      width:node.offsetWidth,height:node.offsetHeight,
      horizontalOverflow:node.scrollWidth>node.clientWidth,
      widgetOverflow:[...(node.matches('.widget')?[node]:[]),...node.querySelectorAll('.widget')].filter(widget=>widget.scrollHeight>widget.clientHeight || widget.scrollWidth>widget.clientWidth).map(widget=>widget.className),
    }));
    if(metrics.horizontalOverflow || metrics.widgetOverflow.length) throw new Error(proposal.id+' overflows: '+JSON.stringify(metrics));
    await element.screenshot({path:path.join(root,'images',proposal.id+'.png'),omitBackground:true});
    records.push({...proposal,...metrics,file:'images/'+proposal.id+'.png'});
  }
  fs.writeFileSync(path.join(root,'manifest.json'),JSON.stringify({kind:'Static HTML/CSS proposals, not simulator captures',scale:3,screens:records},null,2)+'\n');
  await browser.close();
  if(errors.length) throw new Error(errors.join('\n'));
  console.log('Rendered '+records.length+' proposals with no horizontal or widget-content overflow.');
})().catch(async error=>{console.error(error);await activeBrowser?.close();process.exitCode=1;});
