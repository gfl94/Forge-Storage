/**
 * upload.js – File upload management using signed URLs.
 */
const Upload = (() => {
    const uploadBtn = document.getElementById('upload-btn');
    const uploadModal = document.getElementById('upload-modal');
    const fileInput = document.getElementById('file-input');
    const uploadStartBtn = document.getElementById('upload-start-btn');
    const uploadCancelBtn = document.getElementById('upload-cancel-btn');
    const uploadProgress = document.getElementById('upload-progress');
    const progressFill = document.getElementById('progress-fill');
    const uploadStatus = document.getElementById('upload-status');

    function init() {
        uploadBtn.addEventListener('click', () => {
            uploadModal.hidden = false;
            fileInput.value = '';
            uploadStartBtn.disabled = true;
            uploadProgress.hidden = true;
            progressFill.style.width = '0%';
            uploadStatus.textContent = '';
        });

        uploadCancelBtn.addEventListener('click', closeModal);
        uploadModal.querySelector('.modal-overlay').addEventListener('click', closeModal);

        fileInput.addEventListener('change', () => {
            uploadStartBtn.disabled = fileInput.files.length === 0;
        });

        uploadStartBtn.addEventListener('click', startUpload);
    }

    function closeModal() {
        uploadModal.hidden = true;
    }

    async function startUpload() {
        const files = fileInput.files;
        if (files.length === 0) return;

        uploadStartBtn.disabled = true;
        uploadProgress.hidden = false;

        const currentPath = Browser.getCurrentPath();
        let completed = 0;

        for (const file of files) {
            const targetPath = currentPath === '/'
                ? '/' + file.name
                : currentPath + '/' + file.name;

            uploadStatus.textContent = `Uploading ${file.name}...`;

            try {
                // Get signed upload URL from backend
                const target = await API.uploadUrl(targetPath, file.size, file.type || 'application/octet-stream');

                // Upload directly to cloud storage
                await uploadToProvider(file, target);

                completed++;
                const pct = Math.round((completed / files.length) * 100);
                progressFill.style.width = pct + '%';
            } catch (err) {
                uploadStatus.textContent = `Failed to upload ${file.name}: ${err.message}`;
                return;
            }
        }

        uploadStatus.textContent = `Successfully uploaded ${completed} file(s).`;

        // Refresh the file list after a short delay
        setTimeout(() => {
            closeModal();
            Browser.navigate(currentPath);
        }, 1500);
    }

    function uploadToProvider(file, target) {
        return new Promise((resolve, reject) => {
            const xhr = new XMLHttpRequest();
            xhr.open(target.method, target.url);

            // Set required headers from the signed URL response
            if (target.headers) {
                for (const [key, value] of Object.entries(target.headers)) {
                    xhr.setRequestHeader(key, value);
                }
            }

            xhr.upload.addEventListener('progress', (e) => {
                if (e.lengthComputable) {
                    const pct = Math.round((e.loaded / e.total) * 100);
                    uploadStatus.textContent = `Uploading ${file.name}... ${pct}%`;
                }
            });

            xhr.addEventListener('load', () => {
                if (xhr.status >= 200 && xhr.status < 300) {
                    resolve();
                } else {
                    reject(new Error(`Upload returned status ${xhr.status}`));
                }
            });

            xhr.addEventListener('error', () => {
                reject(new Error('Network error during upload'));
            });

            xhr.send(file);
        });
    }

    return { init };
})();
