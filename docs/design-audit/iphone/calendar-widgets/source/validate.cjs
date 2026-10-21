const fs = require('fs');
const path = require('path');
const { fileURLToPath } = require('url');
const assert = require('assert/strict');
const { chromium } = require(process.env.PLAYWRIGHT_MODULE || '/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const root = path.resolve(__dirname,'..');
const manifest = JSON.parse(fs.readFileSync(path.join(root,'manifest.json'),'utf8'));
let browser;
(async()=>{
  const names = new Set(manifest.screens.map(screen=>path.basename(screen.file)));
  assert.equal(names.size,manifest.screens.length);
  assert.deepEqual(new Set(fs.readdirSync(path.join(root,'images')).filter(name=>name.endsWith('.png'))),names);
  for(const screen of manifest.screens) {
    const png = fs.readFileSync(path.join(root,screen.file));
    assert.equal(png.subarray(1,4).toString(),'PNG');
    assert.equal(png.readUInt32BE(16),screen.width*manifest.scale);
    assert.equal(png.readUInt32BE(20),screen.height*manifest.scale);
  }
  browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH || '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const page=await browser.newPage({viewport:{width:1440,height:1100}});
  const errors=[];page.on('pageerror',error=>errors.push(String(error)));
  const url='file://'+path.join(root,'index.html');
  await page.goto(url);
  assert.equal(await page.locator('article.card').count(),manifest.screens.length);
  const links=await page.locator('a[href],img[src]').evaluateAll(nodes=>nodes.map(node=>node.href||node.src));
  for(const link of links) {
    const parsed=new URL(link);
    if(parsed.protocol!=='file:')continue;
    parsed.search='';parsed.hash='';
    const file=fileURLToPath(parsed);
    if(file===path.join(root,'validation.json'))continue;
    assert(fs.existsSync(file),'Missing file: '+file);
  }
  await page.locator('#group').selectOption('Calendar');
  assert.equal(await page.locator('article.card:visible').count(),16);
  await page.locator('#search').fill('overlapping');
  assert.equal(await page.locator('article.card:visible').count(),1);
  await page.locator('#search').fill('no-such-state');
  assert.equal(await page.locator('article.card:visible').count(),0);
  await page.locator('#group').selectOption('all');
  await page.locator('#search').fill('');
  for(const width of [1440,768,390]) {
    await page.setViewportSize({width,height:1100});
    assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'Gallery overflows at '+width);
  }
  await page.screenshot({path:'/private/tmp/family-hub-calendar-widget-gallery-phone.png'});
  const footers=[];
  for(const screen of manifest.screens) {
    if(screen.group==='Calendar')continue;
    await page.goto('file://'+path.join(__dirname,'mockups.html')+'?screen='+screen.id);
    const collisions=await page.evaluate(()=>[...document.querySelectorAll('.widget')].flatMap(widget=>{
      const footer=widget.querySelector('.widget-foot');
      if(!footer)return [];
      const top=footer.getBoundingClientRect().top;
      return [...widget.querySelectorAll('h2,.widget-chore,.meal-line')].filter(node=>node.getBoundingClientRect().bottom>top-5).map(node=>({text:node.textContent.trim(),gap:top-node.getBoundingClientRect().bottom}));
    }));
    assert.equal(collisions.length,0,screen.id+' overlaps its footer: '+JSON.stringify(collisions));
    footers.push(screen.id);
  }
  assert.equal(errors.length,0,errors.join('\n'));
  const result={
    result:'passed',kind:'Static artifact checks; not native UI tests',
    proposals:manifest.screens.length,calendar:16,widgetAndContext:manifest.screens.length-16,
    pngDimensions:'passed',imageManifest:'passed',localGalleryLinks:'passed',groupFilter:'passed',search:'passed',
    galleryWidths:[1440,768,390],horizontalOverflow:false,widgetFooterSeparation:'passed',widgetStatesChecked:footers.length,
    limits:['Static HTML does not validate native gestures or VoiceOver.','Clear, tinted, and system chrome samples are approximations.'],
  };
  fs.writeFileSync(path.join(root,'validation.json'),JSON.stringify(result,null,2)+'\n');
  console.log(JSON.stringify(result));
})().catch(error=>{console.error(error);process.exitCode=1;}).finally(async()=>{await browser?.close();});
