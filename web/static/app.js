/* Har page pe chalne wali cheezein: theme, search palette, QR, import.
   Vanilla JS. Koi React, koi jQuery - kuch nahi. Isliye page turant load hota hai. */

(function () {
  'use strict';

  /* ---------- dark / light ---------- */
  // theme localStorage me yaad rehta hai, warna har refresh pe white flash hota hai
  var saved = localStorage.getItem('likho-theme');
  if (!saved) {
    // pehli baar aaye ho? Toh system ki setting follow karo
    saved = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  document.documentElement.setAttribute('data-theme', saved);

  var themeBtn = document.getElementById('themeBtn');
  if (themeBtn) {
    themeBtn.onclick = function () {
      var next = document.documentElement.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
      document.documentElement.setAttribute('data-theme', next);
      localStorage.setItem('likho-theme', next);
    };
  }

  /* ---------- QR wala popup ---------- */
  var qrOverlay = document.getElementById('qrOverlay');
  var qrBtn = document.getElementById('qrBtn');
  if (qrBtn) qrBtn.onclick = function () { qrOverlay.hidden = false; };

  // overlay ke bahar click karo ya "Theek hai" dabao toh band ho jaye
  document.querySelectorAll('.overlay').forEach(function (ov) {
    ov.addEventListener('click', function (e) {
      if (e.target === ov || e.target.hasAttribute('data-close')) ov.hidden = true;
    });
  });

  /* ---------- import ---------- */
  var bundleInput = document.getElementById('bundleInput');
  var importForm = document.getElementById('importForm');
  if (bundleInput) {
    // label ke andar hidden form hai, toh click manually forward karna padta hai
    var lbl = bundleInput.closest('label');
    if (lbl) lbl.addEventListener('click', function (e) {
      if (e.target !== bundleInput) { e.preventDefault(); bundleInput.click(); }
    });
    bundleInput.onchange = function () {
      if (!bundleInput.files.length) return;
      if (confirm('Import karne se same naam ke posts overwrite ho jayenge. Aage badhein?')) {
        importForm.submit();
      } else {
        bundleInput.value = '';
      }
    };
  }

  /* ---------- Ctrl+K search palette ---------- */
  var overlay = document.getElementById('searchOverlay');
  var input = document.getElementById('searchInput');
  var results = document.getElementById('searchResults');
  var sel = -1;       // abhi kaunsa result highlight hai
  var timer = null;   // debounce ke liye

  function openPalette() {
    overlay.hidden = false;
    input.value = '';
    results.innerHTML = '<div class="no-res">Post ka naam ya andar ka koi word type karo...</div>';
    sel = -1;
    input.focus();
  }
  function closePalette() { overlay.hidden = true; }

  var openBtn = document.getElementById('openSearch');
  if (openBtn) openBtn.onclick = openPalette;

  document.addEventListener('keydown', function (e) {
    // Ctrl+K (ya Mac pe Cmd+K)
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      overlay.hidden ? openPalette() : closePalette();
      return;
    }
    if (e.key === 'Escape') {
      document.querySelectorAll('.overlay').forEach(function (o) { o.hidden = true; });
    }
  });

  if (input) {
    input.addEventListener('input', function () {
      // har keystroke pe request mat bhejo, thoda ruk jao.
      // 120ms itna kam hai ki laga hi nahi ki wait kiya, par server aadhi requests bach gaya.
      clearTimeout(timer);
      timer = setTimeout(runSearch, 120);
    });

    input.addEventListener('keydown', function (e) {
      var items = results.querySelectorAll('.res');
      if (!items.length) return;

      if (e.key === 'ArrowDown') {
        e.preventDefault();
        sel = (sel + 1) % items.length;
        paint(items);
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        sel = sel <= 0 ? items.length - 1 : sel - 1;
        paint(items);
      } else if (e.key === 'Enter') {
        e.preventDefault();
        // kuch select nahi kiya toh pehla wala hi khol do
        (items[sel < 0 ? 0 : sel]).click();
      }
    });
  }

  function paint(items) {
    items.forEach(function (el, i) { el.classList.toggle('sel', i === sel); });
    if (items[sel]) items[sel].scrollIntoView({ block: 'nearest' });
  }

  function runSearch() {
    var q = input.value.trim();
    if (!q) {
      results.innerHTML = '<div class="no-res">Post ka naam ya andar ka koi word type karo...</div>';
      return;
    }

    fetch('/api/search?q=' + encodeURIComponent(q))
      .then(function (r) { return r.json(); })
      .then(function (data) {
        sel = -1;
        if (!data.results || !data.results.length) {
          results.innerHTML = '<div class="no-res">"' + esc(q) + '" pe kuch nahi mila 🤷</div>';
          return;
        }
        results.innerHTML = data.results.map(function (r) {
          return '<a class="res" href="/p/' + r.slug + '">' +
                 '<strong>' + esc(r.title) + (r.draft ? ' <em class="muted">(draft)</em>' : '') + '</strong>' +
                 '<span>' + esc(r.snippet || '') + '</span></a>';
        }).join('');
      })
      .catch(function () {
        results.innerHTML = '<div class="no-res">Search nahi chala. Server band toh nahi ho gaya?</div>';
      });
  }

  // XSS se bachne ke liye. Post ka title user ka likha hua hai,
  // aur agar usme <script> hua toh yahin ghus jayega.
  function esc(s) {
    return String(s).replace(/[&<>"']/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c];
    });
  }

  // dusri script files bhi isko use kar sakein
  window.likhoEsc = esc;

  /* ---------- CSRF token, mutating fetch() calls ke liye ---------- */
  // base.html <head> me ek <meta> tag me daala hua hai. Login/logout jaisi
  // plain form-submit actions ko ye nahi chahiye (wo hidden input use karte
  // hain), par fetch() se jaane wale saare POST/DELETE ko header me ye bhejna hai.
  var csrfMeta = document.querySelector('meta[name="csrf-token"]');
  window.likhoCSRF = csrfMeta ? csrfMeta.content : '';
})();
