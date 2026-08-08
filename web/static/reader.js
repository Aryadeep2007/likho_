/* Do kaam:
   1. left wali patti me dikhao ki pichhle readers kahan tak padhe the
   2. ye reader kitna neeche gaya, wo server ko bhejo (jaate waqt)

   Ye Google Analytics nahi hai - koi cookie nahi, koi ID nahi, kisi ka
   naam nahi. Bas ek counter: "itne log yahan tak pahunche". Sab file me,
   isi computer pe. */

(function () {
  'use strict';

  var post = document.querySelector('.post');
  if (!post) return;

  var slug = post.dataset.slug;
  var strip = document.getElementById('heatstrip');
  var maxSeen = 0;

  /* ---------- purana data patti me dikhao ---------- */
  var depth = [];
  try {
    depth = JSON.parse(post.dataset.depth) || [];
  } catch (err) {
    depth = [];   // data kharab hai toh patti khali chhod do, page toh chalna chahiye
  }

  if (strip && depth.length === 10) {
    var top = depth[0] || 0;
    if (top > 0) {
      for (var i = 0; i < 10; i++) {
        var seg = document.createElement('div');
        // pehle hisse ke mukable kitne log yahan tak aaye
        var ratio = depth[i] / top;
        seg.style.opacity = (0.12 + ratio * 0.75).toFixed(2);
        seg.title = Math.round(ratio * 100) + '% log yahan tak padhe';
        strip.appendChild(seg);
      }
    } else {
      strip.style.display = 'none';   // abhi tak koi padha hi nahi, patti bekar hai
    }
  }

  /* ---------- ye reader kitna scroll kiya ---------- */
  function currentDecile() {
    var body = post.querySelector('.post-body');
    if (!body) return 0;

    var rect = body.getBoundingClientRect();
    var total = rect.height;
    if (total <= 0) return 0;

    // screen ke neeche wala kinara post me kahan hai
    var seen = window.innerHeight - rect.top;
    var frac = seen / total;

    if (frac < 0) frac = 0;
    if (frac > 1) frac = 1;

    var d = Math.floor(frac * 10);
    return d > 9 ? 9 : d;
  }

  // scroll pe har baar calculate karna waste hai, throttle kar dete hain
  var busy = false;
  window.addEventListener('scroll', function () {
    if (busy) return;
    busy = true;
    setTimeout(function () {
      var d = currentDecile();
      if (d > maxSeen) maxSeen = d;
      busy = false;
    }, 200);
  }, { passive: true });

  /* ---------- jaate waqt bhej do ---------- */
  // sendBeacon isliye kyunki normal fetch tab band hone pe cancel ho jata hai.
  // Beacon browser ko bolta hai "ye bhej dena, main ja raha hoon".
  function report() {
    if (maxSeen <= 0) return;

    var payload = JSON.stringify({ slug: slug, decile: maxSeen });
    if (navigator.sendBeacon) {
      navigator.sendBeacon('/api/depth', new Blob([payload], { type: 'application/json' }));
    } else {
      // purane browsers ke liye. keepalive wahi kaam karta hai.
      fetch('/api/depth', { method: 'POST', body: payload, keepalive: true });
    }
    maxSeen = 0;   // do baar count na ho jaye
  }

  // 'visibilitychange' zyada bharosemand hai 'unload' se, khaaskar phone pe
  // jahan tab switch karne pe unload kabhi kabhi chalta hi nahi.
  document.addEventListener('visibilitychange', function () {
    if (document.visibilityState === 'hidden') report();
  });
  window.addEventListener('pagehide', report);
})();
