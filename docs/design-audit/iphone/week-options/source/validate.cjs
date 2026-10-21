const fs=require('fs');
const path=require('path');
const assert=require('assert/strict');
const {fileURLToPath}=require('url');
const {chromium}=require(process.env.PLAYWRIGHT_MODULE||'/Users/bensuskins/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/node_modules/playwright');
const root=path.resolve(__dirname,'..');
const manifest=JSON.parse(fs.readFileSync(path.join(root,'manifest.json'),'utf8'));
let browser;
(async()=>{
  const actual=new Set(fs.readdirSync(path.join(root,'images')).filter(name=>name.endsWith('.png')));
  assert.deepEqual(actual,new Set(manifest.screens.map(screen=>path.basename(screen.file))));
  for(const screen of manifest.screens){
    const png=fs.readFileSync(path.join(root,screen.file));
    assert.equal(png.subarray(1,4).toString(),'PNG');
    assert.equal(png.readUInt32BE(16),screen.width*3);
    assert.equal(png.readUInt32BE(20),screen.height*3);
  }
  browser=await chromium.launch({headless:true,executablePath:process.env.CHROME_PATH||'/Applications/Google Chrome.app/Contents/MacOS/Google Chrome'});
  const page=await browser.newPage({viewport:{width:1440,height:1100}});
  const errors=[];page.on('pageerror',error=>errors.push(String(error)));
  await page.goto('file://'+path.join(root,'index.html'));
  assert.equal(await page.locator('#compare article').count(),4);
  assert.equal(await page.locator('article.card').count(),11);
  assert.equal(await page.locator('article.card.recommended').getAttribute('id'),'week-compact');
  const links=await page.locator('a[href],img[src]').evaluateAll(nodes=>nodes.map(node=>node.href||node.src));
  for(const link of links){
    const parsed=new URL(link);
    if(parsed.protocol!=='file:')continue;
    parsed.search='';parsed.hash='';
    const target=fileURLToPath(parsed);
    if(target===path.join(root,'validation.json'))continue;
    assert(fs.existsSync(target),'Missing file: '+target);
  }
  for(const width of [1440,768,390]){
    await page.setViewportSize({width,height:1100});
    assert(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth),'Gallery overflows at '+width);
  }
  await page.locator('#interactions > summary').click();
  assert.equal(await page.locator('#interactions article:visible').count(),4);
  await page.locator('#adaptations > summary').click();
  assert.equal(await page.locator('#adaptations article:visible').count(),3);
  await page.screenshot({path:'/private/tmp/family-hub-week-options-gallery-phone.png'});
  const metrics={};
  for(const screen of manifest.screens){
    await page.goto('file://'+path.join(__dirname,'mockups.html')+'?screen=calendar-week&variant='+screen.id);
    metrics[screen.id]=await page.evaluate(()=>({
      dates:document.querySelectorAll('.date-strip .date-choice').length,
      canvas:document.querySelector('.week-canvas')?.getBoundingClientRect().height,
      allDay:document.querySelector('.compact-all-day,.all-day')?.getBoundingClientRect().height,
      agendaRows:document.querySelectorAll('.pane-row').length,
      sheetRows:document.querySelectorAll('.all-day-row').length,
      currentTime:!!document.querySelector('.now-line'),
      invalidHours:[...document.querySelectorAll('.hour time')].filter(node=>Number(node.textContent.split(':')[0])>24).map(node=>node.textContent),
      textOverflow:[...document.querySelectorAll('.pane-copy strong,.pane-copy small,.all-day-row strong,.sheet-header strong')].filter(node=>node.scrollWidth>node.clientWidth).map(node=>node.textContent),
    }));
    assert.equal(metrics[screen.id].textOverflow.length,0,screen.id+' clips detail text');
    assert.equal(metrics[screen.id].invalidHours.length,0,screen.id+' shows an invalid hour');
  }
  assert.equal(metrics['week-baseline'].allDay,160);
  assert.equal(metrics['week-compact'].allDay,44);
  assert.equal(metrics['week-compact'].canvas-metrics['week-baseline'].canvas,116);
  assert.equal(metrics['week-three-days'].dates,3);
  assert.equal(metrics['week-agenda'].dates,7);
  assert.equal(metrics['week-agenda'].agendaRows,4);
  assert.equal(metrics['week-all-day-expanded'].sheetRows,5);
  assert.equal(metrics['week-smart-start'].currentTime,false);
  assert.equal(metrics['week-large-text'].agendaRows,0);
  assert.equal(errors.length,0,errors.join('\n'));
  const result={result:'passed',scope:'Static proposal checks, not native UI tests',screens:11,mainAlternatives:3,manifest:'passed',pngDimensions:'passed',localLinks:'passed',galleryWidths:[1440,768,390],horizontalOverflow:false,disclosures:'passed',allDaySpaceRecovered:116,detailsText:'passed',layoutMetrics:metrics};
  fs.writeFileSync(path.join(root,'validation.json'),JSON.stringify(result,null,2)+'\n');
  console.log('Validated 11 images, three alternatives, interaction states, and gallery widths.');
})().catch(error=>{console.error(error);process.exitCode=1;}).finally(async()=>{await browser?.close();});
