/* Dashboard ke drop-off bars. Har post ke 10 hisse, aur har hisse ki
   opacity batati hai kitne log wahan tak pahunche. */

(function () {
  'use strict';

  document.querySelectorAll('.drow').forEach(function (row) {
    var depth = [];
    try {
      depth = JSON.parse(row.dataset.depth) || [];
    } catch (e) {
      return;   // is post ka data kharab hai, baaki rows chalti rahein
    }

    var bars = row.querySelector('.dbars');
    var pct  = row.querySelector('.dpct');
    var top  = depth[0] || 0;

    if (!top) {
      bars.innerHTML = '<span class="muted small">abhi koi nahi padha</span>';
      return;
    }

    for (var i = 0; i < depth.length; i++) {
      var d = document.createElement('div');
      var ratio = depth[i] / top;
      d.style.opacity = (0.12 + ratio * 0.8).toFixed(2);
      d.title = (i * 10) + '–' + ((i + 1) * 10) + '% : ' + depth[i] + ' log';
      bars.appendChild(d);
    }

    // kitne log aakhir tak pahunche - yahi sabse kaam ka number hai
    var finished = Math.round((depth[9] / top) * 100);
    pct.textContent = finished + '%';
    pct.title = finished + '% log post poora padhte hain';
  });
})();
