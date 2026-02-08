/**
 * Modal Component
 * Create/Edit modals, confirmation dialogs with form submission handling
 */

class Modal {
  constructor(options = {}) {
    this.options = {
      title: 'Modal',
      size: 'md', // sm, md, lg, xl
      closable: true,
      onClose: null,
      onConfirm: null,
      showFooter: true,
      confirmText: 'Confirm',
      cancelText: 'Cancel',
      confirmClass: 'btn-primary',
      cancelClass: 'btn-secondary',
      ...options
    };

    this.element = null;
    this.backdrop = null;
  }

  /**
   * Open a modal with custom content
   */
  open(content) {
    this.createBackdrop();
    this.createModal(content);
    this.attachEvents();
    this.animateIn();

    // Focus first focusable element
    setTimeout(() => {
      const focusable = this.element.querySelector('input, textarea, select, button:not(.modal-close)');
      if (focusable) focusable.focus();
    }, 100);

    return this;
  }

  /**
   * Create modal backdrop
   */
  createBackdrop() {
    this.backdrop = document.createElement('div');
    this.backdrop.className = 'modal-backdrop';
    this.backdrop.style.cssText = `
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-color: rgba(0, 0, 0, 0.7);
      display: flex;
      align-items: center;
      justify-content: center;
      z-index: ${getComputedStyle(document.documentElement).getPropertyValue('--z-modal-backdrop') || 300};
      padding: 1rem;
      opacity: 0;
      transition: opacity 0.2s ease;
    `;
    document.body.appendChild(this.backdrop);
  }

  /**
   * Create modal element
   */
  createModal(content) {
    const sizeClass = `modal-${this.options.size}`;

    this.element = document.createElement('div');
    this.element.className = `modal ${sizeClass}`;
    this.element.style.cssText = `
      background-color: var(--color-surface);
      border: 1px solid var(--color-border);
      border-radius: var(--radius-xl);
      width: 100%;
      max-width: ${this.getMaxWidth()};
      max-height: 90vh;
      display: flex;
      flex-direction: column;
      box-shadow: var(--shadow-xl);
      transform: scale(0.95);
      opacity: 0;
      transition: transform 0.2s ease, opacity 0.2s ease;
    `;

    this.element.innerHTML = `
      <div class="modal-header" style="
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 1rem 1.5rem;
        border-bottom: 1px solid var(--color-border);
      ">
        <h3 class="modal-title" style="
          font-size: var(--text-lg);
          font-weight: var(--font-semibold);
          color: var(--color-text-primary);
          margin: 0;
        ">${this.options.title}</h3>
        ${this.options.closable ? `
          <button class="modal-close" style="
            background: none;
            border: none;
            color: var(--color-text-tertiary);
            cursor: pointer;
            padding: 0.25rem;
            display: flex;
            align-items: center;
            justify-content: center;
            border-radius: var(--radius-md);
            transition: all 0.15s ease;
          " aria-label="Close">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="6" x2="6" y2="18"/>
              <line x1="6" y1="6" x2="18" y2="18"/>
            </svg>
          </button>
        ` : ''}
      </div>
      <div class="modal-body" style="
        padding: 1.5rem;
        overflow-y: auto;
        flex: 1;
      ">
        ${content}
      </div>
      ${this.options.showFooter ? `
        <div class="modal-footer" style="
          display: flex;
          align-items: center;
          justify-content: flex-end;
          gap: 0.75rem;
          padding: 1rem 1.5rem;
          border-top: 1px solid var(--color-border);
          background-color: rgba(0, 0, 0, 0.2);
        ">
          <button class="btn ${this.options.cancelClass} modal-cancel">${this.options.cancelText}</button>
          <button class="btn ${this.options.confirmClass} modal-confirm">${this.options.confirmText}</button>
        </div>
      ` : ''}
    `;

    this.backdrop.appendChild(this.element);
  }

  /**
   * Get max width based on size
   */
  getMaxWidth() {
    switch (this.options.size) {
      case 'sm': return '400px';
      case 'lg': return '800px';
      case 'xl': return '1000px';
      default: return '600px';
    }
  }

  /**
   * Attach event listeners
   */
  attachEvents() {
    // Close button
    const closeBtn = this.element.querySelector('.modal-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => this.close());
    }

    // Cancel button
    const cancelBtn = this.element.querySelector('.modal-cancel');
    if (cancelBtn) {
      cancelBtn.addEventListener('click', () => this.close());
    }

    // Confirm button
    const confirmBtn = this.element.querySelector('.modal-confirm');
    if (confirmBtn) {
      confirmBtn.addEventListener('click', () => {
        if (this.options.onConfirm) {
          this.options.onConfirm(this);
        } else {
          this.close();
        }
      });
    }

    // Backdrop click
    this.backdrop.addEventListener('click', (e) => {
      if (e.target === this.backdrop && this.options.closable) {
        this.close();
      }
    });

    // Escape key
    this.handleEscape = (e) => {
      if (e.key === 'Escape' && this.options.closable) {
        this.close();
      }
    };
    document.addEventListener('keydown', this.handleEscape);
  }

  /**
   * Animate modal in
   */
  animateIn() {
    requestAnimationFrame(() => {
      this.backdrop.style.opacity = '1';
      this.element.style.transform = 'scale(1)';
      this.element.style.opacity = '1';
    });
  }

  /**
   * Animate modal out and remove
   */
  close() {
    if (this.options.onClose) {
      this.options.onClose();
    }

    this.backdrop.style.opacity = '0';
    this.element.style.transform = 'scale(0.95)';
    this.element.style.opacity = '0';

    setTimeout(() => {
      document.removeEventListener('keydown', this.handleEscape);
      this.backdrop.remove();
    }, 200);
  }

  /**
   * Get form data from modal
   */
  getFormData() {
    const form = this.element.querySelector('form');
    if (form) {
      return Forms.getData(form);
    }
    return {};
  }

  /**
   * Set loading state on confirm button
   */
  setLoading(loading) {
    const confirmBtn = this.element.querySelector('.modal-confirm');
    if (confirmBtn) {
      confirmBtn.disabled = loading;
      confirmBtn.innerHTML = loading ? `
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation: spin 1s linear infinite;">
          <circle cx="12" cy="12" r="10" stroke-dasharray="60" stroke-dashoffset="20"/>
        </svg>
        Loading...
      ` : this.options.confirmText;
    }
  }

  /**
   * Show validation errors
   */
  showErrors(errors) {
    // Clear existing errors
    this.element.querySelectorAll('.form-error').forEach(el => el.remove());
    this.element.querySelectorAll('.is-error').forEach(el => el.classList.remove('is-error'));

    // Show new errors
    for (const [field, message] of Object.entries(errors)) {
      const input = this.element.querySelector(`[name="${field}"]`);
      if (input) {
        input.classList.add('is-error');
        const errorEl = document.createElement('p');
        errorEl.className = 'form-error';
        errorEl.textContent = message;
        input.parentElement.appendChild(errorEl);
      }
    }
  }

  // ==================== STATIC METHODS ====================

  /**
   * Show a confirmation dialog
   */
  static confirm(options = {}) {
    const {
      title = 'Confirm',
      message = 'Are you sure?',
      confirmText = 'Confirm',
      cancelText = 'Cancel',
      confirmClass = 'btn-danger',
      onConfirm = () => {},
      onCancel = () => {}
    } = options;

    const modal = new Modal({
      title,
      size: 'sm',
      confirmText,
      cancelText,
      confirmClass,
      onConfirm: (m) => {
        onConfirm();
        m.close();
      },
      onClose: onCancel
    });

    modal.open(`<p style="color: var(--color-text-secondary); margin: 0;">${message}</p>`);
    return modal;
  }

  /**
   * Show an alert dialog
   */
  static alert(options = {}) {
    const {
      title = 'Alert',
      message = '',
      confirmText = 'OK',
      onConfirm = () => {}
    } = options;

    const modal = new Modal({
      title,
      size: 'sm',
      confirmText,
      cancelText: null,
      onConfirm: (m) => {
        onConfirm();
        m.close();
      }
    });

    modal.open(`<p style="color: var(--color-text-secondary); margin: 0;">${message}</p>`);
    return modal;
  }

  /**
   * Show a form modal
   */
  static form(options = {}) {
    const {
      title = 'Form',
      fields = '',
      validate = null,
      onSubmit = () => {},
      ...modalOptions
    } = options;

    const formContent = `
      <form id="modal-form">
        ${fields}
      </form>
    `;

    const modal = new Modal({
      title,
      ...modalOptions,
      onConfirm: async (m) => {
        const data = m.getFormData();

        // Validate if validator provided
        if (validate) {
          const validation = validate(data);
          if (!validation.valid) {
            m.showErrors(validation.errors);
            return;
          }
        }

        m.setLoading(true);
        try {
          await onSubmit(data, m);
        } catch (error) {
          m.setLoading(false);
          Toast.error(error.message || 'An error occurred');
        }
      }
    });

    modal.open(formContent);
    return modal;
  }
}

window.Modal = Modal;
