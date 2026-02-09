/**
 * Main Application
 * Entry point for the Guardrail Web UI SPA
 */

class App {
  constructor() {
    this.container = null;
    this.navigation = null;
    this.main = null;
    this.apiKeyModal = null;
    this.updateNotifier = null;
  }

  init() {
    // Check if API key is configured
    if (!window.api.apiKey) {
      this.showApiKeyPrompt();
    } else {
      this.initializeApp();
    }
  }

  showApiKeyPrompt() {
    document.body.innerHTML = '';
    document.body.className = '';

    const prompt = document.createElement('div');
    prompt.className = 'api-key-prompt';
    prompt.innerHTML = `
      <div class="api-key-card">
        <div class="api-key-logo">G</div>
        <h1 class="api-key-title">Guardrail MCP</h1>
        <p class="api-key-description">
          Enter your API key to access the guardrail management interface.
        </p>
        <form id="api-key-form">
          <div class="form-group">
            <input
              type="password"
              id="api-key-input"
              class="form-input"
              placeholder="Enter API key..."
              required
              autofocus
              style="text-align: center;"
            >
          </div>
          <button type="submit" class="btn btn-primary" style="width: 100%;">
            Continue
          </button>
        </form>
        <p style="font-size: var(--text-xs); color: var(--color-text-tertiary); margin-top: var(--space-4);">
          The API key is stored locally in your browser.
        </p>
      </div>
    `;

    document.body.appendChild(prompt);

    // Handle form submission
    const form = prompt.querySelector('#api-key-form');
    form.addEventListener('submit', (e) => {
      e.preventDefault();
      const key = prompt.querySelector('#api-key-input').value.trim();
      if (key) {
        window.api.setApiKey(key);
        this.initializeApp();
      }
    });

    // Add enter key handler
    const input = prompt.querySelector('#api-key-input');
    input.focus();
  }

  initializeApp() {
    // Clear body
    document.body.innerHTML = '';

    // Create app structure
    const app = document.createElement('div');
    app.className = 'app';

    // Navigation sidebar
    const navContainer = document.createElement('div');
    navContainer.id = 'navigation';
    app.appendChild(navContainer);

    // Main content area
    this.main = document.createElement('main');
    this.main.className = 'main';
    this.main.id = 'main';
    app.appendChild(this.main);

    document.body.appendChild(app);

    // Initialize navigation
    this.navigation = new Navigation(navContainer, {
      currentPath: window.location.hash.slice(1) || '/',
      version: '1.0.0'
    });

    // Initialize router with references
    window.router.setNavigation(this.navigation);
    window.router.setMainContainer(this.main);

    // Handle API key status changes
    this.setupApiKeyHandler();

    // Setup global error handler
    this.setupErrorHandler();

    // Initialize update notifier (checks for updates on load)
    this.updateNotifier = new UpdateNotifier();

    // Initial route
    window.router.handleRoute();
  }

  setupApiKeyHandler() {
    // Handle change API key button clicks
    document.addEventListener('click', (e) => {
      if (e.target.closest('#change-api-key-btn')) {
        e.preventDefault();
        this.showApiKeyChangeModal();
      }
    });
  }

  setupErrorHandler() {
    // Handle unhandled errors
    window.addEventListener('error', (e) => {
      console.error('Unhandled error:', e.error);
      Toast.error('An unexpected error occurred. Please refresh the page.');
    });

    // Handle unhandled promise rejections
    window.addEventListener('unhandledrejection', (e) => {
      console.error('Unhandled promise rejection:', e.reason);
      Toast.error('An unexpected error occurred. Please refresh the page.');
    });
  }

  showApiKeyChangeModal() {
    Modal.form({
      title: 'Change API Key',
      fields: `
        <div class="form-group">
          <label class="form-label">Current API Key</label>
          <input type="text" class="form-input" value="${window.api.apiKey.substring(0, 8)}..." disabled>
        </div>
        <div class="form-group">
          <label class="form-label form-label-required">New API Key</label>
          <input type="password" name="api_key" class="form-input" placeholder="Enter new API key..." required autofocus>
        </div>
      `,
      confirmText: 'Update Key',
      onSubmit: (data, modal) => {
        window.api.setApiKey(data.api_key);
        Toast.success('API key updated successfully');
        modal.close();

        // Refresh page to reload data with new key
        window.router.refresh();
      }
    });
  }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
  window.app = new App();
  window.app.init();
});

// Handle API errors globally
window.addEventListener('unhandledrejection', (event) => {
  const error = event.reason;
  if (error?.message?.includes('401') || error?.message?.includes('Unauthorized')) {
    Toast.error('API key is invalid or expired. Please update your API key.');
    // Show API key prompt after a delay
    setTimeout(() => {
      if (window.app) {
        window.app.showApiKeyChangeModal();
      }
    }, 1000);
  }
});
