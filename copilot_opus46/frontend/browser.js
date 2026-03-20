/**
 * browser.js – File browser: listing, breadcrumbs, preview, download.
 */
const Browser = (() => {
    let currentPath = '/';

    const breadcrumb = document.getElementById('breadcrumb');
    const fileListBody = document.getElementById('file-list-body');
    const fileList = document.getElementById('file-list');
    const emptyMessage = document.getElementById('empty-message');
    const loading = document.getElementById('loading');
    const previewModal = document.getElementById('preview-modal');
    const previewTitle = document.getElementById('preview-title');
    const previewBody = document.getElementById('preview-body');
    const previewCloseBtn = document.getElementById('preview-close-btn');

    function renderBreadcrumb(path) {
        const parts = path.split('/').filter(Boolean);
        let html = '<a href="#" data-path="/">Home</a>';
        let accumulated = '';
        for (const part of parts) {
            accumulated += '/' + part;
            html += `<span>/</span><a href="#" data-path="${escapeHtml(accumulated)}">${escapeHtml(part)}</a>`;
        }
        breadcrumb.innerHTML = html;

        breadcrumb.querySelectorAll('a').forEach(a => {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                navigate(a.dataset.path);
            });
        });
    }

    function renderEntries(entries) {
        if (!entries || entries.length === 0) {
            fileList.hidden = true;
            emptyMessage.hidden = false;
            return;
        }

        fileList.hidden = false;
        emptyMessage.hidden = true;

        fileListBody.innerHTML = entries.map(entry => {
            const icon = entry.type === 'directory' ? '📁' : fileIcon(entry.mimeType);
            const size = entry.type === 'file' ? formatSize(entry.size) : '—';
            const mimeType = entry.mimeType || '—';
            const modified = entry.modifiedAt ? formatDate(entry.modifiedAt) : '—';

            let actions = '';
            if (entry.type === 'directory') {
                actions = `<a href="#" class="action-open" data-path="${escapeHtml(entry.path)}">Open</a>`;
            } else {
                actions = `<a href="#" class="action-download" data-path="${escapeHtml(entry.path)}">Download</a>`;
                if (isPreviewable(entry.mimeType)) {
                    actions += ` <a href="#" class="action-preview" data-path="${escapeHtml(entry.path)}" data-name="${escapeHtml(entry.name)}">Preview</a>`;
                }
            }

            return `<tr>
                <td><div class="file-name"><span class="file-icon">${icon}</span><a href="#" class="action-${entry.type === 'directory' ? 'open' : 'preview-or-download'}" data-path="${escapeHtml(entry.path)}" data-name="${escapeHtml(entry.name)}" data-type="${entry.type}" data-mime="${escapeHtml(entry.mimeType || '')}">${escapeHtml(entry.name)}</a></div></td>
                <td>${size}</td>
                <td>${escapeHtml(mimeType)}</td>
                <td>${modified}</td>
                <td><div class="action-links">${actions}</div></td>
            </tr>`;
        }).join('');

        bindEntryActions();
    }

    function bindEntryActions() {
        fileListBody.querySelectorAll('.action-open').forEach(a => {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                navigate(a.dataset.path);
            });
        });

        fileListBody.querySelectorAll('.action-download').forEach(a => {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                downloadFile(a.dataset.path);
            });
        });

        fileListBody.querySelectorAll('.action-preview').forEach(a => {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                previewFile(a.dataset.path, a.dataset.name);
            });
        });

        fileListBody.querySelectorAll('.action-preview-or-download').forEach(a => {
            a.addEventListener('click', (e) => {
                e.preventDefault();
                if (a.dataset.type === 'directory') {
                    navigate(a.dataset.path);
                } else if (isPreviewable(a.dataset.mime)) {
                    previewFile(a.dataset.path, a.dataset.name);
                } else {
                    downloadFile(a.dataset.path);
                }
            });
        });
    }

    async function navigate(path) {
        currentPath = path || '/';
        renderBreadcrumb(currentPath);
        showLoading(true);
        try {
            const data = await API.list(currentPath);
            renderEntries(data.entries);
        } catch (err) {
            if (err.status === 401) {
                Auth.showLogin();
                return;
            }
            fileListBody.innerHTML = `<tr><td colspan="5" class="error-message">${escapeHtml(err.message)}</td></tr>`;
            fileList.hidden = false;
            emptyMessage.hidden = true;
        } finally {
            showLoading(false);
        }
    }

    async function downloadFile(path) {
        try {
            const data = await API.downloadUrl(path);
            const a = document.createElement('a');
            a.href = data.url;
            a.download = '';
            a.target = '_blank';
            a.rel = 'noopener noreferrer';
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
        } catch (err) {
            alert('Download failed: ' + err.message);
        }
    }

    async function previewFile(path, name) {
        try {
            const data = await API.preview(path);
            previewTitle.textContent = name || path;
            previewBody.innerHTML = '';

            const url = data.url;
            const ext = (path.split('.').pop() || '').toLowerCase();
            const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'];
            const pdfExts = ['pdf'];

            if (imageExts.includes(ext)) {
                const img = document.createElement('img');
                img.src = url;
                img.alt = name;
                previewBody.appendChild(img);
            } else if (pdfExts.includes(ext)) {
                const iframe = document.createElement('iframe');
                iframe.src = url;
                previewBody.appendChild(iframe);
            } else {
                const iframe = document.createElement('iframe');
                iframe.src = url;
                previewBody.appendChild(iframe);
            }

            previewModal.hidden = false;
        } catch (err) {
            alert('Preview failed: ' + err.message);
        }
    }

    function showLoading(show) {
        loading.hidden = !show;
        if (show) {
            fileList.hidden = true;
            emptyMessage.hidden = true;
        }
    }

    function getCurrentPath() {
        return currentPath;
    }

    function init() {
        previewCloseBtn.addEventListener('click', () => {
            previewModal.hidden = true;
            previewBody.innerHTML = '';
        });

        previewModal.querySelector('.modal-overlay').addEventListener('click', () => {
            previewModal.hidden = true;
            previewBody.innerHTML = '';
        });

        document.getElementById('refresh-btn').addEventListener('click', () => {
            navigate(currentPath);
        });
    }

    // ── Helpers ───────────────────────────────────────────────

    function escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    function formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const units = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(1024));
        return (bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1) + ' ' + units[i];
    }

    function formatDate(iso) {
        try {
            return new Date(iso).toLocaleString();
        } catch (_) {
            return iso;
        }
    }

    function fileIcon(mimeType) {
        if (!mimeType) return '📄';
        if (mimeType.startsWith('image/')) return '🖼️';
        if (mimeType === 'application/pdf') return '📕';
        if (mimeType.startsWith('video/')) return '🎬';
        if (mimeType.startsWith('audio/')) return '🎵';
        if (mimeType.startsWith('text/')) return '📝';
        return '📄';
    }

    function isPreviewable(mimeType) {
        if (!mimeType) return false;
        return mimeType.startsWith('image/') ||
               mimeType === 'application/pdf' ||
               mimeType.startsWith('text/');
    }

    return { init, navigate, getCurrentPath };
})();
