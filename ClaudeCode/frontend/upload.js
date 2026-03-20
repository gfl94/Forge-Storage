// Upload handler

class UploadHandler {
    constructor(browser) {
        this.browser = browser;
        this.modal = document.getElementById('uploadModal');
        this.form = document.getElementById('uploadForm');
        this.fileInput = document.getElementById('fileInput');
        this.pathInput = document.getElementById('filePath');
        this.progressDiv = document.getElementById('uploadProgress');
        this.progressFill = document.getElementById('progressFill');
        this.progressText = document.getElementById('progressText');
        this.errorDiv = document.getElementById('uploadError');

        this.setupEventListeners();
    }

    setupEventListeners() {
        document.getElementById('uploadBtn').onclick = () => this.show();
        document.getElementById('cancelUploadBtn').onclick = () => this.hide();

        this.form.onsubmit = async (e) => {
            e.preventDefault();
            await this.upload();
        };

        this.fileInput.onchange = () => {
            if (this.fileInput.files.length > 0) {
                const file = this.fileInput.files[0];
                const fileName = file.name;
                let targetPath = this.browser.currentPath;
                if (!targetPath.endsWith('/')) {
                    targetPath += '/';
                }
                targetPath += fileName;
                this.pathInput.value = targetPath;
            }
        };
    }

    show() {
        this.modal.style.display = 'flex';
        this.fileInput.value = '';
        this.pathInput.value = '';
        this.progressDiv.style.display = 'none';
        this.errorDiv.style.display = 'none';
    }

    hide() {
        this.modal.style.display = 'none';
    }

    async upload() {
        const file = this.fileInput.files[0];
        if (!file) {
            this.showError('Please select a file');
            return;
        }

        const targetPath = this.pathInput.value;
        if (!targetPath) {
            this.showError('Invalid upload path');
            return;
        }

        try {
            // Get upload URL
            const uploadTarget = await FileAPI.getUploadURL(
                targetPath,
                file.size,
                file.type || 'application/octet-stream'
            );

            // Show progress
            this.progressDiv.style.display = 'block';
            this.errorDiv.style.display = 'none';

            // Upload file
            await uploadFile(
                uploadTarget.url,
                file,
                uploadTarget.headers,
                (percentage) => {
                    this.progressFill.style.width = percentage + '%';
                    this.progressText.textContent = Math.round(percentage) + '%';
                }
            );

            // Success
            this.hide();
            await this.browser.refresh();
        } catch (error) {
            this.showError('Upload failed: ' + error.message);
        }
    }

    showError(message) {
        this.errorDiv.textContent = message;
        this.errorDiv.style.display = 'block';
    }
}

// Export for use in other modules
window.UploadHandler = UploadHandler;
