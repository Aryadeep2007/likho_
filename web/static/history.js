/* "Isko wapas lao" button. Purana version restore kar deta hai. */

(function () {
  'use strict';

  document.querySelectorAll('.restore').forEach(function (btn) {
    btn.onclick = function (e) {
      // button <summary> ke andar hai, warna click details ko toggle kar deta
      e.preventDefault();
      e.stopPropagation();

      if (!confirm('Is version pe wapas jaana hai?\n\nAbhi wala version bhi history me save ho jayega, toh ye undo ho sakta hai.')) return;

      btn.disabled = true;
      btn.textContent = 'ho raha hai...';

      fetch('/api/restore', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': window.likhoCSRF },
        body: JSON.stringify({ slug: btn.dataset.slug, revisionId: parseInt(btn.dataset.id, 10) })
      })
        .then(function (r) { return r.json(); })
        .then(function (d) {
          if (d.ok) {
            location.href = '/p/' + btn.dataset.slug;
          } else {
            alert('Restore nahi hua: ' + d.error);
            btn.disabled = false;
            btn.textContent = 'Isko wapas lao';
          }
        })
        .catch(function () {
          alert('Server se baat nahi ho payi.');
          btn.disabled = false;
          btn.textContent = 'Isko wapas lao';
        });
    };
  });
})();
