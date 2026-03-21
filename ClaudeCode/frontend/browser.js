// File browser functionality

class FileBrowser {
    constructor() {
        this.currentPath = '/';
    }

    async navigate(path) {
        this.currentPath = path;
        await this.refresh();
    }

    async refresh() {
        try {
            const result = await FileAPI.list(this.currentPath);
            this.render(result);
            this.updateBreadcrumb();
        } catch (error) {
            this.showError(error.message);
        }
    }

    render(result) {
        const tbody = document.getElementById('fileListBody');
        tbody.innerHTML = '';

        if (result.entries.length === 0) {
            tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">No files or folders</td></tr>';
            return;
        }

        result.entries.forEach(entry => {
            const row = document.createElement('tr');

            // Name
            const nameCell = document.createElement('td');
            if (entry.type === 'directory') {
                const link = document.createElement('a');
                link.href = '#';
                link.className = 'file-name';
                link.textContent = '📁 ' + entry.name;
                link.onclick = (e) => {
                    e.preventDefault();
                    this.navigate(entry.path);
                };
                nameCell.appendChild(link);
            } else {
                const span = document.createElement('span');
                span.className = 'file-name';
                span.textContent = '📄 ' + entry.name;
                nameCell.appendChild(span);
            }
            row.appendChild(nameCell);

            // Type
            const typeCell = document.createElement('td');
            const typeBadge = document.createElement('span');
            typeBadge.className = `type-badge type-${entry.type}`;
            typeBadge.textContent = entry.type;
            typeCell.appendChild(typeBadge);
            row.appendChild(typeCell);

            // Size
            const sizeCell = document.createElement('td');
            sizeCell.textContent = entry.type === 'file' ? this.formatSize(entry.size) : '-';
            row.appendChild(sizeCell);

            // Modified
            const modifiedCell = document.createElement('td');
            modifiedCell.textContent = entry.modifiedAt ? new Date(entry.modifiedAt).toLocaleString() : '-';
            row.appendChild(modifiedCell);

            // Actions
            const actionsCell = document.createElement('td');
            const actionsDiv = document.createElement('div');
            actionsDiv.className = 'file-actions';

            if (entry.type === 'file') {
                if (entry.previewable) {
                    const previewBtn = document.createElement('button');
                    previewBtn.className = 'btn btn-secondary btn-small';
                    previewBtn.textContent = 'Preview';
                    previewBtn.onclick = () => this.previewFile(entry.path);
                    actionsDiv.appendChild(previewBtn);
                }

                const downloadBtn = document.createElement('button');
                downloadBtn.className = 'btn btn-primary btn-small';
                downloadBtn.textContent = 'Download';
                downloadBtn.onclick = () => this.downloadFile(entry.path);
                actionsDiv.appendChild(downloadBtn);
            }

            actionsCell.appendChild(actionsDiv);
            row.appendChild(actionsCell);

            tbody.appendChild(row);
        });
    }

    updateBreadcrumb() {
        const breadcrumb = document.getElementById('breadcrumb');
        breadcrumb.innerHTML = '';

        const parts = this.currentPath.split('/').filter(p => p);
        let path = '';

        // Root
        const rootLink = document.createElement('a');
        rootLink.href = '#';
        rootLink.textContent = 'Home';
        rootLink.onclick = (e) => {
            e.preventDefault();
            this.navigate('/');
        };
        breadcrumb.appendChild(rootLink);

        // Parts
        parts.forEach((part, index) => {
            const separator = document.createElement('span');
            separator.textContent = ' / ';
            breadcrumb.appendChild(separator);

            path += '/' + part;
            const currentPath = path;

            if (index === parts.length - 1) {
                const span = document.createElement('span');
                span.textContent = part;
                breadcrumb.appendChild(span);
            } else {
                const link = document.createElement('a');
                link.href = '#';
                link.textContent = part;
                link.onclick = (e) => {
                    e.preventDefault();
                    this.navigate(currentPath);
                };
                breadcrumb.appendChild(link);
            }
        });
    }

    async downloadFile(path) {
        try {
            const result = await FileAPI.getDownloadURL(path);
            window.open(result.url, '_blank');
        } catch (error) {
            this.showError('Failed to download file: ' + error.message);
        }
    }

    async previewFile(path) {
        try {
            const result = await FileAPI.getPreviewURL(path);
            window.open(result.url, '_blank');
        } catch (error) {
            this.showError('Failed to preview file: ' + error.message);
        }
    }

    formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    showError(message) {
        const errorDiv = document.getElementById('errorMessage');
        errorDiv.textContent = message;
        errorDiv.style.display = 'block';
        setTimeout(() => {
            errorDiv.style.display = 'none';
        }, 5000);
    }
}

// Export for use in other modules
window.FileBrowser = FileBrowser;
