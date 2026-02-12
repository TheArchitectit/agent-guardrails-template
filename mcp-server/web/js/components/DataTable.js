/**
 * DataTable Component
 * Sortable, paginated table with row actions and empty states
 */

class DataTable {
  constructor(container, options = {}) {
    this.container = container;
    this.options = {
      columns: [],
      data: [],
      sortable: true,
      pagination: true,
      pageSize: 20,
      pageSizeOptions: [10, 20, 50, 100],
      onRowClick: null,
      rowActions: [],
      emptyText: 'No data available',
      emptyDescription: 'There are no items to display.',
      ...options
    };

    this.state = {
      data: [],
      filteredData: [],
      sortColumn: null,
      sortDirection: 'asc',
      currentPage: 1,
      pageSize: this.options.pageSize,
      loading: false
    };

    this.init();
  }

  init() {
    this.render();
    this.setData(this.options.data);
  }

  render() {
    this.container.innerHTML = `
      <div class="data-table-wrapper">
        <div class="table-container">
          <table class="table">
            <thead>
              <tr>
                ${this.options.columns.map(col => this.renderHeader(col)).join('')}
                ${this.options.rowActions.length ? '<th>Actions</th>' : ''}
              </tr>
            </thead>
            <tbody id="table-body">
              <tr><td colspan="${this.options.columns.length + (this.options.rowActions.length ? 1 : 0)}" class="table-empty">
                <div class="loading-state">
                  <div class="spinner"></div>
                  <p class="loading-text">Loading...</p>
                </div>
              </td></tr>
            </tbody>
          </table>
        </div>
        ${this.options.pagination ? this.renderPagination() : ''}
      </div>
    `;

    this.attachEvents();
  }

  renderHeader(column) {
    const sortable = this.options.sortable && column.sortable !== false;
    const isSorted = this.state.sortColumn === column.key;
    const sortClass = isSorted ? this.state.sortDirection : '';

    return `
      <th class="${sortable ? 'sortable' : ''} ${sortClass}"
          ${sortable ? `data-sort="${column.key}"` : ''}
          style="${column.width ? `width: ${column.width}` : ''}">
        ${column.title}
      </th>
    `;
  }

  renderRow(item, index) {
    return `
      <tr data-index="${index}" class="${this.options.onRowClick ? 'clickable' : ''}">
        ${this.options.columns.map(col => this.renderCell(item, col)).join('')}
        ${this.options.rowActions.length ? this.renderRowActions(item, index) : ''}
      </tr>
    `;
  }

  renderCell(item, column) {
    let value = this.getNestedValue(item, column.key);

    if (column.formatter) {
      value = column.formatter(value, item);
    } else if (value === null || value === undefined) {
      value = '-';
    }

    return `<td>${value}</td>`;
  }

  renderRowActions(item, index) {
    return `
      <td>
        <div class="row-actions">
          ${this.options.rowActions.map(action => `
            <button class="btn btn-sm btn-${action.type || 'ghost'}"
                    data-action="${action.key}"
                    data-index="${index}"
                    title="${action.label}">
              ${action.icon || action.label}
            </button>
          `).join('')}
        </div>
      </td>
    `;
  }

  renderPagination() {
    const totalPages = Math.ceil(this.state.filteredData.length / this.state.pageSize);
    const start = (this.state.currentPage - 1) * this.state.pageSize + 1;
    const end = Math.min(this.state.currentPage * this.state.pageSize, this.state.filteredData.length);

    return `
      <div class="table-pagination">
        <div class="pagination-info">
          Showing ${this.state.filteredData.length ? start : 0} to ${end} of ${this.state.filteredData.length} entries
        </div>
        <div class="pagination">
          <button class="pagination-btn" data-page="prev" ${this.state.currentPage === 1 ? 'disabled' : ''}>
            Previous
          </button>
          ${this.renderPageButtons(totalPages)}
          <button class="pagination-btn" data-page="next" ${this.state.currentPage === totalPages || totalPages === 0 ? 'disabled' : ''}>
            Next
          </button>
        </div>
        <div class="page-size-selector">
          <select class="form-select" id="page-size" style="width: auto;">
            ${this.options.pageSizeOptions.map(size => `
              <option value="${size}" ${size === this.state.pageSize ? 'selected' : ''}>${size} / page</option>
            `).join('')}
          </select>
        </div>
      </div>
    `;
  }

  renderPageButtons(totalPages) {
    if (totalPages <= 1) return '';

    const buttons = [];
    const maxVisible = 5;
    let start = Math.max(1, this.state.currentPage - Math.floor(maxVisible / 2));
    let end = Math.min(totalPages, start + maxVisible - 1);

    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1);
    }

    if (start > 1) {
      buttons.push(`<button class="pagination-btn" data-page="1">1</button>`);
      if (start > 2) buttons.push(`<span class="pagination-ellipsis">...</span>`);
    }

    for (let i = start; i <= end; i++) {
      buttons.push(`
        <button class="pagination-btn ${i === this.state.currentPage ? 'active' : ''}" data-page="${i}">
          ${i}
        </button>
      `);
    }

    if (end < totalPages) {
      if (end < totalPages - 1) buttons.push(`<span class="pagination-ellipsis">...</span>`);
      buttons.push(`<button class="pagination-btn" data-page="${totalPages}">${totalPages}</button>`);
    }

    return buttons.join('');
  }

  renderEmpty() {
    return `
      <tr>
        <td colspan="${this.options.columns.length + (this.options.rowActions.length ? 1 : 0)}" class="table-empty">
          <div class="empty-state">
            <svg class="empty-state-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
              <line x1="9" y1="9" x2="15" y2="15"/>
              <line x1="15" y1="9" x2="9" y2="15"/>
            </svg>
            <h3 class="empty-state-title">${this.options.emptyText}</h3>
            <p class="empty-state-description">${this.options.emptyDescription}</p>
          </div>
        </td>
      </tr>
    `;
  }

  getNestedValue(obj, path) {
    return path.split('.').reduce((acc, part) => acc && acc[part], obj);
  }

  attachEvents() {
    // Sort
    if (this.options.sortable) {
      this.container.querySelectorAll('th.sortable').forEach(th => {
        th.addEventListener('click', (e) => {
          const column = e.currentTarget.dataset.sort;
          this.handleSort(column);
        });
      });
    }

    // Row click
    if (this.options.onRowClick) {
      this.container.addEventListener('click', (e) => {
        const row = e.target.closest('tr[data-index]');
        if (row && !e.target.closest('.row-actions')) {
          const index = parseInt(row.dataset.index);
          const item = this.getCurrentPageData()[index];
          this.options.onRowClick(item);
        }
      });
    }

    // Row actions
    this.container.addEventListener('click', (e) => {
      const btn = e.target.closest('button[data-action]');
      if (btn) {
        const action = btn.dataset.action;
        const index = parseInt(btn.dataset.index);
        const item = this.getCurrentPageData()[index];
        const actionConfig = this.options.rowActions.find(a => a.key === action);
        if (actionConfig && actionConfig.handler) {
          actionConfig.handler(item, index);
        }
      }
    });

    // Pagination
    this.container.addEventListener('click', (e) => {
      const btn = e.target.closest('button[data-page]');
      if (btn) {
        const page = btn.dataset.page;
        this.handlePageChange(page);
      }
    });

    // Page size
    const pageSizeSelect = this.container.querySelector('#page-size');
    if (pageSizeSelect) {
      pageSizeSelect.addEventListener('change', (e) => {
        this.state.pageSize = parseInt(e.target.value);
        this.state.currentPage = 1;
        this.refresh();
      });
    }
  }

  handleSort(column) {
    if (this.state.sortColumn === column) {
      this.state.sortDirection = this.state.sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      this.state.sortColumn = column;
      this.state.sortDirection = 'asc';
    }

    this.sortData();
    this.refresh();
  }

  handlePageChange(page) {
    const totalPages = Math.ceil(this.state.filteredData.length / this.state.pageSize);

    if (page === 'prev') {
      this.state.currentPage = Math.max(1, this.state.currentPage - 1);
    } else if (page === 'next') {
      this.state.currentPage = Math.min(totalPages, this.state.currentPage + 1);
    } else {
      this.state.currentPage = parseInt(page);
    }

    this.refresh();
  }

  sortData() {
    if (!this.state.sortColumn) return;

    const column = this.options.columns.find(c => c.key === this.state.sortColumn);
    if (!column) return;

    this.state.filteredData.sort((a, b) => {
      let aVal = this.getNestedValue(a, this.state.sortColumn);
      let bVal = this.getNestedValue(b, this.state.sortColumn);

      if (column.sortFn) {
        return this.state.sortDirection === 'asc'
          ? column.sortFn(aVal, bVal)
          : column.sortFn(bVal, aVal);
      }

      if (typeof aVal === 'string') {
        aVal = aVal.toLowerCase();
        bVal = bVal.toLowerCase();
      }

      if (aVal < bVal) return this.state.sortDirection === 'asc' ? -1 : 1;
      if (aVal > bVal) return this.state.sortDirection === 'asc' ? 1 : -1;
      return 0;
    });
  }

  getCurrentPageData() {
    const start = (this.state.currentPage - 1) * this.state.pageSize;
    const end = start + this.state.pageSize;
    return this.state.filteredData.slice(start, end);
  }

  refresh() {
    const tbody = this.container.querySelector('#table-body');
    const data = this.getCurrentPageData();

    if (data.length === 0) {
      tbody.innerHTML = this.renderEmpty();
    } else {
      tbody.innerHTML = data.map((item, index) => this.renderRow(item, index)).join('');
    }

    // Update header sort indicators
    this.container.querySelectorAll('th.sortable').forEach(th => {
      th.classList.remove('asc', 'desc');
      if (th.dataset.sort === this.state.sortColumn) {
        th.classList.add(this.state.sortDirection);
      }
    });

    // Update pagination
    if (this.options.pagination) {
      const paginationContainer = this.container.querySelector('.table-pagination');
      if (paginationContainer) {
        paginationContainer.outerHTML = this.renderPagination();
      }
    }
  }

  setData(data) {
    this.state.data = [...data];
    this.state.filteredData = [...data];
    this.state.currentPage = 1;
    if (this.state.sortColumn) {
      this.sortData();
    }
    this.refresh();
  }

  filter(fn) {
    if (fn) {
      this.state.filteredData = this.state.data.filter(fn);
    } else {
      this.state.filteredData = [...this.state.data];
    }
    this.state.currentPage = 1;
    this.refresh();
  }

  setLoading(loading) {
    this.state.loading = loading;
    if (loading) {
      const tbody = this.container.querySelector('#table-body');
      tbody.innerHTML = `
        <tr><td colspan="${this.options.columns.length + (this.options.rowActions.length ? 1 : 0)}" class="table-empty">
          <div class="loading-state">
            <div class="spinner"></div>
            <p class="loading-text">Loading...</p>
          </div>
        </td></tr>
      `;
    }
  }
}

window.DataTable = DataTable;
