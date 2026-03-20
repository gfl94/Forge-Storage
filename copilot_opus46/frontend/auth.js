/**
 * auth.js – Authentication state management.
 */
const Auth = (() => {
    const loginScreen = document.getElementById('login-screen');
    const appScreen = document.getElementById('app-screen');
    const loginForm = document.getElementById('login-form');
    const loginError = document.getElementById('login-error');
    const logoutBtn = document.getElementById('logout-btn');

    function showLogin() {
        loginScreen.hidden = false;
        appScreen.hidden = true;
    }

    function showApp() {
        loginScreen.hidden = true;
        appScreen.hidden = false;
    }

    async function checkSession() {
        try {
            const data = await API.session();
            if (data.authenticated) {
                showApp();
                return true;
            }
        } catch (_) {
            // not authenticated
        }
        showLogin();
        return false;
    }

    function init() {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            loginError.hidden = true;
            const username = document.getElementById('username').value.trim();
            const password = document.getElementById('password').value;
            try {
                await API.login(username, password);
                showApp();
                Browser.navigate('/');
            } catch (err) {
                loginError.textContent = err.message || 'Login failed.';
                loginError.hidden = false;
            }
        });

        logoutBtn.addEventListener('click', async () => {
            try {
                await API.logout();
            } catch (_) {
                // ignore
            }
            showLogin();
        });
    }

    return { init, checkSession, showLogin };
})();
