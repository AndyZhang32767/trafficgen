package main

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>上行流量面板</title>
<style>
  :root { color-scheme: dark; }
  body { margin:0; font-family: -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif;
         background:#0f1216; color:#e6e6e6; display:flex; min-height:100vh;
         align-items:center; justify-content:center; padding:24px 0; }
  .wrap { width:min(680px,92vw); }
  h1 { font-size:20px; font-weight:600; margin:0 0 20px; color:#8ab4f8; }
  .grid { display:grid; grid-template-columns:1fr 1fr; gap:16px; }
  .card { background:#171b21; border:1px solid #232833; border-radius:14px; padding:20px; }
  .card.full { grid-column:1 / -1; }
  .label { font-size:13px; color:#8b97a7; margin-bottom:8px; }
  .value { font-size:30px; font-weight:700; letter-spacing:.5px; }
  .unit { font-size:15px; color:#8b97a7; margin-left:6px; font-weight:400; }
  .speed .value { color:#6ee7a8; }
  .client .value { color:#f0b45a; }
  .foot { margin-top:18px; font-size:12px; color:#5b6675; text-align:center; }
  .ctrl { display:flex; align-items:center; gap:12px; margin-top:14px; flex-wrap:wrap; }
  button { border:0; border-radius:10px; padding:10px 18px; font-size:14px; font-weight:600;
           cursor:pointer; color:#0f1216; background:#6ee7a8; }
  button.stop { background:#f0715a; }
  input { width:64px; background:#0f1216; border:1px solid #2a303b; color:#e6e6e6;
          border-radius:8px; padding:8px 10px; font-size:14px; }
  .status { font-size:13px; color:#8b97a7; }
  .dot { display:inline-block; width:9px; height:9px; border-radius:50%; margin-right:6px;
         background:#5b6675; vertical-align:middle; }
  .dot.on { background:#6ee7a8; box-shadow:0 0 8px #6ee7a8; }
</style>
</head>
<body>
<div class="wrap">
  <h1>上行流量面板</h1>
  <div class="grid">
    <div class="card speed full">
      <div class="label">服务端当前上行速率</div>
      <div class="value" id="speed">-</div>
    </div>
    <div class="card">
      <div class="label">当月流量 <span id="month"></span></div>
      <div class="value" id="monthly">-</div>
    </div>
    <div class="card">
      <div class="label">累计流量</div>
      <div class="value" id="total">-</div>
    </div>

    <div class="card client full">
      <div class="label">网页客户端（本浏览器直接刷流量）</div>
      <div class="value" id="localSpeed">未运行</div>
      <div class="ctrl">
        <span class="status"><span class="dot" id="dot"></span><span id="state">已停止</span></span>
        <label class="status">并发连接
          <input type="number" id="conns" value="3" min="1" max="64">
        </label>
        <button id="toggle">开始刷流量</button>
      </div>
    </div>
  </div>
  <div class="foot">点“开始刷流量”后本浏览器即作为客户端连接 · 数据到达即丢弃 · 断线自动重连</div>
</div>
<script>
function fmtBytes(n){
  if(n<1024) return n.toFixed(0)+' <span class="unit">B</span>';
  const u=['KB','MB','GB','TB','PB']; let i=-1;
  do{ n/=1024; i++; }while(n>=1024 && i<u.length-1);
  return n.toFixed(2)+' <span class="unit">'+u[i]+'</span>';
}
function fmtSpeed(bps){
  const bits=bps*8;
  if(bits<1000) return bits.toFixed(0)+' <span class="unit">bps</span>';
  const u=['Kbps','Mbps','Gbps','Tbps']; let v=bits/1000,i=0;
  while(v>=1000 && i<u.length-1){ v/=1000; i++; }
  return v.toFixed(2)+' <span class="unit">'+u[i]+'</span>';
}

// ---- 服务端统计轮询 ----
// 速率由累计字节的时间差算出，轮询间隔不均也能得到正确均值，避免缓冲导致的假高/卡顿。
let lastTotal=null, lastT=0;
async function tick(){
  try{
    const r=await fetch('/stats',{cache:'no-store'});
    const d=await r.json();
    const now=performance.now();
    if(lastTotal!==null && now>lastT){
      const bps=(d.total-lastTotal)/((now-lastT)/1000);
      document.getElementById('speed').innerHTML=fmtSpeed(Math.max(0,bps));
    }
    lastTotal=d.total; lastT=now;
    document.getElementById('monthly').innerHTML=fmtBytes(d.monthly);
    document.getElementById('total').innerHTML=fmtBytes(d.total);
    document.getElementById('month').textContent='('+d.month+')';
  }catch(e){}
}
tick(); setInterval(tick,1000);

// ---- 网页客户端：持续下载 /stream 并丢弃，断线自动重连 ----
let running=false;
let controllers=[];
let localBytes=0;

async function puller(){
  while(running){
    const ac=new AbortController();
    controllers.push(ac);
    try{
      const resp=await fetch('/stream?ts='+Date.now(),{cache:'no-store',signal:ac.signal});
      const reader=resp.body.getReader();
      while(running){
        const {done,value}=await reader.read();
        if(done) break;              // 服务端断开
        if(value) localBytes+=value.length;  // 计数后丢弃，不保存
      }
    }catch(e){
      // 网络中断/被 abort，稍后重连
    }finally{
      controllers=controllers.filter(c=>c!==ac);
    }
    if(running) await new Promise(r=>setTimeout(r,1000)); // 重连间隔
  }
}

function start(){
  if(running) return;
  running=true;
  let n=Math.max(1,Math.min(64,parseInt(document.getElementById('conns').value)||3));
  // 浏览器对同源 HTTP/1.1 最多约 6 个连接，留一个槽给面板刷新，避免速率显示卡住
  let eff=Math.min(n,5);
  for(let i=0;i<eff;i++) puller();
  document.getElementById('dot').classList.add('on');
  document.getElementById('state').textContent='运行中 · '+eff+' 连接'+(eff<n?'（浏览器上限已封顶）':'');
  const b=document.getElementById('toggle'); b.textContent='停止'; b.classList.add('stop');
}
function stop(){
  running=false;
  controllers.forEach(c=>{try{c.abort()}catch(e){}}); controllers=[];
  document.getElementById('dot').classList.remove('on');
  document.getElementById('state').textContent='已停止';
  document.getElementById('localSpeed').textContent='未运行';
  const b=document.getElementById('toggle'); b.textContent='开始刷流量'; b.classList.remove('stop');
}
document.getElementById('toggle').addEventListener('click',()=>running?stop():start());

// 每秒计算本地下载速率
setInterval(()=>{
  if(running){
    document.getElementById('localSpeed').innerHTML=fmtSpeed(localBytes);
  }
  localBytes=0;
},1000);
</script>
</body>
</html>`
