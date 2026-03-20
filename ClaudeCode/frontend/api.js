// API client for the storage gateway

const API_BASE = window.location.origin + '/api';

async function apiCall(endpoint, options = {}) {
    const url = `${API_BASE}${endpoint}`;
    const config = {
        headers: {
            'Content-Type': 'application/json',
            ...options.headers
        },
        credentials: 'include',
        ...options
    };

    try {
        const response = await fetch(url, config);
        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error?.message || 'Request failed');
        }

        return data;
    } catch (error) {
        console.error('API call failed:', error);
        throw error;
    }
}

// Auth APIs
const AuthAPI = {
    async login(username, password) {
        return apiCall('/login', {
            method: 'POST',
            body: JSON.stringify({ username, password })
        });
    },

    async logout() {
        return apiCall('/logout', {
            method: 'POST'
        });
    },

    async getSession() {
        return apiCall('/session');
    }
};

// File APIs
const FileAPI = {
    async list(path = '/', cursor = '', limit = 100) {
        const params = new URLSearchParams({ path });
        if (cursor) params.append('cursor', cursor);
        if (limit) params.append('limit', limit);

        return apiCall(`/list?${params}`);
    },

    async getMeta(path) {
        const params = new URLSearchParams({ path });
        return apiCall(`/meta?${params}`);
    },

    async getUploadURL(path, size, mimeType) {
        return apiCall('/upload-url', {
            method: 'POST',
            body: JSON.stringify({ path, size, mimeType })
        });
    },

    async getDownloadURL(path) {
        return apiCall('/download-url', {
            method: 'POST',
            body: JSON.stringify({ path })
        });
    },

    async getPreviewURL(path) {
        const params = new URLSearchParams({ path });
        return apiCall(`/preview?${params}`);
    }
};

// Upload file directly to storage provider
async function uploadFile(url, file, headers = {}, onProgress = null) {
    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();

        xhr.upload.addEventListener('progress', (e) => {
            if (e.lengthComputable && onProgress) {
                const percentage = (e.loaded / e.total) * 100;
                onProgress(percentage);
            }
        });

        xhr.addEventListener('load', () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                resolve();
            } else {
                reject(new Error(`Upload failed with status ${xhr.status}`));
            }
        });

        xhr.addEventListener('error', () => {
            reject(new Error('Upload failed'));
        });

        xhr.open('PUT', url);

        // Set required headers
        Object.entries(headers).forEach(([key, value]) => {
            xhr.setRequestHeader(key, value);
        });

        xhr.send(file);
    });
}
