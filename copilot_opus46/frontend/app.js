/**
 * app.js – Application initialization.
 */
document.addEventListener('DOMContentLoaded', async () => {
    Auth.init();
    Browser.init();
    Upload.init();

    // Check if there is an existing session
    const authenticated = await Auth.checkSession();
    if (authenticated) {
        Browser.navigate('/');
    }
});
