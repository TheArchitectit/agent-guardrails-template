/**
 * Toast Component
 * Success/error notifications with auto-dismiss and stacking support
 */

class Toast {
  constructor() {
    throw new Error('Toast is a static class. Use Toast.success(), Toast.error(), etc.');
  }

  /**
   * Get or create toast container
   */
  static getContainer() {
    let container = document.getElementById('toast-container');
    if (!container) {
      container = document.createElement('div');
      container.id = 'toast-container';
      container.className = 'toast-container';
      document.body.appendChild(container);
    }
    return container;
  }

  /**
   * Show a toast notification
   */
  static show(options = {}) {
    const {
      type = 'info',
      title = '',
      message = '',
      duration = 5000,
      closable = true
    } = options;

    const container = this.getContainer();

    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.style.cssText = `
      display: flex;
      align-items: flex-start;
      gap: 0.75rem;
      padding: 1rem;
      background-color: var(--color-surface);
      border: 1px solid var(--color-border);
      border-left: 4px solid var(--color-${type === 'error' ? 'error' : type === 'success' ? 'success' : type === 'warning' ? 'warning' : 'info'});
      border-radius: var(--radius-lg);
      box-shadow: var(--shadow-lg);
      min-width: 300px;
      max-width: 400px;
      animation: slideIn 0.3s ease;
    `;

    const icons = {
      success: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-success)" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>`,
      error: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-error)" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="15" y1="9" x2="9" y2="15"/><line x1="9" y1="9" x2="15" y2="15"/></svg>`,
      warning: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-warning)" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>`,
      info: `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--color-info)" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12.01" y2="8"/></svg>`
    };

    toast.innerHTML = `
      <span class="toast-icon" style="flex-shrink: 0; margin-top: 0.125rem;">${icons[type]}</span>
      <div class="toast-content" style="flex: 1; min-width: 0;">
        ${title ? `<div class="toast-title" style="font-weight: 500; color: var(--color-text-primary); margin-bottom: 0.25rem;">${title}</div>` : ''}
        ${message ? `<div class="toast-message" style="font-size: 0.875rem; color: var(--color-text-secondary);">${message}</div>` : ''}
      </div>
      ${closable ? `
        <button class="toast-close" style="
          background: none;
          border: none;
          color: var(--color-text-tertiary);
          cursor: pointer;
          padding: 0;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;
          margin-top: 0.125rem;
        " aria-label="Close">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="18" y1="6" x2="6" y2="18"/>
            <line x1="6" y1="6" x2="18" y2="18"/>
          </svg>
        </button>
      ` : ''}
    `;

    // Add close handler
    const closeBtn = toast.querySelector('.toast-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.dismiss(toast));
    }

    // Add progress bar for auto-dismiss
    if (duration > 0) {
      const progressBar = document.createElement('div');
      progressBar.style.cssText = `
        position: absolute;
        bottom: 0;
        left: 0;
        height: 3px;
        background-color: var(--color-${type === 'error' ? 'error' : type === 'success' ? 'success' : type === 'warning' ? 'warning' : 'info'});
        opacity: 0.3;
        animation: progress ${duration}ms linear;
      `;
      toast.style.position = 'relative';
      toast.appendChild(progressBar);

      // Add keyframes if not present
      if (!document.getElementById('toast-keyframes')) {
        const style = document.createElement('style');
        style.id = 'toast-keyframes';
        style.textContent = `
          @keyframes progress {
            from { width: 100%; }
            to { width: 0%; }
          }
        `;
        document.head.appendChild(style);
      }
    }

    container.appendChild(toast);

    // Auto dismiss
    if (duration > 0) {
      setTimeout(() => this.dismiss(toast), duration);
    }

    return toast;
  }

  /**
   * Dismiss a toast
   */
  static dismiss(toast) {
    toast.style.animation = 'slideOut 0.3s ease forwards';

    // Add keyframes if not present
    if (!document.getElementById('toast-slide-out-keyframes')) {
      const style = document.createElement('style');
      style.id = 'toast-slide-out-keyframes';
      style.textContent = `
        @keyframes slideOut {
          to {
            transform: translateX(100%);
            opacity: 0;
          }
        }
      `;
      document.head.appendChild(style);
    }

    setTimeout(() => {
      toast.remove();
      // Remove container if empty
      const container = document.getElementById('toast-container');
      if (container && !container.children.length) {
        container.remove();
      }
    }, 300);
  }

  /**
   * Show a success toast
   */
  static success(message, title = 'Success') {
    return this.show({ type: 'success', title, message });
  }

  /**
   * Show an error toast
   */
  static error(message, title = 'Error') {
    return this.show({ type: 'error', title, message, duration: 8000 });
  }

  /**
   * Show a warning toast
   */
  static warning(message, title = 'Warning') {
    return this.show({ type: 'warning', title, message, duration: 7000 });
  }

  /**
   * Show an info toast
   */
  static info(message, title = 'Info') {
    return this.show({ type: 'info', title, message });
  }

  /**
   * Clear all toasts
   */
  static clear() {
    const container = document.getElementById('toast-container');
    if (container) {
      Array.from(container.children).forEach(toast => this.dismiss(toast));
    }
  }
}

window.Toast = Toast;
