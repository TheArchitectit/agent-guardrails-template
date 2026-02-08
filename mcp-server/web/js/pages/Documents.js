/**
 * Documents Page
 * Browse, search, view and edit documentation
 */

class Documents {
  constructor(container) {
    this.container = container;
    this.documents = [];
    this.categories = ['workflow', 'standard', 'guide', 'reference'];
    this.table = null;
    this.currentDoc = null;
    this.render();
    this.loadData();
  }

  render() {
    this.container.innerHTML = `
      <div class="page">
        <div class="page-header">
          <div>
            <h1 class="page-title">Documents</h1>
            <p class="page-description">Browse and manage documentation</p>
          </div>
        </div>

        <div class="toolbar">
          <div class="toolbar-left">
            ${Forms.search({ placeholder: 'Search documents...', id: 'doc-search' })}
          </div>
          <div class="toolbar-right">
            <select class="form-select" id="category-filter" style="width: auto;">
              <option value="">All Categories</option>
              ${this.categories.map(c => `<option value="${c}">${this.capitalize(c)}</option>`).join('')}
            </select>
          </div>
        </div>

        <div id="documents-table"></div>
      </div>
    `;

    this.attachEvents();
  }

  attachEvents() {
    // Search
    const searchInput = this.container.querySelector('#doc-search');
    if (searchInput) {
      let debounceTimer;
      searchInput.addEventListener('input', (e) => {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(() => {
          this.handleSearch(e.target.value);
        }, 300);
      });
    }

    // Category filter
    const categoryFilter = this.container.querySelector('#category-filter');
    if (categoryFilter) {
      categoryFilter.addEventListener('change', (e) => {
        this.loadData({ category: e.target.value });
      });
    }
  }

  async loadData(params = {}) {
    try {
      if (this.table) {
        this.table.setLoading(true);
      }

      const response = await window.api.listDocuments({
        limit: 100,
        ...params
      });

      this.documents = response.data || [];
      this.renderTable();
    } catch (error) {
      Toast.error('Failed to load documents: ' + error.message);
    }
  }

  renderTable() {
    const tableContainer = this.container.querySelector('#documents-table');

    this.table = new DataTable(tableContainer, {
      columns: [
        {
          key: 'title',
          title: 'Title',
          sortable: true,
          formatter: (val, row) => `<span style="font-weight: 500;">${val}</span>`
        },
        {
          key: 'slug',
          title: 'Slug',
          sortable: true
        },
        {
          key: 'category',
          title: 'Category',
          sortable: true,
          formatter: (val) => `<span class="badge badge-${this.getCategoryColor(val)}">${this.capitalize(val)}</span>`
        },
        {
          key: 'version',
          title: 'Version',
          sortable: true,
          width: '100px'
        },
        {
          key: 'updated_at',
          title: 'Updated',
          sortable: true,
          formatter: (val) => this.formatDate(val)
        }
      ],
      data: this.documents,
      onRowClick: (doc) => this.openDocument(doc),
      rowActions: [
        {
          key: 'edit',
          label: 'Edit',
          type: 'ghost',
          icon: `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>`,
          handler: (doc) => this.openDocument(doc, true)
        }
      ],
      emptyText: 'No documents found',
      emptyDescription: 'Try adjusting your search or filter criteria.'
    });
  }

  async handleSearch(query) {
    if (!query.trim()) {
      this.loadData();
      return;
    }

    try {
      this.table.setLoading(true);
      const response = await window.api.searchDocuments(query);
      this.documents = response.data || [];
      this.table.setData(this.documents);
    } catch (error) {
      Toast.error('Search failed: ' + error.message);
    }
  }

  openDocument(doc, editMode = false) {
    this.currentDoc = doc;

    const content = editMode ? this.renderEditForm(doc) : this.renderViewContent(doc);

    Modal.form({
      title: editMode ? `Edit: ${doc.title}` : doc.title,
      size: 'lg',
      fields: content,
      confirmText: editMode ? 'Save Changes' : 'Close',
      cancelText: editMode ? 'Cancel' : null,
      confirmClass: editMode ? 'btn-primary' : 'btn-secondary',
      onConfirm: editMode ? (data, modal) => this.saveDocument(doc.id, data, modal) : null
    });
  }

  renderViewContent(doc) {
    return `
      <div style="display: flex; flex-direction: column; gap: var(--space-4);">
        <div style="display: flex; gap: var(--space-4); flex-wrap: wrap;">
          <span class="badge badge-${this.getCategoryColor(doc.category)}">${this.capitalize(doc.category)}</span>
          <span style="font-size: var(--text-sm); color: var(--color-text-secondary);">
            Version ${doc.version}
          </span>
          <span style="font-size: var(--text-sm); color: var(--color-text-secondary);">
            Updated ${this.formatDate(doc.updated_at)}
          </span>
        </div>
        <div class="document-content" style="
          background-color: var(--color-bg-secondary);
          border: 1px solid var(--color-border);
          border-radius: var(--radius-md);
          padding: var(--space-4);
          max-height: 500px;
          overflow-y: auto;
          font-family: var(--font-family-mono);
          font-size: var(--text-sm);
          white-space: pre-wrap;
          line-height: 1.6;
        ">${this.escapeHtml(doc.content)}</div>
      </div>
    `;
  }

  renderEditForm(doc) {
    return `
      ${Forms.input({
        name: 'title',
        label: 'Title',
        value: doc.title,
        required: true
      })}
      ${Forms.select({
        name: 'category',
        label: 'Category',
        options: this.categories.map(c => ({ value: c, label: this.capitalize(c) })),
        value: doc.category,
        required: true
      })}
      ${Forms.textarea({
        name: 'content',
        label: 'Content',
        value: doc.content,
        rows: 20,
        required: true,
        hint: 'Markdown content'
      })}
    `;
  }

  async saveDocument(id, data, modal) {
    try {
      modal.setLoading(true);
      await window.api.updateDocument(id, data);
      Toast.success('Document updated successfully');
      modal.close();
      this.loadData();
    } catch (error) {
      modal.setLoading(false);
      throw error;
    }
  }

  getCategoryColor(category) {
    const colors = {
      workflow: 'primary',
      standard: 'success',
      guide: 'warning',
      reference: 'info'
    };
    return colors[category] || 'neutral';
  }

  capitalize(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
  }

  formatDate(dateStr) {
    if (!dateStr) return '-';
    const date = new Date(dateStr);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }
}

window.Documents = Documents;
