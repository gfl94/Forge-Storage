/**
 * api.js – Low-level API client for the storage gateway backend.
 * All backend communication goes through this module.
 */
const API = (() => {
    const BASE = '/api';

    async function request(method, path, body) {
        const opts = {
            method,
            credentials: 'same-origin',
            headers: {},
        };
        if (body !== undefined) {
            opts.headers['Content-Type'] = 'application/json';
            opts.body = JSON.stringify(body);
        }
        const res = await fetch(BASE + path, opts);
        const data = await res.json();
        if (!res.ok) {
            const msg = data?.error?.message || res.statusText;
            const err = new Error(msg);
            err.code = data?.error?.code || 'unknown';
            err.status = res.status;
            throw err;
        }
        return data;
    }

    return {
        login(username, password) {
            return request('POST', '/login', { username, password });
        },

        logout() {
            return request('POST', '/logout');
        },

        session() {
            return request('GET', '/session');
        },

        list(path, cursor, limit) {
            const params = new URLSearchParams({ path: path || '/' });
            if (cursor) params.set('cursor', cursor);
            if (limit) params.set('limit', String(limit));
            return request('GET', '/list?' + params.toString());
        },

        meta(path) {
            return request('GET', '/meta?path=' + encodeURIComponent(path));
        },

        uploadUrl(path, size, mimeType) {
            return request('POST', '/upload-url', { path, size, mimeType });
        },

        downloadUrl(path) {
            return request('POST', '/download-url', { path });
        },

        preview(path) {
            return request('GET', '/preview?path=' + encodeURIComponent(path));
        },
    };
})();
