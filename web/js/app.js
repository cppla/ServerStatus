// 简洁现代前端 - 仅使用原生 JS
const S = { updated:0, servers:[], ssl:[], error:false, hist:{}, metricHist:{}, loadHist:{} };// hist latency; metricHist: {key:{cpu:[],mem:[],hdd:[]}}; loadHist: {key:[]}
const els = {
  notice: ()=>document.getElementById('notice'),
  last: ()=>document.getElementById('lastUpdate'),
  serversBody: ()=>document.getElementById('serversBody'),
  monitorsBody: ()=>document.getElementById('monitorsBody'),
  sslBody: ()=>document.getElementById('sslBody')
};

// (清理) 已移除 bytes / humanAuto 等未使用的通用进位函数
// 最小单位 MB：
function humanMinMBFromKB(kb){ if(kb==null||isNaN(kb)) return '-'; // 输入单位: KB
  let mb = kb/1000; const units=['MB','GB','TB','PB']; let i=0; while(mb>=1000 && i<units.length-1){ mb/=1000;i++; }
  const out = mb>=100? mb.toFixed(0): mb.toFixed(1); return out+units[i]; }
function humanMinMBFromMB(mbVal){ if(mbVal==null||isNaN(mbVal)) return '-'; // 输入单位: MB
  let v=mbVal; const units=['MB','GB','TB','PB']; let i=0; while(v>=1000 && i<units.length-1){ v/=1000;i++; }
  const out = v>=100? v.toFixed(0): v.toFixed(1); return out+units[i]; }
function humanMinMBFromB(bytes){ if(bytes==null||isNaN(bytes)) return '-'; // 输入单位: B
  let mb = bytes/1000/1000; const units=['MB','GB','TB','PB']; let i=0; while(mb>=1000 && i<units.length-1){ mb/=1000;i++; }
  const out = mb>=100? mb.toFixed(0): mb.toFixed(1); return out+units[i]; }
function humanRateMinMBFromB(bytes){ if(bytes==null||isNaN(bytes)) return '-'; if(bytes<=0) return '0.0MB'; return humanMinMBFromB(bytes); }
function humanMinKBFromB(bytes){ if(bytes==null||isNaN(bytes)) return '-'; // 输入单位: B; 最小单位 KB
  let kb = bytes/1000; const units=['KB','MB','GB','TB','PB']; let i=0; while(kb>=1000 && i<units.length-1){ kb/=1000; i++; }
  const out = kb>=100? kb.toFixed(0): kb.toFixed(1); return out+units[i]; }
// (清理) pct / clsBy 已不再使用
function humanAgo(ts){ if(!ts) return '-'; const s=Math.floor((Date.now()/1000 - ts)); const m=Math.floor(s/60); return m>0? m+' 分钟前':'几秒前'; }
function num(v){ return (typeof v==='number' && !isNaN(v)) ? v : '-'; }

// 将服务端上报的 os 映射为样式类名（用于为行/卡片着色）
function osClass(os){
  if(!os) return '';
  const v = String(os).toLowerCase();
  const pick = (k)=>' os-'+k;
  if(v.includes('ubuntu')) return pick('ubuntu');
  if(v.includes('debian')) return pick('debian');
  if(v.includes('centos')) return pick('centos');
  if(v.includes('rocky')) return pick('rocky');
  if(v.includes('alma')) return pick('almalinux');
  if(v.includes('arch')) return pick('arch');
  if(v.includes('alpine')) return pick('alpine');
  if(v.includes('fedora')) return pick('fedora');
  if(v.includes('rhel') || v.includes('redhat')) return pick('rhel');
  if(v.includes('suse')) return pick('suse');
  if(v.includes('amazon')) return pick('amazon');
  if(v.includes('freebsd')) return pick('freebsd');
  if(v.includes('openbsd')) return pick('openbsd');
  if(v.includes('netbsd') || v.includes('bsd')) return pick('bsd');
  if(v.includes('darwin') || v.includes('mac')) return pick('darwin');
  if(v.includes('win')) return pick('windows');
  if(v.includes('linux')) return pick('linux');
  return pick(v.replace(/[^a-z0-9_-]+/g,'-').slice(0,20));
}

async function fetchData(){
  try {
    const r = await fetch('json/stats.json?_='+Date.now());
    if(!r.ok) throw new Error(r.status);
    const j = await r.json();
    if(j.reload) location.reload();
  S.updated = j.updated; S.servers = j.servers||[]; S.ssl = j.sslcerts||[]; S.error=false;
  // 为每个服务器生成唯一 key（基于 name|location|type + 顺序号），避免同名节点写入同一历史
  const keyCount = Object.create(null);
  S.servers.forEach((s, idx)=>{
    const base = [s.name||'-', s.location||'-', s.type||'-'].join('|');
    const seq = (keyCount[base]||0) + 1; keyCount[base] = seq;
    const key = `${base}#${seq}`;
    s._key = key; // 挂到对象上，后续查找/弹窗均用它
      if(!S.hist[key]) S.hist[key] = {cu:[],ct:[],cm:[]};
      const H = S.hist[key];
      // 使用 time_ 字段 (ms) 若不存在则跳过
      if(typeof s.time_10010 === 'number') H.cu.push(s.time_10010);
      if(typeof s.time_189 === 'number') H.ct.push(s.time_189);
      if(typeof s.time_10086 === 'number') H.cm.push(s.time_10086);
  const MAX=120; // 保留最多 120 条
      ['cu','ct','cm'].forEach(k=>{ if(H[k].length>MAX) H[k].splice(0,H[k].length-MAX); });
      // 指标历史 (仅在线时记录)
      if(!S.metricHist[key]) S.metricHist[key] = {cpu:[],mem:[],hdd:[]};
      const MH = S.metricHist[key];
      if(s.online4||s.online6){
        const memPct = s.memory_total? (s.memory_used/s.memory_total*100):0;
        const hddPct = s.hdd_total? (s.hdd_used/s.hdd_total*100):0;
        MH.cpu.push(s.cpu||0);
        MH.mem.push(memPct||0);
        MH.hdd.push(hddPct||0);
        const MAXM=120; ['cpu','mem','hdd'].forEach(k=>{ if(MH[k].length>MAXM) MH[k].splice(0,MH[k].length-MAXM); });
      }
  // 负载历史 (记录 load_1 / load_5 / load_15)
  if(!S.loadHist[key]) S.loadHist[key] = {l1:[],l5:[],l15:[]};
  const LH = S.loadHist[key];
  const pushLoad = (arr,val)=>{ if(typeof val === 'number' && val >= 0){ arr.push(val); if(arr.length>120) arr.splice(0,arr.length-120); } };
  pushLoad(LH.l1, s.load_1);
  pushLoad(LH.l5, s.load_5);
  pushLoad(LH.l15, s.load_15);
    });
    render();
  }catch(e){ S.error=true; els.notice().textContent = '数据获取失败'; console.error(e); }
}

function render(){
  els.notice().style.display='none';
  renderServers();
  renderServersCards();
  renderMonitors();
  renderMonitorsCards();
  renderSSL();
  renderSSLCards();
  updateTime();
}
function renderServers(){
  const tbody = els.serversBody();
  let html='';
  S.servers.forEach((s,idx)=>{
    const online = s.online4||s.online6;
  const proto = online ? (s.online4 && s.online6? '双栈': s.online4? 'IPv4':'IPv6') : '离线';
  const statusPill = online ? `<span class="pill on">${proto}</span>` : `<span class="pill off">${proto}</span>`;
  const memPct = s.memory_total? (s.memory_used/s.memory_total*100):0;
  const hddPct = s.hdd_total? (s.hdd_used/s.hdd_total*100):0;
  const monthInBytes = (s.network_in - s.last_network_in) || 0; // 原始: B
  const monthOutBytes = (s.network_out - s.last_network_out) || 0;
  const monthIn = humanMinMBFromB(monthInBytes); // 最小单位 MB
  const monthOut = humanMinMBFromB(monthOutBytes);
  const HEAVY_THRESHOLD = 500 * 1000 * 1000 * 1000; // 500GB
  const heavy = monthInBytes >= HEAVY_THRESHOLD || monthOutBytes >= HEAVY_THRESHOLD;
  const trafficCls = heavy ? 'caps-traffic duo heavy' : 'caps-traffic duo normal';
    const netNow = humanMinKBFromB(s.network_rx) + ' | ' + humanMinKBFromB(s.network_tx); // 最小单位 KB
    const netTotal = humanMinMBFromB(s.network_in)+' | '+humanMinMBFromB(s.network_out); // 最小单位 MB
  const p1 = (s.ping_10010||0); const p2 = (s.ping_189||0); const p3 = (s.ping_10086||0);
  function bucket(p){ const v = Math.max(0, Math.min(100, p)); const level = v>=20?'bad':(v>=10?'warn':'ok'); return `<div class=\"bucket\" data-lv=\"${level}\"><span style=\"--h:${v}%\"></span><label>${v.toFixed(0)}%</label></div>`; }
  const pingBuckets = `<div class=\"buckets\" title=\"CU/CT/CM\">${bucket(p1)}${bucket(p2)}${bucket(p3)}</div>`;
  // 唯一 key 已附加为 s._key（如需使用）
  const rowCursor = online? 'pointer':'default';
    const highLoad = online && ( (s.cpu||0)>=90 || (memPct)>=90 || (hddPct)>=90 );
  html += `<tr data-idx="${idx}" data-online="${online?1:0}" class="row-server${highLoad?' high-load':''}${osClass(s.os)}" style="cursor:${rowCursor};${online?'':'opacity:.65;'}">
  <td>${statusPill}</td>
  <td><span class="${trafficCls}" title="本月下行 | 上行 (≥500GB 触发红黄)"><span class="half in">${monthIn}</span><span class="half out">${monthOut}</span></span></td>
      <td>${s.name||'-'}</td>
      <td>${s.type||'-'}</td>
      <td>${s.location||'-'}</td>
      <td>${s.uptime||'-'}</td>
  <td>${s.load_1==-1?'–':Math.max(0,(s.load_1||0)).toFixed(2)}</td>
      <td>${netNow}</td>
      <td>${netTotal}</td>
  <td>${online?gaugeHTML('cpu', s.cpu||0):'-'}</td>
  <td>${online?gaugeHTML('mem', memPct):'-'}</td>
  <td>${online?gaugeHTML('hdd', hddPct):'-'}</td>
  <td>${pingBuckets}</td>
    </tr>`;
  });
  tbody.innerHTML = html || `<tr><td colspan="13" class="muted" style="text-align:center;padding:1rem;">无数据</td></tr>`;

  // 绑定行点击
  tbody.querySelectorAll('tr.row-server').forEach(tr=>{
    tr.addEventListener('click',()=>{
      if(tr.getAttribute('data-online')!=='1') return; // 离线不弹出
      const i = parseInt(tr.getAttribute('data-idx'));
      openDetail(i);
    });
  });

  // 仪表盘无需历史 spark 小图
}
// 生成仪表盘 (圆形 conic-gradient)
function gaugeHTML(type,val){
  const pct = Math.max(0,Math.min(100,val));
  const p = (pct/100).toFixed(3);
  const warnAttr = pct>=90? 'data-bad' : (pct>=50? 'data-warn' : '');
    return `<div class="gauge-half" data-type="${type}" ${warnAttr} style="--p:${p}" title="${labelOf(type)} ${pct.toFixed(0)}%">
      <svg viewBox="0 0 100 50" preserveAspectRatio="xMidYMid meet" aria-hidden="true">
        <path class="track" d="M10 50 A40 40 0 0 1 90 50" />
        <path class="arc" d="M10 50 A40 40 0 0 1 90 50" />
      </svg>
      <span>${pct.toFixed(0)}%</span>
    </div>`;
}
function labelOf(t){ return t==='cpu'?'CPU': t==='mem'?'内存':'硬盘'; }
function renderServersCards(){
  const wrap = document.getElementById('serversCards');
  if(!wrap) return;
  // 仅在窄屏时显示 (和 CSS 一致判断, 可稍放宽避免闪烁)
  if(window.innerWidth>700){ wrap.innerHTML=''; return; }
  let html='';
  S.servers.forEach((s,idx)=>{
    const online = s.online4||s.online6;
    const proto = online ? (s.online4 && s.online6? '双栈': s.online4? 'IPv4':'IPv6') : '离线';
    const pill = `<span class="status-pill ${online?'on':'off'}">${proto}</span>`;
  const memPct = s.memory_total? (s.memory_used/s.memory_total*100):0;
  const hddPct = s.hdd_total? (s.hdd_used/s.hdd_total*100):0;
  // 月流量（移动端）并应用 500GB 阈值配色逻辑
  const monthInBytes = (s.network_in - s.last_network_in) || 0; // B
  const monthOutBytes = (s.network_out - s.last_network_out) || 0;
  const monthIn = humanMinMBFromB(monthInBytes);
  const monthOut = humanMinMBFromB(monthOutBytes);
  const HEAVY_THRESHOLD = 500 * 1000 * 1000 * 1000; // 500GB
  const heavy = monthInBytes >= HEAVY_THRESHOLD || monthOutBytes >= HEAVY_THRESHOLD;
  const trafficCls = heavy ? 'caps-traffic duo heavy sm' : 'caps-traffic duo normal sm';
    const netNow = humanMinKBFromB(s.network_rx)+' | '+humanMinKBFromB(s.network_tx);
    const netTotal = humanMinMBFromB(s.network_in)+' | '+humanMinMBFromB(s.network_out);
    const p1 = (s.ping_10010||0); const p2=(s.ping_189||0); const p3=(s.ping_10086||0);
    function bucket(p){ const v=Math.max(0,Math.min(100,p)); const level = v>=20?'bad':(v>=10?'warn':'ok'); return `<div class=\"bucket\" data-lv=\"${level}\"><span style=\"--h:${v}%\"></span><label>${v.toFixed(0)}%</label></div>`; }
    const buckets = `<div class=\"buckets\">${bucket(p1)}${bucket(p2)}${bucket(p3)}</div>`;
  // 唯一 key 已附加为 s._key（如需使用）
  const highLoad = online && ( (s.cpu||0)>=90 || (memPct)>=90 || (hddPct)>=90 );
  html += `<div class=\"card${online?'':' offline'}${highLoad?' high-load':''}${osClass(s.os)}\" data-idx=\"${idx}\" data-online=\"${online?1:0}\">\n      <button class=\"expand-btn\" aria-label=\"展开\">▼</button>\n      <div class=\"card-header\">\n        <div class=\"card-title\">${s.name||'-'} <span class=\"tag\">${s.location||'-'}</span></div>\n        ${pill}\n      </div>\n      <div class=\"kvlist\">\n        <div><span class=\"key\">负载</span><span>${s.load_1==-1?'–':s.load_1?.toFixed(2)}</span></div>\n        <div><span class=\"key\">在线</span><span>${s.uptime||'-'}</span></div>\n        <div><span class=\"key\">月流量</span><span><span class=\"${trafficCls}\" title=\"本月下行 | 上行 (≥500GB 触发红黄)\"><span class=\"half in\">${monthIn}</span><span class=\"half out\">${monthOut}</span></span></span></div>\n        <div><span class=\"key\">网络</span><span>${netNow}</span></div>\n        <div><span class=\"key\">总流量</span><span>${netTotal}</span></div>\n        <div><span class=\"key\">CPU</span><span>${s.cpu||0}%</span></div>\n        <div><span class=\"key\">内存</span><span>${memPct.toFixed(0)}%</span></div>\n        <div><span class=\"key\">硬盘</span><span>${hddPct.toFixed(0)}%</span></div>\n      </div>\n      ${buckets}\n      <div class=\"expand-area\">\n        <div style=\"font-size:.65rem;opacity:.7;margin-top:.3rem\">${online?'点击卡片可查看详情':'离线，不可查看详情'}</div>\n      </div>\n    </div>`;
  });
  wrap.innerHTML = html || '<div class="muted" style="font-size:.75rem;text-align:center;padding:1rem;">无数据</div>';
  wrap.querySelectorAll('.card').forEach(card=>{
    const idx = parseInt(card.getAttribute('data-idx'));
    card.addEventListener('click', e=>{ 
      if(e.target.classList.contains('expand-btn')){ card.classList.toggle('expanded'); e.stopPropagation(); return;}
      if(card.getAttribute('data-online')!=='1') return; // 离线不弹
      openDetail(idx); 
    });
  });
}

function renderMonitors(){
  const tbody = els.monitorsBody();
  let html='';
  S.servers.forEach(s=>{
    html += `<tr>
      <td>${(s.online4||s.online6)?'在线':'离线'}</td>
      <td>${s.name||'-'}</td>
      <td>${s.location||'-'}</td>
      <td>${s.custom||'-'}</td>
    </tr>`;
  });
  tbody.innerHTML = html || `<tr><td colspan="4" class="muted" style="text-align:center;padding:1rem;">无数据</td></tr>`;
}

// 服务卡片 (移动端)
function renderMonitorsCards(){
  const wrap = document.getElementById('monitorsCards');
  if(!wrap) return; if(window.innerWidth>700){ wrap.innerHTML=''; return; }
  let html='';
  S.servers.forEach(s=>{
    const online = (s.online4||s.online6)?'在线':'离线';
    const pill = `<span class="status-pill ${online==='在线'?'on':'off'}">${online}</span>`;
    html += `<div class="card">
      <div class="card-header"><div class="card-title">${s.name||'-'} <span class="tag">${s.location||'-'}</span></div>${pill}</div>
      <div class="kvlist" style="grid-template-columns:repeat(2,minmax(0,1fr));">
        <div><span class="key">监测内容</span><span>${s.custom||'-'}</span></div>
        <div><span class="key">协议</span><span>${online}</span></div>
      </div>
    </div>`;
  });
  wrap.innerHTML = html || '<div class="muted" style="font-size:.75rem;text-align:center;padding:1rem;">无数据</div>';
}

function renderSSL(){
  const tbody = els.sslBody();
  let html='';
  S.ssl.forEach(c=>{
    const cls = c.expire_days<=0? 'err': c.expire_days<=7? 'warn':'ok';
    const status = c.expire_days<=0? '已过期': c.expire_days<=7? '将到期':'正常';
    const dt = c.expire_ts? new Date(c.expire_ts*1000).toISOString().replace('T',' ').replace(/\.\d+Z/,''):'-';
    html += `<tr>
      <td>${c.name||'-'}</td>
      <td>${(c.domain||'').replace(/^https?:\/\//,'')}</td>
      <td>${c.port||443}</td>
      <td><span class="badge ${cls}">${c.expire_days??'-'}</span></td>
      <td>${dt}</td>
      <td><span class="badge ${cls}">${status}</span></td>
    </tr>`;
  });
  tbody.innerHTML = html || `<tr><td colspan="6" class="muted" style="text-align:center;padding:1rem;">无证书数据</td></tr>`;
}

// 证书卡片 (移动端)
function renderSSLCards(){
  const wrap = document.getElementById('sslCards');
  if(!wrap) return; if(window.innerWidth>700){ wrap.innerHTML=''; return; }
  let html='';
  S.ssl.forEach(c=>{
    const cls = c.expire_days<=0? 'err': c.expire_days<=7? 'warn':'ok';
    const status = c.expire_days<=0? '已过期': c.expire_days<=7? '将到期':'正常';
    const dt = c.expire_ts? new Date(c.expire_ts*1000).toISOString().replace('T',' ').replace(/\.\d+Z/,''):'-';
    html += `<div class="card">
      <div class="card-header"><div class="card-title">${c.name||'-'}</div><span class="status-pill ${cls==='err'?'off':'on'}">${status}</span></div>
      <div class="kvlist" style="grid-template-columns:repeat(2,minmax(0,1fr));">
        <div><span class="key">域名</span><span>${(c.domain||'').replace(/^https?:\/\//,'')}</span></div>
        <div><span class="key">端口</span><span>${c.port||443}</span></div>
        <div><span class="key">剩余(天)</span><span>${c.expire_days??'-'}</span></div>
        <div><span class="key">到期</span><span>${dt.split(' ')[0]||dt}</span></div>
      </div>
    </div>`;
  });
  wrap.innerHTML = html || '<div class="muted" style="font-size:.75rem;text-align:center;padding:1rem;">无证书数据</div>';
}

function updateTime(){
  const el = els.last();
  if(S.updated){ el.textContent = '最后更新: '+ humanAgo(S.updated); }
}

function bindTabs(){
  document.getElementById('navTabs').addEventListener('click',e=>{
    if(e.target.tagName!=='BUTTON') return; const tab=e.target.dataset.tab; 
    document.querySelectorAll('.nav button').forEach(b=>b.classList.toggle('active',b===e.target));
    document.querySelectorAll('.panel').forEach(p=>p.classList.toggle('active', p.id==='panel-'+tab));
  });
}
function bindTheme(){
  const btn = document.getElementById('themeToggle');
  const mql = window.matchMedia('(prefers-color-scheme: light)');
  const saved = localStorage.getItem('theme'); // 'light' | 'dark' | null (auto)

  const apply = (isLight)=>{ document.body.classList.toggle('light', isLight); document.documentElement.classList.toggle('light', isLight); };

  if(!saved){
    // 自动跟随系统
    apply(mql.matches);
    // 监听系统偏好变化（仅在未手动选择时）
    mql.addEventListener('change', e=>{ if(!localStorage.getItem('theme')) apply(e.matches); });
  } else {
    apply(saved==='light');
  }

  btn.addEventListener('click',()=>{
    // 用户手动切换后即固定，不再自动
    const toLight = !document.body.classList.contains('light');
    apply(toLight);
    localStorage.setItem('theme', toLight?'light':'dark');
  });
}

bindTabs();
bindTheme();
fetchData();
setInterval(fetchData, 1000);
setInterval(updateTime, 60000);

// 详情弹窗逻辑
function openDetail(i){
  const s = S.servers[i]; if(!s) return;
  const box = document.getElementById('detailContent');
  const modal = document.getElementById('detailModal');
  document.getElementById('detailTitle').textContent = s.name + ' 详情';
  const offline = !(s.online4||s.online6);
  const memPct = s.memory_total? (s.memory_used/s.memory_total*100):0;
  const swapPct = s.swap_total? (s.swap_used/s.swap_total*100):0;
  const hddPct = s.hdd_total? (s.hdd_used/s.hdd_total*100):0;
  const ioRead = (typeof s.io_read==='number')? s.io_read:0;
  const ioWrite = (typeof s.io_write==='number')? s.io_write:0;
  const procLine = `${num(s.tcp_count)} / ${num(s.udp_count)} / ${num(s.process_count)} / ${num(s.thread_count)}`;
  // 保留延迟数据用于图表，但不再展示当前延迟文字行
  const latText = offline ? '离线' : `CU/CT/CM: ${num(s.time_10010)}ms (${(s.ping_10010||0).toFixed(0)}%) / ${num(s.time_189)}ms (${(s.ping_189||0).toFixed(0)}%) / ${num(s.time_10086)}ms (${(s.ping_10086||0).toFixed(0)}%)`;
  const key = s._key || [s.name||'-', s.location||'-', s.type||'-'].join('|')+'#1';

  let latencyBlock = '';
  if(!offline){
    latencyBlock = `
    <div style="display:flex;flex-direction:column;gap:.4rem;">
      <canvas id="latChart" height="150" style="width:100%;border:1px solid var(--border);border-radius:10px;background:linear-gradient(145deg,var(--bg),var(--bg-alt));"></canvas>
      <div class="mono" style="font-size:11px;display:flex;gap:1rem;flex-wrap:wrap;">
        <span style="color:#3b82f6">● 联通 (<span id="lat-cu">${num(s.time_10010)}ms</span>)</span>
        <span style="color:#10b981">● 电信 (<span id="lat-ct">${num(s.time_189)}ms</span>)</span>
        <span style="color:#f59e0b">● 移动 (<span id="lat-cm">${num(s.time_10086)}ms</span>)</span>
  <span style="opacity:.6"> (~<span id="lat-count">${(S.hist[key]?Math.max(S.hist[key].cu.length, S.hist[key].ct.length, S.hist[key].cm.length):0)}</span> 条)</span>
      </div>
    </div>`;
  } else {
    latencyBlock = `
    <div style="display:flex;flex-direction:column;gap:.4rem;">
      <canvas id="latChart" height="150" style="width:100%;border:1px solid var(--border);border-radius:10px;background:linear-gradient(145deg,var(--bg),var(--bg-alt));"></canvas>
      <div class="mono" style="font-size:11px;opacity:.6;">离线，无联通/电信/移动延迟数据</div>
    </div>`;
  }

  // 旧进度条函数 barHTML/ioBar 已弃用
  // 资源行（移除百分比显示，仅显示 已用 / 总量）
  // 资源行（单独拆分 span 便于后续动态刷新）
  // 单位来源：memory/swap: KB; hdd: MB; io: B (速率) -> 统一最小单位显示为 MB，并向上进位 (MB/GB/TB)
  const memUsed = s.memory_total!=null? humanMinMBFromKB(s.memory_used||0):'-';
  const memTotal = s.memory_total!=null? humanMinMBFromKB(s.memory_total):'-';
  const swapUsed = s.swap_total!=null? humanMinMBFromKB(s.swap_used||0):'-';
  const swapTotal = s.swap_total!=null? humanMinMBFromKB(s.swap_total):'-';
  const hddUsed = s.hdd_total!=null? humanMinMBFromMB(s.hdd_used||0):'-';
  const hddTotal = s.hdd_total!=null? humanMinMBFromMB(s.hdd_total):'-';
  const ioReadLine = (ioRead!=null)? humanRateMinMBFromB(ioRead):'-';
  const ioWriteLine = (ioWrite!=null)? humanRateMinMBFromB(ioWrite):'-';
  const memColor = memPct>80? ' style="color:var(--danger)"':''; // 已用/总量显示为单一块，所以对已用着色
  const swapColor = swapPct>80? ' style="color:var(--danger)"':'';
  const hddColor = hddPct>80? ' style="color:var(--danger)"':'';
  // IO 阈值：>100MB (原始单位 B) -> >100*1000*1000 B
  const readColor = ioRead>100*1000*1000? ' style="color:var(--danger)"':'';
  const writeColor = ioWrite>100*1000*1000? ' style="color:var(--danger)"':'';
  box.innerHTML = `
    <div class="kv"><span>TCP/UDP/进/线</span><span class="mono" id="detail-proc">${procLine}</span></div>
  <div class="kv"><span>内存 / 虚存</span><span class="mono">
    <span id="mem-line"${memColor}><span id="mem-used">${memUsed}</span> / <span id="mem-total">${memTotal}</span></span>
    | <span id="swap-line"${swapColor}><span id="swap-used">${swapUsed}</span> / <span id="swap-total">${swapTotal}</span></span>
  </span></div>
  <div class="kv"><span>硬盘 / 读写</span><span class="mono">
    <span id="disk-line"${hddColor}><span id="hdd-used">${hddUsed}</span> / <span id="hdd-total">${hddTotal}</span></span>
    | <span id="io-read"${readColor}>${ioReadLine}</span> / <span id="io-write"${writeColor}>${ioWriteLine}</span>
  </span></div>
    <div style="display:flex;flex-direction:column;gap:.35rem;">
      <canvas id="loadChart" height="120" style="width:100%;border:1px solid var(--border);border-radius:10px;background:linear-gradient(145deg,var(--bg),var(--bg-alt));"></canvas>
      <div class="mono" style="font-size:11px;display:flex;gap:.9rem;flex-wrap:wrap;align-items:center;opacity:.8;">
        <span style="color:#8b5cf6">● load1 (<span id="load1-val">${s.load_1==-1?'–':Math.max(0,(s.load_1||0)).toFixed(2)}</span>)</span>
        <span style="color:#10b981">● load5 (<span id="load5-val">${s.load_5==-1?'–':Math.max(0,(s.load_5||0)).toFixed(2)}</span>)</span>
        <span style="color:#f59e0b">● load15 (<span id="load15-val">${s.load_15==-1?'–':Math.max(0,(s.load_15||0)).toFixed(2)}</span>)</span>
  <span style="opacity:.6">(~<span id="load-count">${(S.loadHist[key]?Math.max(S.loadHist[key].l1.length, S.loadHist[key].l5.length, S.loadHist[key].l15.length):0)}</span> 条)</span>
      </div>
    </div>
  <!-- 进度条移除：读/写/虚存以文本形式显示于上方合并行 -->
    ${latencyBlock}
  `;
  modal.style.display='flex';
  document.addEventListener('keydown', escCloseOnce);
  if(!offline){
    drawLatencyChart(key);
    drawLoadChart(key);
  S._openDetailKey = key; // 记录当前弹窗对应节点（唯一 key）
    startDetailAutoUpdate();
  } else {
    S._openDetailKey = null;
    stopDetailAutoUpdate();
  }
}
function escCloseOnce(e){ if(e.key==='Escape'){ closeDetail(); } }
function closeDetail(){ const m=document.getElementById('detailModal'); m.style.display='none'; document.removeEventListener('keydown', escCloseOnce); stopDetailAutoUpdate(); }
document.getElementById('detailClose').addEventListener('click', closeDetail);
document.getElementById('detailModal').addEventListener('click', e=>{ if(e.target.id==='detailModal') closeDetail(); });

// 绘制三网延迟折线图 (简易实现)
function drawLatencyChart(key){
  const data = S.hist[key];
  const canvas = document.getElementById('latChart');
  if(!canvas || !data) return;
  const ctx = canvas.getContext('2d');
  const W = canvas.clientWidth; const H = canvas.height; canvas.width = W; // 适配宽度
  ctx.clearRect(0,0,W,H);
  const padL=40, padR=10, padT=10, padB=18;
  const isLight = document.body.classList.contains('light');
  const axisColor = isLight? 'rgba(0,0,0,0.22)' : 'rgba(255,255,255,0.18)';
  const gridColor = isLight? 'rgba(0,0,0,0.08)' : 'rgba(255,255,255,0.10)';
  const textColor = isLight? 'var(--text-dim)' : 'rgba(226,232,240,0.85)';
  const series = [ {arr:data.cu,color:'#3b82f6'}, {arr:data.ct,color:'#10b981'}, {arr:data.cm,color:'#f59e0b'} ];
  const allVals = series.flatMap(s=>s.arr);
  if(!allVals.length){ ctx.fillStyle='var(--text-dim)'; ctx.font='12px system-ui'; ctx.fillText('暂无数据', W/2-30, H/2); return; }
  const max = Math.max(...allVals);
  const min = Math.min(...allVals);
  const range = Math.max(1, max-min);
  const n = Math.max(...series.map(s=>s.arr.length));
  const xStep = (W - padL - padR) / Math.max(1,n-1);
  // 网格与轴 (增强暗色对比)
  ctx.strokeStyle=axisColor; ctx.lineWidth=1.1;
  ctx.beginPath(); ctx.moveTo(padL,padT); ctx.lineTo(padL,H-padB); ctx.lineTo(W-padR,H-padB); ctx.stroke();
  ctx.fillStyle=textColor; ctx.font='10px system-ui';
  const yMarks=4; for(let i=0;i<=yMarks;i++){ const y = padT + (H-padT-padB)*i/yMarks; const val = (max - range*i/yMarks).toFixed(0)+'ms'; ctx.fillText(val,4,y+3); ctx.strokeStyle=gridColor; ctx.beginPath(); ctx.moveTo(padL,y); ctx.lineTo(W-padR,y); ctx.stroke(); }
  // 绘制线
  series.forEach(s=>{
    if(s.arr.length<2) return;
    ctx.strokeStyle = s.color; ctx.lineWidth=1.6; ctx.beginPath();
    s.arr.forEach((v,i)=>{ const x = padL + xStep*i; const y = padT + (H-padT-padB)*(1-(v-min)/range); if(i===0) ctx.moveTo(x,y); else ctx.lineTo(x,y); });
    ctx.stroke();
  });
}

// 在每次 render 后若弹窗打开则重绘最新图
const _oldRender = render;
render = function(){ _oldRender(); if(S._openDetailKey){ drawLatencyChart(S._openDetailKey); drawLoadChart(S._openDetailKey); } };
window.addEventListener('resize', ()=>{ 
  if(S._openDetailKey){ drawLatencyChart(S._openDetailKey); drawLoadChart(S._openDetailKey); }
  renderServersCards();
  renderMonitorsCards();
  renderSSLCards();
});

// (清理) drawSparks 已移除

// 负载折线图 (load1 历史)
function drawLoadChart(key){
  const L = S.loadHist[key];
  const canvas = document.getElementById('loadChart');
  if(!canvas) return; const ctx = canvas.getContext('2d');
  if(!L){ ctx.clearRect(0,0,canvas.width,canvas.height); return; }
  const l1=L.l1||[], l5=L.l5||[], l15=L.l15||[];
  const canvasW = canvas.clientWidth; const H = canvas.height; canvas.width = canvasW; const W=canvasW;
  ctx.clearRect(0,0,W,H);
  if(l1.length<2){ ctx.fillStyle='var(--text-dim)'; ctx.font='12px system-ui'; ctx.fillText('暂无负载数据', W/2-42, H/2); return; }
  const all = [...l1,...l5,...l15];
  const padL=38,padR=8,padT=8,padB=16;
  // 修正：纵轴下限不小于 0，且当真实 range <0.5 时向上扩展 max 而不是向下产生负刻度
  const rawMax = all.length? Math.max(...all):0;
  const rawMin = all.length? Math.min(...all):0;
  const min = 0; // 我们只显示 >=0
  let max = Math.max(rawMax,0);
  let range = max - min;
  const MIN_RANGE = 0.5;
  if(range < MIN_RANGE){ max = MIN_RANGE; range = MIN_RANGE; }
  const n = Math.max(l1.length,l5.length,l15.length); const xStep=(W-padL-padR)/Math.max(1,n-1);
  const isLight = document.body.classList.contains('light');
  const axisColor = isLight? 'rgba(0,0,0,0.22)' : 'rgba(255,255,255,0.18)';
  const gridColor = isLight? 'rgba(0,0,0,0.08)' : 'rgba(255,255,255,0.10)';
  const textColor = isLight? 'var(--text-dim)' : 'rgba(226,232,240,0.85)';
  // 轴 & 网格 (增强暗色对比)
  ctx.strokeStyle=axisColor; ctx.lineWidth=1.1; ctx.beginPath(); ctx.moveTo(padL,padT); ctx.lineTo(padL,H-padB); ctx.lineTo(W-padR,H-padB); ctx.stroke();
  ctx.fillStyle=textColor; ctx.font='10px system-ui';
  const yMarks=4; for(let i=0;i<=yMarks;i++){
    const y=padT+(H-padT-padB)*i/yMarks;
    const val=(max - range*i/yMarks); // top -> bottom
    const labelVal = (Math.abs(val) < 0.005 ? 0 : val).toFixed(2);
    ctx.fillText(labelVal,4,y+3);
    ctx.strokeStyle=gridColor; ctx.beginPath(); ctx.moveTo(padL,y); ctx.lineTo(W-padR,y); ctx.stroke();
  }
  const series=[{arr:l1,color:'#8b5cf6',fill:true},{arr:l5,color:'#10b981'},{arr:l15,color:'#f59e0b'}];
  // 面积先画 load1
  series.forEach(s=>{
    if(s.arr.length<2) return;
    ctx.beginPath(); ctx.lineWidth=1.5; ctx.strokeStyle=s.color;
  s.arr.forEach((v,i)=>{ const vClamped = Math.max(0, v); const x=padL+xStep*i; const y=padT+(H-padT-padB)*(1-(vClamped-min)/range); if(i===0) ctx.moveTo(x,y); else ctx.lineTo(x,y); });
    ctx.stroke();
    if(s.fill){
      const lastX = padL + xStep*(s.arr.length-1);
      ctx.lineTo(lastX,H-padB); ctx.lineTo(padL,H-padB); ctx.closePath();
      const grd = ctx.createLinearGradient(0,padT,0,H-padB); grd.addColorStop(0,'rgba(139,92,246,0.25)'); grd.addColorStop(1,'rgba(139,92,246,0)');
      ctx.fillStyle=grd; ctx.fill();
    }
  });
}

//# sourceMappingURL=app.js.map

// ====== 详情动态刷新 ======
function findServerByKey(key){ return S.servers.find(x=> (x._key)===key); }
function updateDetailMetrics(key){
  const s = findServerByKey(key); if(!s) return; if(!(s.online4||s.online6)) return; // 离线不更新
    const procLine = `${num(s.tcp_count)} / ${num(s.udp_count)} / ${num(s.process_count)} / ${num(s.thread_count)}`;
    const procEl = document.getElementById('detail-proc'); if(procEl) procEl.textContent = procLine;
    const cuEl=document.getElementById('lat-cu'); if(cuEl) cuEl.textContent = num(s.time_10010)+'ms';
    const ctEl=document.getElementById('lat-ct'); if(ctEl) ctEl.textContent = num(s.time_189)+'ms';
    const cmEl=document.getElementById('lat-cm'); if(cmEl) cmEl.textContent = num(s.time_10086)+'ms';
  // 延迟动态刷新 (若存在)
  const cuE1=document.getElementById('lat-cu'); if(cuE1) cuE1.textContent = num(s.time_10010)+'ms';
  const ctE1=document.getElementById('lat-ct'); if(ctE1) ctE1.textContent = num(s.time_189)+'ms';
  const cmE1=document.getElementById('lat-cm'); if(cmE1) cmE1.textContent = num(s.time_10086)+'ms';
  // 刷新联通/电信/移动历史计数（取三者最大长度）
  const latCntEl = document.getElementById('lat-count');
  if(latCntEl){
    const H = S.hist[key];
    const n = H ? Math.max(H.cu.length||0, H.ct.length||0, H.cm.length||0) : 0;
    latCntEl.textContent = n;
  }
  // 资源动态刷新
  const memLineEl = document.getElementById('mem-line');
  if(memLineEl){
    const pct = s.memory_total? (s.memory_used/s.memory_total*100):0;
  document.getElementById('mem-used').textContent = s.memory_total!=null? humanMinMBFromKB(s.memory_used||0):'-';
  document.getElementById('mem-total').textContent = s.memory_total!=null? humanMinMBFromKB(s.memory_total):'-';
    if(pct>80) memLineEl.style.color='var(--danger)'; else memLineEl.style.color='';
  }
  const swapLineEl = document.getElementById('swap-line');
  if(swapLineEl){
    const pct = s.swap_total? (s.swap_used/s.swap_total*100):0;
  document.getElementById('swap-used').textContent = s.swap_total!=null? humanMinMBFromKB(s.swap_used||0):'-';
  document.getElementById('swap-total').textContent = s.swap_total!=null? humanMinMBFromKB(s.swap_total):'-';
    if(pct>80) swapLineEl.style.color='var(--danger)'; else swapLineEl.style.color='';
  }
  const diskLineEl = document.getElementById('disk-line');
  if(diskLineEl){
    const pct = s.hdd_total? (s.hdd_used/s.hdd_total*100):0;
  document.getElementById('hdd-used').textContent = s.hdd_total!=null? humanMinMBFromMB(s.hdd_used||0):'-';
  document.getElementById('hdd-total').textContent = s.hdd_total!=null? humanMinMBFromMB(s.hdd_total):'-';
    if(pct>80) diskLineEl.style.color='var(--danger)'; else diskLineEl.style.color='';
  }
  const ioReadEl = document.getElementById('io-read'); if(ioReadEl){ const v = (typeof s.io_read==='number')? s.io_read:0; ioReadEl.textContent = humanRateMinMBFromB(v); ioReadEl.style.color = v>100*1000*1000? 'var(--danger)':''; }
  const ioWriteEl = document.getElementById('io-write'); if(ioWriteEl){ const v = (typeof s.io_write==='number')? s.io_write:0; ioWriteEl.textContent = humanRateMinMBFromB(v); ioWriteEl.style.color = v>100*1000*1000? 'var(--danger)':''; }
  // 动态刷新负载标签与条数
  const l1El = document.getElementById('load1-val'); if(l1El) l1El.textContent = s.load_1==-1?'–':Math.max(0,(s.load_1||0)).toFixed(2);
  const l5El = document.getElementById('load5-val'); if(l5El) l5El.textContent = s.load_5==-1?'–':Math.max(0,(s.load_5||0)).toFixed(2);
  const l15El = document.getElementById('load15-val'); if(l15El) l15El.textContent = s.load_15==-1?'–':Math.max(0,(s.load_15||0)).toFixed(2);
  const cntEl = document.getElementById('load-count');
  if(cntEl){ const L=S.loadHist[key]; const n = L? Math.max(L.l1.length, L.l5.length, L.l15.length):0; cntEl.textContent = n; }
}
function startDetailAutoUpdate(){ stopDetailAutoUpdate(); S._detailTimer = setInterval(()=>{ if(S._openDetailKey) updateDetailMetrics(S._openDetailKey); }, 1000); }
function stopDetailAutoUpdate(){ if(S._detailTimer){ clearInterval(S._detailTimer); S._detailTimer=null; } }
