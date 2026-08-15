// Nexya Local Printer Dashboard App
const API_BASE = window.location.origin;

let currentPrinters = [];
let currentConfig = {};
let jobsCount = 0;

document.addEventListener('DOMContentLoaded', () => {
  initTabs();
  initActionButtons();
  initForms();
  
  // Initial data load
  fetchHealth();
  fetchPrinters();
  fetchJobs();
  fetchAutostartStatus();

  // Periodic polling every 3 seconds for live job stream & health
  setInterval(() => {
    fetchHealth();
    fetchJobs();
  }, 3000);
});

// Tab Navigation
function initTabs() {
  const tabBtns = document.querySelectorAll('.nav-btn');
  const tabPanes = document.querySelectorAll('.tab-pane');

  tabBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const targetTab = btn.getAttribute('data-tab');
      
      tabBtns.forEach(b => b.classList.remove('active'));
      tabPanes.forEach(p => p.classList.remove('active'));

      btn.classList.add('active');
      const targetPane = document.getElementById(`tab-${targetTab}`);
      if (targetPane) targetPane.classList.add('active');

      if (targetTab === 'printers') fetchPrinters();
      if (targetTab === 'jobs') fetchJobs();
      if (targetTab === 'settings') fetchAutostartStatus();
    });
  });
}

// Action Buttons
function initActionButtons() {
  document.getElementById('btnTestPrint')?.addEventListener('click', handleTestPrint);
  document.getElementById('btnOpenDrawer')?.addEventListener('click', handleOpenDrawer);
  document.getElementById('btnRefreshPrinters')?.addEventListener('click', () => {
    fetchPrinters(true);
    showToast('Lista de impresoras actualizada', 'success');
  });
  document.getElementById('btnReloadPrintersList')?.addEventListener('click', () => {
    fetchPrinters(true);
    showToast('Lista de impresoras actualizada', 'success');
  });
  document.getElementById('btnClearJobs')?.addEventListener('click', handleClearJobs);
}

// Forms
function initForms() {
  document.getElementById('btnSavePrinterConfig')?.addEventListener('click', handleSavePrinterConfig);
  document.getElementById('btnSaveSystemSettings')?.addEventListener('click', handleSaveSystemSettings);
  document.getElementById('chkAutostart')?.addEventListener('change', handleToggleAutostart);
}

// Fetch Health
async function fetchHealth() {
  try {
    const res = await fetch(`${API_BASE}/api/health`);
    if (res.ok) {
      const data = await res.json();
      currentConfig = data.config || {};
      
      // Update UI
      document.getElementById('globalStatusPill').className = 'status-indicator online';
      document.getElementById('globalStatusText').textContent = 'Servicio Activo';
      document.getElementById('statDefaultPrinter').textContent = data.default_printer || 'Predefinida';
      
      const copies = currentConfig.default_copies || 1;
      const width = currentConfig.paper_width || '80mm';
      document.getElementById('statPaperWidth').textContent = `${copies} ${copies === 1 ? 'copia' : 'copias'} • ${width}`;
      document.getElementById('portBadge').textContent = `:${currentConfig.port || '18181'}`;
      
      // Format uptime
      const sec = data.uptime_seconds || 0;
      const m = Math.floor(sec / 60);
      const h = Math.floor(m / 60);
      const d = Math.floor(h / 24);
      let uptimeStr = `${sec % 60}s`;
      if (m > 0) uptimeStr = `${m % 60}m ${uptimeStr}`;
      if (h > 0) uptimeStr = `${h % 24}h ${uptimeStr}`;
      if (d > 0) uptimeStr = `${d}d ${uptimeStr}`;
      document.getElementById('statUptime').textContent = uptimeStr;
    } else {
      setOfflineStatus();
    }
  } catch (err) {
    setOfflineStatus();
  }
}

function setOfflineStatus() {
  document.getElementById('globalStatusPill').className = 'status-indicator offline';
  document.getElementById('globalStatusText').textContent = 'Desconectado';
}

// Fetch Printers
async function fetchPrinters(force = false) {
  try {
    const res = await fetch(`${API_BASE}/api/printers`);
    if (res.ok) {
      const data = await res.json();
      currentPrinters = data.printers || [];
      renderPrintersList(currentPrinters, data.default);
      populatePrinterSelects(currentPrinters);
    }
  } catch (err) {
    console.error('Error fetching printers:', err);
  }
}

function renderPrintersList(printers, defPrinter) {
  const container = document.getElementById('printersListContainer');
  if (!container) return;

  if (!printers || printers.length === 0) {
    container.innerHTML = '<div class="loading-spinner">No se detectaron impresoras instaladas en Windows.</div>';
    return;
  }

  container.innerHTML = printers.map(p => `
    <div class="printer-card-item ${p.is_default ? 'is-default' : ''}">
      <div class="printer-item-info">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="${p.is_default ? '#34d399' : '#818cf8'}" stroke-width="2">
          <polyline points="6 9 6 2 18 2 18 9"></polyline>
          <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"></path>
          <rect x="6" y="14" width="12" height="8"></rect>
        </svg>
        <span class="printer-item-name">${p.name}</span>
      </div>
      <div>
        ${p.is_default ? '<span class="printer-badge-default">PREDETERMINADA</span>' : ''}
      </div>
    </div>
  `).join('');
}

function populatePrinterSelects(printers) {
  const select = document.getElementById('selectDefaultPrinter');
  if (!select) return;

  const currentVal = currentConfig.default_printer || select.value || 'Predefinida';
  let html = '<option value="Predefinida">Impresora Predefinida de Windows</option>';
  
  printers.forEach(p => {
    html += `<option value="${p.name}" ${p.name === currentVal ? 'selected' : ''}>${p.name} ${p.is_default ? '(Predeterminada)' : ''}</option>`;
  });
  
  select.innerHTML = html;

  if (currentConfig.paper_width) {
    document.getElementById('selectPaperWidth').value = currentConfig.paper_width;
  }
  if (currentConfig.default_copies) {
    document.getElementById('selectDefaultCopies').value = String(currentConfig.default_copies);
  }
  if (currentConfig.auto_cut !== undefined) {
    document.getElementById('chkAutoCut').checked = currentConfig.auto_cut;
  }
  if (currentConfig.open_drawer !== undefined) {
    document.getElementById('chkOpenDrawer').checked = currentConfig.open_drawer;
  }
}

// Fetch Jobs History
async function fetchJobs() {
  try {
    const res = await fetch(`${API_BASE}/api/jobs`);
    if (res.ok) {
      const data = await res.json();
      const jobs = data.jobs || [];
      document.getElementById('jobsCountBadge').textContent = jobs.length;
      renderJobsTable(jobs);
    }
  } catch {
    // Silent
  }
}

function renderJobsTable(jobs) {
  const tbody = document.getElementById('jobsTableBody');
  if (!tbody) return;

  if (!jobs || jobs.length === 0) {
    tbody.innerHTML = `
      <tr class="empty-row">
        <td colspan="7">No se han recibido trabajos de impresión en esta sesión.</td>
      </tr>
    `;
    return;
  }

  tbody.innerHTML = jobs.map(job => `
    <tr>
      <td style="font-family: 'JetBrains Mono', monospace; font-size: 0.75rem;">${job.time}</td>
      <td><strong>#${job.order_code || 'TICKET'}</strong></td>
      <td>${job.printer_name || 'Predeterminada'}</td>
      <td><strong>${job.copies || 1}</strong></td>
      <td style="font-weight: 700; color: #34d399;">$ ${Number(job.total || 0).toLocaleString('es-CO')}</td>
      <td>
        <span class="${job.success ? 'badge-success' : 'badge-failed'}">
          ${job.success ? 'Impreso' : 'Error'}
        </span>
      </td>
      <td>
        <button class="btn btn-sm btn-secondary" onclick="handleReprintJob('${job.id}')" title="Volver a enviar a la impresora">
          Re-imprimir
        </button>
      </td>
    </tr>
  `).join('');
}

// Actions
async function handleTestPrint() {
  try {
    const printer = document.getElementById('selectDefaultPrinter')?.value || 'Predefinida';
    const copies = parseInt(document.getElementById('selectDefaultCopies')?.value, 10) || 1;
    const res = await fetch(`${API_BASE}/api/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ printer_name: printer, copies: copies })
    });
    const data = await res.json();
    if (data.success) {
      showToast(`¡Ticket de prueba enviado (${copies} copia/s)!`, 'success');
      fetchJobs();
    } else {
      showToast(`Error: ${data.error || data.message}`, 'error');
    }
  } catch (err) {
    showToast('No se pudo comunicar con el agente local', 'error');
  }
}

async function handleOpenDrawer() {
  try {
    const printer = document.getElementById('selectDefaultPrinter')?.value || 'Predefinida';
    const res = await fetch(`${API_BASE}/api/drawer/open`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ printer_name: printer })
    });
    const data = await res.json();
    if (data.success) {
      showToast('Pulso de apertura de cajón enviado exitosamente', 'success');
    } else {
      showToast(`Error abriendo cajón: ${data.error}`, 'error');
    }
  } catch (err) {
    showToast('Error al enviar comando de cajón', 'error');
  }
}

async function handleReprintJob(jobId) {
  try {
    const res = await fetch(`${API_BASE}/api/jobs/reprint`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ job_id: jobId })
    });
    const data = await res.json();
    if (data.success) {
      showToast('Trabajo re-enviado a la impresora', 'success');
      fetchJobs();
    } else {
      showToast(`Error al re-imprimir: ${data.error}`, 'error');
    }
  } catch (err) {
    showToast('Error de conexión', 'error');
  }
}

async function handleClearJobs() {
  try {
    await fetch(`${API_BASE}/api/jobs`, { method: 'DELETE' });
    fetchJobs();
    showToast('Historial de trabajos limpiado', 'success');
  } catch (err) {
    showToast('Error al limpiar historial', 'error');
  }
}

async function handleSavePrinterConfig() {
  try {
    const payload = {
      ...currentConfig,
      default_printer: document.getElementById('selectDefaultPrinter').value,
      paper_width: document.getElementById('selectPaperWidth').value,
      default_copies: parseInt(document.getElementById('selectDefaultCopies').value, 10) || 1,
      auto_cut: document.getElementById('chkAutoCut').checked,
      open_drawer: document.getElementById('chkOpenDrawer').checked,
    };

    const res = await fetch(`${API_BASE}/api/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      showToast('Configuración de impresora guardada', 'success');
      fetchHealth();
    } else {
      showToast('Error al guardar configuración', 'error');
    }
  } catch (err) {
    showToast('Error de conexión', 'error');
  }
}

async function handleSaveSystemSettings() {
  try {
    const port = document.getElementById('inputPort').value;
    const logLevel = document.getElementById('selectLogLevel').value;

    const payload = {
      ...currentConfig,
      port: port || '18181',
      log_level: logLevel || 'info',
    };

    const res = await fetch(`${API_BASE}/api/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    if (res.ok) {
      showToast('Ajustes del sistema guardados', 'success');
      fetchHealth();
    } else {
      showToast('Error al guardar ajustes', 'error');
    }
  } catch (err) {
    showToast('Error de conexión', 'error');
  }
}

// Autostart with Windows
async function fetchAutostartStatus() {
  try {
    const res = await fetch(`${API_BASE}/api/autostart`);
    if (res.ok) {
      const data = await res.json();
      document.getElementById('chkAutostart').checked = data.enabled === true;
    }
  } catch {
    // Silent
  }
}

async function handleToggleAutostart(e) {
  const enabled = e.target.checked;
  try {
    const res = await fetch(`${API_BASE}/api/autostart`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ enabled })
    });
    const data = await res.json();
    if (data.success) {
      showToast(enabled ? 'Inicio automático con Windows ACTIVADO' : 'Inicio automático con Windows DESACTIVADO', 'success');
    } else {
      showToast(`Error: ${data.error}`, 'error');
      e.target.checked = !enabled;
    }
  } catch (err) {
    showToast('Error al configurar inicio con Windows', 'error');
    e.target.checked = !enabled;
  }
}

// Toast Helper
function showToast(msg, type = 'success') {
  const container = document.getElementById('toastContainer');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerHTML = `
    <span>${type === 'success' ? '✅' : '❌'}</span>
    <span>${msg}</span>
  `;

  container.appendChild(toast);
  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transform = 'translateY(10px)';
    toast.style.transition = 'all 0.3s ease';
    setTimeout(() => toast.remove(), 300);
  }, 3500);
}
