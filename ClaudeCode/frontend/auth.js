// Authentication handler

document.addEventListener('DOMContentLoaded', () => {
    const loginForm = document.getElementById('loginForm');
    const errorMessage = document.getElementById('errorMessage');

    if (loginForm) {
        loginForm.addEventListener('submit', async (e) => {
            e.preventDefault();

            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;

            try {
                await AuthAPI.login(username, password);
                window.location.href = '/';
            } catch (error) {
                errorMessage.textContent = error.message || 'Login failed';
                errorMessage.style.display = 'block';
            }
        });
    }
});
