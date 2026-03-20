import './style.css';

type SessionResponse = {
  authenticated: boolean;
  user?: { id: string; username: string };
};

type ListEntry = {
  name: string;
  path: string;
  type: 'directory' | 'file';
  size?: number;
  mimeType?: string;
  modifiedAt?: string;
};

type ListingResponse = {
  path: string;
  entries: ListEntry[];
  nextCursor?: string | null;
  source: string;
};

type MetadataResponse = {
  path: string;
  type: string;
  size?: number;
  mimeType?: string;
  modifiedAt?: string;
  etag?: string;
  previewable?: boolean;
};

type UploadTarget = {
  provider: string;
  method: string;
  url: string;
  headers?: Record<string, string>;
  expiresAt: string;
};

type DownloadTarget = {
  url: string;
  expiresAt: string;
};

type AppState = {
  session: SessionResponse | null;
  listing: ListingResponse | null;
  selected: ListEntry | null;
  metadata: MetadataResponse | null;
  currentPath: string;
};

const state: AppState = {
  session: null,
  listing: null,
  selected: null,
  metadata: null,
  currentPath: '/',
};

const app = document.querySelector<HTMLDivElement>('#app');

if (!app) {
  throw new Error('App container missing');
}

app.innerHTML = `
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">
        <div class="mark"></div>
        <div>
          <div class="title">Forge Storage</div>
          <div class="subtitle">Thin gateway for your blobs</div>
        </div>
      </div>
      <div class="session-controls">
        <span id="userLabel" class="user-label"></span>
        <button id="logoutBtn" class="ghost">Logout</button>
      </div>
    </header>

    <section id="loginPanel" class="card">
      <div class="card-header">
        <h1>Sign in</h1>
        <p>Authenticate to browse and manage your storage.</p>
      </div>
      <form id="loginForm" class="form-grid">
        <label>
          <span>Username</span>
          <input id="usernameInput" name="username" autocomplete="username" required />
        </label>
        <label>
          <span>Password</span>
          <input id="passwordInput" type="password" name="password" autocomplete="current-password" required />
        </label>
        <button type="submit" class="primary">Login</button>
        <div id="loginError" class="error-text"></div>
      </form>
    </section>

    <section id="mainPanel" class="hidden">
      <div class="toolbar card">
        <div class="path-controls">
          <div class="label">Path</div>
          <div class="path-input">
            <input id="pathInput" value="/" spellcheck="false" />
            <button id="goBtn" class="ghost">Go</button>
            <button id="upBtn" class="ghost">Up</button>
            <button id="refreshBtn" class="ghost">Refresh</button>
          </div>
        </div>
        <div class="upload-block">
          <form id="uploadForm" class="upload-form">
            <label class="file-pill">
              <input type="file" id="fileInput" />
              <span id="fileLabel">Choose a file</span>
            </label>
            <button type="submit" class="primary">Upload</button>
          </form>
          <div id="uploadStatus" class="muted"></div>
        </div>
      </div>

      <div class="grid">
        <div class="panel card">
          <div class="panel-header">
            <div>
              <div class="label muted">Browsing</div>
              <div id="currentPathLabel" class="strong">/</div>
            </div>
            <div class="label muted" id="sourceLabel"></div>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Type</th>
                  <th>Size</th>
                  <th>Modified</th>
                </tr>
              </thead>
              <tbody id="listingBody"></tbody>
            </table>
          </div>
        </div>

        <div class="panel card">
          <div class="panel-header">
            <div>
              <div class="label muted">Details</div>
              <div id="detailTitle" class="strong">Select an item</div>
            </div>
          </div>
          <div id="detailBody" class="detail-body muted">Metadata will appear here.</div>
          <div class="detail-actions">
            <button id="previewBtn" class="ghost" disabled>Preview</button>
            <button id="downloadBtn" class="ghost" disabled>Download</button>
          </div>
        </div>
      </div>
      <div id="statusBar" class="status-bar muted"></div>
    </section>
  </div>
`;

const loginPanel = document.querySelector<HTMLDivElement>('#loginPanel')!;
const mainPanel = document.querySelector<HTMLElement>('#mainPanel')!;
const loginForm = document.querySelector<HTMLFormElement>('#loginForm')!;
const usernameInput = document.querySelector<HTMLInputElement>('#usernameInput')!;
const passwordInput = document.querySelector<HTMLInputElement>('#passwordInput')!;
const loginError = document.querySelector<HTMLDivElement>('#loginError')!;
const userLabel = document.querySelector<HTMLSpanElement>('#userLabel')!;
const logoutBtn = document.querySelector<HTMLButtonElement>('#logoutBtn')!;
const pathInput = document.querySelector<HTMLInputElement>('#pathInput')!;
const goBtn = document.querySelector<HTMLButtonElement>('#goBtn')!;
const upBtn = document.querySelector<HTMLButtonElement>('#upBtn')!;
const refreshBtn = document.querySelector<HTMLButtonElement>('#refreshBtn')!;
const listingBody = document.querySelector<HTMLTableSectionElement>('#listingBody')!;
const detailBody = document.querySelector<HTMLDivElement>('#detailBody')!;
const detailTitle = document.querySelector<HTMLDivElement>('#detailTitle')!;
const sourceLabel = document.querySelector<HTMLDivElement>('#sourceLabel')!;
const currentPathLabel = document.querySelector<HTMLDivElement>('#currentPathLabel')!;
const previewBtn = document.querySelector<HTMLButtonElement>('#previewBtn')!;
const downloadBtn = document.querySelector<HTMLButtonElement>('#downloadBtn')!;
const statusBar = document.querySelector<HTMLDivElement>('#statusBar')!;
const uploadForm = document.querySelector<HTMLFormElement>('#uploadForm')!;
const fileInput = document.querySelector<HTMLInputElement>('#fileInput')!;
const fileLabel = document.querySelector<HTMLSpanElement>('#fileLabel')!;
const uploadStatus = document.querySelector<HTMLDivElement>('#uploadStatus')!;

loginForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  loginError.textContent = '';
  setStatus('Signing in...');
  try {
    const res = await apiFetch<SessionResponse>('/api/login', {
      method: 'POST',
      body: JSON.stringify({ username: usernameInput.value, password: passwordInput.value }),
    });
    state.session = { authenticated: true, user: res.user };
    usernameInput.value = '';
    passwordInput.value = '';
    await refreshSession();
    setStatus('Signed in');
  } catch (err) {
    const message = parseError(err);
    loginError.textContent = message;
    setStatus(message);
  }
});

logoutBtn.addEventListener('click', async () => {
  await apiFetch('/api/logout', { method: 'POST' }).catch(() => {});
  state.session = null;
  state.listing = null;
  state.selected = null;
  state.metadata = null;
  renderAuthState();
  setStatus('Logged out');
});

goBtn.addEventListener('click', () => {
  moveToPath(pathInput.value || '/');
});

upBtn.addEventListener('click', () => {
  if (state.currentPath === '/' || state.currentPath === '') return;
  const parts = state.currentPath.split('/').filter(Boolean);
  parts.pop();
  const next = '/' + parts.join('/');
  moveToPath(next || '/');
});

refreshBtn.addEventListener('click', () => moveToPath(state.currentPath));

listingBody.addEventListener('click', (e) => {
  const target = e.target as HTMLElement;
  const row = target.closest<HTMLTableRowElement>('tr[data-path]');
  if (!row) return;
  const itemPath = row.dataset.path!;
  const entry = state.listing?.entries.find((it) => it.path === itemPath);
  if (!entry) return;
  if (entry.type === 'directory') {
    moveToPath(entry.path);
  } else {
    selectEntry(entry);
  }
});

uploadForm.addEventListener('submit', async (e) => {
  e.preventDefault();
  uploadStatus.textContent = '';
  const file = fileInput.files?.[0];
  if (!file) {
    uploadStatus.textContent = 'Choose a file first.';
    return;
  }
  const targetPath = buildTargetPath(file.name);
  try {
    uploadStatus.textContent = 'Requesting upload URL...';
    const target = await apiFetch<UploadTarget>('/api/upload-url', {
      method: 'POST',
      body: JSON.stringify({ path: targetPath, size: file.size, mimeType: file.type }),
    });
    uploadStatus.textContent = 'Uploading to storage...';
    await uploadToStorage(target, file);
    uploadStatus.textContent = 'Upload complete.';
    fileInput.value = '';
    fileLabel.textContent = 'Choose a file';
    await moveToPath(state.currentPath);
  } catch (err) {
    uploadStatus.textContent = parseError(err);
  }
});

fileInput.addEventListener('change', () => {
  fileLabel.textContent = fileInput.files?.[0]?.name ?? 'Choose a file';
});

previewBtn.addEventListener('click', async () => {
  if (!state.selected) return;
  try {
    const preview = await apiFetch<{ path: string; mode: string; url: string }>('/api/preview?path=' + encodeURIComponent(state.selected.path));
    window.open(preview.url, '_blank');
  } catch (err) {
    setStatus(parseError(err));
  }
});

downloadBtn.addEventListener('click', async () => {
  if (!state.selected) return;
  try {
    const target = await apiFetch<DownloadTarget>('/api/download-url', {
      method: 'POST',
      body: JSON.stringify({ path: state.selected.path }),
    });
    window.open(target.url, '_blank');
  } catch (err) {
    setStatus(parseError(err));
  }
});

async function refreshSession() {
  try {
    const res = await apiFetch<SessionResponse>('/api/session');
    state.session = res;
    renderAuthState();
    if (res.authenticated) {
      await moveToPath(state.currentPath);
    }
  } catch (err) {
    state.session = null;
    renderAuthState();
    setStatus(parseError(err));
  }
}

function renderAuthState() {
  const authed = !!state.session?.authenticated;
  loginPanel.classList.toggle('hidden', authed);
  mainPanel.classList.toggle('hidden', !authed);
  userLabel.textContent = authed ? state.session?.user?.username ?? '' : '';
}

async function moveToPath(nextPath: string) {
  const normalized = normalizePath(nextPath);
  state.currentPath = normalized;
  pathInput.value = normalized;
  currentPathLabel.textContent = normalized;
  setStatus('Loading ' + normalized);
  const listing = await apiFetch<ListingResponse>('/api/list?path=' + encodeURIComponent(normalized));
  state.listing = listing;
  state.selected = null;
  state.metadata = null;
  renderListing();
  renderDetails();
  setStatus(`Loaded ${normalized} (${listing.entries.length} items)`);
}

function renderListing() {
  sourceLabel.textContent = state.listing ? `Source: ${state.listing.source}` : '';
  listingBody.innerHTML = '';
  const items = state.listing?.entries ?? [];
  if (!items.length) {
    listingBody.innerHTML = `<tr><td colspan="4" class="muted">No entries</td></tr>`;
    return;
  }
  for (const entry of items) {
    const row = document.createElement('tr');
    row.dataset.path = entry.path;
    row.innerHTML = `
      <td>${entry.name}</td>
      <td>${entry.type}</td>
      <td>${entry.type === 'file' ? formatBytes(entry.size ?? 0) : '—'}</td>
      <td>${entry.modifiedAt ? formatDate(entry.modifiedAt) : '—'}</td>
    `;
    listingBody.appendChild(row);
  }
}

async function selectEntry(entry: ListEntry) {
  state.selected = entry;
  detailTitle.textContent = entry.name;
  detailBody.textContent = 'Loading metadata...';
  previewBtn.disabled = true;
  downloadBtn.disabled = true;
  try {
    const meta = await apiFetch<MetadataResponse>('/api/meta?path=' + encodeURIComponent(entry.path));
    state.metadata = meta;
    renderDetails();
  } catch (err) {
    detailBody.textContent = parseError(err);
  }
}

function renderDetails() {
  if (!state.selected || !state.metadata) {
    detailTitle.textContent = state.selected?.name ?? 'Select an item';
    detailBody.textContent = state.selected ? 'Loading...' : 'Metadata will appear here.';
    previewBtn.disabled = true;
    downloadBtn.disabled = true;
    return;
  }
  const meta = state.metadata;
  detailBody.innerHTML = `
    <div class="meta-grid">
      <div><span class="label muted">Path</span><div>${meta.path}</div></div>
      <div><span class="label muted">Type</span><div>${meta.mimeType ?? 'n/a'}</div></div>
      <div><span class="label muted">Size</span><div>${formatBytes(meta.size ?? 0)}</div></div>
      <div><span class="label muted">Modified</span><div>${meta.modifiedAt ? formatDate(meta.modifiedAt) : '—'}</div></div>
      <div><span class="label muted">ETag</span><div>${meta.etag ?? '—'}</div></div>
    </div>
  `;
  previewBtn.disabled = !meta.previewable;
  downloadBtn.disabled = false;
}

async function apiFetch<T = any>(url: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(url, {
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(options?.headers ?? {}),
    },
    ...options,
  });
  if (!resp.ok) {
    const body = await safeJson(resp);
    const message = (body as any)?.error?.message ?? resp.statusText;
    throw new Error(message);
  }
  return (await safeJson(resp)) as T;
}

async function safeJson(resp: Response) {
  const text = await resp.text();
  try {
    return text ? JSON.parse(text) : {};
  } catch {
    return {};
  }
}

function normalizePath(p: string): string {
  if (!p) return '/';
  if (!p.startsWith('/')) p = '/' + p;
  if (p.length > 1 && p.endsWith('/')) p = p.slice(0, -1);
  return p;
}

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(1024));
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
}

function formatDate(input: string): string {
  const d = new Date(input);
  return isNaN(d.getTime()) ? '—' : d.toLocaleString();
}

function parseError(err: unknown): string {
  if (err instanceof Error) return err.message;
  return 'Unexpected error';
}

function setStatus(msg: string) {
  statusBar.textContent = msg;
}

function buildTargetPath(filename: string): string {
  if (state.currentPath === '/') {
    return `/${filename}`;
  }
  return `${state.currentPath}/${filename}`;
}

async function uploadToStorage(target: UploadTarget, file: File) {
  const headers = new Headers(target.headers ?? {});
  await fetch(target.url, {
    method: target.method,
    headers,
    body: file,
  });
}

refreshSession();
