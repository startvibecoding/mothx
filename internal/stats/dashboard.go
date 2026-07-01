package stats

const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>VibeCoding Stats</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #c9d1d9; min-height: 100vh; }
.header { background: #161b22; border-bottom: 1px solid #30363d; padding: 16px 24px; display: flex; align-items: center; justify-content: space-between; }
.header h1 { font-size: 20px; color: #58a6ff; }
.filters { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.filters label { font-size: 12px; color: #8b949e; }
.filters select, .filters input { background: #0d1117; border: 1px solid #30363d; color: #c9d1d9; padding: 4px 8px; border-radius: 4px; font-size: 13px; }
.filters select:focus, .filters input:focus { outline: none; border-color: #58a6ff; }
.container { max-width: 1200px; margin: 0 auto; padding: 24px; }
.summary-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 16px; }
.card .label { font-size: 12px; color: #8b949e; margin-bottom: 4px; }
.card .value { font-size: 24px; font-weight: 600; color: #f0f6fc; }
.card .value.blue { color: #58a6ff; }
.card .value.green { color: #3fb950; }
.card .value.orange { color: #d29922; }
.section { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 20px; margin-bottom: 24px; }
.section h2 { font-size: 16px; color: #f0f6fc; margin-bottom: 16px; }
.chart-container { position: relative; width: 100%; height: 260px; }
canvas { width: 100%; height: 100%; display: block; }
.table-wrap { overflow-x: auto; }
table { width: 100%; border-collapse: collapse; font-size: 13px; }
th { text-align: left; padding: 8px 12px; border-bottom: 1px solid #30363d; color: #8b949e; font-weight: 500; }
td { padding: 8px 12px; border-bottom: 1px solid #21262d; }
tr:hover td { background: #1c2128; }
.pagination { display: flex; align-items: center; justify-content: center; gap: 6px; margin-top: 12px; }
.page-btn { background: #21262d; border: 1px solid #30363d; color: #c9d1d9; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 13px; }
.page-btn:hover { background: #30363d; }
.page-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.page-btn.active { background: #1f6feb22; color: #58a6ff; border-color: #1f6feb44; }
.page-info { font-size: 13px; color: #8b949e; padding: 0 8px; }
.bar-inline { display: inline-block; height: 12px; border-radius: 2px; background: #58a6ff; vertical-align: middle; margin-right: 6px; }
.loading { text-align: center; padding: 40px; color: #8b949e; }
.tabs { display: flex; gap: 4px; margin-bottom: 16px; }
.tab { padding: 6px 14px; border-radius: 4px; cursor: pointer; font-size: 13px; color: #8b949e; border: 1px solid transparent; }
.tab:hover { color: #c9d1d9; }
.tab.active { background: #1f6feb22; color: #58a6ff; border-color: #1f6feb44; }
</style>
</head>
<body>
<div class="header">
<h1>VibeCoding Stats</h1>
<div class="filters">
<label>Range:</label>
<select id="range">
<option value="today">Today</option>
<option value="week">This Week</option>
<option value="month" selected>This Month</option>
<option value="all">All Time</option>
</select>
<label>Provider:</label>
<select id="vendor"><option value="">All</option></select>
<label>Protocol:</label>
<select id="protocol"><option value="">All</option></select>
<label>Model:</label>
<select id="model"><option value="">All</option></select>
</div>
</div>
<div class="container">
<div class="summary-cards" id="summary">
<div class="card"><div class="label">Requests</div><div class="value blue" id="stat-requests">-</div></div>
<div class="card"><div class="label">Input Tokens</div><div class="value green" id="stat-input">-</div></div>
<div class="card"><div class="label">Output Tokens</div><div class="value orange" id="stat-output">-</div></div>
<div class="card"><div class="label">Total Tokens</div><div class="value" id="stat-total">-</div></div>
</div>

<div class="section">
<h2>Token Usage Over Time</h2>
<div class="tabs" id="timeTabs">
<div class="tab active" data-group="day">Daily</div>
<div class="tab" data-group="week">Weekly</div>
<div class="tab" data-group="month">Monthly</div>
</div>
<div class="chart-container"><canvas id="chartTime"></canvas></div>
</div>

<div style="display:grid;grid-template-columns:1fr 1fr;gap:24px;">
<div class="section">
<h2>By Provider</h2>
<div class="chart-container"><canvas id="chartProvider"></canvas></div>
</div>
<div class="section">
<h2>By Model</h2>
<div class="chart-container"><canvas id="chartModel"></canvas></div>
</div>
</div>

<div class="section">
<h2>Recent Requests</h2>
<div class="table-wrap"><table><thead><tr><th>Time</th><th>Provider</th><th>Protocol</th><th>Model</th><th>Input</th><th>Output</th><th>Duration</th></tr></thead><tbody id="recentTable"><tr><td colspan="7" class="loading">Loading...</td></tr></tbody></table></div>
<div class="pagination" id="recentPagination"></div>
</div>
</div>

<script>
const $ = id => document.getElementById(id);
const fmt = n => n >= 1e6 ? (n/1e6).toFixed(1)+'M' : n >= 1e3 ? (n/1e3).toFixed(1)+'K' : String(n);

function getRange() {
	const r = $('range').value;
	if (r === 'all') return { from: '', to: '' };
	const now = new Date();
	let from, to;
	if (r === 'today') {
		from = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		to = new Date(from.getTime() + 86400000);
	} else if (r === 'week') {
		const dow = now.getDay() || 7;
		from = new Date(now.getFullYear(), now.getMonth(), now.getDate() - dow + 1);
		to = new Date(from.getTime() + 7 * 86400000);
	} else {
		from = new Date(now.getFullYear(), now.getMonth(), 1);
		to = new Date(now.getFullYear(), now.getMonth() + 1, 1);
	}
	return { from: from.toISOString().slice(0,10), to: to.toISOString().slice(0,10) };
}

function buildQS(extra) {
	const { from, to } = getRange();
	const params = new URLSearchParams();
	if (from) params.set('from', from);
	if (to) params.set('to', to);
	const vendor = $('vendor').value;
	if (vendor) params.set('vendor', vendor);
	const protocol = $('protocol').value;
	if (protocol) params.set('protocol', protocol);
	const model = $('model').value;
	if (model) params.set('model', model);
	if (extra) for (const [k,v] of Object.entries(extra)) params.set(k, v);
	return params.toString();
}

async function fetchAPI(path) {
	const qs = buildQS();
	const r = await fetch(path + (qs ? '?' + qs : ''));
	return r.json();
}

function drawBarChart(canvas, labels, datasets, colors) {
	const ctx = canvas.getContext('2d');
	const dpr = window.devicePixelRatio || 1;
	const rect = canvas.parentElement.getBoundingClientRect();
	canvas.width = rect.width * dpr;
	canvas.height = rect.height * dpr;
	ctx.scale(dpr, dpr);
	const W = rect.width, H = rect.height;
	ctx.clearRect(0, 0, W, H);

	const padL = 50, padR = 16, padT = 16, padB = 40;
	const chartW = W - padL - padR;
	const chartH = H - padT - padB;

	// Find max
	let maxVal = 0;
	for (const ds of datasets) for (const v of ds) if (v > maxVal) maxVal = v;
	if (maxVal === 0) maxVal = 1;

	// Y axis
	ctx.strokeStyle = '#30363d';
	ctx.lineWidth = 1;
	ctx.beginPath();
	for (let i = 0; i <= 4; i++) {
		const y = padT + chartH - (i / 4) * chartH;
		ctx.moveTo(padL, y);
		ctx.lineTo(padL + chartW, y);
	}
	ctx.stroke();

	ctx.font = '11px -apple-system, sans-serif';
	ctx.fillStyle = '#8b949e';
	ctx.textAlign = 'right';
	for (let i = 0; i <= 4; i++) {
		const val = Math.round(maxVal * i / 4);
		const y = padT + chartH - (i / 4) * chartH;
		ctx.fillText(fmt(val), padL - 6, y + 4);
	}

	// Bars
	const n = labels.length;
	if (n === 0) {
		ctx.fillStyle = '#8b949e';
		ctx.textAlign = 'center';
		ctx.fillText('No data', W/2, H/2);
		return;
	}
	const groupW = chartW / n;
	const barW = Math.min(groupW * 0.7 / datasets.length, 24);
	const groupPad = groupW * 0.15;

	for (let i = 0; i < n; i++) {
		const gx = padL + i * groupW + groupPad;
		for (let d = 0; d < datasets.length; d++) {
			const val = datasets[d][i] || 0;
			const bh = (val / maxVal) * chartH;
			const x = gx + d * (barW + 2);
			const y = padT + chartH - bh;
			ctx.fillStyle = colors[d] || '#58a6ff';
			ctx.fillRect(x, y, barW, bh);
		}
		// X label
		ctx.fillStyle = '#8b949e';
		ctx.textAlign = 'center';
		const label = labels[i];
		const shortLabel = label.length > 10 ? label.slice(5) : label;
		ctx.fillText(shortLabel, gx + (datasets.length * barW) / 2, padT + chartH + 16);
	}
}

function drawPieChart(canvas, labels, values, colors) {
	const ctx = canvas.getContext('2d');
	const dpr = window.devicePixelRatio || 1;
	const rect = canvas.parentElement.getBoundingClientRect();
	canvas.width = rect.width * dpr;
	canvas.height = rect.height * dpr;
	ctx.scale(dpr, dpr);
	const W = rect.width, H = rect.height;
	ctx.clearRect(0, 0, W, H);

	const total = values.reduce((a, b) => a + b, 0);
	if (total === 0) {
		ctx.fillStyle = '#8b949e';
		ctx.textAlign = 'center';
		ctx.font = '13px -apple-system, sans-serif';
		ctx.fillText('No data', W/2, H/2);
		return;
	}

	const cx = W * 0.35, cy = H / 2, r = Math.min(W * 0.28, H * 0.4);
	let startAngle = -Math.PI / 2;

	for (let i = 0; i < values.length; i++) {
		const slice = values[i] / total * Math.PI * 2;
		ctx.beginPath();
		ctx.moveTo(cx, cy);
		ctx.arc(cx, cy, r, startAngle, startAngle + slice);
		ctx.closePath();
		ctx.fillStyle = colors[i % colors.length];
		ctx.fill();
		startAngle += slice;
	}

	// Legend
	const legendX = W * 0.68;
	ctx.font = '12px -apple-system, sans-serif';
	for (let i = 0; i < labels.length; i++) {
		const ly = 20 + i * 22;
		ctx.fillStyle = colors[i % colors.length];
		ctx.fillRect(legendX, ly, 12, 12);
		ctx.fillStyle = '#c9d1d9';
		ctx.textAlign = 'left';
		const pct = (values[i] / total * 100).toFixed(1);
		const label = labels[i].length > 18 ? labels[i].slice(0, 16) + '..' : labels[i];
		ctx.fillText(label + ' (' + pct + '%)', legendX + 18, ly + 10);
	}
}

async function loadSummary() {
	const s = await fetchAPI('/api/summary');
	$('stat-requests').textContent = fmt(s.totalRequests || 0);
	$('stat-input').textContent = fmt(s.inputTokens || 0);
	$('stat-output').textContent = fmt(s.outputTokens || 0);
	$('stat-total').textContent = fmt(s.totalTokens || 0);
}

async function loadTimeSeries(groupBy) {
	const { from, to } = getRange();
	const params = new URLSearchParams();
	if (from) params.set('from', from);
	if (to) params.set('to', to);
	const vendor = $('vendor').value;
	if (vendor) params.set('vendor', vendor);
	const protocol = $('protocol').value;
	if (protocol) params.set('protocol', protocol);
	const model = $('model').value;
	if (model) params.set('model', model);
	params.set('groupBy', groupBy);
	const data = await fetch('/api/timeseries?' + params.toString()).then(r => r.json());
	const labels = data.map(d => d.label);
	const input = data.map(d => d.inputTokens);
	const output = data.map(d => d.outputTokens);
	drawBarChart($('chartTime'), labels, [input, output], ['#58a6ff', '#3fb950']);
}

async function loadByProvider() {
	const data = await fetchAPI('/api/by-provider');
	const labels = data.map(d => d.protocol ? d.vendor + ' (' + d.protocol + ')' : d.vendor);
	const values = data.map(d => d.totalTokens);
	drawPieChart($('chartProvider'), labels, values, ['#58a6ff','#3fb950','#d29922','#f78166','#a371f7','#8b949e','#1f6feb','#56d4dd']);
}

async function loadByModel() {
	const data = await fetchAPI('/api/by-model');
	const labels = data.map(d => d.label);
	const values = data.map(d => d.totalTokens);
	drawPieChart($('chartModel'), labels, values, ['#58a6ff','#3fb950','#d29922','#f78166','#a371f7','#8b949e','#1f6feb','#56d4dd']);
}

let recentPage = 1;
let recentPageSize = 20;
let recentTotal = 0;

function renderPagination() {
	const container = $('recentPagination');
	const totalPages = Math.ceil(recentTotal / recentPageSize) || 1;
	if (totalPages <= 1) { container.innerHTML = ''; return; }

	let html = '';html += '<button class="page-btn" onclick="goRecentPage(1)"' + (recentPage <= 1 ? ' disabled' : '') + '>« First</button>';html += '<button class="page-btn" onclick="goRecentPage(' + (recentPage - 1) + ')"' + (recentPage <= 1 ? ' disabled' : '') + '>‹ Prev</button>';const maxVisible = 5;
	let start = Math.max(1, recentPage - Math.floor(maxVisible / 2));
	let end = Math.min(totalPages, start + maxVisible - 1);
	if (end - start + 1 < maxVisible) start = Math.max(1, end - maxVisible + 1);
	if (start > 1) html += '<span class="page-info">...</span>';for (let i = start; i <= end; i++) {html += '<button class="page-btn' + (i === recentPage ? ' active' : '') + '" onclick="goRecentPage(' + i + ')">' + i + '</button>';}
	if (end < totalPages) html += '<span class="page-info">...</span>';html += '<button class="page-btn" onclick="goRecentPage(' + (recentPage + 1) + ')"' + (recentPage >= totalPages ? ' disabled' : '') + '>Next ›</button>';html += '<button class="page-btn" onclick="goRecentPage(' + totalPages + ')"' + (recentPage >= totalPages ? ' disabled' : '') + '>Last »</button>';html += '<span class="page-info">' + recentPage + '/' + totalPages + '</span>';container.innerHTML = html;
}

function goRecentPage(p) {
	const totalPages = Math.ceil(recentTotal / recentPageSize) || 1;
	if (p < 1 || p > totalPages) return;
	recentPage = p;
	loadRecent();
}

async function loadRecent() {
	const r = await fetch('/api/recent?page=' + recentPage + '&pageSize=' + recentPageSize);
	const data = await r.json();
	recentTotal = data.total || 0;
	recentPage = data.page || 1;
	const tbody = $('recentTable');
	if (!data.items || !data.items.length) { tbody.innerHTML = '<tr><td colspan="7" class="loading">No data</td></tr>'; $('recentPagination').innerHTML = ''; return; }
	tbody.innerHTML = data.items.map(e => '<tr>' +
		'<td>' + new Date(e.timestamp).toLocaleString() + '</td>' +
		'<td>' + (e.vendor || '-') + '</td>' +
		'<td>' + (e.protocol || '-') + '</td>' +
		'<td>' + e.model + '</td>' +
		'<td>' + fmt(e.inputTokens) + '</td>' +
		'<td>' + fmt(e.outputTokens) + '</td>' +
		'<td>' + (e.durationMs > 0 ? (e.durationMs / 1000).toFixed(1) + 's' : '-') + '</td>' +
		'</tr>').join('');
	renderPagination();
}

async function loadFilters() {
	const provData = await fetchAPI('/api/by-provider');
	const vendorSel = $('vendor');
	const protocolSel = $('protocol');
	const seenProtocols = new Set();
	provData.forEach(d => {
		const opt = document.createElement('option');
		opt.value = d.vendor; opt.textContent = d.vendor;
		vendorSel.appendChild(opt);
		if (d.protocol && !seenProtocols.has(d.protocol)) {
			seenProtocols.add(d.protocol);
			const pOpt = document.createElement('option');
			pOpt.value = d.protocol; pOpt.textContent = d.protocol;
			protocolSel.appendChild(pOpt);
		}
	});
	const modelData = await fetchAPI('/api/by-model');
	const modelSel = $('model');
	modelData.forEach(d => {
		const opt = document.createElement('option');
		opt.value = d.model; opt.textContent = d.model;
		modelSel.appendChild(opt);
	});
}

let currentTimeGroupBy = 'month';
async function refresh() {
	recentPage = 1;
	await loadSummary();
	await loadTimeSeries(currentTimeGroupBy);
	await loadByProvider();
	await loadByModel();
	await loadRecent();
}

$('range').addEventListener('change', refresh);
$('vendor').addEventListener('change', refresh);
$('protocol').addEventListener('change', refresh);
$('model').addEventListener('change', refresh);

document.querySelectorAll('#timeTabs .tab').forEach(tab => {
	tab.addEventListener('click', () => {
		document.querySelectorAll('#timeTabs .tab').forEach(t => t.classList.remove('active'));
		tab.classList.add('active');
		currentTimeGroupBy = tab.dataset.group;
		loadTimeSeries(currentTimeGroupBy);
	});
});

window.addEventListener('resize', () => {
	loadTimeSeries(currentTimeGroupBy);
	loadByProvider();
	loadByModel();
});

loadFilters().then(refresh);
</script>
</body>
</html>`
