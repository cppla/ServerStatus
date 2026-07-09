const S = {
  updated: 0,
  servers: [],
  ssl: [],
  hist: {},
  loadHist: {},
  openDetailKey: null,
  activeTab: 'servers',
  layoutCompact: null,
  osOptionsSignature: '',
  suppressStatsReloadUntil: 0,
  filters: { query: '', status: 'all', os: 'all', sort: 'name', dir: 'desc' },
  admin: {
    token: localStorage.getItem('serverstatusAdminToken') || '',
    enabled: false,
    connected: false,
    config: null,
    selectedType: 'servers',
    selectedIndex: -1,
    saving: false
  }
};

const $ = (id) => document.getElementById(id);
const esc = (v) => String(v ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const clamp = (v, min, max) => Math.max(min, Math.min(max, v));
const num = (v) => typeof v === 'number' && Number.isFinite(v) ? v : 0;

function debounce(fn, wait){
  let timer = 0;
  const wrapped = (...args) => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      timer = 0;
      fn(...args);
    }, wait);
  };
  wrapped.cancel = () => {
    clearTimeout(timer);
    timer = 0;
  };
  return wrapped;
}
function isCompactLayout(){ return window.innerWidth <= 700; }
function clearHTML(id){
  const el = $(id);
  if(el && el.innerHTML) el.innerHTML = '';
}
function htmlElement(html){
  const tpl = document.createElement('template');
  tpl.innerHTML = html.trim();
  return tpl.content.firstElementChild;
}

function humanMinMBFromKB(kb){ return humanBytes(num(kb) * 1000, 1000 * 1000); }
function humanMinMBFromMB(mb){ return humanBytes(num(mb) * 1000 * 1000, 1000 * 1000); }
function humanMinMBFromB(bytes){ return humanBytes(num(bytes), 1000 * 1000); }
function humanMinKBFromB(bytes){ return humanBytes(num(bytes), 1000); }
function humanBytes(bytes, minUnit){
  if(!Number.isFinite(bytes)) return '-';
  const units = ['B','KB','MB','GB','TB','PB'];
  let index = minUnit >= 1000 * 1000 ? 2 : minUnit >= 1000 ? 1 : 0;
  let divisor = Math.pow(1000, index);
  let value = Math.max(0, bytes) / divisor;
  while(value >= 1000 && index < units.length - 1){ value /= 1000; index++; }
  const out = value >= 100 ? value.toFixed(0) : value.toFixed(1);
  return out + units[index];
}
function cpuCores(s){
  const value = Number(s?.cpu_cores ?? s?.cpu_core ?? s?.cpu_count ?? s?.cores ?? 0);
  return Number.isFinite(value) && value > 0 ? Math.round(value) : 0;
}
function cpuCoreLabel(s){
  const cores = cpuCores(s);
  return cores > 0 ? `${cores}C` : '';
}
function memoryGbLabel(s){
  const kb = Number(s?.memory_total ?? 0);
  if(!Number.isFinite(kb) || kb <= 0) return '';
  return `${Math.max(1, Math.round(kb / 1024 / 1024))}G`;
}
function serverSpecLabel(s){
  const cores = cpuCoreLabel(s);
  if(!cores) return '';
  return `${cores}${memoryGbLabel(s)}`;
}
function cpuModelLabel(s){
  return String(s?.cpu_model || '').trim();
}
function humanAgo(ts){
  if(!ts) return '-';
  const sec = Math.max(0, Math.floor(Date.now() / 1000 - Number(ts)));
  if(sec < 60) return '几秒前';
  const min = Math.floor(sec / 60);
  if(min < 60) return `${min} 分钟前`;
  return `${Math.floor(min / 60)} 小时前`;
}

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
  if(v.includes('darwin') || v.includes('macos') || v.includes('os x') || v.includes('osx') || v.includes('apple') || v.includes('mac')) return pick('darwin');
  if(v.includes('win')) return pick('windows');
  if(v.includes('linux')) return pick('linux');
  return pick(v.replace(/[^a-z0-9_-]+/g,'-').slice(0,20));
}
function osLabel(os){
  if(!os) return '';
  const v = String(os).toLowerCase();
  if(v.includes('ubuntu')) return 'Ubuntu';
  if(v.includes('debian')) return 'Debian';
  if(v.includes('centos')) return 'CentOS';
  if(v.includes('rocky')) return 'Rocky Linux';
  if(v.includes('alma')) return 'AlmaLinux';
  if(v.includes('rhel') || v.includes('redhat')) return 'Red Hat Enterprise Linux';
  if(v.includes('arch')) return 'Arch Linux';
  if(v.includes('alpine')) return 'Alpine Linux';
  if(v.includes('fedora')) return 'Fedora';
  if(v.includes('amazon')) return 'Amazon Linux';
  if(v.includes('suse')) return 'SUSE Linux';
  if(v.includes('freebsd')) return 'FreeBSD';
  if(v.includes('openbsd')) return 'OpenBSD';
  if(v.includes('netbsd') || v.includes('bsd')) return 'BSD';
  if(v.includes('darwin') || v.includes('macos') || v.includes('os x') || v.includes('osx') || v.includes('apple') || v.includes('mac')) return 'macOS';
  if(v.includes('win')) return 'Windows';
  if(v.includes('linux')) return 'Linux';
  return String(os).charAt(0).toUpperCase() + String(os).slice(1);
}

function lossValues(s){
  return [s.ping_10010, s.ping_189, s.ping_10086].map(p => clamp(num(p), 0, 100));
}

function metrics(s){
  const online = !!(s.online4 || s.online6);
  const memPct = s.memory_total ? s.memory_used / s.memory_total * 100 : 0;
  const hddPct = s.hdd_total ? s.hdd_used / s.hdd_total * 100 : 0;
  const monthIn = Math.max(0, num(s.network_in) - num(s.last_network_in));
  const monthOut = Math.max(0, num(s.network_out) - num(s.last_network_out));
  const traffic = monthIn + monthOut;
  const losses = lossValues(s);
  const loss = Math.max(...losses);
  const lossBadCount = losses.filter(p => p >= 40).length;
  const lossWarnCount = losses.filter(p => p >= 30).length;
  const blocked = online && losses.every(p => p >= 100);
  const resourceCritical = online && (num(s.cpu) >= 90 || memPct >= 90 || hddPct >= 90);
  const resourceWarning = online && (num(s.cpu) >= 75 || memPct >= 80 || hddPct >= 85);
  const trafficWarning = online && traffic >= 1000 * 1000 * 1000 * 1000;
  const lossCritical = online && loss >= 40;
  const lossWarning = online && !lossCritical && loss >= 30;
  const critical = online && (resourceCritical || blocked || lossCritical);
  const warning = online && (resourceWarning || lossWarning);
  const rowLevel = !online || critical ? 'critical' : (resourceWarning || lossWarning ? 'warning' : '');
  return { online, memPct, hddPct, monthIn, monthOut, traffic, loss, lossBadCount, lossWarnCount, blocked, resourceCritical, resourceWarning, trafficWarning, lossCritical, lossWarning, critical, warning, alert: critical || warning, highlight: !!rowLevel, rowLevel };
}

async function fetchData(){
  try{
    const res = await fetch('json/stats.json?_=' + Date.now(), { cache: 'no-store' });
    if(!res.ok) throw new Error(String(res.status));
    const data = await res.json();
    if(data.reload && !shouldSuppressStatsReload()) {
      location.reload();
      return;
    }
    S.updated = Number(data.updated || 0);
    S.servers = (data.servers || []).map((server, index) => {
      const base = [server.name || '-', server.location || '-', server.type || '-'].join('|');
      const key = `${base}#${index + 1}`;
      server._key = key;
      if(!S.hist[key]) S.hist[key] = { cu: [], ct: [], cm: [] };
      if(!S.loadHist[key]) S.loadHist[key] = { l1: [], l5: [], l15: [] };
      pushHistory(S.hist[key].cu, server.time_10010);
      pushHistory(S.hist[key].ct, server.time_189);
      pushHistory(S.hist[key].cm, server.time_10086);
      pushHistory(S.loadHist[key].l1, server.load_1);
      pushHistory(S.loadHist[key].l5, server.load_5);
      pushHistory(S.loadHist[key].l15, server.load_15);
      return server;
    });
    S.ssl = data.sslcerts || [];
    render();
  }catch(err){
    const notice = $('notice');
    notice.style.display = 'flex';
    notice.textContent = '数据获取失败: ' + err.message;
  }
}
function shouldSuppressStatsReload(){
  return S.admin.saving || S.activeTab === 'config' || Date.now() < S.suppressStatsReloadUntil;
}
function pushHistory(arr, value){
  if(typeof value === 'number' && Number.isFinite(value) && value >= 0) arr.push(value);
  if(arr.length > 120) arr.splice(0, arr.length - 120);
}

function visibleServers(){
  const q = S.filters.query.trim().toLowerCase();
  let rows = S.servers.filter(server => {
    const m = metrics(server);
    if(S.filters.status === 'online' && !m.online) return false;
    if(S.filters.status === 'offline' && m.online) return false;
    if(S.filters.status === 'alert' && !m.alert) return false;
    const os = osLabel(server.os);
    if(S.filters.os !== 'all' && os !== S.filters.os) return false;
    if(!q) return true;
    return [server.name, server.type, server.host, server.location, os].some(v => String(v || '').toLowerCase().includes(q));
  });
  rows.sort((a, b) => {
    const dir = S.filters.dir === 'asc' ? 1 : -1;
    const ma = metrics(a), mb = metrics(b);
    const value = (server, m) => ({
      name: String(server.name || ''),
      type: String(server.type || ''),
      location: String(server.location || ''),
      status: m.online ? 1 : 0,
      load: num(server.load_1),
      cpu: num(server.cpu),
      memory: m.memPct,
      hdd: m.hddPct,
      traffic: m.traffic,
      loss: m.loss
    })[S.filters.sort];
    const va = value(a, ma), vb = value(b, mb);
    if(typeof va === 'string') return va.localeCompare(vb, 'zh-CN') * dir;
    return (va - vb) * dir;
  });
  return rows;
}

function render(){
  $('notice').style.display = 'none';
  renderOverview();
  renderActivePanel();
  updateTime();
  if(S.openDetailKey) refreshDetail();
}

function renderActivePanel(){
  normalizeServersToolbarState();
  if(S.activeTab === 'servers') renderServersView();
  else if(S.activeTab === 'monitors') renderMonitorsView();
  else if(S.activeTab === 'ssl') renderSSLView();
}

function renderServersView(){
  const compact = isCompactLayout();
  S.layoutCompact = compact;
  normalizeServersToolbarState();
  renderOsOptions();
  if(compact){
    clearHTML('serversBody');
    renderServersCards();
  }else{
    clearHTML('serversCards');
    renderServers();
  }
}

function renderMonitorsView(){
  const compact = isCompactLayout();
  S.layoutCompact = compact;
  if(compact){
    clearHTML('monitorsBody');
    renderMonitorsCards();
  }else{
    clearHTML('monitorsCards');
    renderMonitors();
  }
}

function renderSSLView(){
  const compact = isCompactLayout();
  S.layoutCompact = compact;
  if(compact){
    clearHTML('sslBody');
    renderSSLCards();
  }else{
    clearHTML('sslCards');
    renderSSL();
  }
}

const scheduleServersRender = debounce(() => {
  if(S.activeTab === 'servers') renderServersView();
}, 160);
function renderServersViewNow(){
  scheduleServersRender.cancel();
  if(S.activeTab === 'servers') renderServersView();
}

function renderOverview(){
  const total = S.servers.length;
  const online = S.servers.filter(s => metrics(s).online).length;
  const alerts = alertStats();
  const sslWarn = S.ssl.filter(c => c.mismatch || c.expire_days <= 7).length;
  const monthDown = S.servers.reduce((sum, s) => sum + metrics(s).monthIn, 0);
  const monthUp = S.servers.reduce((sum, s) => sum + metrics(s).monthOut, 0);
  $('overviewCards').innerHTML = [
    card('在线主机', `${online}/${total}`, '当前在线节点', online === total ? 'ok' : 'warn'),
    card('证书风险', sslWarn, sslWarn ? '过期或域名不匹配' : '证书正常', sslWarn ? 'warn' : 'ok'),
    card('本月下行', humanMinMBFromB(monthDown), '下载累计', 'traffic-down'),
    card('本月上行', humanMinMBFromB(monthUp), '上传累计', 'traffic-up'),
    card('活跃告警', alerts.total, `离线 ${alerts.offline} / 异常 ${alerts.abnormal} / 被墙 ${alerts.blocked}`, alerts.total ? (alerts.offline || alerts.blocked ? 'err' : 'warn') : 'ok')
  ].join('');
}
function card(label, value, hint, cls){ return `<div class="overview-card ${cls}"><span class="label">${esc(label)}</span><span class="value">${esc(value)}</span><span class="hint">${esc(hint)}</span></div>`; }

function alertStats(){
  const stats = { offline: 0, abnormal: 0, blocked: 0, total: 0 };
  S.servers.forEach(s => {
    const m = metrics(s);
    if(!m.online) stats.offline++;
    else if(m.blocked) stats.blocked++;
    else if(m.alert) stats.abnormal++;
  });
  stats.total = stats.offline + stats.abnormal + stats.blocked;
  return stats;
}

function normalizeServersToolbarState(){
  const show = S.activeTab === 'servers' && S.servers.length > 10;
  $('serversToolbar').style.display = show ? 'flex' : 'none';
}

function renderOsOptions(){
  const select = $('osFilter');
  const labels = [...new Set(S.servers.map(s => osLabel(s.os)).filter(Boolean))].sort((a,b)=>a.localeCompare(b,'zh-CN'));
  const signature = labels.join('\u001f');
  const current = labels.includes(S.filters.os) ? S.filters.os : 'all';
  if(document.activeElement === select) return;
  if(S.osOptionsSignature !== signature){
    select.innerHTML = '<option value="all">全部系统</option>' + labels.map(label => `<option value="${esc(label)}">${esc(label)}</option>`).join('');
    S.osOptionsSignature = signature;
  }
  if(select.value !== current) select.value = current;
  S.filters.os = select.value;
}

function protoPill(s){
  const m = metrics(s);
  const proto = m.online ? (s.online4 && s.online6 ? '双栈' : (s.online4 ? 'IPv4' : 'IPv6')) : '离线';
  return `<span class="pill ${m.online ? 'on' : 'off'}">${proto}</span>`;
}
const VIRT_PALETTE = ['#06b6d4','#8b5cf6','#10b981','#f59e0b','#3b82f6','#14b8a6','#6366f1','#a855f7','#22c55e','#eab308'];
function stableHash(value){
  let hash = 2166136261;
  const text = String(value || '').trim().toLowerCase();
  for(let i = 0; i < text.length; i++){
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return hash >>> 0;
}
function virtColor(type){
  const key = String(type || 'unknown').trim().toLowerCase() || 'unknown';
  return VIRT_PALETTE[stableHash(key) % VIRT_PALETTE.length];
}
function virtPill(type){
  const raw = String(type ?? '').trim();
  const text = raw || '-';
  return `<span class="virt-pill" style="--virt:${virtColor(raw)}" title="虚拟化：${esc(text)}">${esc(text)}</span>`;
}
function trafficCaps(s, small){
  const m = metrics(s);
  const heavy = m.traffic >= 1000 * 1000 * 1000 * 1000;
  return `<span class="caps-traffic duo ${heavy ? 'heavy' : 'normal'}${small ? ' sm' : ''}" title="本月下行 | 上行"><span class="half in">${humanMinMBFromB(m.monthIn)}</span><span class="half out">${humanMinMBFromB(m.monthOut)}</span></span>`;
}
function loadCellHTML(s){
  const load = s.load_1 === -1 ? '–' : num(s.load_1).toFixed(2);
  const cores = cpuCoreLabel(s);
  if(!cores) return esc(load);
  return `<span class="load-with-cores" title="负载 ${esc(load)} / CPU ${esc(cores)}"><span class="load-value">${esc(load)}</span><span class="load-core-bubble">${esc(cores)}</span></span>`;
}
function gaugeHTML(type, value){
  const pct = clamp(num(value), 0, 100);
  const thresholds = {
    cpu: { warn: 75, bad: 90 },
    mem: { warn: 80, bad: 90 },
    hdd: { warn: 85, bad: 90 }
  }[type] || { warn: 75, bad: 90 };
  const warnAttr = pct >= thresholds.bad ? 'data-bad' : (pct >= thresholds.warn ? 'data-warn' : '');
  const label = type === 'cpu' ? 'CPU' : type === 'mem' ? '内存' : '硬盘';
  return `<div class="gauge-half" data-type="${type}" ${warnAttr} style="--p:${(pct / 100).toFixed(3)}" title="${label} ${pct.toFixed(0)}%">
    <svg viewBox="0 0 100 50" preserveAspectRatio="xMidYMid meet" aria-hidden="true"><path class="track" d="M10 50 A40 40 0 0 1 90 50" /><path class="arc" d="M10 50 A40 40 0 0 1 90 50" /></svg>
    <span>${pct.toFixed(0)}%</span>
  </div>`;
}
function resourceMeter(label, value, pct, kind){
  const safePct = clamp(num(pct), 0, 100);
  const thresholds = {
    cpu: { warn: 75, bad: 90 },
    mem: { warn: 80, bad: 90 },
    swap: { warn: 80, bad: 90 },
    hdd: { warn: 85, bad: 90 }
  }[kind] || { warn: 75, bad: 90 };
  const level = safePct >= thresholds.bad ? 'bad' : (safePct >= thresholds.warn ? 'warn' : 'ok');
  return `<div class="resource-meter" data-kind="${esc(kind)}" data-level="${level}" style="--p:${safePct.toFixed(1)}%">
    <div class="resource-meter-head"><span>${esc(label)}</span><strong>${esc(value)}</strong></div>
    <div class="resource-track"><i></i></div>
  </div>`;
}
function detailResourceHTML(s, m){
  const swapPct = s.swap_total ? num(s.swap_used) / num(s.swap_total) * 100 : 0;
  return `
    ${resourceMeter('CPU', `${num(s.cpu).toFixed(0)}%`, num(s.cpu), 'cpu')}
    ${resourceMeter('内存', `${humanMinMBFromKB(s.memory_used)} / ${humanMinMBFromKB(s.memory_total)}`, m.memPct, 'mem')}
    ${resourceMeter('虚存', `${humanMinMBFromKB(s.swap_used)} / ${humanMinMBFromKB(s.swap_total)}`, swapPct, 'swap')}
    ${resourceMeter('硬盘', `${humanMinMBFromMB(s.hdd_used)} / ${humanMinMBFromMB(s.hdd_total)}`, m.hddPct, 'hdd')}
    <div class="resource-mini"><span>IO</span><strong>读 ${humanMinMBFromB(s.io_read)} / 写 ${humanMinMBFromB(s.io_write)}</strong></div>`;
}
function packetLossLine(s){
  return [s.ping_10010, s.ping_189, s.ping_10086].map(p => `${clamp(num(p), 0, 100).toFixed(0)}%`).join('|');
}
function chartLegend(series){
  return `<div class="chart-legend">${series.map(item => `<span><i style="background:${esc(item.color)}"></i>${esc(item.label)}</span>`).join('')}</div>`;
}
function buckets(s){
  return `<div class="buckets" title="联通 / 电信 / 移动">${[s.ping_10010, s.ping_189, s.ping_10086].map(p => {
    const v = clamp(num(p), 0, 100);
    const level = v >= 40 ? 'bad' : (v >= 30 ? 'warn' : 'ok');
    return `<div class="bucket" data-lv="${level}"><span style="--h:${v}%"></span><label>${v.toFixed(0)}%</label></div>`;
  }).join('')}</div>`;
}

function serverRowSignature(s, m){
  const text = [
    m.online ? 1 : 0, m.rowLevel, s.online4, s.online6, s.os,
    s.name, s.type, s.location, s.uptime, s.load_1, cpuCores(s),
    s.network_rx, s.network_tx, s.network_in, s.network_out, s.last_network_in, s.last_network_out,
    s.cpu, s.memory_used, s.memory_total, s.hdd_used, s.hdd_total,
    s.ping_10010, s.ping_189, s.ping_10086
  ].map(v => String(v ?? '')).join('\u001f');
  return `${stableHash(text).toString(36)}:${text.length}`;
}

function serverRowHTML(s, m, signature){
  const netNow = `${humanMinKBFromB(s.network_rx)} | ${humanMinKBFromB(s.network_tx)}`;
  const netTotal = `${humanMinMBFromB(s.network_in)} | ${humanMinMBFromB(s.network_out)}`;
  const alertClass = m.rowLevel ? ` alert-${m.rowLevel}` : '';
  return `<tr data-key="${esc(s._key)}" data-online="${m.online ? 1 : 0}" data-sig="${esc(signature)}" class="row-server${alertClass}${osClass(s.os)}" style="cursor:${m.online ? 'pointer' : 'default'};">
    <td>${protoPill(s)}</td>
    <td>${trafficCaps(s)}</td>
    <td><span class="node-name" title="${esc(s.name || '-')}">${esc(s.name || '-')}</span></td>
    <td>${virtPill(s.type)}</td>
    <td>${esc(s.location || '-')}</td>
    <td>${esc(s.uptime || '-')}</td>
    <td>${loadCellHTML(s)}</td>
    <td>${netNow}</td>
    <td>${netTotal}</td>
    <td>${m.online ? gaugeHTML('cpu', s.cpu) : '-'}</td>
    <td>${m.online ? gaugeHTML('mem', m.memPct) : '-'}</td>
    <td>${m.online ? gaugeHTML('hdd', m.hddPct) : '-'}</td>
    <td>${buckets(s)}</td>
  </tr>`;
}

function renderServers(){
  const rows = visibleServers();
  const tbody = $('serversBody');
  document.querySelectorAll('#serversTable th[data-sort]').forEach(th => {
    th.classList.toggle('sorted-asc', th.dataset.sort === S.filters.sort && S.filters.dir === 'asc');
    th.classList.toggle('sorted-desc', th.dataset.sort === S.filters.sort && S.filters.dir === 'desc');
  });
  if(!rows.length){
    if(tbody.dataset.empty !== 'servers'){
      tbody.innerHTML = `<tr class="empty-row"><td colspan="13" class="muted" style="text-align:center;padding:1rem;">无数据</td></tr>`;
      tbody.dataset.empty = 'servers';
    }
    return;
  }
  delete tbody.dataset.empty;
  tbody.querySelector('.empty-row')?.remove();
  const existing = new Map([...tbody.querySelectorAll('tr.row-server')].map(row => [row.dataset.key, row]));
  const desiredKeys = new Set(rows.map(s => s._key));
  existing.forEach((row, key) => { if(!desiredKeys.has(key)) row.remove(); });
  rows.forEach(s => {
    const key = String(s._key);
    const m = metrics(s);
    const signature = serverRowSignature(s, m);
    const current = existing.get(key);
    let row = current;
    if(!row || row.dataset.sig !== signature){
      row = htmlElement(serverRowHTML(s, m, signature));
      if(current) current.replaceWith(row);
    }
    tbody.appendChild(row);
  });
}

function renderServersCards(){
  const wrap = $('serversCards');
  wrap.innerHTML = visibleServers().map(s => {
    const m = metrics(s);
    const alertClass = m.rowLevel ? ` alert-${m.rowLevel}` : '';
    const spec = serverSpecLabel(s);
    return `<div class="card${m.online ? '' : ' offline'}${alertClass}${osClass(s.os)}" data-key="${esc(s._key)}" data-online="${m.online ? 1 : 0}">
      <div class="card-header"><div class="card-title">${esc(s.name || '-')}${spec ? `<span class="card-spec-chip" title="CPU 核心 / 总内存">${esc(spec)}</span>` : ''} <span class="tag">${esc(s.location || '-')}</span></div>${protoPill(s)}</div>
      <div class="kvlist">
        <div><span class="key">负载</span><span>${s.load_1 === -1 ? '–' : num(s.load_1).toFixed(2)}</span></div>
        <div><span class="key">在线</span><span>${esc(s.uptime || '-')}</span></div>
        <div><span class="key">月流量</span><span>${trafficCaps(s, true)}</span></div>
        <div><span class="key">网络</span><span>${humanMinKBFromB(s.network_rx)} | ${humanMinKBFromB(s.network_tx)}</span></div>
        <div><span class="key">CPU</span><span>${num(s.cpu).toFixed(0)}%</span></div>
        <div><span class="key">内存</span><span>${m.memPct.toFixed(0)}%</span></div>
      </div>
      ${buckets(s)}
    </div>`;
  }).join('') || '<div class="empty-state">无数据</div>';
}

function parseCustom(str){
  if(typeof str !== 'string' || !str.trim()) return [];
  return str.split(';').map(seg => {
    const [name, val] = seg.split('=');
    const ms = parseInt((val || '').trim(), 10);
    return name && Number.isFinite(ms) ? { name: name.trim(), ms: Math.max(0, ms) } : null;
  }).filter(Boolean);
}
function signalBars(ms){
  const levels = [20, 50, 100, 160];
  let on = typeof ms === 'number' ? (ms <= levels[0] ? 5 : ms <= levels[1] ? 4 : ms <= levels[2] ? 3 : ms <= levels[3] ? 2 : 1) : 0;
  return `<span class="sig">${[0,1,2,3,4].map(i => `<i class="b ${i < on ? 'on' : 'off'}"></i>`).join('')}</span>`;
}
function monitorItems(s){
  const items = parseCustom(s.custom);
  return items.map(item => `<span class="mon-item"><span class="name">${esc(item.name)}</span>${signalBars(item.ms)}<span class="ms">${item.ms}ms</span></span>`).join('') || '-';
}
function renderMonitors(){
  $('monitorsBody').innerHTML = S.servers.map(s => `<tr><td>${protoPill(s)}</td><td>${esc(s.name || '-')}</td><td>${esc(s.location || '-')}</td><td><div class="mon-items">${monitorItems(s)}</div></td></tr>`).join('') || `<tr><td colspan="4" class="muted" style="text-align:center;padding:1rem;">无数据</td></tr>`;
}
function renderMonitorsCards(){
  const wrap = $('monitorsCards');
  if(window.innerWidth > 700){ wrap.innerHTML = ''; return; }
  wrap.innerHTML = S.servers.map(s => `<div class="card"><div class="card-header"><div class="card-title">${esc(s.name || '-')} <span class="tag">${esc(s.location || '-')}</span></div>${protoPill(s)}</div><div class="kvlist"><div><span class="key">监测内容</span><span class="mon-items">${monitorItems(s)}</span></div></div></div>`).join('') || '<div class="empty-state">无数据</div>';
}
function renderSSL(){
  $('sslBody').innerHTML = S.ssl.map(c => {
    const cls = c.mismatch || c.expire_days <= 0 ? 'err' : c.expire_days <= 7 ? 'warn' : 'ok';
    const status = c.mismatch ? '域名不匹配' : c.expire_days <= 0 ? '已过期' : c.expire_days <= 7 ? '将到期' : '正常';
    const dt = c.expire_ts ? new Date(c.expire_ts * 1000).toISOString().replace('T',' ').replace(/\.\d+Z/,'') : '-';
    const alertClass = cls === 'err' ? 'alert-critical' : cls === 'warn' ? 'alert-warning' : '';
    return `<tr class="${alertClass}"><td>${esc(c.name || '-')}</td><td>${esc(String(c.domain || '').replace(/^https?:\/\//,''))}</td><td>${esc(c.port || 443)}</td><td><span class="badge ${cls}">${esc(c.expire_days ?? '-')}</span></td><td>${dt}</td><td><span class="badge ${cls}">${status}</span></td></tr>`;
  }).join('') || `<tr><td colspan="6" class="muted" style="text-align:center;padding:1rem;">无证书数据</td></tr>`;
}
function renderSSLCards(){
  const wrap = $('sslCards');
  if(window.innerWidth > 700){ wrap.innerHTML = ''; return; }
  wrap.innerHTML = S.ssl.map(c => {
    const cls = c.mismatch || c.expire_days <= 0 ? 'err' : c.expire_days <= 7 ? 'warn' : 'ok';
    const status = c.mismatch ? '域名不匹配' : c.expire_days <= 0 ? '已过期' : c.expire_days <= 7 ? '将到期' : '正常';
    const dt = c.expire_ts ? new Date(c.expire_ts * 1000).toISOString().slice(0,10) : '-';
    const alertClass = cls === 'err' ? ' alert-critical' : cls === 'warn' ? ' alert-warning' : '';
    return `<div class="card${alertClass}"><div class="card-header"><div class="card-title">${esc(c.name || '-')}</div><span class="status-pill ${cls === 'err' ? 'off' : 'on'}">${status}</span></div><div class="kvlist"><div><span class="key">域名</span><span>${esc(String(c.domain || '').replace(/^https?:\/\//,''))}</span></div><div><span class="key">端口</span><span>${esc(c.port || 443)}</span></div><div><span class="key">剩余</span><span>${esc(c.expire_days ?? '-')} 天</span></div><div><span class="key">到期</span><span>${dt}</span></div></div></div>`;
  }).join('') || '<div class="empty-state">无证书数据</div>';
}

function openDetail(key){
  S.openDetailKey = key;
  $('detailModal').style.display = 'flex';
  refreshDetail();
  document.addEventListener('keydown', escCloseOnce);
}
function findServer(key){ return S.servers.find(s => s._key === key); }
function refreshDetail(){
  const s = findServer(S.openDetailKey);
  if(!s){ closeDetail(); return; }
  const m = metrics(s);
  const title = $('detailTitle');
  const spec = serverSpecLabel(s);
  const cpuModel = cpuModelLabel(s);
  title.innerHTML = `${esc(s.name || '-')} 详情${s.os ? `<span class="os-chip${osClass(s.os)}">${esc(osLabel(s.os))}</span>` : ''}${spec ? `<span class="spec-chip" title="CPU 核心 / 总内存">${esc(spec)}</span>` : ''}${cpuModel ? `<span class="cpu-model-chip" title="${esc(cpuModel)}">${esc(cpuModel)}</span>` : ''}`;
  const modalBox = document.querySelector('#detailModal .modal-box');
  if(modalBox){
    modalBox.classList.toggle('alert-critical', m.rowLevel === 'critical');
    modalBox.classList.toggle('alert-warning', m.rowLevel === 'warning');
  }
  const loadSeries = [
    { data: S.loadHist[s._key]?.l1 || [], color:'#8b5cf6', label:'load1' },
    { data: S.loadHist[s._key]?.l5 || [], color:'#10b981', label:'load5' },
    { data: S.loadHist[s._key]?.l15 || [], color:'#f59e0b', label:'load15' }
  ];
  const latencySeries = [
    { data: S.hist[s._key]?.cu || [], color:'#3b82f6', label:'联通' },
    { data: S.hist[s._key]?.ct || [], color:'#10b981', label:'电信' },
    { data: S.hist[s._key]?.cm || [], color:'#f59e0b', label:'移动' }
  ];
  $('detailContent').innerHTML = `
    <div class="detail-grid">
      <section class="detail-section"><h4>身份</h4>
        <div class="kv"><span>节点 / 位置</span><span class="mono">${esc(s.name || '-')} / ${esc(s.location || '-')}</span></div>
        <div class="kv"><span>虚拟化 / 主机</span><span class="mono">${esc(s.type || '-')} / ${esc(s.host || '-')}</span></div>
        <div class="kv"><span>协议 / 在线</span><span class="detail-inline">${protoPill(s)}<span class="mono">${esc(s.uptime || '-')}</span></span></div>
      </section>
      <section class="detail-section resource-section"><h4>资源</h4>
        ${detailResourceHTML(s, m)}
      </section>
      <section class="detail-section"><h4>网络</h4>
        <div class="kv"><span>当前 ↓|↑</span><span class="mono">${humanMinKBFromB(s.network_rx)} | ${humanMinKBFromB(s.network_tx)}</span></div>
        <div class="kv"><span>总流量 ↓|↑</span><span class="mono">${humanMinMBFromB(s.network_in)} | ${humanMinMBFromB(s.network_out)}</span></div>
        <div class="kv"><span>本月 ↓|↑</span><span>${trafficCaps(s, true)}</span></div>
      </section>
      <section class="detail-section"><h4>连接</h4>
        <div class="kv"><span>TCP / UDP</span><span class="mono">${num(s.tcp_count)} / ${num(s.udp_count)}</span></div>
        <div class="kv"><span>进程 / 线程</span><span class="mono">${num(s.process_count)} / ${num(s.thread_count)}</span></div>
        <div class="kv"><span>联通|电信|移动</span><span class="mono">${packetLossLine(s)}</span></div>
      </section>
    </div>
    <section class="detail-section chart-section"><div class="chart-head"><h4>负载趋势</h4>${chartLegend(loadSeries)}</div><canvas id="loadChart" class="detail-chart" height="130"></canvas></section>
    <section class="detail-section chart-section"><div class="chart-head"><h4>三网延迟</h4>${chartLegend(latencySeries)}</div><canvas id="latChart" class="detail-chart" height="150"></canvas></section>`;
  drawLineChart('loadChart', loadSeries, '暂无负载数据');
  drawLineChart('latChart', latencySeries, '暂无延迟数据', 'ms');
}
function closeDetail(){
  $('detailModal').style.display = 'none';
  S.openDetailKey = null;
  document.removeEventListener('keydown', escCloseOnce);
}
function escCloseOnce(e){ if(e.key === 'Escape') closeDetail(); }

function drawLineChart(id, series, emptyText, unit = ''){
  const canvas = $(id);
  if(!canvas) return;
  const ctx = canvas.getContext('2d');
  const W = canvas.clientWidth || canvas.width;
  const H = canvas.height;
  canvas.width = W;
  ctx.clearRect(0,0,W,H);
  const all = series.flatMap(s => s.data).filter(v => Number.isFinite(v));
  if(all.length < 2){
    ctx.fillStyle = getComputedStyle(document.body).getPropertyValue('--text-dim') || '#7a899d';
    ctx.font = '12px system-ui';
    ctx.fillText(emptyText, Math.max(12, W / 2 - 42), H / 2);
    return;
  }
  const padL = unit ? 50 : 42, padR = 10, padT = 12, padB = 20;
  let min = Math.min(0, ...all), max = Math.max(...all);
  if(max - min < 1) max = min + 1;
  const range = max - min;
  const n = Math.max(...series.map(s => s.data.length));
  const xStep = (W - padL - padR) / Math.max(1, n - 1);
  const light = document.body.classList.contains('light');
  const axis = light ? 'rgba(0,0,0,.22)' : 'rgba(255,255,255,.18)';
  const grid = light ? 'rgba(0,0,0,.08)' : 'rgba(255,255,255,.10)';
  const text = light ? 'rgba(30,41,59,.7)' : 'rgba(226,232,240,.82)';
  ctx.strokeStyle = axis; ctx.lineWidth = 1; ctx.beginPath(); ctx.moveTo(padL,padT); ctx.lineTo(padL,H-padB); ctx.lineTo(W-padR,H-padB); ctx.stroke();
  ctx.fillStyle = text; ctx.font = '10px system-ui';
  for(let i=0;i<=4;i++){
    const y = padT + (H-padT-padB) * i / 4;
    const val = max - range * i / 4;
    ctx.fillText(val.toFixed(max < 10 ? 1 : 0) + unit, 4, y + 3);
    ctx.strokeStyle = grid; ctx.beginPath(); ctx.moveTo(padL,y); ctx.lineTo(W-padR,y); ctx.stroke();
  }
  series.forEach(item => {
    if(item.data.length < 2) return;
    ctx.strokeStyle = item.color; ctx.lineWidth = 1.7; ctx.beginPath();
    item.data.forEach((v, i) => {
      const x = padL + xStep * i;
      const y = padT + (H-padT-padB) * (1 - (v - min) / range);
      if(i === 0) ctx.moveTo(x,y); else ctx.lineTo(x,y);
    });
    ctx.stroke();
  });
}

function updateTime(){
  const last = $('lastUpdate');
  if(last) last.textContent = '最后更新: ' + humanAgo(S.updated);
}

function bindTabs(){
  $('navTabs').addEventListener('click', e => {
    if(e.target.tagName !== 'BUTTON') return;
    const tab = e.target.dataset.tab;
    S.activeTab = tab;
    scheduleServersRender.cancel();
    document.querySelectorAll('.nav button').forEach(btn => btn.classList.toggle('active', btn === e.target));
    document.querySelectorAll('.panel').forEach(panel => panel.classList.toggle('active', panel.id === 'panel-' + tab));
    renderActivePanel();
    if(tab === 'config') ensureAdminChecked();
  });
}
function bindTheme(){
  const btn = $('themeToggle');
  const mql = window.matchMedia('(prefers-color-scheme: light)');
  const apply = (light) => { document.body.classList.toggle('light', light); document.documentElement.classList.toggle('light', light); };
  const saved = localStorage.getItem('theme');
  apply(saved ? saved === 'light' : mql.matches);
  mql.addEventListener('change', e => { if(!localStorage.getItem('theme')) apply(e.matches); });
  btn.addEventListener('click', () => {
    const toLight = !document.body.classList.contains('light');
    apply(toLight);
    localStorage.setItem('theme', toLight ? 'light' : 'dark');
    if(S.openDetailKey) refreshDetail();
  });
}
function bindFilters(){
  $('serverSearch').addEventListener('input', e => { S.filters.query = e.target.value; scheduleServersRender(); });
  $('statusFilter').addEventListener('click', e => {
    if(e.target.tagName !== 'BUTTON') return;
    S.filters.status = e.target.dataset.filter;
    document.querySelectorAll('#statusFilter button').forEach(btn => btn.classList.toggle('active', btn === e.target));
    renderServersViewNow();
  });
  $('osFilter').addEventListener('change', e => { S.filters.os = e.target.value; renderServersViewNow(); });
  $('osFilter').addEventListener('blur', renderOsOptions);
  $('sortSelect').addEventListener('change', e => { S.filters.sort = e.target.value; renderServersViewNow(); });
  $('sortDirection').addEventListener('click', () => {
    S.filters.dir = S.filters.dir === 'desc' ? 'asc' : 'desc';
    $('sortDirection').textContent = S.filters.dir === 'desc' ? '降序' : '升序';
    renderServersViewNow();
  });
  document.querySelectorAll('#serversTable th[data-sort]').forEach(th => th.addEventListener('click', () => {
    if(S.filters.sort === th.dataset.sort) S.filters.dir = S.filters.dir === 'desc' ? 'asc' : 'desc';
    else S.filters.sort = th.dataset.sort;
    $('sortSelect').value = S.filters.sort;
    $('sortDirection').textContent = S.filters.dir === 'desc' ? '降序' : '升序';
    renderServersViewNow();
  }));
}

function bindServerInteractions(){
  $('serversBody').addEventListener('click', e => {
    const row = e.target.closest('.row-server');
    if(!row || row.dataset.online !== '1') return;
    openDetail(row.dataset.key);
  });
  $('serversCards').addEventListener('click', e => {
    const card = e.target.closest('.card[data-key]');
    if(!card || card.dataset.online !== '1') return;
    openDetail(card.dataset.key);
  });
}

function adminHeaders(){
  return { 'Content-Type': 'application/json', 'Authorization': `Bearer ${S.admin.token}` };
}
async function api(path, options = {}){
  const res = await fetch(path, { ...options, headers: { ...(options.headers || {}), ...(options.auth === false ? {} : adminHeaders()) } });
  const data = await res.json().catch(() => ({}));
  if(!res.ok || data.ok === false) throw new Error(data.error || res.statusText);
  return data;
}
async function ensureAdminChecked(){
  if(S.admin._checking) return;
  S.admin._checking = true;
  try{
    const health = await api('/api/health', { auth:false });
    S.admin.enabled = !!health.enabled;
    setAdminStatus(health.enabled ? '管理 API 已启用，输入 token 后可编辑配置。' : '管理 API 未启用：请在容器环境变量设置 ADMIN_TOKEN。', health.enabled ? '' : 'err');
    if(S.admin.enabled && S.admin.token) await loadConfig();
  }catch(err){
    setAdminStatus('无法连接管理 API：' + err.message, 'err');
  }finally{
    S.admin._checking = false;
  }
}
function setAdminStatus(text, cls){
  const el = $('adminStatus');
  el.className = 'admin-status' + (cls ? ' ' + cls : '');
  el.textContent = text;
}
async function loadConfig(){
  const data = await api('/api/config');
  S.admin.config = normalizeAdminConfig(data.config);
  S.admin.connected = true;
  setAdminStatus('已连接管理 API，配置可编辑。', 'ok');
  renderConfigList();
  renderConfigEditor();
}

const CONFIG_TYPES = {
  servers: {
    label: '节点',
    addLabel: '新增节点',
    empty: '暂无节点配置',
    hint: '客户端登录使用 username/password，保存后自动重载 sergate。',
    fields: [
      { name:'username', label:'用户名', required:true, max:120 },
      { name:'name', label:'节点名', required:true, max:120 },
      { name:'type', label:'虚拟化', required:true, max:120, placeholder:'kvm / xen / vmware' },
      { name:'host', label:'主机名', required:true, max:120 },
      { name:'location', label:'位置', required:true, max:120, placeholder:'🇨🇳 / 上海 / hk-01' },
      { name:'password', label:'密码', required:true, max:120, keepRaw:true },
      { name:'monthstart', label:'月初日', type:'number', min:1, max:28, default:1 },
      { name:'disabled', label:'禁用节点', type:'checkbox' }
    ],
    title: item => item?.name || item?.username || '未命名节点',
    meta: item => `${item?.location || '-'} / ${item?.type || '-'} / ${item?.host || '-'}`,
    badge: item => ({ text: item?.disabled ? '禁用' : '启用', cls: item?.disabled ? 'warn' : 'ok' })
  },
  monitors: {
    label: '监测',
    addLabel: '新增监测',
    empty: '暂无服务监测',
    hint: '服务监测会写入 config.json 的 monitors 数组。',
    fields: [
      { name:'name', label:'名称', required:true, max:120 },
      { name:'host', label:'地址', required:true, max:300, placeholder:'https://example.com' },
      { name:'type', label:'类型', required:true, max:40, placeholder:'http / https / tcp' },
      { name:'interval', label:'间隔秒', type:'number', min:1, default:600 }
    ],
    title: item => item?.name || item?.host || '未命名监测',
    meta: item => `${item?.type || '-'} / ${item?.host || '-'} / ${item?.interval || '-'}s`,
    badge: item => ({ text: item?.type || '监测', cls: 'ok' })
  },
  sslcerts: {
    label: '证书',
    addLabel: '新增证书',
    empty: '暂无证书配置',
    hint: '证书配置会写入 config.json 的 sslcerts 数组。',
    fields: [
      { name:'name', label:'名称', required:true, max:120 },
      { name:'domain', label:'域名', required:true, max:300, placeholder:'https://example.com' },
      { name:'port', label:'端口', type:'number', min:1, max:65535, default:443 },
      { name:'interval', label:'间隔秒', type:'number', min:1, default:7200 },
      { name:'callback', label:'回调地址', max:500, placeholder:'https://yourSMSurl' }
    ],
    title: item => item?.name || item?.domain || '未命名证书',
    meta: item => `${item?.domain || '-'} / ${item?.port || 443} / ${item?.interval || '-'}s`,
    badge: () => ({ text: 'SSL', cls: 'ok' })
  },
  watchdog: {
    label: '告警',
    addLabel: '新增告警',
    empty: '暂无告警规则',
    hint: '告警规则会写入 config.json 的 watchdog 数组。',
    fields: [
      { name:'name', label:'名称', required:true, max:160 },
      { name:'rule', label:'规则表达式', type:'textarea', required:true, placeholder:"online4=0&online6=0" },
      { name:'interval', label:'间隔秒', type:'number', min:1, default:600 },
      { name:'callback', label:'回调地址', max:500, placeholder:'https://yourSMSurl' }
    ],
    title: item => item?.name || '未命名告警',
    meta: item => item?.rule || '-',
    badge: item => ({ text: `${item?.interval || '-'}s`, cls: 'warn' })
  }
};

function normalizeAdminConfig(config){
  const normalized = config && typeof config === 'object' ? config : {};
  ['servers','monitors','sslcerts','watchdog'].forEach(key => {
    if(!Array.isArray(normalized[key])) normalized[key] = [];
  });
  return normalized;
}
function activeConfigDef(){
  return CONFIG_TYPES[S.admin.selectedType] || CONFIG_TYPES.servers;
}
function configItems(){
  if(!S.admin.config) S.admin.config = normalizeAdminConfig({});
  const key = S.admin.selectedType;
  if(!Array.isArray(S.admin.config[key])) S.admin.config[key] = [];
  return S.admin.config[key];
}
function renderConfigList(){
  const list = $('configItemList');
  const def = activeConfigDef();
  const items = configItems();
  if(S.admin.selectedIndex >= items.length) S.admin.selectedIndex = -1;
  $('addConfigItemBtn').textContent = def.addLabel;
  document.querySelectorAll('#configTypeTabs button').forEach(btn => btn.classList.toggle('active', btn.dataset.type === S.admin.selectedType));
  list.innerHTML = items.map((item, index) => {
    const badge = def.badge(item, index);
    return `<div class="config-row${S.admin.selectedIndex === index ? ' active' : ''}" data-index="${index}"><div><div class="name">${esc(def.title(item, index))}</div><div class="meta">${esc(def.meta(item, index))}</div></div><span class="badge ${badge.cls}">${esc(badge.text)}</span></div>`;
  }).join('') || `<div class="empty-state">${def.empty}</div>`;
  list.querySelectorAll('.config-row').forEach(row => row.addEventListener('click', () => selectConfigItem(Number(row.dataset.index))));
}
function selectConfigItem(index){
  const item = configItems()[index];
  if(!item) return;
  S.admin.selectedIndex = index;
  renderConfigList();
  renderConfigEditor(item);
}
function renderConfigEditor(item){
  const def = activeConfigDef();
  const editing = S.admin.selectedIndex >= 0 && item;
  const current = item || {};
  $('configEditorTitle').textContent = `${editing ? '编辑' : '新增'}${def.label}`;
  $('configEditorHint').textContent = def.hint;
  $('configFields').innerHTML = def.fields.map(field => fieldHTML(field, current)).join('');
  const resetTrafficBtn = $('resetTrafficBtn');
  const canResetTraffic = editing && S.admin.selectedType === 'servers';
  resetTrafficBtn.style.display = canResetTraffic ? '' : 'none';
  resetTrafficBtn.disabled = !canResetTraffic || S.admin.saving;
  $('deleteConfigItemBtn').disabled = !editing;
}
function fieldHTML(field, item){
  const value = item[field.name] ?? field.default ?? '';
  if(field.type === 'checkbox'){
    return `<label class="check-row"><input name="${esc(field.name)}" type="checkbox" ${value ? 'checked' : ''} /> <span>${esc(field.label)}</span></label>`;
  }
  const required = field.required ? ' required' : '';
  const min = field.min != null ? ` min="${field.min}"` : '';
  const max = field.max != null ? (field.type === 'number' ? ` max="${field.max}"` : ` maxlength="${field.max}"`) : '';
  const placeholder = field.placeholder ? ` placeholder="${esc(field.placeholder)}"` : '';
  if(field.type === 'textarea'){
    return `<label class="wide"><span>${esc(field.label)}</span><textarea name="${esc(field.name)}"${required}${placeholder}>${esc(value)}</textarea></label>`;
  }
  return `<label><span>${esc(field.label)}</span><input name="${esc(field.name)}" type="${field.type || 'text'}" value="${esc(value)}"${required}${min}${max}${placeholder} /></label>`;
}
function clearConfigForm(){
  S.admin.selectedIndex = -1;
  renderConfigList();
  renderConfigEditor();
}
function formConfigItem(){
  const elements = $('configForm').elements;
  const item = {};
  activeConfigDef().fields.forEach(field => {
    const el = elements[field.name];
    if(!el) return;
    if(field.type === 'checkbox'){
      item[field.name] = el.checked;
      return;
    }
    if(field.type === 'number'){
      let value = el.value === '' ? (field.default ?? 0) : Number(el.value);
      if(!Number.isFinite(value)) value = field.default ?? 0;
      if(field.min != null) value = Math.max(field.min, value);
      if(field.max != null) value = Math.min(field.max, value);
      item[field.name] = value;
      return;
    }
    item[field.name] = field.keepRaw ? el.value : el.value.trim();
  });
  return item;
}
function configItemPath(key, index){
  if(index < 0) return `/api/${key}`;
  const current = configItems()[index] || {};
  const id = key === 'servers' ? current.username : String(index);
  return `/api/${key}/${encodeURIComponent(id || String(index))}`;
}
async function saveConfigItem(key, index, item){
  S.admin.saving = true;
  S.suppressStatsReloadUntil = Date.now() + 8000;
  try{
    const data = await api(configItemPath(key, index), { method: index >= 0 ? 'PUT' : 'POST', body: JSON.stringify(item) });
    if(data.config) S.admin.config = normalizeAdminConfig(data.config);
    S.suppressStatsReloadUntil = Date.now() + 8000;
    return data;
  }finally{
    S.admin.saving = false;
  }
}
async function deleteConfigItem(key, index){
  S.admin.saving = true;
  S.suppressStatsReloadUntil = Date.now() + 8000;
  try{
    const data = await api(configItemPath(key, index), { method:'DELETE' });
    if(data.config) S.admin.config = normalizeAdminConfig(data.config);
    S.suppressStatsReloadUntil = Date.now() + 8000;
    return data;
  }finally{
    S.admin.saving = false;
  }
}
async function resetServerTraffic(index){
  const server = configItems()[index];
  if(!server) return setAdminStatus('请先选择要重置月流量的节点。', 'err');
  const title = server.name || server.username || '未命名节点';
  if(!confirm(`将节点 ${title} 的月流量重置为 0？该操作会重启采集服务以写入当前流量基准。`)) return;
  S.admin.saving = true;
  S.suppressStatsReloadUntil = Date.now() + 10000;
  renderConfigEditor(server);
  try{
    const data = await api(`/api/servers/${encodeURIComponent(server.username)}/reset-traffic`, { method:'POST' });
    const beforeIn = humanMinMBFromB(data.stats?.month_in_before || 0);
    const beforeOut = humanMinMBFromB(data.stats?.month_out_before || 0);
    setAdminStatus(`${title} 月流量已重置为 0（重置前 ↓${beforeIn} / ↑${beforeOut}）。`, 'ok');
    setTimeout(fetchData, 1500);
  }catch(err){
    setAdminStatus('重置月流量失败：' + err.message, 'err');
  }finally{
    S.admin.saving = false;
    S.suppressStatsReloadUntil = Date.now() + 8000;
    renderConfigEditor(configItems()[S.admin.selectedIndex]);
  }
}
function bindAdmin(){
  $('adminToken').value = S.admin.token;
  $('adminTokenForm').addEventListener('submit', async e => {
    e.preventDefault();
    S.admin.token = $('adminToken').value.trim();
    localStorage.setItem('serverstatusAdminToken', S.admin.token);
    try{ await loadConfig(); }catch(err){ setAdminStatus('认证失败：' + err.message, 'err'); }
  });
  $('refreshConfigBtn').addEventListener('click', async () => { try{ await loadConfig(); }catch(err){ setAdminStatus('刷新失败：' + err.message, 'err'); } });
  $('configTypeTabs').addEventListener('click', e => {
    if(e.target.tagName !== 'BUTTON') return;
    S.admin.selectedType = e.target.dataset.type;
    S.admin.selectedIndex = -1;
    renderConfigList();
    renderConfigEditor();
  });
  $('addConfigItemBtn').addEventListener('click', clearConfigForm);
  $('resetConfigFormBtn').addEventListener('click', clearConfigForm);
  $('resetTrafficBtn').addEventListener('click', () => resetServerTraffic(S.admin.selectedIndex));
  $('adminReload').addEventListener('click', async () => {
    try{ await api('/api/reload', { method:'POST' }); setAdminStatus('配置重载已触发。', 'ok'); }
    catch(err){ setAdminStatus('重载失败：' + err.message, 'err'); }
  });
  $('adminRestart').addEventListener('click', async () => {
    if(!confirm('重启 sergate 采集服务？客户端会短暂断开后自动重连。')) return;
    try{ await api('/api/restart', { method:'POST' }); setAdminStatus('服务重启已触发，等待容器入口脚本拉起 sergate。', 'ok'); }
    catch(err){ setAdminStatus('重启失败：' + err.message, 'err'); }
  });
  $('configForm').addEventListener('submit', async e => {
    e.preventDefault();
    const def = activeConfigDef();
    const key = S.admin.selectedType;
    const item = formConfigItem();
    const index = S.admin.selectedIndex;
    try{
      await saveConfigItem(key, index, item);
      S.admin.selectedIndex = index >= 0 ? index : configItems().length - 1;
      renderConfigList();
      renderConfigEditor(configItems()[S.admin.selectedIndex]);
      setAdminStatus(`${def.label}配置已保存并重载。`, 'ok');
    }catch(err){ setAdminStatus('保存失败：' + err.message, 'err'); }
  });
  $('deleteConfigItemBtn').addEventListener('click', async () => {
    const def = activeConfigDef();
    const key = S.admin.selectedType;
    const index = S.admin.selectedIndex;
    if(index < 0) return setAdminStatus(`请先选择要删除的${def.label}。`, 'err');
    const item = configItems()[index];
    if(!confirm(`删除${def.label} ${def.title(item, index)}？`)) return;
    try{
      await deleteConfigItem(key, index);
      S.admin.selectedIndex = -1;
      renderConfigList();
      renderConfigEditor();
      setAdminStatus(`${def.label}配置已删除并重载。`, 'ok');
    }catch(err){ setAdminStatus('删除失败：' + err.message, 'err'); }
  });
}

$('detailClose').addEventListener('click', closeDetail);
$('detailModal').addEventListener('click', e => { if(e.target.id === 'detailModal') closeDetail(); });
window.addEventListener('resize', debounce(() => {
  const compact = isCompactLayout();
  if(S.layoutCompact !== compact){
    S.layoutCompact = compact;
    renderActivePanel();
  }
  if(S.openDetailKey) refreshDetail();
}, 120));

bindTabs();
bindTheme();
bindFilters();
bindServerInteractions();
bindAdmin();
fetchData();
setInterval(fetchData, 1000);
setInterval(updateTime, 60000);
