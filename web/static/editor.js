/* Editor: likho, turant preview dekho, Ctrl+S se save.
   Preview yahin browser me banta hai - server ko puchne jayenge toh laggy lagega. */

(function () {
  'use strict';

  var box       = document.querySelector('.editor');
  var titleEl   = document.getElementById('title');
  var bodyEl    = document.getElementById('body');
  var tagsEl    = document.getElementById('tags');
  var pubEl     = document.getElementById('published');
  var preview   = document.getElementById('preview');
  var statusEl  = document.getElementById('status');
  var wordEl    = document.getElementById('wordcount');
  var saveBtn   = document.getElementById('saveBtn');
  var deleteBtn = document.getElementById('deleteBtn');

  var slug = box.dataset.slug || '';
  var dirty = false;

  /* ---------- chhota markdown renderer ----------
     Ye poora markdown spec nahi hai - sirf wo cheezein jo log actually likhte hain.
     Asli rendering server pe goldmark karta hai, ye sirf preview ke liye hai.
     Isliye ismein thoda farak ho sakta hai, koi badi baat nahi. */
  function mdToHtml(src) {
    var esc = window.likhoEsc;
    var out = [];
    var lines = src.split('\n');
    var inCode = false, inList = false;

    for (var i = 0; i < lines.length; i++) {
      var L = lines[i];

      // ``` wale code blocks
      if (L.indexOf('```') === 0) {
        if (inCode) { out.push('</code></pre>'); inCode = false; }
        else { closeList(); out.push('<pre><code>'); inCode = true; }
        continue;
      }
      if (inCode) { out.push(esc(L) + '\n'); continue; }

      if (!L.trim()) { closeList(); continue; }

      // headings
      var h = L.match(/^(#{1,4})\s+(.*)$/);
      if (h) {
        closeList();
        var n = h[1].length;
        out.push('<h' + n + '>' + inline(h[2]) + '</h' + n + '>');
        continue;
      }

      // quote
      if (L.indexOf('> ') === 0) {
        closeList();
        out.push('<blockquote>' + inline(L.slice(2)) + '</blockquote>');
        continue;
      }

      // bullet list
      if (/^[-*]\s+/.test(L)) {
        if (!inList) { out.push('<ul>'); inList = true; }
        out.push('<li>' + inline(L.replace(/^[-*]\s+/, '')) + '</li>');
        continue;
      }

      closeList();
      out.push('<p>' + inline(L) + '</p>');
    }

    if (inCode) out.push('</code></pre>');
    closeList();
    return out.join('\n');

    function closeList() { if (inList) { out.push('</ul>'); inList = false; } }

    // line ke andar ki cheezein: bold, italic, code, link, wikilink
    function inline(s) {
      s = esc(s);
      // {fx:preset}...{/fx} - text-animation presets. Naam ki list yahan
      // fx.go ke fxPresets whitelist se match honi chahiye, warna preview
      // aur asli published post alag dikhenge.
      s = s.replace(/\{fx:(legendary|champion|holo)\}([\s\S]*?)\{\/fx\}/g, '<span class="fx fx-$1">$2</span>');
      s = s.replace(/`([^`]+)`/g, '<code>$1</code>');
      s = s.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
      s = s.replace(/\*([^*]+)\*/g, '<em>$1</em>');
      s = s.replace(/\[\[([^\]]+)\]\]/g, '<a href="#">$1</a>');   // preview me link dead rakha hai
      s = s.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>');
      return s;
    }
  }

  function refresh() {
    preview.innerHTML = mdToHtml(bodyEl.value);
    var n = bodyEl.value.trim() ? bodyEl.value.trim().split(/\s+/).length : 0;
    wordEl.textContent = n + ' words · ~' + Math.max(1, Math.round(n / 200)) + ' min';
  }

  bodyEl.addEventListener('input', function () { dirty = true; refresh(); });
  titleEl.addEventListener('input', function () { dirty = true; });
  tagsEl.addEventListener('input', function () { dirty = true; });
  refresh();

  /* ---------- tag suggestion buttons ---------- */
  document.querySelectorAll('.addtag').forEach(function (b) {
    b.onclick = function () {
      var t = b.dataset.tag;
      var cur = tagsEl.value.split(',').map(function (x) { return x.trim(); }).filter(Boolean);
      if (cur.indexOf(t) === -1) {          // do baar mat daalo
        cur.push(t);
        tagsEl.value = cur.join(', ');
        dirty = true;
      }
      b.remove();
    };
  });

  /* ---------- Tab key ---------- */
  // Normally Tab focus hata deta hai. Code likhte waqt ye irritating hai.
  bodyEl.addEventListener('keydown', function (e) {
    if (e.key !== 'Tab') return;
    e.preventDefault();
    var s = bodyEl.selectionStart, en = bodyEl.selectionEnd;
    bodyEl.value = bodyEl.value.slice(0, s) + '  ' + bodyEl.value.slice(en);
    bodyEl.selectionStart = bodyEl.selectionEnd = s + 2;
    dirty = true;
  });

  /* ---------- fx presets: selected text ko glow/shine/holo laga do ---------- */
  // Floating "selection ke upar toolbar" jaanbujhke nahi banaya - phone pe
  // uska apna native selection-bubble pehle se hota hai, do bubble ek saath
  // ladte hain. Isliye ek fixed row hai, aur ye Tab key wale trick jaisa hi
  // kaam karta hai: bas selectionStart/End padho, text wrap karo, cursor wapas rakho.
  function wrapSelectionWithFx(preset) {
    var s = bodyEl.selectionStart, en = bodyEl.selectionEnd;
    if (s === en) {
      say('Pehle thoda text select karo, phir preset lagao ✨');
      return;
    }
    var selected = bodyEl.value.slice(s, en);
    var open = '{fx:' + preset + '}', close = '{/fx}';

    bodyEl.value = bodyEl.value.slice(0, s) + open + selected + close + bodyEl.value.slice(en);
    bodyEl.focus();
    bodyEl.selectionStart = s + open.length;
    bodyEl.selectionEnd = s + open.length + selected.length;

    dirty = true;
    refresh();
  }

  document.querySelectorAll('.fx-btn').forEach(function (b) {
    b.onclick = function () { wrapSelectionWithFx(b.dataset.fx); };
  });

  /* ---------- save ---------- */
  function save(silent) {
    if (!titleEl.value.trim()) {
      say('Title toh daal do 🙃');
      titleEl.focus();
      return;
    }
    say('save ho raha hai...');

    fetch('/api/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': window.likhoCSRF },
      body: JSON.stringify({
        slug: slug,
        title: titleEl.value,
        body: bodyEl.value,
        tags: tagsEl.value,
        published: pubEl.checked
      })
    })
      .then(function (r) { return r.json(); })
      .then(function (d) {
        if (!d.ok) { say('❌ ' + (d.error || 'save nahi hua')); return; }

        dirty = false;
        say('✓ save ho gaya');

        // naya post tha? Ab uska URL mil gaya - address bar update kar do
        // taki dobara save karne pe duplicate na bane.
        if (!slug) {
          slug = d.slug;
          box.dataset.slug = slug;
          history.replaceState({}, '', '/edit/' + slug);
        }
        if (!silent) setTimeout(function () { say(''); }, 2000);
      })
      .catch(function () { say('❌ server se baat nahi ho payi'); });
  }

  saveBtn.onclick = function () { save(false); };

  document.addEventListener('keydown', function (e) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 's') {
      e.preventDefault();
      save(false);
    }
  });

  /* ---------- delete ---------- */
  if (deleteBtn) {
    deleteBtn.onclick = function () {
      if (!confirm('Ye post permanently delete ho jayega. Pakka?')) return;

      fetch('/api/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': window.likhoCSRF },
        body: JSON.stringify({ slug: slug })
      })
        .then(function (r) { return r.json(); })
        .then(function (d) {
          if (d.ok) { dirty = false; location.href = '/'; }
          else say('❌ ' + d.error);
        });
    };
  }

  /* ---------- bina save kiye band mat karo ---------- */
  window.addEventListener('beforeunload', function (e) {
    if (!dirty) return;
    e.preventDefault();
    e.returnValue = '';   // browser apna default message dikhayega
  });

  function say(msg) { statusEl.textContent = msg; }

  // naya post hai toh seedha title me cursor rakho, ek click bacha
  if (box.dataset.new === 'true' && !titleEl.value) titleEl.focus();
  else bodyEl.focus();
})();
