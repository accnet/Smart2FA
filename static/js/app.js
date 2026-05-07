// ============================================================
// Smart2FA — Client JS
// ============================================================

// --- Service Worker registration (PWA) ---
if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker.register('/static/sw.js').catch(() => {});
  });
}

// --- Active Group state ---
let currentGroup = localStorage.getItem('activeGroup') || 'Default';
let refreshInterval = null;

// ============================================================
// Tab Group logic
// ============================================================

function switchTab(btn, group) {
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  currentGroup = group;
  localStorage.setItem('activeGroup', group);
  loadCodes();
}

function loadCodes() {
  htmx.ajax('GET', '/partial/codes?group=' + encodeURIComponent(currentGroup), {
    target: '#codes-container',
    swap: 'innerHTML settle:0ms',
  });
}

function startRefresh() {
  if (refreshInterval) clearInterval(refreshInterval);
  refreshInterval = setInterval(loadCodes, 5000);
}

// Open inline editable new-group tab
function openNewGroupTab() {
  // Prevent duplicate
  if (document.querySelector('.tab-editing')) return;

  const tabsBar = document.getElementById('tabs-bar');
  const newBtn  = document.getElementById('new-group-btn');
  const prevGroup = currentGroup;

  // Deactivate all existing tabs
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));

  // Create the editing tab element
  const tab = document.createElement('button');
  tab.className = 'tab-btn tab-editing active';

  const input = document.createElement('input');
  input.type = 'text';
  input.className = 'tab-edit-input';
  input.placeholder = 'Group name…';
  input.maxLength = 40;
  input.setAttribute('spellcheck', 'false');

  tab.appendChild(input);
  tabsBar.insertBefore(tab, newBtn);
  input.focus();

  // Clear content immediately — don't show old group's entries while typing
  const container = document.getElementById('codes-container');
  if (container) container.innerHTML = '';

  let committed = false;

  function commit() {
    if (committed) return;
    committed = true;

    const name = input.value.trim();

    // Cancel: empty name
    if (!name) {
      tab.remove();
      const prevBtn = document.querySelector(`.tab-btn[data-group="${CSS.escape(prevGroup)}"]`);
      if (prevBtn) prevBtn.classList.add('active');
      currentGroup = prevGroup;
      loadCodes();
      return;
    }

    // Duplicate: group already exists → switch to it
    const existing = document.querySelector(`.tab-btn[data-group="${CSS.escape(name)}"]`);
    if (existing) {
      tab.remove();
      switchTab(existing, name);
      return;
    }

    // Finalize tab: replace input with label, keep active class
    tab.classList.remove('tab-editing');
    tab.dataset.group = name;
    tab.innerHTML = `<span class="tab-label">${escHtml(name)}</span>`;
    tab.addEventListener('click', () => switchTab(tab, name));

    // Switch to the new group
    currentGroup = name;
    localStorage.setItem('activeGroup', name);

    // Scroll new tab into view
    tab.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });

    // Load codes (will return empty state from server)
    loadCodes();
    startRefresh();
  }

  input.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); commit(); }
    if (e.key === 'Escape') {
      committed = true;
      tab.remove();
      const prevBtn = document.querySelector(`.tab-btn[data-group="${CSS.escape(prevGroup)}"]`);
      if (prevBtn) prevBtn.classList.add('active');
      currentGroup = prevGroup;
      loadCodes();
    }
  });

  // Commit on blur (click away), small delay to avoid race with Enter
  input.addEventListener('blur', () => setTimeout(commit, 120));
}

function escHtml(str) {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// ============================================================
// Restore active tab on page load
// ============================================================
document.addEventListener('DOMContentLoaded', () => {
  const saved = localStorage.getItem('activeGroup') || 'Default';
  let found = false;

  document.querySelectorAll('.tab-btn').forEach(btn => {
    if (btn.dataset.group === saved) {
      btn.classList.add('active');
      found = true;
    }
  });

  if (!found) {
    // Saved group no longer exists → reset to Default
    const defBtn = document.querySelector('.tab-btn[data-group="Default"]');
    if (defBtn) defBtn.classList.add('active');
    currentGroup = 'Default';
    localStorage.setItem('activeGroup', 'Default');
  } else {
    currentGroup = saved;
    // If not Default, override the hx-get URL before HTMX fires
    if (saved !== 'Default') {
      const container = document.getElementById('codes-container');
      if (container) {
        container.setAttribute('hx-get', '/partial/codes?group=' + encodeURIComponent(saved));
        htmx.process(container);
        htmx.trigger(container, 'load');
      }
    }
  }

  startRefresh();

  // Double-click on any tab to rename it (except Default)
  document.getElementById('tabs-bar')?.addEventListener('dblclick', e => {
    const btn = e.target.closest('.tab-btn');
    if (btn && btn.id !== 'new-group-btn' && !btn.classList.contains('tab-editing')) {
      startTabRename(btn);
    }
  });
});

// ============================================================
// Tab Rename (double-click)
// ============================================================
function startTabRename(tabBtn) {
  const group = tabBtn.dataset.group;
  if (!group || group === 'Default') return; // Default is immutable

  const label = tabBtn.querySelector('.tab-label');
  const badge = tabBtn.querySelector('.tab-badge');
  if (!label) return;

  const origName = group;

  // Hide label+badge, show input
  label.style.display = 'none';
  if (badge) badge.style.display = 'none';
  tabBtn.classList.add('tab-editing');

  const input = document.createElement('input');
  input.type = 'text';
  input.className = 'tab-edit-input';
  input.value = origName;
  input.setAttribute('spellcheck', 'false');
  tabBtn.insertBefore(input, label);
  input.select();

  let committed = false;

  function restore() {
    input.remove();
    label.style.display = '';
    if (badge) badge.style.display = '';
    tabBtn.classList.remove('tab-editing');
  }

  function commit() {
    if (committed) return;
    committed = true;

    const newName = input.value.trim();
    restore();

    if (!newName || newName === origName) return;

    // Duplicate check
    const dup = document.querySelector(`.tab-btn[data-group="${CSS.escape(newName)}"]`);
    if (dup) { showToast('Group already exists!'); return; }

    // Update DOM immediately
    tabBtn.dataset.group = newName;
    label.textContent = newName;
    if (currentGroup === origName) {
      currentGroup = newName;
      localStorage.setItem('activeGroup', newName);
    }

    // Persist to server (non-blocking)
    const fd = new FormData();
    fd.append('old_name', origName);
    fd.append('new_name', newName);
    fetch('/group/rename', { method: 'POST', body: fd })
      .then(res => { if (!res.ok) { showToast('Rename failed!'); } })
      .catch(() => showToast('Rename failed!'));
  }

  input.addEventListener('keydown', e => {
    if (e.key === 'Enter') { e.preventDefault(); commit(); }
    if (e.key === 'Escape') { committed = true; restore(); }
  });
  input.addEventListener('blur', () => setTimeout(commit, 120));
}

// ============================================================
// Add Account Modal
// ============================================================
function openAddModal() {
  const m = document.getElementById('add-modal');
  if (!m) return;
  m.classList.remove('hidden');
  document.body.style.overflow = 'hidden';

  // Set the group to the currently active tab
  const gInput = document.getElementById('entry-group');
  const gTag   = document.getElementById('modal-group-tag');
  if (gInput) gInput.value = currentGroup;
  if (gTag)   gTag.textContent = currentGroup;

  setTimeout(() => document.getElementById('entry-name')?.focus(), 60);
}

function closeAddModal() {
  const m = document.getElementById('add-modal');
  if (m) { m.classList.add('hidden'); document.body.style.overflow = ''; }
  const form = document.getElementById('add-entry-form');
  if (form) form.reset();
}

function closeModalOnBg(e) {
  if (e.target === document.getElementById('add-modal')) closeAddModal();
}

// ============================================================
// Edit Modal
// ============================================================
function openEditModal(btn) {
  const name  = btn.dataset.name  || '';
  const group = btn.dataset.group || 'Default';

  document.getElementById('edit-orig-name').value  = name;
  document.getElementById('edit-entry-name').value  = name;
  document.getElementById('edit-entry-secret').value = '';
  document.getElementById('edit-entry-group').value  = group;

  const m = document.getElementById('edit-modal');
  if (m) { m.classList.remove('hidden'); document.body.style.overflow = 'hidden'; }
  setTimeout(() => document.getElementById('edit-entry-name')?.select(), 60);
}

function closeEditModal() {
  const m = document.getElementById('edit-modal');
  if (m) { m.classList.add('hidden'); document.body.style.overflow = ''; }
  const form = document.getElementById('edit-entry-form');
  if (form) form.reset();
}

function closeEditModalOnBg(e) {
  if (e.target === document.getElementById('edit-modal')) closeEditModal();
}

document.addEventListener('keydown', e => {
  if (e.key === 'Escape') { closeAddModal(); closeEditModal(); }
});

// ============================================================
// Copy to clipboard
// ============================================================
function copyCode(btn, code) {
  const raw = code.replace(/\s/g, '');
  navigator.clipboard.writeText(raw).then(() => {
    btn.textContent = '✓ Copied';
    btn.style.background = 'var(--accent)';
    btn.style.color = 'white';
    showToast('Copied!');
    setTimeout(() => { btn.textContent = 'Copy'; btn.style.background = ''; btn.style.color = ''; }, 1500);
  }).catch(() => {
    const ta = document.createElement('textarea');
    ta.value = raw; ta.style.position = 'fixed'; ta.style.opacity = '0';
    document.body.appendChild(ta); ta.select(); document.execCommand('copy');
    document.body.removeChild(ta);
    showToast('Copied!');
  });
}

// ============================================================
// Toast
// ============================================================
function showToast(msg) {
  const t = document.getElementById('toast');
  if (!t) return;
  t.textContent = msg;
  t.classList.remove('hidden');
  clearTimeout(t._timer);
  t._timer = setTimeout(() => t.classList.add('hidden'), 1800);
}

// ============================================================
// Smooth client-side timer bars
// ============================================================
let timerInterval = null;

function startTimerBars() {
  if (timerInterval) clearInterval(timerInterval);
  timerInterval = setInterval(updateTimerBars, 1000);
}

function updateTimerBars() {
  const now = Math.floor(Date.now() / 1000);
  const remaining = 30 - (now % 30);
  const pct = (remaining / 30) * 100;

  document.querySelectorAll('.timer-bar').forEach(bar => {
    bar.style.width = pct.toFixed(1) + '%';
    bar.style.background = remaining <= 7
      ? 'linear-gradient(90deg, #ff5a5a, #ff8a8a)'
      : '';
  });

  document.querySelectorAll('.entry-code').forEach(el => {
    el.classList.toggle('code-expiring', remaining <= 7);
  });
}

startTimerBars();

document.body.addEventListener('htmx:afterSwap', e => {
  if (e.detail.target?.id === 'codes-container') {
    startTimerBars();
    updateTimerBars();
  }
});

// ============================================================
// Unlock form spinner
// ============================================================
const unlockForm = document.getElementById('unlock-form');
if (unlockForm) {
  unlockForm.addEventListener('submit', () => {
    const btn = document.getElementById('unlock-btn');
    if (btn) {
      btn.querySelector('.btn-text').textContent = 'Unlocking…';
      btn.disabled = true;
    }
  });
}

// ============================================================
// PWA Install
// ============================================================
let _deferredInstallPrompt = null;

// Android Chrome: capture install prompt
window.addEventListener('beforeinstallprompt', e => {
  e.preventDefault();
  _deferredInstallPrompt = e;
  const btn = document.getElementById('install-btn');
  if (btn) btn.style.display = '';
});

window.addEventListener('appinstalled', () => {
  _deferredInstallPrompt = null;
  const btn = document.getElementById('install-btn');
  if (btn) btn.style.display = 'none';
  showToast('App installed! 🎉');
});

function installPWA() {
  if (!_deferredInstallPrompt) return;
  _deferredInstallPrompt.prompt();
  _deferredInstallPrompt.userChoice.then(() => {
    _deferredInstallPrompt = null;
    const btn = document.getElementById('install-btn');
    if (btn) btn.style.display = 'none';
  });
}

// iOS Safari: show banner if not already in standalone
(function checkIOS() {
  const isIOS = /iphone|ipad|ipod/i.test(navigator.userAgent);
  const isStandalone = window.navigator.standalone ||
                       window.matchMedia('(display-mode: standalone)').matches;
  if (isIOS && !isStandalone) {
    const dismissed = sessionStorage.getItem('ios-banner-dismissed');
    if (!dismissed) {
      document.addEventListener('DOMContentLoaded', () => {
        const banner = document.getElementById('ios-banner');
        if (banner) {
          banner.classList.remove('hidden');
          // Auto-hide after 8s
          setTimeout(() => banner.classList.add('hidden'), 8000);
        }
      });
    }
  }
})();
