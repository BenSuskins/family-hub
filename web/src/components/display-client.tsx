"use client";

import { QRCodeSVG } from "qrcode.react";
import { FormEvent, useCallback, useEffect, useRef, useState } from "react";
import { KioskScratchpad } from "@/components/kiosk-scratchpad";
import { getApiBaseUrl } from "@/lib/api";

type DisplayEvent = { ID: string; Title: string; StartTime: string; Location: string };
type DisplayChore = { ID: string; Name: string; Status: string; DueDate?: string | null };
type DisplayMeal = { Date: string; MealType: string; Name: string };
type DisplayRecipe = { ID: string; Title: string; Steps?: string[]; Instructions?: string; Ingredients?: { name: string; items: string[] }[] };
type Weather = { status: string; current?: { temperature: number; code: number }; high?: number; low?: number; hours?: { time: string; temperature: number; code: number; rainProbability: number }[]; days?: { date: string; high: number; low: number; code: number; rainProbability?: number }[] };
type Snapshot = { events: DisplayEvent[]; chores: DisplayChore[]; meals: DisplayMeal[]; weather: Weather; timezone: string; updatedAt: string };
type Pairing = { challengeId: string; pairingCode: string; expiresAt: string };
type OverviewSection = "agenda" | "meals" | "chores" | "weather";
const overviewSections: { id: OverviewSection; title: string; description: string }[] = [
  { id: "agenda", title: "Calendar", description: "Today’s events" },
  { id: "meals", title: "Meals", description: "Today’s meal plan" },
  { id: "chores", title: "Chores", description: "Shared routines" },
  { id: "weather", title: "Weather", description: "Local forecast" },
];
const displayTokenKey = "family-hub-display-token";
const snapshotKey = "family-hub-display-snapshot";
const displaySectionsKey = "family-hub-display-sections";
const apiBase = getApiBaseUrl();
class DisplayAccessRevoked extends Error {}

async function displayRequest<T>(path: string, token: string): Promise<T> {
  const response = await fetch(apiBase + path, { headers: { Authorization: "Bearer " + token } });
  if (response.status === 401 || response.status === 403) throw new DisplayAccessRevoked();
  if (!response.ok) throw new Error("Display data is temporarily unavailable.");
  return response.json() as Promise<T>;
}

function weatherLabel(code:number){if(code===0)return "Clear";if([1,2].includes(code))return "Partly cloudy";if(code===3)return "Cloudy";if([45,48].includes(code))return "Fog";if([51,53,55,56,57].includes(code))return "Drizzle";if([61,63,65,66,67,80,81,82].includes(code))return "Rain";if([71,73,75,77,85,86].includes(code))return "Snow";if([95,96,99].includes(code))return "Thunderstorms";return "Forecast";}
function dayKeyAt(date: Date, timezone: string) { const parts=new Intl.DateTimeFormat("en-CA",{timeZone:timezone,year:"numeric",month:"2-digit",day:"2-digit"}).formatToParts(date);const value=(type:string)=>parts.find(part=>part.type===type)?.value??"00";return value("year")+"-"+value("month")+"-"+value("day"); }
function nextDayKey(key:string) { const [year,month,day]=key.split("-").map(Number);const next=new Date(Date.UTC(year,month-1,day+1));return next.getUTCFullYear()+"-"+String(next.getUTCMonth()+1).padStart(2,"0")+"-"+String(next.getUTCDate()).padStart(2,"0"); }

function PairDisplay({ onClaimed }: { onClaimed: (token: string) => void }) {
  const [pairing, setPairing] = useState<Pairing | null>(null);
  const [error, setError] = useState("");
  const [renewing, setRenewing] = useState(false);
  const create = useCallback(async () => {
    setRenewing(true); setError("");
    try { const response = await fetch(apiBase + "/api/display-pairing/challenges", { method: "POST" }); if (!response.ok) throw new Error("We could not start pairing. Please try again."); setPairing(await response.json() as Pairing); }
    catch (reason) { setError(reason instanceof Error ? reason.message : "We could not start pairing. Please try again."); }
    finally { setRenewing(false); }
  }, []);
  useEffect(() => { void create(); }, [create]);
  useEffect(() => {
    if (!pairing) return;
    let stopped = false;
    async function checkClaim() {
      try {
        const response = await fetch(apiBase + "/api/display-pairing/challenges/" + pairing!.challengeId + "/claim", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ pairingCode: pairing!.pairingCode }) });
        if (response.ok) { const body = await response.json() as { token: string }; if (!stopped) onClaimed(body.token); }
        else if (response.status !== 409 && !stopped) setError("We lost the pairing connection. Please try again.");
      } catch { if (!stopped) setError("We lost the pairing connection. Please try again."); }
    }
    const timer = window.setInterval(() => { if (Date.parse(pairing.expiresAt) <= Date.now()) { setPairing(null); setError("This pairing code expired. Create a new one to continue."); } else void checkClaim(); }, 8000);
    return () => { stopped = true; window.clearInterval(timer); };
  }, [pairing, onClaimed]);
  const qrValue = pairing ? JSON.stringify({ apiBaseUrl: apiBase, challengeId: pairing.challengeId, pairingCode: pairing.pairingCode }) : "";
  return <main className="pair-screen"><section className="pair-card"><span className="brand-mark">f</span><p className="kicker">PAIR WITH YOUR HOME</p><h1>One screen for everyone.</h1><p>Open Family Hub on an iPhone or iPad, scan this code, then approve this display.</p>{pairing ? <div className="qr-wrap"><QRCodeSVG value={qrValue} size={228} level="M" includeMargin /></div> : <div className="qr-empty" role="status">{renewing ? "Preparing a new code…" : "Pairing code is unavailable"}</div>}{error && <p className="error-message" role="alert">{error}</p>}<button className="action-button" disabled={renewing} onClick={() => void create()}>{renewing ? "Preparing…" : "Create a new pairing code"}</button><small>Each code expires after five minutes and can be used once.</small></section></main>;
}

export function DisplayClient() {
  const [token, setToken] = useState("");
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const snapshotRef = useRef<Snapshot | null>(null);
  const [offline, setOffline] = useState(false);
  const [clock, setClock] = useState(new Date());
  const [selectedApp, setSelectedApp] = useState("");
  const [visibleSections, setVisibleSections] = useState<OverviewSection[]>(overviewSections.map(section => section.id));
  const [sectionsLoaded, setSectionsLoaded] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const accept = useCallback((value: string) => { localStorage.setItem(displayTokenKey, value); setToken(value); }, []);
  useEffect(() => { const saved = localStorage.getItem(displayTokenKey) ?? ""; setToken(saved); const cached = localStorage.getItem(snapshotKey); if (cached) { try { const value=JSON.parse(cached) as Snapshot; snapshotRef.current=value; setSnapshot(value); } catch { localStorage.removeItem(snapshotKey); } } }, []);
  useEffect(() => {
    try {
      const saved = JSON.parse(localStorage.getItem(displaySectionsKey) ?? "null") as unknown;
      if (Array.isArray(saved)) setVisibleSections(saved.filter((section): section is OverviewSection => overviewSections.some(option => option.id === section)));
    } catch {
      localStorage.removeItem(displaySectionsKey);
    }
    setSectionsLoaded(true);
  }, []);
  useEffect(() => { if (sectionsLoaded) localStorage.setItem(displaySectionsKey, JSON.stringify(visibleSections)); }, [sectionsLoaded, visibleSections]);
  useEffect(() => { const timer=window.setInterval(()=>setClock(new Date()),30_000);return()=>window.clearInterval(timer); }, []);
  const refresh=useCallback(async()=>{
    if(!token)return;
    const now=new Date();
    const [homeResult]=await Promise.allSettled([displayRequest<{timezone:string}>("/api/display/home-info",token)]);
    const timezone=homeResult.status==="fulfilled"?homeResult.value.timezone:snapshotRef.current?.timezone??"Europe/London";
    const calendarDate=dayKeyAt(now,timezone);
    setError("");
    const [calendarResult,weatherResult]=await Promise.allSettled([
      displayRequest<{events:DisplayEvent[];chores:DisplayChore[];meals:DisplayMeal[]}>("/api/display/calendar?view=week&date="+calendarDate,token),
      displayRequest<Weather>("/api/display/weather",token),
    ]);
    if(calendarResult.status==="fulfilled"||weatherResult.status==="fulfilled"){
      setSnapshot(current=>{
        const next:Snapshot={events:calendarResult.status==="fulfilled"?calendarResult.value.events:current?.events??[],chores:calendarResult.status==="fulfilled"?calendarResult.value.chores:current?.chores??[],meals:calendarResult.status==="fulfilled"?calendarResult.value.meals:current?.meals??[],weather:weatherResult.status==="fulfilled"?weatherResult.value:current?.weather??{status:"unavailable"},timezone:timezone,updatedAt:new Date().toISOString()};
        snapshotRef.current=next;localStorage.setItem(snapshotKey,JSON.stringify(next));return next;
      });
    }
    const hasFailure=homeResult.status==="rejected"||calendarResult.status==="rejected"||weatherResult.status==="rejected";
    const revoked=[homeResult,calendarResult,weatherResult].some(result=>result.status==="rejected"&&result.reason instanceof DisplayAccessRevoked);
    if(revoked){localStorage.removeItem(displayTokenKey);localStorage.removeItem(snapshotKey);snapshotRef.current=null;setSnapshot(null);setToken("");setOffline(false);setError("");setLoading(false);return}
    setOffline(hasFailure);if(hasFailure&&!snapshotRef.current)setError("We’re having trouble connecting. The display will keep trying.");setLoading(false);
  },[token]);
  useEffect(()=>{if(!token)return;void refresh();const timer=window.setInterval(()=>void refresh(),60_000);return()=>window.clearInterval(timer)},[token,refresh]);
  if(!token)return <PairDisplay onClaimed={accept}/>;
  if(selectedApp==="settings")return <main className="kiosk-app"><header><button className="kiosk-back" onClick={()=>setSelectedApp("")}>← Home</button><h1>Display settings</h1></header><section className="kiosk-settings"><h2>What appears here</h2><p>Choose which sections this display shows on its home screen. You can still open every family app from the launcher.</p>{overviewSections.map(section=><button className="kiosk-setting-row" key={section.id} aria-pressed={visibleSections.includes(section.id)} onClick={()=>setVisibleSections(current=>current.includes(section.id)?current.filter(item=>item!==section.id):[...current,section.id])}><span><strong>{section.title}</strong><small>{section.description}</small></span><span className={visibleSections.includes(section.id)?"setting-toggle is-on":"setting-toggle"} aria-hidden="true" /></button>)}</section></main>;
  if(selectedApp==="launcher")return <main className="kiosk-app"><header><button className="kiosk-back" onClick={()=>setSelectedApp("")}>← Home</button><h1>Family apps</h1></header><div className="launcher-grid">{["Shopping","Scratchpad","Calendar","Recipes","Meal plan","Inventory"].map((app,index)=><button className="launcher-tile" key={app} onClick={()=>setSelectedApp(app)}><span aria-hidden="true">{["＋","✎","▦","♧","♨","▤"][index]}</span>{app}</button>)}</div></main>;
  if(selectedApp){return <main className="kiosk-app"><header><button className="kiosk-back" onClick={()=>setSelectedApp("")}>← Home</button><h1>{selectedApp}</h1></header><KioskApp app={selectedApp} token={token}/></main>}
  const timezone=snapshot?.timezone??"Europe/London";
  const date=clock.toLocaleDateString("en-GB",{weekday:"long",month:"long",day:"numeric",timeZone:timezone});
  const today=dayKeyAt(clock,timezone);
  const hour=Number(new Intl.DateTimeFormat("en-GB",{timeZone:timezone,hour:"2-digit",hourCycle:"h23"}).format(clock));
  const showTomorrowMeal=hour>=18;
  const mealDate=showTomorrowMeal?nextDayKey(today):today;
  const events=(snapshot?.events??[]).filter(item=>dayKeyAt(new Date(item.StartTime),timezone)===today).sort((a,b)=>a.StartTime.localeCompare(b.StartTime));
  const chores=(snapshot?.chores??[]).filter(item=>item.Status==="pending"||item.Status==="overdue").slice(0,4);
  const meals=snapshot?.meals??[];
  const visible=new Set(visibleSections);
  return <main className="kiosk-screen"><header className="kiosk-header"><div><div className="kiosk-time">{clock.toLocaleTimeString("en-GB", {hour:"numeric",minute:"2-digit",timeZone:timezone})}</div><div className="kiosk-date">{date}</div></div><div className="kiosk-header-actions"><button className="kiosk-weather" onClick={()=>setSelectedApp("Weather")}>{snapshot?.weather.current?<><strong>{Math.round(snapshot.weather.current.temperature)}°</strong><span>{offline?"Saved forecast":"At home · tap for forecast"}</span></>:<span>{loading?"Connecting…":"Weather unavailable"}</span>}</button><button className="display-settings-button" aria-label="Display settings" onClick={()=>setSelectedApp("settings")}>⚙</button></div></header>{offline&&<div className="offline-banner" role="status">Showing the last saved family update · Trying to reconnect</div>}{error&&<div className="offline-banner" role="alert">{error}</div>}{visible.size===0?<section className="kiosk-no-sections"><h2>No overview sections selected</h2><p>Choose what this display shows in display settings.</p><button className="launcher-button" onClick={()=>setSelectedApp("settings")}>Choose sections</button></section>:<div className="kiosk-columns">{visible.has("agenda")&&<button className="kiosk-card kiosk-card-action agenda-card" onClick={()=>setSelectedApp("Calendar")} aria-label="Open Calendar"><p className="kicker">THE DAY AHEAD</p><h2>On the family calendar</h2>{events.length?events.map(item=><div className="kiosk-event" key={item.ID}><time>{new Date(item.StartTime).toLocaleTimeString([], {hour:"numeric",minute:"2-digit",timeZone:timezone})}</time><div><strong>{item.Title}</strong>{item.Location&&<small>{item.Location}</small>}</div></div>):<p className="kiosk-empty">A little space in the day.</p>}</button>}{visible.has("meals")&&<button className="kiosk-card kiosk-card-action" onClick={()=>setSelectedApp("Meal plan")} aria-label="Open meal plan"><p className="kicker">AT THE TABLE</p><h2>Meals</h2><MealPreview label={showTomorrowMeal?"Tomorrow’s preview":"Today"} date={mealDate} meals={meals}/></button>}{visible.has("chores")&&<button className="kiosk-card kiosk-card-action" onClick={()=>setSelectedApp("Calendar")} aria-label="Open Calendar for shared routines"><p className="kicker">SHARED ROUTINES</p><h2>Things to do</h2>{chores.length?chores.map(item=><div className="kiosk-chore" key={item.ID}><span className="kiosk-check"/><span>{item.Name}</span></div>):<p className="kiosk-empty">Nothing waiting right now.</p>}</button>}{visible.has("weather")&&<button className="kiosk-card kiosk-card-action forecast-card" onClick={()=>setSelectedApp("Weather")} aria-label="Open weather details"><p className="kicker">THE WEATHER</p><h2>At home</h2><div className="forecast-strip">{(snapshot?.weather.hours??[]).slice(0,6).map(hour=><div key={hour.time}><small>{new Date(hour.time).toLocaleTimeString([], {hour:"numeric"})}</small><strong>{Math.round(hour.temperature)}°</strong><span>{Math.round(hour.rainProbability)}% rain</span></div>)}</div><div className="forecast-week">{(snapshot?.weather.days??[]).slice(0,7).map(day=><div key={day.date}><small>{new Date(day.date).toLocaleDateString([], {weekday:"short"})}</small><strong>{Math.round(day.high)}°</strong><span>{Math.round(day.low)}° · {weatherLabel(day.code)}{day.rainProbability===undefined?"":" · "+Math.round(day.rainProbability)+"% rain"}</span></div>)}</div></button>}</div>}<footer className="kiosk-footer"><span>{snapshot?.updatedAt?"Updated "+new Date(snapshot.updatedAt).toLocaleTimeString([], {hour:"numeric",minute:"2-digit",timeZone:timezone}):"Family Hub"}</span><div className="kiosk-footer-actions"><button className="kiosk-settings-link" onClick={()=>setSelectedApp("settings")}>Display settings</button><button className="launcher-button" onClick={()=>setSelectedApp("launcher")}>Open family apps</button></div></footer></main>;
}

function MealPreview({label,date,meals}:{label:string;date:string;meals:DisplayMeal[]}){const items=meals.filter(item=>item.Date===date);return <div className="kiosk-meal-day"><strong>{label}</strong>{items.length?items.map(item=><div key={item.MealType}><small>{item.MealType}</small><span>{item.Name}</span></div>):<p>No meals planned</p>}</div>}

function KioskApp({app,token}:{app:string;token:string}){return app==="Scratchpad"?<KioskScratchpad token={token}/>:<KioskDataApp app={app} token={token}/>}
function KioskDataApp({app,token}:{app:string;token:string}){
  const [data,setData]=useState<unknown>(null);const [error,setError]=useState("");const [busy,setBusy]=useState("");const [week,setWeek]=useState(()=>dayKeyAt(new Date(),"Europe/London"));const [selected,setSelected]=useState("");const [mealName,setMealName]=useState("");const [recipeDetails,setRecipeDetails]=useState<Record<string,DisplayRecipe>>({});const [recipeErrors,setRecipeErrors]=useState<Record<string,string>>({});const [newMealDate,setNewMealDate]=useState(()=>dayKeyAt(new Date(),"Europe/London"));const [newMealType,setNewMealType]=useState("dinner");const [newMealName,setNewMealName]=useState("");
  const path=app==="Shopping"?"/api/display/shopping":app==="Calendar"?"/api/display/calendar?view=week&date="+week:app==="Recipes"?"/api/display/recipes":app==="Meal plan"?"/api/display/meals":app==="Inventory"?"/api/display/inventory":"/api/display/weather";
  const load=useCallback(async()=>{setError("");try{setData(await displayRequest<unknown>(path,token))}catch{setError("This family information is unavailable right now.")}},[path,token]);useEffect(()=>{void load()},[load]);
  async function updateShopping(item:{id:string;name:string;purchased:boolean;version:number}){setBusy(item.id);setError("");try{await fetch(apiBase+"/api/display/shopping/items/"+item.id,{method:"PATCH",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({purchased:!item.purchased,version:item.version})}).then(async response=>{if(!response.ok)throw new Error();return response.json()});await load()}catch{setError("We couldn’t update the shopping list. Please try again.")}finally{setBusy("")}}
  async function addMeal(event:FormEvent<HTMLFormElement>){event.preventDefault();const name=newMealName.trim();if(!name)return;setBusy("new-meal");setError("");try{const response=await fetch(apiBase+"/api/display/meals",{method:"POST",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({date:newMealDate,mealType:newMealType,name})});if(!response.ok)throw new Error();setNewMealName("");await load()}catch{setError("We couldn’t save that meal. Please try again.")}finally{setBusy("")}}
  async function loadRecipe(id:string){try{const recipe=await displayRequest<DisplayRecipe>("/api/display/recipes/"+id,token);setRecipeDetails(current=>({...current,[id]:recipe}))}catch{setRecipeErrors(current=>({...current,[id]:"This recipe is unavailable right now."}))}}
  async function saveMeal(event:FormEvent<HTMLFormElement>,meal:DisplayMeal){event.preventDefault();const name=mealName.trim();if(!name)return;setBusy(meal.Date+meal.MealType);setError("");try{const response=await fetch(apiBase+"/api/display/meals",{method:"POST",headers:{Authorization:"Bearer "+token,"Content-Type":"application/json"},body:JSON.stringify({date:meal.Date,mealType:meal.MealType,name})});if(!response.ok)throw new Error();setSelected("");setMealName("");await load()}catch{setError("We couldn’t save that meal. Please try again.")}finally{setBusy("")}}
  if(error&&!data)return <p role="alert">{error}</p>;if(!data)return <p role="status">Loading…</p>;
  if(app==="Shopping")return <div className="kiosk-list">{error&&<p role="alert">{error}</p>}{(data as {id:string;name:string;purchased:boolean;version:number}[]).map(item=><label className="shopping-check" key={item.id}><input type="checkbox" checked={item.purchased} disabled={busy===item.id} onChange={()=>void updateShopping(item)}/><span>{item.name}</span></label>)}</div>;
  if(app==="Inventory")return <div className="kiosk-list">{(data as {ID:string;Name:string;Items:{ID:string;Name:string;Quantity:number;Unit:string}[]}[]).map(area=><section key={area.ID}><h2>{area.Name}</h2>{area.Items.map(item=><p key={item.ID}>{item.Name} · {item.Quantity} {item.Unit}</p>)}</section>)}</div>;
  if(app==="Recipes")return <div className="kiosk-list">{(data as {ID:string;Title:string}[]).map(recipe=>{const detail=recipeDetails[recipe.ID];return <details key={recipe.ID} onToggle={event=>{if(event.currentTarget.open&&!detail&&!recipeErrors[recipe.ID])void loadRecipe(recipe.ID)}}><summary>{recipe.Title}</summary>{recipeErrors[recipe.ID]?<><p role="alert">{recipeErrors[recipe.ID]}</p><button className="kiosk-back" onClick={()=>{setRecipeErrors(current=>({...current,[recipe.ID]:""}));void loadRecipe(recipe.ID)}}>Try again</button></>:!detail?<p role="status">Loading recipe…</p>:<><h3>Ingredients</h3>{(detail.Ingredients??[]).flatMap(group=>group.items).map((ingredient,index)=><p key={index}>{ingredient}</p>)}<h3>Method</h3>{(detail.Steps??(detail.Instructions?[detail.Instructions]:[])).map((step,index)=><p key={index}>{index+1}. {step}</p>)}</>}</details>})}</div>;
  if(app==="Calendar"){const calendar=data as {events:DisplayEvent[];chores:DisplayChore[]};return <div className="kiosk-list"><div className="calendar-toolbar"><button className="icon-action" aria-label="Previous week" onClick={()=>setWeek(dayKeyAt(new Date(Date.parse(week+"T12:00:00Z")-7*86400000),"UTC"))}>←</button><strong>Week of {week}</strong><button className="icon-action" aria-label="Next week" onClick={()=>setWeek(dayKeyAt(new Date(Date.parse(week+"T12:00:00Z")+7*86400000),"UTC"))}>→</button><button className="text-action" onClick={()=>setWeek(dayKeyAt(new Date(),"Europe/London"))}>Today</button></div>{calendar.events.map(event=><details key={event.ID}><summary>{new Date(event.StartTime).toLocaleString()} · {event.Title}</summary>{event.Location&&<p>{event.Location}</p>}</details>)}{calendar.chores.map(chore=><p key={chore.ID}>Chore · {chore.Name}</p>)}</div>}
  if(app==="Weather"){const weather=data as Weather;return <div className="kiosk-list"><h2>{weather.current?.temperature}° at home</h2><div className="forecast-week">{weather.days?.map(day=><div key={day.date}><small>{new Date(day.date).toLocaleDateString([], {weekday:"long"})}</small><strong>{Math.round(day.high)}°</strong><span>{Math.round(day.low)}° · {weatherLabel(day.code)}{day.rainProbability===undefined?"":" · "+Math.round(day.rainProbability)+"% rain"}</span></div>)}</div></div>}
  return <div className="kiosk-list"><form className="inline-form" onSubmit={event=>void addMeal(event)}><label>Date<input aria-label="Meal date" type="date" value={newMealDate} onChange={event=>setNewMealDate(event.target.value)}/></label><label>Meal<select aria-label="Meal type" value={newMealType} onChange={event=>setNewMealType(event.target.value)}><option value="breakfast">Breakfast</option><option value="lunch">Lunch</option><option value="dinner">Dinner</option></select></label><label>Meal idea<input aria-label="Meal idea" value={newMealName} onChange={event=>setNewMealName(event.target.value)} placeholder="Add a meal idea"/></label><button className="action-button" disabled={!newMealName.trim()||busy==="new-meal"}>{busy==="new-meal"?"Saving…":"Add meal"}</button></form>{error&&<p role="alert">{error}</p>}{(data as DisplayMeal[]).map(meal=><section key={meal.Date+meal.MealType}><p>{meal.Date} · {meal.MealType}</p>{selected===meal.Date+meal.MealType?<form className="inline-form" onSubmit={event=>void saveMeal(event,meal)}><input aria-label="Meal name" value={mealName} onChange={event=>setMealName(event.target.value)} /><button className="action-button" disabled={busy===selected}>Save</button><button type="button" className="text-action" onClick={()=>setSelected("")}>Cancel</button></form>:<><strong>{meal.Name}</strong><button className="text-action" onClick={()=>{setSelected(meal.Date+meal.MealType);setMealName(meal.Name)}}>Edit</button></>}</section>)}</div>
}
