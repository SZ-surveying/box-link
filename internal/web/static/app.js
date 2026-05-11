const configList = document.getElementById('configList');
const statusView = document.getElementById('statusView');
const doctorView = document.getElementById('doctorView');
const logView = document.getElementById('logView');
const configState = document.getElementById('configState');
const statusState = document.getElementById('statusState');
const doctorState = document.getElementById('doctorState');
const logState = document.getElementById('logState');

const actionButtons = new Map(
  Array.from(document.querySelectorAll('[data-action]')).map((button) => [button.dataset.action, button])
);
const actionLabels = new Map(
  Array.from(actionButtons.entries()).map(([action, button]) => [action, button.textContent])
);
const busyLabels = {
  status: 'Refreshing...',
  doctor: 'Diagnosing...',
  on: 'Connecting...',
  off: 'Restoring...',
};
const activeActions = new Set();
const levelClasses = new Set(['DEBUG', 'INFO', 'WARN', 'ERROR']);

let currentConfig = null;
let logEntries = [];
let stream = null;

function setBadgeState(node, text, quiet = false) {
  node.textContent = text;
  node.className = quiet ? 'badge quiet' : 'badge';
}

function normalizeLevel(level) {
  const upper = String(level || '').toUpperCase();
  return levelClasses.has(upper) ? upper : 'INFO';
}

function toneFromLevel(level) {
  switch (normalizeLevel(level)) {
    case 'ERROR':
      return 'danger';
    case 'WARN':
      return 'warn';
    default:
      return 'ok';
  }
}

function formatTimestamp(value) {
  if (!value) {
    return '';
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }

  return date.toLocaleString();
}

function formatNetmask(value) {
  if (!value) {
    return '';
  }

  if (!/^0x[0-9a-f]{8}$/i.test(value)) {
    return value;
  }

  const numeric = Number.parseInt(value.slice(2), 16);
  if (!Number.isFinite(numeric)) {
    return value;
  }

  const octets = [
    (numeric >>> 24) & 255,
    (numeric >>> 16) & 255,
    (numeric >>> 8) & 255,
    numeric & 255,
  ];
  const binary = octets
    .map((octet) => octet.toString(2).padStart(8, '0'))
    .join('');
  const cidrLike = /^1*0*$/.test(binary);
  const prefix = binary.replaceAll('0', '').length;
  const dotted = octets.join('.');

  return cidrLike ? `${dotted} (/${prefix})` : dotted;
}

function appendLevelTag(parent, level) {
  const tag = document.createElement('span');
  const normalized = normalizeLevel(level);
  tag.classList.add(`level-${normalized}`);
  tag.textContent = `[${normalized}]`;
  parent.appendChild(tag);
}

function setPanelMessage(node, message) {
  node.replaceChildren(document.createTextNode(message));
}

function syncActionState() {
  const mutationBusy = activeActions.has('on') || activeActions.has('off');

  actionButtons.forEach((button, action) => {
    const isBusy = activeActions.has(action);
    button.disabled = isBusy || (mutationBusy && action !== 'on' && action !== 'off');
    button.classList.toggle('is-busy', isBusy);
    button.textContent = isBusy ? busyLabels[action] || actionLabels.get(action) : actionLabels.get(action);
  });
}

async function withAction(action, run) {
  if (activeActions.has(action)) {
    return;
  }

  activeActions.add(action);
  syncActionState();

  try {
    await run();
  } finally {
    activeActions.delete(action);
    syncActionState();
  }
}

async function apiRequest(path, options = {}) {
  let response;
  try {
    response = await fetch(path, {
      headers: { Accept: 'application/json' },
      ...options,
    });
  } catch (error) {
    throw new Error(`network error: ${error.message}`);
  }

  const raw = await response.text();
  let payload = null;

  if (raw.trim() !== '') {
    try {
      payload = JSON.parse(raw);
    } catch (error) {
      if (!response.ok) {
        throw new Error(`request failed with status ${response.status}`);
      }
      throw new Error('server returned an invalid response');
    }
  }

  if (!response.ok || !payload?.ok) {
    const message = payload?.message || `request failed with status ${response.status}`;
    const error = new Error(message);
    error.payload = payload;
    throw error;
  }

  return payload;
}

function parseStreamPayload(event) {
  try {
    return JSON.parse(event.data);
  } catch (error) {
    return null;
  }
}

function createStatePill(text, tone = 'neutral') {
  const pill = document.createElement('span');
  pill.className = `state-pill tone-${tone}`;
  pill.textContent = text;
  return pill;
}

function createRawDetail(label, content) {
  const details = document.createElement('details');
  details.className = 'raw-detail';

  const summary = document.createElement('summary');
  summary.textContent = label;

  const pre = document.createElement('pre');
  pre.textContent = content;

  details.append(summary, pre);
  return details;
}

function createFactList(items) {
  const rows = items.filter((item) => item.value);
  if (rows.length === 0) {
    return null;
  }

  const list = document.createElement('dl');
  list.className = 'fact-list';

  rows.forEach((item) => {
    const dt = document.createElement('dt');
    dt.textContent = item.label;

    const dd = document.createElement('dd');
    dd.textContent = item.value;

    list.append(dt, dd);
  });

  return list;
}

function createStatusCard({ title, pill, tone, value, note, facts = [], rawLabel, rawContent }) {
  const card = document.createElement('article');
  card.className = 'status-card';

  const head = document.createElement('div');
  head.className = 'status-card-head';

  const heading = document.createElement('h3');
  heading.textContent = title;

  head.append(heading, createStatePill(pill, tone));

  const valueNode = document.createElement('div');
  valueNode.className = 'status-card-value';
  valueNode.textContent = value;

  const noteNode = document.createElement('p');
  noteNode.className = 'status-card-note';
  noteNode.textContent = note;

  card.append(head, valueNode, noteNode);

  const factList = createFactList(facts);
  if (factList) {
    card.appendChild(factList);
  }

  if (rawContent) {
    card.appendChild(createRawDetail(rawLabel, rawContent));
  }

  return card;
}

function createDoctorCard(check) {
  const tone = toneFromLevel(check.level);
  const card = document.createElement('article');
  card.className = 'doctor-card';

  const head = document.createElement('div');
  head.className = 'doctor-card-head';

  const heading = document.createElement('h3');
  heading.textContent = check.name || 'check';

  const meta = document.createElement('div');
  meta.className = 'doctor-card-meta';
  meta.append(
    createStatePill(normalizeLevel(check.level), tone),
    createStatePill(check.ok ? 'PASS' : 'CHECK', check.ok ? 'ok' : tone)
  );

  head.append(heading, meta);

  const body = document.createElement('p');
  body.className = 'doctor-card-body';
  body.textContent = check.detail || 'No details available.';

  card.append(head, body);
  return card;
}

function replaceLogs(logs) {
  logEntries = Array.isArray(logs) ? logs.slice(-30) : [];
  renderLogs(logEntries);
}

function appendLog(entry) {
  logEntries = [...logEntries, entry].slice(-30);
  renderLogs(logEntries);
}

function renderConfig(data) {
  currentConfig = data;
  configList.replaceChildren();

  const rows = [
    ['Config path', data.config_path],
    ['Iface override', data.iface || '(auto)'],
    ['Host IP', data.host_ip],
    ['Box IP', data.box_ip],
    ['Netmask', data.netmask],
    ['Hardware match', data.hardware_port_pattern],
    ['Log level', data.log_level],
    ['Listen', data.listen_addr],
  ];

  rows.forEach(([key, value]) => {
    const dt = document.createElement('dt');
    dt.textContent = key;

    const dd = document.createElement('dd');
    dd.textContent = value || '-';

    configList.append(dt, dd);
  });

  setBadgeState(configState, 'ready');
}

function renderStatus(data) {
  statusView.replaceChildren();

  const boxIP = currentConfig?.box_ip || 'box';
  const hostIP = currentConfig?.host_ip || 'host ip';
  const interfaceName = data.iface || 'unresolved';
  const interfaceDetails = data.interfaceDetails || {};
  const routeDetails = data.routeDetails || {};
  const pingDetails = data.pingDetails || {};

  const cards = [
    createStatusCard({
      title: 'Interface',
      pill: interfaceName,
      tone: data.interface ? 'ok' : 'warn',
      value: interfaceDetails.ipv4 || `Watching ${interfaceName}`,
      note: interfaceDetails.status
        ? `Link status is ${interfaceDetails.status}. Expected host address: ${hostIP}.`
        : `Expected host address: ${hostIP}.`,
      facts: [
        { label: 'Name', value: interfaceDetails.name || interfaceName },
        { label: 'IPv4', value: interfaceDetails.ipv4 },
        { label: 'Netmask', value: formatNetmask(interfaceDetails.netmask) },
        { label: 'MAC', value: interfaceDetails.ether },
        { label: 'Media', value: interfaceDetails.media },
        { label: 'Status', value: interfaceDetails.status },
      ],
      rawLabel: 'ifconfig snapshot',
      rawContent: data.interface || '(no data)',
    }),
    createStatusCard({
      title: 'Route',
      pill: data.routeFound ? 'Route found' : 'Needs check',
      tone: data.routeFound ? 'ok' : 'warn',
      value: routeDetails.gateway || (data.routeFound ? `Route to ${boxIP} is available` : `Route to ${boxIP} is missing`),
      note: routeDetails.interface
        ? `Traffic is using ${routeDetails.interface}.`
        : (data.routeFound ? 'macOS resolved a path to the box.' : 'Verify cable, IP, and interface binding.'),
      facts: [
        { label: 'Target', value: routeDetails.destination || boxIP },
        { label: 'Gateway', value: routeDetails.gateway },
        { label: 'Interface', value: routeDetails.interface },
        { label: 'Flags', value: routeDetails.flags },
      ],
      rawLabel: 'route output',
      rawContent: data.route || '(no data)',
    }),
    createStatusCard({
      title: 'Reachability',
      pill: data.reachable ? 'Reachable' : 'Unreachable',
      tone: data.reachable ? 'ok' : 'danger',
      value: pingDetails.latency || (data.reachable ? `${boxIP} answered ping` : `${boxIP} did not answer ping`),
      note: pingDetails.packetLoss
        ? pingDetails.packetLoss
        : (data.reachable ? 'The box is responding on the direct link.' : 'The link is up but the box did not respond.'),
      facts: [
        { label: 'Target', value: pingDetails.target || boxIP },
        { label: 'Responder', value: pingDetails.responder },
        { label: 'Latency', value: pingDetails.latency },
        { label: 'Packet loss', value: pingDetails.packetLoss },
        { label: 'Round trip', value: pingDetails.roundTrip },
      ],
      rawLabel: 'ping output',
      rawContent: data.ping || '(no data)',
    }),
  ];

  statusView.append(...cards);
  setBadgeState(statusState, data.reachable ? 'healthy' : 'attention', !data.reachable);
}

function renderStatusError(message) {
  setPanelMessage(statusView, message);
  setBadgeState(statusState, 'error', true);
}

function renderDoctor(data) {
  doctorView.replaceChildren();

  if (!data.checks || data.checks.length === 0) {
    setPanelMessage(doctorView, 'No diagnostic checks were returned.');
    setBadgeState(doctorState, 'empty', true);
    return;
  }

  const fragment = document.createDocumentFragment();

  if (data.iface) {
    const summary = document.createElement('div');
    summary.className = 'doctor-summary';
    summary.append(
      createStatePill('Interface', 'neutral'),
      document.createTextNode(`Diagnostics are targeting ${data.iface}.`)
    );
    fragment.appendChild(summary);
  }

  const grid = document.createElement('div');
  grid.className = 'doctor-grid';
  data.checks.forEach((check) => {
    grid.appendChild(createDoctorCard(check));
  });

  fragment.appendChild(grid);
  doctorView.appendChild(fragment);

  const hasFailures = data.checks.some((check) => normalizeLevel(check.level) === 'ERROR');
  const hasWarnings = data.checks.some((check) => normalizeLevel(check.level) === 'WARN');
  if (hasFailures) {
    setBadgeState(doctorState, 'error', true);
  } else if (hasWarnings) {
    setBadgeState(doctorState, 'review', true);
  } else {
    setBadgeState(doctorState, 'good');
  }
}

function renderLogs(logs) {
  logView.replaceChildren();

  if (!logs || logs.length === 0) {
    setPanelMessage(logView, 'No logs yet.');
    return;
  }

  const fragment = document.createDocumentFragment();

  logs.slice(-30).reverse().forEach((entry) => {
    const row = document.createElement('div');
    row.className = 'log-row';

    const meta = document.createElement('div');
    meta.className = 'log-meta';

    const time = document.createElement('span');
    time.textContent = formatTimestamp(entry.time);
    meta.appendChild(time);
    appendLevelTag(meta, entry.level);

    const message = document.createElement('div');
    message.textContent = entry.message || '';

    row.append(meta, message);
    fragment.appendChild(row);
  });

  logView.appendChild(fragment);
}

function connectEventStream() {
  if (!window.EventSource) {
    setBadgeState(logState, 'manual', true);
    void refreshStatus();
    return;
  }

  if (stream) {
    stream.close();
  }

  setBadgeState(logState, 'connecting', true);
  stream = new EventSource('/api/events');

  stream.addEventListener('open', () => {
    setBadgeState(logState, 'live');
  });

  stream.addEventListener('logs', (event) => {
    const payload = parseStreamPayload(event);
    replaceLogs(payload || []);
  });

  stream.addEventListener('log', (event) => {
    const payload = parseStreamPayload(event);
    if (payload) {
      appendLog(payload);
    }
  });

  stream.addEventListener('status', (event) => {
    const payload = parseStreamPayload(event);
    if (payload?.ok) {
      renderStatus(payload.data);
      return;
    }
    renderStatusError(payload?.message || 'Live status unavailable.');
  });

  stream.onerror = () => {
    setBadgeState(logState, 'reconnecting', true);
  };
}

async function loadConfig() {
  try {
    const payload = await apiRequest('/api/config');
    renderConfig(payload.data);
    replaceLogs(payload.logs || []);
  } catch (error) {
    setBadgeState(configState, 'error', true);
    setPanelMessage(configList, error.message);
    replaceLogs(error.payload?.logs || []);
  }
}

async function refreshStatus() {
  setBadgeState(statusState, 'loading', true);

  try {
    const payload = await apiRequest('/api/status');
    renderStatus(payload.data);
    replaceLogs(payload.logs || []);
  } catch (error) {
    renderStatusError(error.message);
    replaceLogs(error.payload?.logs || []);
  }
}

async function runDoctor() {
  setBadgeState(doctorState, 'running', true);
  setPanelMessage(doctorView, 'Running diagnostics...');

  try {
    const payload = await apiRequest('/api/doctor');
    renderDoctor(payload.data);
    replaceLogs(payload.logs || []);
  } catch (error) {
    setPanelMessage(doctorView, error.message);
    replaceLogs(error.payload?.logs || []);
    setBadgeState(doctorState, 'error', true);
  }
}

async function triggerAction(action) {
  await withAction(action, async () => {
    const method = action === 'on' || action === 'off' ? 'POST' : 'GET';

    try {
      const payload = await apiRequest(`/api/${action}`, { method });
      replaceLogs(payload.logs || []);

      if (action === 'status') {
        renderStatus(payload.data);
        return;
      }

      if (action === 'doctor') {
        renderDoctor(payload.data);
        return;
      }

      await refreshStatus();
    } catch (error) {
      replaceLogs(error.payload?.logs || []);

      if (action === 'doctor') {
        setPanelMessage(doctorView, error.message);
        setBadgeState(doctorState, 'error', true);
      } else if (action === 'status') {
        renderStatusError(error.message);
      } else {
        window.alert(error.message);
      }
    }
  });
}

actionButtons.forEach((button, action) => {
  button.addEventListener('click', () => {
    void triggerAction(action);
  });
});

loadConfig();
connectEventStream();
