// Main app initialization

document.addEventListener('DOMContentLoaded', async () => {
    // Check if we're on the main app page
    if (!document.getElementById('fileList')) {
        return;
    }

    // Check session
    try {
        const session = await AuthAPI.getSession();
        if (!session.authenticated) {
            window.location.href = '/login.html';
            return;
        }

        // Display username
        document.getElementById('username').textContent = session.user.username;

        // Initialize file browser
        const browser = new FileBrowser();
        await browser.navigate('/');

        // Initialize upload handler
        new UploadHandler(browser);

        // Setup logout
        document.getElementById('logoutBtn').onclick = async () => {
            await AuthAPI.logout();
            window.location.href = '/login.html';
        };

    } catch (error) {
        console.error('Session check failed:', error);
        window.location.href = '/login.html';
    }
});
