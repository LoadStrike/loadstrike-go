package loadstrike

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type htmlTab struct {
	ID    string
	Title string
	HTML  string
}

type reportTable struct {
	Headers []string
	Rows    []map[string]any
}

type htmlChartData struct {
	OverallOutcome       []chartPoint      `json:"overallOutcome"`
	ScenarioRequests     []chartPoint      `json:"scenarioRequests"`
	ScenarioP95Latency   []chartPoint      `json:"scenarioP95Latency"`
	ScenarioRps          []chartPoint      `json:"scenarioRps"`
	ScenarioFailRate     []chartPoint      `json:"scenarioFailRate"`
	ScenarioBytes        []chartPoint      `json:"scenarioBytes"`
	StatusCodeClasses    []chartPoint      `json:"statusCodeClasses"`
	ScenarioLatencyTrend latencyTrendChart `json:"scenarioLatencyTrend"`
}

type latencyTrendChart struct {
	Labels []string             `json:"labels"`
	Series []latencyTrendSeries `json:"series"`
}

type latencyTrendSeries struct {
	Name   string    `json:"name"`
	Color  string    `json:"color"`
	Values []float64 `json:"values"`
}

type chartPoint struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
	Color string  `json:"color"`
}

type groupedCorrelationValueChart struct {
	GatherByField string
	GatherByValue string
	SourceRows    int
	Chart         latencyTrendChart
}

type groupedCorrelationChartRow struct {
	GroupLabel    string
	GatherByField string
	GatherByValue string
	LatencyP50Ms  *float64
	LatencyP80Ms  *float64
	LatencyP85Ms  *float64
	LatencyP90Ms  *float64
	LatencyP95Ms  *float64
	LatencyP99Ms  *float64
}

var (
	reportLogosOnce        sync.Once
	reportLogoDarkDataURI  string
	reportLogoLightDataURI string
)

const reportHTMLCSS = `:root{--bg:#f4f7fc;--bgTop:#fff3d0;--bgRight:#dbe8ff;--bgBottom:#edf2fb;--panel:#ffffff;--panelAlt:#eef3fb;--line:#d3deed;--text:#0f172a;--muted:#4b5563;--accent:#2563eb;--ok:#138a4a;--fail:#c63636;--warn:#b7791f;--chip:#edf3fc;--card:#ffffff;--stat:#f5f9ff;--chartPanel:#0f172a;--chartGrid:#334155;--chartAxis:#b5c2d3;--chartLabel:#dbe6f4;--chartDot:#0f172a;--chartPieCenter:#0f172a;--chartPieCenterText:#e5eefc;--chartLegendText:#e6edf3;--shadow:0 10px 24px rgba(15,23,42,.1)}
body[data-theme='dark']{--bg:#0d1117;--bgTop:#18243b;--bgRight:#102235;--bgBottom:#0b0f14;--panel:#111827;--panelAlt:#0f172a;--line:#263241;--text:#e6edf3;--muted:#9fb0c3;--accent:#2563eb;--ok:#18a957;--fail:#d14343;--warn:#f59e0b;--chip:rgba(31,41,55,.45);--card:linear-gradient(180deg,rgba(17,24,39,.92) 0,rgba(13,19,32,.92) 100%);--stat:rgba(20,30,47,.65);--chartPanel:rgba(15,23,42,.72);--chartGrid:#334155;--chartAxis:#b5c2d3;--chartLabel:#dbe6f4;--chartDot:#0f172a;--chartPieCenter:#0f172a;--chartPieCenterText:#e5eefc;--chartLegendText:#e6edf3;--shadow:0 10px 24px rgba(0,0,0,.2)}
body{margin:0;background:radial-gradient(1200px 700px at 10% -10%,var(--bgTop) 0,transparent 58%),radial-gradient(900px 620px at 100% 0,var(--bgRight) 0,transparent 57%),var(--bgBottom);color:var(--text);font-family:'Segoe UI',Tahoma,sans-serif}
.wrap{max-width:1440px;margin:0 auto;padding:20px}
.report-brand{display:flex;align-items:center;gap:16px;margin:0 0 10px 0;overflow:visible;padding-top:8px;flex-wrap:wrap}
.report-logo-slot{width:380px;height:184px;max-width:100%;flex:0 0 auto;display:flex;align-items:flex-start;justify-content:flex-start;overflow:visible}
.report-logo{width:100%;height:100%;object-fit:contain;object-position:left top;border:none;background:transparent;box-shadow:none;border-radius:0;padding:0;margin:0;overflow:visible;transform:scale(var(--report-logo-scale,1));transform-origin:left top;transition:transform .2s ease}
body[data-theme='dark'] .report-logo{--report-logo-scale:1.175}
.theme-toggle-report{position:fixed;top:12px;right:12px;z-index:1200;display:inline-flex;align-items:center;justify-content:center;width:40px;height:40px;appearance:none;border:1px solid var(--line);background:var(--chip);color:var(--text);padding:0;border-radius:999px;font-size:18px;font-weight:700;cursor:pointer;transition:all .2s ease}
.theme-toggle-report:hover{border-color:var(--accent);transform:translateY(-1px)}
.theme-toggle-report span{line-height:1;pointer-events:none}
h1{margin:0 0 10px 0;font-size:28px;letter-spacing:.2px}
h2{margin:0 0 12px 0;font-size:18px}
h3{margin:0 0 10px 0;font-size:15px;color:var(--text)}
.meta{display:flex;gap:14px;flex-wrap:wrap;color:var(--muted);font-size:13px;margin-bottom:16px}
.meta span{background:var(--chip);border:1px solid var(--line);padding:6px 10px;border-radius:999px}
.report-layout{display:grid;grid-template-columns:280px minmax(0,1fr);gap:14px;align-items:start}
.tabs-pane{position:sticky;top:12px;max-height:calc(100vh - 24px);overflow:auto;overscroll-behavior:contain;padding-right:4px;cursor:grab}
.tabs-pane.panning{cursor:grabbing;user-select:none}
.tabs{display:flex;flex-direction:column;gap:8px;margin:12px 0 14px 0}
.tab-btn{background:var(--panelAlt);color:var(--text);border:1px solid var(--line);padding:8px 12px;border-radius:8px;cursor:pointer;transition:all .2s ease;text-align:left;width:100%}
.tab-btn:hover{border-color:var(--accent);background:var(--chip)}
.tab-btn.active{background:linear-gradient(180deg,#2f66db 0,#2754b8 100%);border-color:#3f73e0}
.tabs-content{min-width:0}
.tab{display:none}
.tab.active{display:block}
table{width:100%;border-collapse:collapse;font-size:12px}
th,td{border:1px solid var(--line);padding:7px 8px;text-align:left;vertical-align:top}
th{background:var(--panelAlt);position:sticky;top:0;z-index:1}
.card{background:var(--card);border:1px solid var(--line);border-radius:12px;padding:14px;margin-bottom:14px;box-shadow:var(--shadow)}
.card-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px;margin-bottom:14px}
.stat-card{padding:10px 12px;border-radius:10px;border:1px solid var(--line);background:var(--stat)}
.stat-label{font-size:12px;color:var(--muted)}
.stat-value{font-size:22px;font-weight:700;margin-top:4px}
.value-ok{color:var(--ok)}
.value-fail{color:var(--fail)}
.chart-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(360px,1fr));gap:12px}
.chart-card{padding:12px;border:1px solid var(--line);border-radius:10px;background:var(--chartPanel)}
.chart-canvas{width:100%;height:260px;display:block}
.correlation-chart-grid{grid-template-columns:repeat(auto-fit,minmax(420px,720px));justify-content:center}
.correlation-chart-card{max-width:720px;width:100%}
.correlation-chart-card .chart-canvas{height:320px}
.chart-card h3,.chart-card p{color:var(--chartLegendText)}
.table-wrap{overflow:auto;max-height:70vh}
@media (max-width:980px){.wrap{padding:14px}.report-brand{margin-bottom:8px;padding-top:4px}.report-logo-slot{width:300px;height:146px}.theme-toggle-report{top:10px;right:10px}.report-layout{grid-template-columns:1fr}.tabs-pane{position:static;max-height:none;cursor:auto}.tabs{flex-direction:row;flex-wrap:wrap}.tab-btn{width:auto}.chart-canvas{height:220px}.correlation-chart-grid{grid-template-columns:1fr;justify-content:stretch}.correlation-chart-card{max-width:none}.correlation-chart-card .chart-canvas{height:280px}.stat-value{font-size:18px}}`

const reportHTMLScriptTemplate = `const reportCharts=__REPORT_CHARTS__;
const btns=[...document.querySelectorAll('.tab-btn')];
const tabSections=[...document.querySelectorAll('.tab')];
const tabsPane=document.getElementById('tab-pane');
const linePalette=['#38bdf8','#22c55e','#f59e0b','#a855f7','#f43f5e','#14b8a6','#eab308','#818cf8','#06b6d4','#84cc16'];
const reportThemeKey='loadstrike-report-theme';
const reportThemeToggle=document.querySelector('[data-report-theme-toggle]');
const reportLogo=document.querySelector('[data-report-logo]');
function applyReportTheme(theme){const normalized=theme==='dark'?'dark':'light';document.body.setAttribute('data-theme',normalized);if(reportLogo){const lightLogo=reportLogo.dataset.logoLight||reportLogo.getAttribute('src');const darkLogo=reportLogo.dataset.logoDark||reportLogo.getAttribute('src');reportLogo.setAttribute('src',normalized==='dark'?darkLogo:lightLogo);}if(reportThemeToggle){const darkActive=normalized==='dark';const nextLabel=darkActive?'light':'dark';reportThemeToggle.innerHTML=darkActive?'&#x2600;':'&#x263E;';reportThemeToggle.setAttribute('aria-pressed',darkActive?'true':'false');reportThemeToggle.setAttribute('aria-label','Switch to '+nextLabel+' theme');reportThemeToggle.setAttribute('title','Switch to '+nextLabel+' theme');}}
function show(id){btns.forEach(b=>b.classList.toggle('active',b.dataset.tab===id));tabSections.forEach(t=>t.classList.toggle('active',t.id===id));}
function formatMetric(v){if(!Number.isFinite(v))return '0';return Math.abs(v)>=100?v.toFixed(0):v.toFixed(2);}
function setupCanvas(canvas){const dpr=window.devicePixelRatio||1;const w=Math.max(320,canvas.clientWidth||320);const h=Math.max(220,canvas.clientHeight||220);canvas.width=Math.floor(w*dpr);canvas.height=Math.floor(h*dpr);const ctx=canvas.getContext('2d');ctx.setTransform(dpr,0,0,dpr,0,0);return {ctx,w,h};}
function drawNoData(ctx,w,h,msg){ctx.fillStyle='#9fb0c3';ctx.font='13px Segoe UI';ctx.textAlign='center';ctx.fillText(msg,w/2,h/2);}
function drawBar(canvasId,points){const canvas=document.getElementById(canvasId);if(!canvas)return;const c=setupCanvas(canvas);const ctx=c.ctx,w=c.w,h=c.h;ctx.clearRect(0,0,w,h);if(!points||points.length===0){drawNoData(ctx,w,h,'No data');return;}const left=46,right=14,top=16,bottom=62;const pw=w-left-right;const ph=h-top-bottom;const max=Math.max(...points.map(p=>p.value),1);ctx.strokeStyle='#334155';ctx.lineWidth=1;for(let i=0;i<=4;i++){const y=top+(ph*(i/4));ctx.beginPath();ctx.moveTo(left,y);ctx.lineTo(w-right,y);ctx.stroke();}const slot=pw/points.length;const bar=Math.max(8,slot*0.58);ctx.font='11px Segoe UI';for(let i=0;i<points.length;i++){const p=points[i];const x=left+i*slot+(slot-bar)/2;const bh=(p.value/max)*ph;const y=top+ph-bh;ctx.fillStyle=p.color||'#3b82f6';ctx.fillRect(x,y,bar,bh);ctx.fillStyle='#dbe6f4';ctx.textAlign='center';ctx.fillText(formatMetric(p.value),x+bar/2,Math.max(12,y-4));ctx.save();ctx.translate(x+bar/2,h-bottom+14);ctx.rotate(-0.6);ctx.fillStyle='#b5c2d3';ctx.fillText((p.label||'').slice(0,26),0,0);ctx.restore();}ctx.fillStyle='#b5c2d3';ctx.textAlign='right';for(let i=0;i<=4;i++){const value=max*(1-i/4);const y=top+(ph*(i/4))+4;ctx.fillText(formatMetric(value),left-6,y);}}
function drawPie(canvasId,points){const canvas=document.getElementById(canvasId);if(!canvas)return;const c=setupCanvas(canvas);const ctx=c.ctx,w=c.w,h=c.h;ctx.clearRect(0,0,w,h);if(!points||points.length===0){drawNoData(ctx,w,h,'No data');return;}const total=points.reduce((s,p)=>s+(p.value||0),0);if(total<=0){drawNoData(ctx,w,h,'No data');return;}const cx=w*0.35,cy=h*0.5,r=Math.min(w,h)*0.28;let angle=-Math.PI/2;for(const p of points){const val=Math.max(0,p.value||0);const delta=(val/total)*Math.PI*2;ctx.beginPath();ctx.moveTo(cx,cy);ctx.arc(cx,cy,r,angle,angle+delta);ctx.closePath();ctx.fillStyle=p.color||'#3b82f6';ctx.fill();angle+=delta;}ctx.fillStyle='#0f172a';ctx.beginPath();ctx.arc(cx,cy,r*0.54,0,Math.PI*2);ctx.fill();ctx.fillStyle='#e5eefc';ctx.font='bold 18px Segoe UI';ctx.textAlign='center';ctx.fillText(total.toString(),cx,cy+6);ctx.font='12px Segoe UI';ctx.fillStyle='#9fb0c3';ctx.fillText('requests',cx,cy+24);ctx.textAlign='left';let y=cy-r+10;for(const p of points){ctx.fillStyle=p.color||'#3b82f6';ctx.fillRect(w*0.64,y-10,12,12);ctx.fillStyle='#e6edf3';ctx.font='12px Segoe UI';const pct=total<=0?0:((p.value/total)*100);ctx.fillText(__BT__${p.label}: ${p.value} (${pct.toFixed(1)}%)__BT__,w*0.64+18,y);y+=20;}}
function drawLatencyLine(canvasRef,chart){const canvas=typeof canvasRef==='string'?document.getElementById(canvasRef):canvasRef;if(!canvas)return;const c=setupCanvas(canvas);const ctx=c.ctx,w=c.w,h=c.h;ctx.clearRect(0,0,w,h);if(!chart||!Array.isArray(chart.labels)||chart.labels.length===0||!Array.isArray(chart.series)||chart.series.length===0){drawNoData(ctx,w,h,'No latency data');return;}const labels=chart.labels;const seriesList=chart.series;const allValues=seriesList.flatMap(s=>(s.values||[]).filter(v=>Number.isFinite(v)));if(allValues.length===0){drawNoData(ctx,w,h,'No latency data');return;}const shortAxisLabels=labels.every(label=>((label||'').toString().length<=4));const rotateAxisLabels=!shortAxisLabels&&labels.length>4;const left=52,right=18,bottom=rotateAxisLabels?78:52;const legendItemWidth=150,legendLineHeight=14,legendTop=12;const plotWidth=Math.max(120,w-left-right);const legendCols=Math.max(1,Math.floor(plotWidth/legendItemWidth));const legendRows=Math.max(1,Math.ceil(seriesList.length/legendCols));const legendHeight=legendRows*legendLineHeight;const top=legendTop+legendHeight+16;const pw=w-left-right;const ph=h-top-bottom;if(ph<=20){drawNoData(ctx,w,h,'No latency data');return;}const max=Math.max(...allValues,1);const scaleMax=max*1.08;ctx.strokeStyle='#334155';ctx.lineWidth=1;for(let i=0;i<=4;i++){const y=top+(ph*(i/4));ctx.beginPath();ctx.moveTo(left,y);ctx.lineTo(w-right,y);ctx.stroke();}ctx.fillStyle='#b5c2d3';ctx.font='11px Segoe UI';ctx.textAlign='right';for(let i=0;i<=4;i++){const value=max*(1-i/4);const y=top+(ph*(i/4))+4;ctx.fillText(formatMetric(value),left-6,y);}const xStep=labels.length<=1?0:pw/(labels.length-1);const xAt=i=>labels.length<=1?left+(pw/2):left+(xStep*i);const overlapOffset=new Map();const epsilon=Math.max(max*0.0005,0.001);for(let pointIndex=0;pointIndex<labels.length;pointIndex++){const points=[];for(let seriesIndex=0;seriesIndex<seriesList.length;seriesIndex++){const value=(seriesList[seriesIndex].values||[])[pointIndex];if(Number.isFinite(value)){points.push({seriesIndex,value});}}points.sort((a,b)=>a.value===b.value?a.seriesIndex-b.seriesIndex:a.value-b.value);let start=0;while(start<points.length){let end=start+1;while(end<points.length&&Math.abs(points[end].value-points[start].value)<=epsilon){end++;}const count=end-start;if(count>1){const mid=(count-1)/2;for(let k=0;k<count;k++){overlapOffset.set(points[start+k].seriesIndex+'|'+pointIndex,(k-mid)*3);}}start=end;}}const dashPatterns=[[0,0],[7,4],[2,3],[10,3,2,3]];for(let seriesIndex=0;seriesIndex<seriesList.length;seriesIndex++){const series=seriesList[seriesIndex];ctx.beginPath();ctx.strokeStyle=series.color||'#38bdf8';ctx.lineWidth=2.2;const dash=dashPatterns[seriesIndex%dashPatterns.length];if(dash[0]===0){ctx.setLineDash([]);}else{ctx.setLineDash(dash);}let started=false;(series.values||[]).forEach((value,index)=>{if(!Number.isFinite(value)){started=false;return;}const x=xAt(index);const offset=overlapOffset.get(seriesIndex+'|'+index)||0;const y=top+ph-((value/scaleMax)*ph)+offset;if(!started){ctx.moveTo(x,y);started=true;}else{ctx.lineTo(x,y);}});ctx.stroke();ctx.setLineDash([]);(series.values||[]).forEach((value,index)=>{if(!Number.isFinite(value))return;const x=xAt(index);const offset=overlapOffset.get(seriesIndex+'|'+index)||0;const y=top+ph-((value/scaleMax)*ph)+offset;ctx.beginPath();ctx.fillStyle='#0f172a';ctx.arc(x,y,3.2,0,Math.PI*2);ctx.fill();ctx.beginPath();ctx.strokeStyle=series.color||'#38bdf8';ctx.lineWidth=2;ctx.arc(x,y,3.2,0,Math.PI*2);ctx.stroke();});}ctx.fillStyle='#b5c2d3';ctx.font='10px Segoe UI';ctx.textAlign='center';labels.forEach((label,index)=>{const x=xAt(index);const text=(label||'').toString().slice(0,34);if(rotateAxisLabels){ctx.save();ctx.translate(x,h-bottom+12);ctx.rotate(-0.6);ctx.fillText(text,0,0);ctx.restore();}else{ctx.fillText(text,x,h-bottom+16);}});let lx=left,ly=legendTop+2;for(let seriesIndex=0;seriesIndex<seriesList.length;seriesIndex++){const series=seriesList[seriesIndex];const rawName=(series.name||'Series').toString();const legendName=rawName.length>30?rawName.slice(0,27)+'...':rawName;ctx.strokeStyle=series.color||'#38bdf8';ctx.lineWidth=2.2;const dash=dashPatterns[seriesIndex%dashPatterns.length];if(dash[0]===0){ctx.setLineDash([]);}else{ctx.setLineDash(dash);}ctx.beginPath();ctx.moveTo(lx,ly+1.5);ctx.lineTo(lx+12,ly+1.5);ctx.stroke();ctx.setLineDash([]);ctx.fillStyle='#b5c2d3';ctx.font='11px Segoe UI';ctx.textAlign='left';ctx.fillText(legendName,lx+16,ly+4);lx+=legendItemWidth;if(lx>w-right-legendItemWidth){lx=left;ly+=legendLineHeight;}}}
function renderCharts(){drawPie('chart-outcome',reportCharts.overallOutcome);drawBar('chart-scenario-requests',reportCharts.scenarioRequests);drawBar('chart-scenario-p95',reportCharts.scenarioP95Latency);drawBar('chart-scenario-rps',reportCharts.scenarioRps);drawBar('chart-scenario-fail-rate',reportCharts.scenarioFailRate);drawBar('chart-scenario-bytes',reportCharts.scenarioBytes);drawPie('chart-status-code-classes',reportCharts.statusCodeClasses);drawLatencyLine('chart-scenario-latency-lines',reportCharts.scenarioLatencyTrend);}
function getGroupedCorrelationChart(key){const node=document.querySelector('script[type="application/json"][data-grouped-correlation-chart="'+key+'"]');if(!node)return {labels:[],series:[]};try{return JSON.parse(node.textContent||'{"labels":[],"series":[]}');}catch{return {labels:[],series:[]};}}
function getUngroupedCorrelationChart(key){const node=document.querySelector('script[type="application/json"][data-ungrouped-correlation="'+key+'"]');if(!node)return {labels:[],series:[]};try{return JSON.parse(node.textContent||'{"labels":[],"series":[]}');}catch{return {labels:[],series:[]};}}
function normalizeUngroupedCorrelationChart(chart){if(!chart||!Array.isArray(chart.labels)||!Array.isArray(chart.series))return {labels:[],series:[]};const compactName=input=>{const raw=(input||'Series').toString();const parts=raw.split('|').map(x=>x.trim()).filter(x=>x.length>0);const tail=parts.length>0?parts[parts.length-1]:raw.trim();if(tail.length===0)return 'Series';return tail.length>22?tail.slice(0,19)+'...':tail;};const labels=chart.labels||[];const series=chart.series||[];const normalizedSeries=series.map((item,index)=>({name:compactName(item&&item.name),color:(item&&item.color)||linePalette[index%linePalette.length],values:Array.isArray(item&&item.values)?item.values.map(v=>Number.isFinite(v)?v:NaN):[]})).filter(s=>s.values.some(v=>Number.isFinite(v)));const isPercentile=value=>/^p\d+$/i.test((value||'').toString().trim());const labelsArePercentiles=labels.length>0&&labels.every(isPercentile);if(labelsArePercentiles){return {labels,series:normalizedSeries};}const seriesArePercentiles=normalizedSeries.length>0&&normalizedSeries.every(s=>isPercentile(s.name));if(!seriesArePercentiles||labels.length===0){return {labels,series:normalizedSeries};}const percentileLabels=normalizedSeries.map(s=>s.name.toUpperCase());const reshaped=labels.map((label,labelIndex)=>{const values=normalizedSeries.map(s=>{const value=s.values[labelIndex];return Number.isFinite(value)?value:NaN;});return {name:compactName(label||('Series '+(labelIndex+1))),color:linePalette[labelIndex%linePalette.length],values};}).filter(s=>s.values.some(v=>Number.isFinite(v)));if(reshaped.length===0){return {labels:percentileLabels,series:[]};}return {labels:percentileLabels,series:reshaped};}
function renderGroupedCorrelationCharts(){const canvases=[...document.querySelectorAll('.grouped-correlation-canvas')];if(canvases.length===0)return;canvases.forEach(canvas=>{const key=canvas.dataset.groupedKey||'';const chart=getGroupedCorrelationChart(key);drawLatencyLine(canvas,chart);});}
function renderUngroupedCorrelationCharts(){const canvases=[...document.querySelectorAll('.ungrouped-correlation-canvas')];if(canvases.length===0)return;canvases.forEach(canvas=>{const key=canvas.dataset.ungroupedKey||'';const rawChart=getUngroupedCorrelationChart(key);const chart=normalizeUngroupedCorrelationChart(rawChart);drawLatencyLine(canvas,chart);});}
function initPanePan(){if(!tabsPane)return;let activePointerId=null;let startY=0;let startScroll=0;tabsPane.addEventListener('pointerdown',event=>{if(event.button!==0)return;if(event.target&&event.target.closest&&event.target.closest('button,a,input,textarea,select,label'))return;activePointerId=event.pointerId;startY=event.clientY;startScroll=tabsPane.scrollTop;tabsPane.classList.add('panning');tabsPane.setPointerCapture(event.pointerId);});tabsPane.addEventListener('pointermove',event=>{if(activePointerId!==event.pointerId)return;const delta=event.clientY-startY;tabsPane.scrollTop=startScroll-delta;});const stopPan=event=>{if(activePointerId!==event.pointerId)return;activePointerId=null;tabsPane.classList.remove('panning');};tabsPane.addEventListener('pointerup',stopPan);tabsPane.addEventListener('pointercancel',stopPan);tabsPane.addEventListener('lostpointercapture',()=>{activePointerId=null;tabsPane.classList.remove('panning');});}
function renderAllCharts(){renderCharts();renderUngroupedCorrelationCharts();renderGroupedCorrelationCharts();}
const storedReportTheme=(()=>{try{return localStorage.getItem(reportThemeKey);}catch{return null;}})();
applyReportTheme(storedReportTheme==='dark'?'dark':'light');
if(reportThemeToggle){reportThemeToggle.addEventListener('click',()=>{const next=document.body.getAttribute('data-theme')==='dark'?'light':'dark';applyReportTheme(next);try{localStorage.setItem(reportThemeKey,next);}catch{}renderAllCharts();});}
btns.forEach(b=>b.addEventListener('click',()=>show(b.dataset.tab)));
if(btns.length>0){show(btns[0].dataset.tab);}renderAllCharts();initPanePan();window.addEventListener('resize',renderAllCharts);`

func renderDotnetHTMLReport(nodeStats map[string]any) string {
	tabs := buildDotnetHTMLTabs(nodeStats)
	logoDarkDataURI := getReportLogoDarkDataURI()
	logoLightDataURI := getReportLogoLightDataURI()
	chartDataJSON := escapeJSONForHTMLScript(marshalJSONNoEscape(buildHTMLChartData(nodeStats)))
	testInfo := reportObject(nodeStats, "testInfo", "testInfo")

	var builder strings.Builder
	appendLine(&builder, "<!doctype html>")
	appendLine(&builder, "<html lang=\"en\">")
	appendLine(&builder, "<head>")
	appendLine(&builder, "<meta charset=\"utf-8\" />")
	appendLine(&builder, "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />")
	appendLine(&builder, "<title>LoadStrike Report</title>")
	appendLine(&builder, "<style>")
	appendBlock(&builder, reportHTMLCSS)
	appendLine(&builder, "</style>")
	appendLine(&builder, "</head>")
	appendLine(&builder, "<body data-theme=\"light\">")
	appendLine(&builder, "<div class=\"wrap\">")
	appendLine(&builder, "<button type=\"button\" class=\"theme-toggle-report\" data-report-theme-toggle aria-label=\"Switch to dark theme\" aria-pressed=\"false\" title=\"Switch to dark theme\"><span aria-hidden=\"true\">&#x263E;</span></button>")
	appendLine(&builder, "<div class=\"report-brand\">")
	appendLine(&builder, "<div class=\"report-logo-slot\">")
	appendLine(&builder, fmt.Sprintf("<img src=\"%s\" alt=\"LoadStrike logo\" class=\"report-logo\" data-report-logo data-logo-light=\"%s\" data-logo-dark=\"%s\" />", escapeHTML(logoLightDataURI), escapeHTML(logoLightDataURI), escapeHTML(logoDarkDataURI)))
	appendLine(&builder, "</div>")
	appendLine(&builder, "<h1>LoadStrike Report</h1>")
	appendLine(&builder, "</div>")
	appendLine(&builder, "<div class=\"meta\">")
	appendLine(&builder, fmt.Sprintf("<span>Suite: <strong>%s</strong></span>", escapeHTML(asString(reportValue(testInfo, "testSuite", "TestSuite")))))
	appendLine(&builder, fmt.Sprintf("<span>Name: <strong>%s</strong></span>", escapeHTML(asString(reportValue(testInfo, "testName", "TestName")))))
	appendLine(&builder, fmt.Sprintf("<span>Session: <strong>%s</strong></span>", escapeHTML(asString(reportValue(testInfo, "sessionId", "SessionId")))))
	appendLine(&builder, fmt.Sprintf("<span>Duration: <strong>%s</strong></span>", escapeHTML(formatDotnetTimeSpan(reportValue(nodeStats, "duration", "Duration", "durationMs", "DurationMs")))))
	appendLine(&builder, "</div>")
	appendLine(&builder, "<div class=\"report-layout\">")
	appendLine(&builder, "<aside class=\"tabs-pane\" id=\"tab-pane\">")
	appendLine(&builder, "<div class=\"tabs\">")
	for _, tab := range tabs {
		appendLine(&builder, fmt.Sprintf("<button class=\"tab-btn\" data-tab=\"%s\">%s</button>", tab.ID, escapeHTML(tab.Title)))
	}
	appendLine(&builder, "</div>")
	appendLine(&builder, "</aside>")
	appendLine(&builder, "<div class=\"tabs-content\">")
	for _, tab := range tabs {
		appendLine(&builder, fmt.Sprintf("<section id=\"%s\" class=\"tab\">%s</section>", tab.ID, tab.HTML))
	}
	appendLine(&builder, "</div>")
	appendLine(&builder, "</div>")
	appendLine(&builder, "</div>")
	appendLine(&builder, "<script>")
	scriptBody := strings.Replace(reportHTMLScriptTemplate, "__REPORT_CHARTS__", chartDataJSON, 1)
	scriptBody = strings.ReplaceAll(scriptBody, "__BT__", "`")
	appendBlock(&builder, scriptBody)
	appendLine(&builder, "</script>")
	appendLine(&builder, "</body>")
	appendLine(&builder, "</html>")
	return normalizeReportText(builder.String())
}

func buildDotnetHTMLTabs(nodeStats map[string]any) []htmlTab {
	tabs := []htmlTab{
		{ID: "summary", Title: "Summary", HTML: buildSummaryHTML(nodeStats)},
	}

	scenarioRows := buildScenarioRows(nodeStats)
	if len(scenarioRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "scenarios", Title: "Scenarios", HTML: buildTableHTML(scenarioRows, true)})
	}

	scenarioMeasurementRows := buildScenarioMeasurementRows(nodeStats)
	if len(scenarioMeasurementRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "scenario-measurements", Title: "Scenario Measurements", HTML: buildTableHTML(scenarioMeasurementRows, true)})
	}

	stepRows := buildStepRows(nodeStats)
	if len(stepRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "steps", Title: "Steps", HTML: buildTableHTML(stepRows, true)})
	}

	stepMeasurementRows := buildStepMeasurementRows(nodeStats)
	if len(stepMeasurementRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "step-measurements", Title: "Step Measurements", HTML: buildTableHTML(stepMeasurementRows, true)})
	}

	statusCodeRows := buildStatusCodeRows(nodeStats)
	if len(statusCodeRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "status-codes", Title: "Status Codes", HTML: buildTableHTML(statusCodeRows, true)})
	}

	failedStatusRows := buildFailedStatusCodeRows(nodeStats)
	failedEventRows := buildFailedEventRows(nodeStats)
	if len(failedStatusRows.Rows) > 0 || len(failedEventRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "failed-responses", Title: "Failed Responses", HTML: buildFailedResponseHTML(failedStatusRows, failedEventRows)})
	}

	thresholdRows := buildThresholdRows(nodeStats)
	if len(thresholdRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "thresholds", Title: "Thresholds", HTML: buildTableHTML(thresholdRows, true)})
	}

	metricRows := buildMetricRows(reportObject(nodeStats, "metrics", "Metrics"))
	if len(metricRows.Rows) > 0 {
		tabs = append(tabs, htmlTab{ID: "metrics", Title: "Metrics", HTML: buildTableHTML(metricRows, true)})
	}

	for _, plugin := range reportRecordList(nodeStats, "pluginsData", "PluginsData") {
		pluginName := asString(reportValue(plugin, "pluginName", "PluginName"))
		hintsHTML := buildHintsHTML(reportStringList(plugin, "hints", "Hints"))
		skipMergedPluginTab := strings.Contains(strings.ToLower(pluginName), "failed") || strings.Contains(strings.ToLower(pluginName), "correlation")
		tables := reportRecordList(plugin, "tables", "Tables")
		if len(tables) == 0 {
			if !skipMergedPluginTab && hintsHTML != "" {
				tabs = append(tabs, htmlTab{ID: fmt.Sprintf("plugin-%d", len(tabs)), Title: pluginName, HTML: hintsHTML})
			}
			continue
		}

		for _, table := range tables {
			tableName := asString(reportValue(table, "tableName", "TableName"))
			rows := normalizePluginTableRows(reportRows(table, "rows", "Rows"))
			isFailedResponses := strings.Contains(strings.ToLower(pluginName), "failed") || strings.Contains(strings.ToLower(tableName), "failed")
			isGroupedCorrelationSummary := strings.Contains(strings.ToLower(pluginName), "correlation") && strings.Contains(strings.ToLower(tableName), "grouped correlation summary")
			isUngroupedCorrelationSummary := strings.Contains(strings.ToLower(pluginName), "correlation") && strings.Contains(strings.ToLower(tableName), "ungrouped correlation rows")
			if isFailedResponses || len(rows) == 0 {
				continue
			}

			title := fmt.Sprintf("%s: %s", pluginName, tableName)
			bodyHTML := buildGenericPluginTableHTML(rows)
			if isGroupedCorrelationSummary {
				title = "Grouped Correlation Summary"
				bodyHTML = buildGroupedCorrelationSummaryHTML(rows, fmt.Sprintf("grouped-correlation-%d", len(tabs)))
			} else if isUngroupedCorrelationSummary {
				title = "Ungrouped Corelation Summary"
				bodyHTML = buildUngroupedCorrelationSummaryHTML(rows, fmt.Sprintf("ungrouped-correlation-%d", len(tabs)))
			}
			tabs = append(tabs, htmlTab{ID: fmt.Sprintf("plugin-%d", len(tabs)), Title: title, HTML: hintsHTML + bodyHTML})
		}
	}

	return tabs
}

func buildSummaryHTML(nodeStats map[string]any) string {
	testInfo := reportObject(nodeStats, "testInfo", "testInfo")
	nodeInfo := reportObject(nodeStats, "nodeInfo", "nodeInfo")
	scenarios := sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats"))
	allRequests := asInt(reportValue(nodeStats, "allRequestCount", "AllRequestCount"))
	allOK := asInt(reportValue(nodeStats, "allOkCount", "AllOkCount"))
	allFail := asInt(reportValue(nodeStats, "allFailCount", "AllFailCount"))
	successRate := 0.0
	failRate := 0.0
	if allRequests > 0 {
		successRate = float64(allOK) * 100 / float64(allRequests)
		failRate = float64(allFail) * 100 / float64(allRequests)
	}
	overallRPS := 0.0
	durationSeconds := reportDurationSeconds(reportValue(nodeStats, "duration", "Duration", "durationMs", "DurationMs"))
	if durationSeconds > 0 {
		overallRPS = float64(allRequests) / durationSeconds
	}

	topScenarioName := "n/a"
	topRequests := -1
	for _, scenario := range scenarios {
		requests := asInt(reportValue(scenario, "allRequestCount", "AllRequestCount"))
		if requests > topRequests {
			topRequests = requests
			topScenarioName = asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		}
	}

	latencyRows := buildSummaryLatencyRows(nodeStats)
	chartData := buildHTMLChartData(nodeStats)

	var builder strings.Builder
	appendLine(&builder, "<div class=\"card-grid\">")
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Total Requests</div><div class=\"stat-value\">%d</div></div>", allRequests))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Success</div><div class=\"stat-value value-ok\">%d (%s%%)</div></div>", allOK, formatSummaryPercent(successRate)))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Fail</div><div class=\"stat-value value-fail\">%d (%s%%)</div></div>", allFail, formatSummaryPercent(failRate)))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Overall RPS</div><div class=\"stat-value\">%s</div></div>", formatReportNumber(overallRPS)))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Duration</div><div class=\"stat-value\">%s</div></div>", escapeHTML(formatDotnetTimeSpan(reportValue(nodeStats, "duration", "Duration", "durationMs", "DurationMs")))))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Total Bytes</div><div class=\"stat-value\">%d</div></div>", asInt(reportValue(nodeStats, "allBytes", "AllBytes"))))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Top Scenario</div><div class=\"stat-value\">%s</div></div>", escapeHTML(topScenarioName)))
	appendLine(&builder, fmt.Sprintf("<div class=\"stat-card\"><div class=\"stat-label\">Node</div><div class=\"stat-value\">%d</div></div>", loadStrikeNodeTypeTag(reportValue(nodeInfo, "nodeType", "NodeType"))))
	appendLine(&builder, "</div>")

	renderedCharts := false
	var chartsBuilder strings.Builder
	if hasPieChartData(chartData.OverallOutcome) {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-outcome", "Success vs Fail")
	}
	if len(chartData.ScenarioRequests) > 0 {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-scenario-requests", "Requests by Scenario")
	}
	if len(chartData.ScenarioP95Latency) > 0 {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-scenario-p95", "P95 Latency by Scenario (ms)")
	}
	if len(chartData.ScenarioRps) > 0 {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-scenario-rps", "RPS by Scenario")
	}
	if len(chartData.ScenarioFailRate) > 0 {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-scenario-fail-rate", "Failure Rate by Scenario (%)")
	}
	if len(chartData.ScenarioBytes) > 0 {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-scenario-bytes", "Bytes by Scenario")
	}
	if hasPieChartData(chartData.StatusCodeClasses) {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-status-code-classes", "Status Code Class Mix")
	}
	if hasLatencyTrendData(chartData.ScenarioLatencyTrend) {
		renderedCharts = true
		appendChartCard(&chartsBuilder, "chart-scenario-latency-lines", "Latency Trend by Scenario (ms)")
	}
	if renderedCharts {
		appendLine(&builder, "<div class=\"card\">")
		appendLine(&builder, "<h2>Charts</h2>")
		appendLine(&builder, "<div class=\"chart-grid\">")
		builder.WriteString(chartsBuilder.String())
		appendLine(&builder, "</div>")
		appendLine(&builder, "</div>")
	}

	if len(latencyRows.Rows) > 0 {
		appendLine(&builder, "<div class=\"card\">")
		appendLine(&builder, "<h2>Latency by Scenario (Table)</h2>")
		appendHTMLLine(&builder, buildTableHTML(latencyRows, false))
		appendLine(&builder, "</div>")
	}

	appendLine(&builder, "<div class=\"card\">")
	appendLine(&builder, "<h2>Environment</h2>")
	appendLine(&builder, "<div class=\"table-wrap\"><table><tbody>")
	appendLine(&builder, fmt.Sprintf("<tr><th>Cluster Id</th><td>%s</td></tr>", escapeHTML(asString(reportValue(testInfo, "clusterId", "ClusterId")))))
	appendLine(&builder, fmt.Sprintf("<tr><th>Created (UTC)</th><td>%s</td></tr>", escapeHTML(formatReportTimestamp(reportValue(testInfo, "created", "Created")))))
	appendLine(&builder, fmt.Sprintf("<tr><th>Machine</th><td>%s</td></tr>", escapeHTML(asString(reportValue(nodeInfo, "machineName", "MachineName")))))
	appendLine(&builder, fmt.Sprintf("<tr><th>OS</th><td>%s</td></tr>", escapeHTML(asString(reportValue(nodeInfo, "os", "OS")))))
	appendLine(&builder, fmt.Sprintf("<tr><th>DotNet</th><td>%s</td></tr>", escapeHTML(asString(reportValue(nodeInfo, "dotNetVersion", "DotNetVersion")))))
	appendLine(&builder, fmt.Sprintf("<tr><th>Processor</th><td>%s</td></tr>", escapeHTML(asString(reportValue(nodeInfo, "processor", "Processor")))))
	appendLine(&builder, fmt.Sprintf("<tr><th>Cores</th><td>%d</td></tr>", asInt(reportValue(nodeInfo, "coresCount", "CoresCount"))))
	appendLine(&builder, fmt.Sprintf("<tr><th>Operation</th><td>%s</td></tr>", escapeHTML(asString(reportValue(nodeInfo, "currentOperation", "CurrentOperation")))))
	appendLine(&builder, "</tbody></table></div>")
	appendLine(&builder, "</div>")
	return builder.String()
}

func buildSummaryLatencyRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		rows = append(rows, buildSummaryLatencyRow(scenarioName, "OK", reportObject(scenario, "ok", "Ok")))
		rows = append(rows, buildSummaryLatencyRow(scenarioName, "FAIL", reportObject(scenario, "fail", "Fail")))
	}
	return reportTable{
		Headers: []string{"scenario", "Result", "Count", "LatencyMinMs", "LatencyMeanMs", "LatencyP50Ms", "LatencyP75Ms", "LatencyP95Ms", "LatencyP99Ms", "LatencyMaxMs", "LatencyStdDev"},
		Rows:    rows,
	}
}

func buildSummaryLatencyRow(scenarioName, result string, measurement map[string]any) map[string]any {
	latency := reportObject(measurement, "latency", "Latency")
	return map[string]any{
		"scenario":      scenarioName,
		"Result":        result,
		"Count":         requestCount(measurement),
		"LatencyMinMs":  formatReportNumber(asDouble(reportValue(latency, "minMs", "MinMs"))),
		"LatencyMeanMs": formatReportNumber(asDouble(reportValue(latency, "meanMs", "MeanMs"))),
		"LatencyP50Ms":  formatReportNumber(asDouble(reportValue(latency, "percent50", "Percent50"))),
		"LatencyP75Ms":  formatReportNumber(asDouble(reportValue(latency, "percent75", "Percent75"))),
		"LatencyP95Ms":  formatReportNumber(asDouble(reportValue(latency, "percent95", "Percent95"))),
		"LatencyP99Ms":  formatReportNumber(asDouble(reportValue(latency, "percent99", "Percent99"))),
		"LatencyMaxMs":  formatReportNumber(asDouble(reportValue(latency, "maxMs", "MaxMs"))),
		"LatencyStdDev": formatReportNumber(asDouble(reportValue(latency, "stdDev", "StdDev"))),
	}
}

func appendChartCard(builder *strings.Builder, id, title string) {
	appendLine(builder, fmt.Sprintf("<div class=\"chart-card\"><h3>%s</h3><canvas id=\"%s\" class=\"chart-canvas\"></canvas></div>", escapeHTML(title), escapeHTML(id)))
}

func hasPieChartData(points []chartPoint) bool {
	if len(points) == 0 {
		return false
	}
	for _, point := range points {
		if point.Value != 0 {
			return true
		}
	}
	return false
}

func hasLatencyTrendData(chart latencyTrendChart) bool {
	if len(chart.Labels) == 0 {
		return false
	}
	for _, series := range chart.Series {
		if len(series.Values) > 0 {
			return true
		}
	}
	return false
}

func buildScenarioRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		durationSeconds := reportDurationSeconds(reportValue(scenario, "duration", "Duration", "durationMs", "DurationMs"))
		requests := asInt(reportValue(scenario, "allRequestCount", "AllRequestCount"))
		rps := 0.0
		if durationSeconds > 0 {
			rps = float64(requests) / durationSeconds
		}
		okLatency := reportObject(reportObject(scenario, "ok", "Ok"), "latency", "Latency")
		failLatency := reportObject(reportObject(scenario, "fail", "Fail"), "latency", "Latency")
		rows = append(rows, map[string]any{
			"scenario":         asString(reportValue(scenario, "scenarioName", "ScenarioName")),
			"Simulation":       asString(reportValue(reportObject(scenario, "loadSimulationStats", "loadSimulationStats"), "simulationName", "SimulationName")),
			"SimulationValue":  asInt(reportValue(reportObject(scenario, "loadSimulationStats", "loadSimulationStats"), "value", "Value")),
			"Requests":         requests,
			"OK":               asInt(reportValue(scenario, "allOkCount", "AllOkCount")),
			"FAIL":             asInt(reportValue(scenario, "allFailCount", "AllFailCount")),
			"Duration":         formatDotnetTimeSpan(reportValue(scenario, "duration", "Duration", "durationMs", "DurationMs")),
			"RPS":              formatReportNumber(rps),
			"LatencyP95Ms":     formatReportNumber(maxDouble(asDouble(reportValue(okLatency, "percent95", "Percent95")), asDouble(reportValue(failLatency, "percent95", "Percent95")))),
			"LatencyP99Ms":     formatReportNumber(maxDouble(asDouble(reportValue(okLatency, "percent99", "Percent99")), asDouble(reportValue(failLatency, "percent99", "Percent99")))),
			"CurrentOperation": asString(reportValue(scenario, "currentOperation", "CurrentOperation")),
		})
	}
	return reportTable{
		Headers: []string{"scenario", "Simulation", "SimulationValue", "Requests", "OK", "FAIL", "Duration", "RPS", "LatencyP95Ms", "LatencyP99Ms", "CurrentOperation"},
		Rows:    rows,
	}
}

func buildStepRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		for _, step := range sortBySortIndex(reportRecordList(scenario, "stepStats", "stepStats")) {
			okMeasurement := reportObject(step, "ok", "Ok")
			failMeasurement := reportObject(step, "fail", "Fail")
			okLatency := reportObject(okMeasurement, "latency", "Latency")
			failLatency := reportObject(failMeasurement, "latency", "Latency")
			rows = append(rows, map[string]any{
				"scenario":      scenarioName,
				"step":          asString(reportValue(step, "stepName", "StepName")),
				"Requests":      requestCount(okMeasurement) + requestCount(failMeasurement),
				"OK":            requestCount(okMeasurement),
				"FAIL":          requestCount(failMeasurement),
				"OK_RPS":        formatReportNumber(requestRps(okMeasurement)),
				"FAIL_RPS":      formatReportNumber(requestRps(failMeasurement)),
				"LatencyMeanMs": formatReportNumber(maxDouble(asDouble(reportValue(okLatency, "meanMs", "MeanMs")), asDouble(reportValue(failLatency, "meanMs", "MeanMs")))),
				"P95LatencyMs":  formatReportNumber(maxDouble(asDouble(reportValue(okLatency, "percent95", "Percent95")), asDouble(reportValue(failLatency, "percent95", "Percent95")))),
			})
		}
	}
	return reportTable{
		Headers: []string{"scenario", "step", "Requests", "OK", "FAIL", "OK_RPS", "FAIL_RPS", "LatencyMeanMs", "P95LatencyMs"},
		Rows:    rows,
	}
}

func buildScenarioMeasurementRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		rows = append(rows, buildMeasurementRow("scenario", scenarioName, "", "OK", reportObject(scenario, "ok", "Ok")))
		rows = append(rows, buildMeasurementRow("scenario", scenarioName, "", "FAIL", reportObject(scenario, "fail", "Fail")))
	}
	return measurementReportTable(rows)
}

func buildStepMeasurementRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		for _, step := range sortBySortIndex(reportRecordList(scenario, "stepStats", "stepStats")) {
			stepName := asString(reportValue(step, "stepName", "StepName"))
			rows = append(rows, buildMeasurementRow("step", scenarioName, stepName, "OK", reportObject(step, "ok", "Ok")))
			rows = append(rows, buildMeasurementRow("step", scenarioName, stepName, "FAIL", reportObject(step, "fail", "Fail")))
		}
	}
	return measurementReportTable(rows)
}

func measurementReportTable(rows []map[string]any) reportTable {
	return reportTable{
		Headers: []string{"Scope", "scenario", "step", "Result", "Count", "Percent", "RPS", "LatencyMinMs", "LatencyMeanMs", "LatencyP50Ms", "LatencyP75Ms", "LatencyP95Ms", "LatencyP99Ms", "LatencyMaxMs", "LatencyStdDev", "Latency<=800ms", "Latency800-1200ms", "Latency>=1200ms", "BytesTotal", "BytesMin", "BytesMean", "BytesP50", "BytesP75", "BytesP95", "BytesP99", "BytesMax", "BytesStdDev"},
		Rows:    rows,
	}
}

func buildMeasurementRow(scope, scenarioName, stepName, result string, measurement map[string]any) map[string]any {
	request := reportObject(measurement, "request", "Request")
	latency := reportObject(measurement, "latency", "Latency")
	latencyCount := reportObject(latency, "latencyCount", "latencyCount")
	dataTransfer := reportObject(measurement, "dataTransfer", "DataTransfer")
	return map[string]any{
		"Scope":             scope,
		"scenario":          scenarioName,
		"step":              stepName,
		"Result":            result,
		"Count":             asInt(reportValue(request, "count", "Count")),
		"Percent":           asInt(reportValue(request, "percent", "Percent")),
		"RPS":               formatReportNumber(asDouble(reportValue(request, "rps", "RPS"))),
		"LatencyMinMs":      formatReportNumber(asDouble(reportValue(latency, "minMs", "MinMs"))),
		"LatencyMeanMs":     formatReportNumber(asDouble(reportValue(latency, "meanMs", "MeanMs"))),
		"LatencyP50Ms":      formatReportNumber(asDouble(reportValue(latency, "percent50", "Percent50"))),
		"LatencyP75Ms":      formatReportNumber(asDouble(reportValue(latency, "percent75", "Percent75"))),
		"LatencyP95Ms":      formatReportNumber(asDouble(reportValue(latency, "percent95", "Percent95"))),
		"LatencyP99Ms":      formatReportNumber(asDouble(reportValue(latency, "percent99", "Percent99"))),
		"LatencyMaxMs":      formatReportNumber(asDouble(reportValue(latency, "maxMs", "MaxMs"))),
		"LatencyStdDev":     formatReportNumber(asDouble(reportValue(latency, "stdDev", "StdDev"))),
		"Latency<=800ms":    asInt(reportValue(latencyCount, "lessOrEq800", "LessOrEq800")),
		"Latency800-1200ms": asInt(reportValue(latencyCount, "more800Less1200", "More800Less1200")),
		"Latency>=1200ms":   asInt(reportValue(latencyCount, "moreOrEq1200", "MoreOrEq1200")),
		"BytesTotal":        asInt(reportValue(dataTransfer, "allBytes", "AllBytes")),
		"BytesMin":          asInt(reportValue(dataTransfer, "minBytes", "MinBytes")),
		"BytesMean":         asInt(reportValue(dataTransfer, "meanBytes", "MeanBytes")),
		"BytesP50":          asInt(reportValue(dataTransfer, "percent50", "Percent50")),
		"BytesP75":          asInt(reportValue(dataTransfer, "percent75", "Percent75")),
		"BytesP95":          asInt(reportValue(dataTransfer, "percent95", "Percent95")),
		"BytesP99":          asInt(reportValue(dataTransfer, "percent99", "Percent99")),
		"BytesMax":          asInt(reportValue(dataTransfer, "maxBytes", "MaxBytes")),
		"BytesStdDev":       formatReportNumber(asDouble(reportValue(dataTransfer, "stdDev", "StdDev"))),
	}
}

func buildStatusCodeRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		rows = append(rows, measurementStatusCodeRows("scenario", scenarioName, "", "OK", reportObject(scenario, "ok", "Ok"))...)
		rows = append(rows, measurementStatusCodeRows("scenario", scenarioName, "", "FAIL", reportObject(scenario, "fail", "Fail"))...)
		for _, step := range sortBySortIndex(reportRecordList(scenario, "stepStats", "stepStats")) {
			stepName := asString(reportValue(step, "stepName", "StepName"))
			rows = append(rows, measurementStatusCodeRows("step", scenarioName, stepName, "OK", reportObject(step, "ok", "Ok"))...)
			rows = append(rows, measurementStatusCodeRows("step", scenarioName, stepName, "FAIL", reportObject(step, "fail", "Fail"))...)
		}
	}
	return reportTable{
		Headers: []string{"Scope", "scenario", "step", "Result", "StatusCode", "Message", "Count", "Percent", "IsError"},
		Rows:    rows,
	}
}

func measurementStatusCodeRows(scope, scenarioName, stepName, result string, measurement map[string]any) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, code := range reportRecordList(measurement, "statusCodes", "StatusCodes") {
		rows = append(rows, map[string]any{
			"Scope":      scope,
			"scenario":   scenarioName,
			"step":       stepName,
			"Result":     result,
			"StatusCode": asString(reportValue(code, "statusCode", "StatusCode")),
			"Message":    asString(reportValue(code, "message", "Message")),
			"Count":      asInt(reportValue(code, "count", "Count")),
			"Percent":    asInt(reportValue(code, "percent", "Percent")),
			"IsError":    asBool(reportValue(code, "isError", "IsError")),
		})
	}
	return rows
}

func buildFailedResponseHTML(failedStatusRows, failedEventRows reportTable) string {
	var builder strings.Builder
	if len(failedStatusRows.Rows) > 0 {
		appendLine(&builder, "<div class=\"card\">")
		appendLine(&builder, "<h2>Failed Status Codes</h2>")
		appendHTMLLine(&builder, buildTableHTML(failedStatusRows, false))
		appendLine(&builder, "</div>")
	}
	if len(failedEventRows.Rows) > 0 {
		appendLine(&builder, "<div class=\"card\">")
		appendLine(&builder, "<h2>Failed and Timed Out Rows</h2>")
		appendHTMLLine(&builder, buildTableHTML(failedEventRows, false))
		appendLine(&builder, "</div>")
	}
	return builder.String()
}

func buildFailedStatusCodeRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, scenario := range sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats")) {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		rows = append(rows, failedMeasurementStatusCodeRows("scenario", scenarioName, "", reportObject(scenario, "fail", "Fail"))...)
		for _, step := range sortBySortIndex(reportRecordList(scenario, "stepStats", "stepStats")) {
			rows = append(rows, failedMeasurementStatusCodeRows("step", scenarioName, asString(reportValue(step, "stepName", "StepName")), reportObject(step, "fail", "Fail"))...)
		}
	}
	return reportTable{
		Headers: []string{"Scope", "scenario", "step", "StatusCode", "Message", "Count", "Percent", "IsError"},
		Rows:    rows,
	}
}

func failedMeasurementStatusCodeRows(scope, scenarioName, stepName string, measurement map[string]any) []map[string]any {
	rows := make([]map[string]any, 0)
	for _, code := range reportRecordList(measurement, "statusCodes", "StatusCodes") {
		count := asInt(reportValue(code, "count", "Count"))
		isError := asBool(reportValue(code, "isError", "IsError"))
		if count <= 0 && !isError {
			continue
		}
		rows = append(rows, map[string]any{
			"Scope":      scope,
			"scenario":   scenarioName,
			"step":       stepName,
			"StatusCode": asString(reportValue(code, "statusCode", "StatusCode")),
			"Message":    asString(reportValue(code, "message", "Message")),
			"Count":      count,
			"Percent":    asInt(reportValue(code, "percent", "Percent")),
			"IsError":    isError,
		})
	}
	return rows
}

func buildFailedEventRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, plugin := range reportRecordList(nodeStats, "pluginsData", "PluginsData") {
		pluginName := asString(reportValue(plugin, "pluginName", "PluginName"))
		for _, table := range reportRecordList(plugin, "tables", "Tables") {
			tableName := asString(reportValue(table, "tableName", "TableName"))
			lowerPlugin := strings.ToLower(pluginName)
			lowerTable := strings.ToLower(tableName)
			failedTable := strings.Contains(lowerPlugin, "fail") || strings.Contains(lowerPlugin, "timeout") || strings.Contains(lowerTable, "fail") || strings.Contains(lowerTable, "timeout")
			if !failedTable {
				continue
			}
			for _, row := range reportRows(table, "rows", "Rows") {
				rows = append(rows, map[string]any{
					"Category":    fmt.Sprintf("%s: %s", pluginName, tableName),
					"OccurredUtc": tryReadRowValue(row, "OccurredUtc"),
					"scenario":    tryReadRowValue(row, "scenario", "Scenario", "scenarioName", "ScenarioName"),
					"step":        tryReadRowValue(row, "step", "Step", "stepName", "StepName"),
					"Source":      tryReadRowValue(row, "Source"),
					"Destination": tryReadRowValue(row, "Destination"),
					"StatusCode":  tryReadRowValue(row, "StatusCode"),
					"Message":     tryReadRowValue(row, "Message"),
					"TrackingId":  tryReadRowValue(row, "TrackingId"),
					"EventId":     tryReadRowValue(row, "EventId"),
					"LatencyMs":   tryReadRowValue(row, "LatencyMs"),
				})
			}
		}
	}
	return reportTable{
		Headers: []string{"Category", "OccurredUtc", "scenario", "step", "Source", "Destination", "StatusCode", "Message", "TrackingId", "EventId", "LatencyMs"},
		Rows:    rows,
	}
}

func buildThresholdRows(nodeStats map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, threshold := range reportRecordList(nodeStats, "thresholds", "Thresholds") {
		rows = append(rows, map[string]any{
			"scenario":   asString(reportValue(threshold, "scenarioName", "ScenarioName")),
			"step":       asString(reportValue(threshold, "stepName", "StepName")),
			"Check":      asString(reportValue(threshold, "checkExpression", "CheckExpression")),
			"Failed":     asBool(reportValue(threshold, "isFailed", "IsFailed")),
			"ErrorCount": asInt(reportValue(threshold, "errorCount", "ErrorCount")),
			"Exception":  asString(reportValue(threshold, "exceptionMessage", "ExceptionMessage", "exceptionMsg", "ExceptionMsg")),
		})
	}
	return reportTable{
		Headers: []string{"scenario", "step", "Check", "Failed", "ErrorCount", "Exception"},
		Rows:    rows,
	}
}

func buildMetricRows(metrics map[string]any) reportTable {
	rows := make([]map[string]any, 0)
	for _, counter := range reportRecordList(metrics, "counters", "Counters") {
		rows = append(rows, map[string]any{
			"Type":     "Counter",
			"scenario": asString(reportValue(counter, "scenarioName", "ScenarioName")),
			"Name":     asString(reportValue(counter, "metricName", "MetricName")),
			"Unit":     asString(reportValue(counter, "unitOfMeasure", "UnitOfMeasure")),
			"Value":    asInt(reportValue(counter, "value", "Value")),
		})
	}
	for _, gauge := range reportRecordList(metrics, "gauges", "Gauges") {
		rows = append(rows, map[string]any{
			"Type":     "Gauge",
			"scenario": asString(reportValue(gauge, "scenarioName", "ScenarioName")),
			"Name":     asString(reportValue(gauge, "metricName", "MetricName")),
			"Unit":     asString(reportValue(gauge, "unitOfMeasure", "UnitOfMeasure")),
			"Value":    formatReportNumber(asDouble(reportValue(gauge, "value", "Value"))),
		})
	}
	return reportTable{
		Headers: []string{"Type", "scenario", "Name", "Unit", "Value"},
		Rows:    rows,
	}
}

func buildHTMLChartData(nodeStats map[string]any) htmlChartData {
	orderedScenarios := sortBySortIndex(reportRecordList(nodeStats, "scenarioStats", "scenarioStats"))
	scenarioRequests := make([]chartPoint, 0, len(orderedScenarios))
	scenarioP95 := make([]chartPoint, 0, len(orderedScenarios))
	scenarioRPS := make([]chartPoint, 0, len(orderedScenarios))
	scenarioFailRate := make([]chartPoint, 0, len(orderedScenarios))
	scenarioBytes := make([]chartPoint, 0, len(orderedScenarios))
	scenarioLabels := make([]string, 0, len(orderedScenarios))
	p50Values := make([]float64, 0, len(orderedScenarios))
	p75Values := make([]float64, 0, len(orderedScenarios))
	p95Values := make([]float64, 0, len(orderedScenarios))
	p99Values := make([]float64, 0, len(orderedScenarios))

	for _, scenario := range orderedScenarios {
		scenarioName := asString(reportValue(scenario, "scenarioName", "ScenarioName"))
		requests := asInt(reportValue(scenario, "allRequestCount", "AllRequestCount"))
		failCount := asInt(reportValue(scenario, "allFailCount", "AllFailCount"))
		durationSeconds := reportDurationSeconds(reportValue(scenario, "duration", "Duration", "durationMs", "DurationMs"))
		okLatency := reportObject(reportObject(scenario, "ok", "Ok"), "latency", "Latency")
		failLatency := reportObject(reportObject(scenario, "fail", "Fail"), "latency", "Latency")
		p50 := maxDouble(asDouble(reportValue(okLatency, "percent50", "Percent50")), asDouble(reportValue(failLatency, "percent50", "Percent50")))
		p75 := maxDouble(asDouble(reportValue(okLatency, "percent75", "Percent75")), asDouble(reportValue(failLatency, "percent75", "Percent75")))
		p95 := maxDouble(asDouble(reportValue(okLatency, "percent95", "Percent95")), asDouble(reportValue(failLatency, "percent95", "Percent95")))
		p99 := maxDouble(asDouble(reportValue(okLatency, "percent99", "Percent99")), asDouble(reportValue(failLatency, "percent99", "Percent99")))
		rps := 0.0
		failRate := 0.0
		if durationSeconds > 0 {
			rps = float64(requests) / durationSeconds
		}
		if requests > 0 {
			failRate = float64(failCount) * 100 / float64(requests)
		}
		scenarioRequests = append(scenarioRequests, chartPoint{Label: scenarioName, Value: float64(requests), Color: "#3b82f6"})
		scenarioP95 = append(scenarioP95, chartPoint{Label: scenarioName, Value: p95, Color: "#8b5cf6"})
		scenarioRPS = append(scenarioRPS, chartPoint{Label: scenarioName, Value: rps, Color: "#10b981"})
		scenarioFailRate = append(scenarioFailRate, chartPoint{Label: scenarioName, Value: failRate, Color: "#ef4444"})
		scenarioBytes = append(scenarioBytes, chartPoint{Label: scenarioName, Value: float64(asInt(reportValue(scenario, "allBytes", "AllBytes"))), Color: "#0ea5e9"})
		scenarioLabels = append(scenarioLabels, scenarioName)
		p50Values = append(p50Values, p50)
		p75Values = append(p75Values, p75)
		p95Values = append(p95Values, p95)
		p99Values = append(p99Values, p99)
	}

	return htmlChartData{
		OverallOutcome: []chartPoint{
			{Label: "OK", Value: float64(asInt(reportValue(nodeStats, "allOkCount", "AllOkCount"))), Color: "#18a957"},
			{Label: "FAIL", Value: float64(asInt(reportValue(nodeStats, "allFailCount", "AllFailCount"))), Color: "#d14343"},
		},
		ScenarioRequests:   scenarioRequests,
		ScenarioP95Latency: scenarioP95,
		ScenarioRps:        scenarioRPS,
		ScenarioFailRate:   scenarioFailRate,
		ScenarioBytes:      scenarioBytes,
		StatusCodeClasses:  buildStatusCodeClassChart(orderedScenarios),
		ScenarioLatencyTrend: latencyTrendChart{
			Labels: scenarioLabels,
			Series: []latencyTrendSeries{
				{Name: "P50", Color: "#38bdf8", Values: p50Values},
				{Name: "P75", Color: "#22c55e", Values: p75Values},
				{Name: "P95", Color: "#f59e0b", Values: p95Values},
				{Name: "P99", Color: "#f43f5e", Values: p99Values},
			},
		},
	}
}

func buildStatusCodeClassChart(scenarios []map[string]any) []chartPoint {
	buckets := map[string]float64{"2xx": 0, "3xx": 0, "4xx": 0, "5xx": 0, "Other": 0}
	for _, scenario := range scenarios {
		for _, measurement := range []map[string]any{reportObject(scenario, "ok", "Ok"), reportObject(scenario, "fail", "Fail")} {
			for _, status := range reportRecordList(measurement, "statusCodes", "StatusCodes") {
				buckets[classifyStatusCode(asString(reportValue(status, "statusCode", "StatusCode")))] += float64(asInt(reportValue(status, "count", "Count")))
			}
		}
	}
	points := []chartPoint{
		{Label: "2xx", Value: buckets["2xx"], Color: "#16a34a"},
		{Label: "3xx", Value: buckets["3xx"], Color: "#0ea5e9"},
		{Label: "4xx", Value: buckets["4xx"], Color: "#f59e0b"},
		{Label: "5xx", Value: buckets["5xx"], Color: "#dc2626"},
		{Label: "Other", Value: buckets["Other"], Color: "#8b5cf6"},
	}
	filtered := make([]chartPoint, 0, len(points))
	for _, point := range points {
		if point.Value > 0 {
			filtered = append(filtered, point)
		}
	}
	return filtered
}

func classifyStatusCode(statusCode string) string {
	trimmed := strings.TrimSpace(statusCode)
	if len(trimmed) >= 3 && trimmed[0] >= '0' && trimmed[0] <= '9' {
		switch trimmed[0] {
		case '2':
			return "2xx"
		case '3':
			return "3xx"
		case '4':
			return "4xx"
		case '5':
			return "5xx"
		}
	}
	return "Other"
}

func buildGroupedCorrelationSummaryHTML(rows []map[string]any, groupedChartKey string) string {
	charts := buildGroupedCorrelationPercentileCharts(rows)
	var builder strings.Builder
	appendHTMLLine(&builder, buildTableHTML(reportTable{
		Headers: []string{"scenario", "Destination", "GatherByField", "GatherByValue", "MatchedCount", "UnmatchedCount", "TimeoutCount", "LatencyP50Ms", "LatencyP80Ms", "LatencyP85Ms", "LatencyP90Ms", "LatencyP95Ms", "LatencyP99Ms"},
		Rows:    rows,
	}, true))
	if len(charts) == 0 {
		return builder.String()
	}
	appendLine(&builder, "<div class=\"card\"><h2>Grouped Correlation Percentile Trends</h2><p>Each chart is one GatherBy value. The line shows latency progression across P50 to P99.</p></div>")
	appendLine(&builder, "<div class=\"chart-grid correlation-chart-grid\">")
	for index, chart := range charts {
		chartKey := fmt.Sprintf("%s-value-%d", groupedChartKey, index)
		chartJSON := escapeJSONForHTMLScript(marshalJSONNoEscape(chart.Chart))
		title := fmt.Sprintf("%s: %s", chart.GatherByField, chart.GatherByValue)
		subtitle := "Single grouped row."
		if chart.SourceRows > 1 {
			subtitle = fmt.Sprintf("Averaged across %d grouped rows.", chart.SourceRows)
		}
		appendLine(&builder, fmt.Sprintf("<script type=\"application/json\" data-grouped-correlation-chart=\"%s\">%s</script>", escapeHTML(chartKey), chartJSON))
		appendLine(&builder, fmt.Sprintf("<div class=\"chart-card correlation-chart-card\"><h3>%s</h3><p>%s</p><canvas class=\"chart-canvas grouped-correlation-canvas\" data-grouped-key=\"%s\"></canvas></div>", escapeHTML(title), escapeHTML(subtitle), escapeHTML(chartKey)))
	}
	appendLine(&builder, "</div>")
	return builder.String()
}

func buildUngroupedCorrelationSummaryHTML(rows []map[string]any, groupedChartKey string) string {
	chart := buildUngroupedCorrelationPercentileChart(rows)
	var builder strings.Builder
	appendHTMLLine(&builder, buildTableHTML(reportTable{
		Headers: []string{"OccurredUtc", "scenario", "Source", "Destination", "RunMode", "StatusCode", "TrackingId", "EventId", "SourceTimestampUtc", "DestinationTimestampUtc", "LatencyMs", "Message"},
		Rows:    rows,
	}, true))
	if hasLatencyTrendData(chart) {
		chartJSON := escapeJSONForHTMLScript(marshalJSONNoEscape(chart))
		appendLine(&builder, "<div class=\"card\"><h2>Ungrouped Correlation Percentile Trends</h2><p>All percentile lines are shown in one graph for direct comparison.</p></div>")
		appendLine(&builder, fmt.Sprintf("<script type=\"application/json\" data-ungrouped-correlation=\"%s\">%s</script>", escapeHTML(groupedChartKey), chartJSON))
		appendLine(&builder, "<div class=\"chart-grid correlation-chart-grid\">")
		appendLine(&builder, fmt.Sprintf("<div class=\"chart-card correlation-chart-card\"><h3>Ungrouped Correlation Percentiles</h3><canvas class=\"chart-canvas ungrouped-correlation-canvas\" data-ungrouped-key=\"%s\"></canvas></div>", escapeHTML(groupedChartKey)))
		appendLine(&builder, "</div>")
	}
	return builder.String()
}

func buildGroupedCorrelationPercentileCharts(rows []map[string]any) []groupedCorrelationValueChart {
	chartRows := buildGroupedCorrelationChartRows(rows)
	filtered := make([]groupedCorrelationChartRow, 0, len(chartRows))
	for _, row := range chartRows {
		if row.LatencyP50Ms != nil || row.LatencyP80Ms != nil || row.LatencyP85Ms != nil || row.LatencyP90Ms != nil || row.LatencyP95Ms != nil || row.LatencyP99Ms != nil {
			filtered = append(filtered, row)
		}
	}
	if len(filtered) == 0 {
		return nil
	}

	type groupKey struct {
		Field string
		Value string
	}
	grouped := map[groupKey][]groupedCorrelationChartRow{}
	keys := make([]groupKey, 0)
	for _, row := range filtered {
		key := groupKey{Field: row.GatherByField, Value: row.GatherByValue}
		if _, exists := grouped[key]; !exists {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], row)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Field == keys[j].Field {
			return keys[i].Value < keys[j].Value
		}
		return keys[i].Field < keys[j].Field
	})

	labels := []string{"P50", "P80", "P85", "P90", "P95", "P99"}
	palette := []string{"#38bdf8", "#22c55e", "#f59e0b", "#a855f7", "#f43f5e", "#06b6d4", "#14b8a6", "#eab308", "#818cf8", "#84cc16"}
	charts := make([]groupedCorrelationValueChart, 0, len(keys))
	for index, key := range keys {
		group := grouped[key]
		charts = append(charts, groupedCorrelationValueChart{
			GatherByField: key.Field,
			GatherByValue: key.Value,
			SourceRows:    len(group),
			Chart: latencyTrendChart{
				Labels: labels,
				Series: []latencyTrendSeries{
					{
						Name:  "Latency",
						Color: palette[index%len(palette)],
						Values: []float64{
							averagePercentile(selectPercentiles(group, func(row groupedCorrelationChartRow) *float64 { return row.LatencyP50Ms })),
							averagePercentile(selectPercentiles(group, func(row groupedCorrelationChartRow) *float64 { return row.LatencyP80Ms })),
							averagePercentile(selectPercentiles(group, func(row groupedCorrelationChartRow) *float64 { return row.LatencyP85Ms })),
							averagePercentile(selectPercentiles(group, func(row groupedCorrelationChartRow) *float64 { return row.LatencyP90Ms })),
							averagePercentile(selectPercentiles(group, func(row groupedCorrelationChartRow) *float64 { return row.LatencyP95Ms })),
							averagePercentile(selectPercentiles(group, func(row groupedCorrelationChartRow) *float64 { return row.LatencyP99Ms })),
						},
					},
				},
			},
		})
	}
	return charts
}

func buildGroupedCorrelationChartRows(rows []map[string]any) []groupedCorrelationChartRow {
	chartRows := make([]groupedCorrelationChartRow, 0, len(rows))
	for _, row := range rows {
		scenario := tryReadRowValue(row, "scenario")
		destination := tryReadRowValue(row, "Destination")
		gatherByField := tryReadRowValue(row, "GatherByField")
		gatherByValue := tryReadRowValue(row, "GatherByValue")
		groupLabel := strings.TrimSpace(fmt.Sprintf("%s | %s", scenario, destination))
		chartRows = append(chartRows, groupedCorrelationChartRow{
			GroupLabel:    groupLabel,
			GatherByField: firstNonBlank(strings.TrimSpace(gatherByField), "<none>"),
			GatherByValue: firstNonBlank(strings.TrimSpace(gatherByValue), "<missing>"),
			LatencyP50Ms:  tryParseDoublePtr(tryReadRowValue(row, "LatencyP50Ms")),
			LatencyP80Ms:  tryParseDoublePtr(tryReadRowValue(row, "LatencyP80Ms")),
			LatencyP85Ms:  tryParseDoublePtr(tryReadRowValue(row, "LatencyP85Ms")),
			LatencyP90Ms:  tryParseDoublePtr(tryReadRowValue(row, "LatencyP90Ms")),
			LatencyP95Ms:  tryParseDoublePtr(tryReadRowValue(row, "LatencyP95Ms")),
			LatencyP99Ms:  tryParseDoublePtr(tryReadRowValue(row, "LatencyP99Ms")),
		})
	}
	filtered := chartRows[:0]
	for _, row := range chartRows {
		if strings.TrimSpace(row.GroupLabel) != "" {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func buildUngroupedCorrelationChartRows(rows []map[string]any) []groupedCorrelationChartRow {
	type rawRow struct {
		scenario    string
		Destination string
		StatusCode  string
		LatencyMs   float64
	}
	grouped := map[string][]rawRow{}
	keys := make([]string, 0)
	for _, row := range rows {
		scenario := tryReadRowValue(row, "scenario")
		destination := tryReadRowValue(row, "Destination")
		statusCode := tryReadRowValue(row, "StatusCode")
		latencyPtr := tryParseDoublePtr(tryReadRowValue(row, "LatencyMs"))
		if scenario == "" || destination == "" || statusCode == "" || latencyPtr == nil {
			continue
		}
		key := scenario + "\x00" + destination + "\x00" + statusCode
		if _, exists := grouped[key]; !exists {
			keys = append(keys, key)
		}
		grouped[key] = append(grouped[key], rawRow{scenario: scenario, Destination: destination, StatusCode: statusCode, LatencyMs: *latencyPtr})
	}
	sort.Strings(keys)
	result := make([]groupedCorrelationChartRow, 0, len(keys))
	for _, key := range keys {
		group := grouped[key]
		values := make([]float64, 0, len(group))
		for _, row := range group {
			values = append(values, row.LatencyMs)
		}
		sort.Float64s(values)
		first := group[0]
		result = append(result, groupedCorrelationChartRow{
			GroupLabel:    fmt.Sprintf("%s | %s", first.scenario, first.Destination),
			GatherByField: first.StatusCode,
			GatherByValue: first.StatusCode,
			LatencyP50Ms:  float64Ptr(percentile(values, 0.50)),
			LatencyP80Ms:  float64Ptr(percentile(values, 0.80)),
			LatencyP85Ms:  float64Ptr(percentile(values, 0.85)),
			LatencyP90Ms:  float64Ptr(percentile(values, 0.90)),
			LatencyP95Ms:  float64Ptr(percentile(values, 0.95)),
			LatencyP99Ms:  float64Ptr(percentile(values, 0.99)),
		})
	}
	return result
}

func buildUngroupedCorrelationPercentileChart(rows []map[string]any) latencyTrendChart {
	chartRows := buildUngroupedCorrelationChartRows(rows)
	labels := []string{"P50", "P80", "P85", "P90", "P95", "P99"}
	palette := []string{"#38bdf8", "#22c55e", "#f59e0b", "#a855f7", "#f43f5e", "#06b6d4", "#14b8a6", "#eab308", "#818cf8", "#84cc16"}
	nameCount := map[string]int{}
	series := make([]latencyTrendSeries, 0, len(chartRows))
	for index, row := range chartRows {
		baseName := strings.TrimSpace(row.GatherByField)
		if baseName == "" {
			baseName = "status"
		}
		count := nameCount[baseName]
		nameCount[baseName] = count + 1
		seriesName := baseName
		if count > 0 {
			seriesName = fmt.Sprintf("%s-%d", baseName, count+1)
		}
		series = append(series, latencyTrendSeries{
			Name:  seriesName,
			Color: palette[index%len(palette)],
			Values: []float64{
				derefFloat64(row.LatencyP50Ms),
				derefFloat64(row.LatencyP80Ms),
				derefFloat64(row.LatencyP85Ms),
				derefFloat64(row.LatencyP90Ms),
				derefFloat64(row.LatencyP95Ms),
				derefFloat64(row.LatencyP99Ms),
			},
		})
	}
	return latencyTrendChart{Labels: labels, Series: series}
}

func getReportLogoDarkDataURI() string {
	initReportLogos()
	return reportLogoDarkDataURI
}

func getReportLogoLightDataURI() string {
	initReportLogos()
	return reportLogoLightDataURI
}

func initReportLogos() {
	reportLogosOnce.Do(func() {
		reportLogoDarkDataURI = buildReportLogoDataURI(reportLogoDarkPNG)
		reportLogoLightDataURI = buildReportLogoDataURI(reportLogoLightPNG)
	})
}

func buildReportLogoDataURI(content []byte) string {
	if len(content) > 0 {
		return "data:image/png;base64," + base64.StdEncoding.EncodeToString(content)
	}
	const fallbackSVG = "<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 128 128'><defs><linearGradient id='g' x1='0' y1='0' x2='1' y2='1'><stop offset='0' stop-color='#64b5ff'/><stop offset='1' stop-color='#2f66db'/></linearGradient></defs><rect width='128' height='128' rx='24' fill='#081325'/><path d='M24 94L56 24l16 34 14-22 18 58h-16l-7-23-9 13-10-21-14 31H24z' fill='url(#g)'/></svg>"
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(fallbackSVG))
}

func escapeJSONForHTMLScript(value string) string {
	value = strings.ReplaceAll(value, "</", "<\\/")
	value = strings.ReplaceAll(value, "<!--", "<\\!--")
	return value
}

func buildTableHTML(table reportTable, wrapInCard bool) string {
	if len(table.Rows) == 0 {
		if wrapInCard {
			return "<div class=\"card\"><div class=\"table-wrap\"><table><tbody><tr><td>No rows.</td></tr></tbody></table></div></div>"
		}
		return "<div class=\"table-wrap\"><table><tbody><tr><td>No rows.</td></tr></tbody></table></div>"
	}

	headers := table.Headers
	if len(headers) == 0 {
		headers = inferHeaders(table.Rows)
	}

	var builder strings.Builder
	if wrapInCard {
		appendLine(&builder, "<div class=\"card\">")
	}
	appendLine(&builder, "<div class=\"table-wrap\">")
	appendLine(&builder, "<table>")
	appendLine(&builder, "<thead><tr>")
	for _, header := range headers {
		appendLine(&builder, fmt.Sprintf("<th>%s</th>", escapeHTML(displayTableHeader(header))))
	}
	appendLine(&builder, "</tr></thead><tbody>")
	for _, row := range table.Rows {
		appendLine(&builder, "<tr>")
		for _, header := range headers {
			appendLine(&builder, fmt.Sprintf("<td>%s</td>", escapeHTML(displayTableCell(header, row[header]))))
		}
		appendLine(&builder, "</tr>")
	}
	appendLine(&builder, "</tbody></table></div>")
	if wrapInCard {
		appendLine(&builder, "</div>")
	}
	return builder.String()
}

func displayTableHeader(header string) string {
	switch header {
	case "scenario":
		return "Scenario"
	case "step":
		return "Step"
	default:
		return header
	}
}

func displayTableCell(header string, value any) string {
	formatted := formatTableValue(value)
	if header != "Scope" {
		return formatted
	}
	switch formatted {
	case "scenario":
		return "Scenario"
	case "step":
		return "Step"
	default:
		return formatted
	}
}

func inferHeaders(rows []map[string]any) []string {
	headerSet := make(map[string]struct{})
	headers := make([]string, 0)
	for _, row := range rows {
		for key := range row {
			if _, exists := headerSet[key]; exists {
				continue
			}
			headerSet[key] = struct{}{}
			headers = append(headers, key)
		}
	}
	sort.Strings(headers)
	return headers
}

func buildGenericPluginTableHTML(rows []map[string]any) string {
	return buildTableHTML(reportTable{Headers: inferHeaders(rows), Rows: rows}, true)
}

func buildHintsHTML(hints []string) string {
	nonEmpty := make([]string, 0, len(hints))
	for _, hint := range hints {
		if strings.TrimSpace(hint) != "" {
			nonEmpty = append(nonEmpty, hint)
		}
	}
	if len(nonEmpty) == 0 {
		return ""
	}
	items := make([]string, 0, len(nonEmpty))
	for _, hint := range nonEmpty {
		items = append(items, fmt.Sprintf("<li>%s</li>", escapeHTML(hint)))
	}
	return "<div class=\"card\"><strong>Hints</strong><ul>" + strings.Join(items, "") + "</ul></div>"
}

func reportRows(source any, keys ...string) []map[string]any {
	return reportRecordList(source, keys...)
}

func reportStringList(source any, keys ...string) []string {
	value := reportValue(source, keys...)
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, asString(item))
		}
		return items
	default:
		return nil
	}
}

func normalizePluginTableRows(rows []map[string]any) []map[string]any {
	normalized := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		copyRow := make(map[string]any, len(row)+2)
		for key, value := range row {
			copyRow[key] = value
		}
		if _, exists := copyRow["scenario"]; !exists {
			for _, key := range []string{"Scenario", "scenarioName", "ScenarioName"} {
				if value, ok := copyRow[key]; ok {
					copyRow["scenario"] = value
					break
				}
			}
		}
		if _, exists := copyRow["step"]; !exists {
			for _, key := range []string{"Step", "stepName", "StepName"} {
				if value, ok := copyRow[key]; ok {
					copyRow["step"] = value
					break
				}
			}
		}
		normalized = append(normalized, copyRow)
	}
	return normalized
}

func tryReadRowValue(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, exists := row[key]; exists {
			return formatTableValue(value)
		}
	}
	return ""
}

func marshalJSONNoEscape(value any) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
	return strings.TrimSuffix(buffer.String(), "\n")
}

func escapeHTML(value string) string {
	return html.EscapeString(value)
}

func normalizeReportText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	if !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	return value
}

func appendLine(builder *strings.Builder, value string) {
	builder.WriteString(value)
	builder.WriteByte('\n')
}

func appendBlock(builder *strings.Builder, value string) {
	if value == "" {
		return
	}
	builder.WriteString(strings.TrimSuffix(value, "\n"))
	builder.WriteByte('\n')
}

func appendHTMLLine(builder *strings.Builder, value string) {
	if value == "" {
		builder.WriteByte('\n')
		return
	}
	builder.WriteString(value)
	builder.WriteByte('\n')
}

func formatSummaryPercent(value float64) string {
	text := strconv.FormatFloat(value, 'f', 2, 64)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "0"
	}
	return text
}

func formatReportTimestamp(value any) string {
	text := strings.TrimSpace(asString(value))
	if text == "" {
		return ""
	}
	if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
		return parsed.UTC().Format("2006-01-02T15:04:05.0000000Z")
	}
	return text
}

func formatTableValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case bool:
		if typed {
			return "True"
		}
		return "False"
	default:
		return fmt.Sprint(value)
	}
}

func maxDouble(left, right float64) float64 {
	if right > left {
		return right
	}
	return left
}

func tryParseDoublePtr(value string) *float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &parsed
}

func averagePercentile(values []*float64) float64 {
	samples := make([]float64, 0, len(values))
	for _, value := range values {
		if value != nil {
			samples = append(samples, *value)
		}
	}
	if len(samples) == 0 {
		return 0
	}
	sum := 0.0
	for _, sample := range samples {
		sum += sample
	}
	return sum / float64(len(samples))
}

func selectPercentiles(rows []groupedCorrelationChartRow, selector func(groupedCorrelationChartRow) *float64) []*float64 {
	values := make([]*float64, 0, len(rows))
	for _, row := range rows {
		values = append(values, selector(row))
	}
	return values
}

func percentile(orderedValues []float64, percentile float64) float64 {
	if len(orderedValues) == 0 {
		return 0
	}
	index := int(percentile*float64(len(orderedValues))+0.999999999) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(orderedValues) {
		index = len(orderedValues) - 1
	}
	return orderedValues[index]
}

func float64Ptr(value float64) *float64 {
	return &value
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
