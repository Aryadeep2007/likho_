/* Blog ka naksha. Force-directed graph, poora canvas pe haath se banaya hua.
   D3 import kar sakte the par wo 250kb ka hai - itne se kaam ke liye zyada hai.

   Physics simple hai:
     1. har gola har dusre gole ko dhakelta hai (repulsion)
     2. jo gole link se jude hain wo ek dusre ko kheenchte hain (spring)
     3. sab beech me aane ki koshish karte hain (gravity)
   Ye teeno har frame chalte hain aur apne aap ek acha layout ban jata hai. */

(function () {
  'use strict';

  var canvas = document.getElementById('graph');
  var ctx = canvas.getContext('2d');
  var emptyMsg = document.getElementById('graphEmpty');

  var nodes = [], links = [];
  var W = 0, H = 0;
  var drag = null, hover = null;
  var alpha = 1;           // simulation ki "energy" - dheere dheere thandi hoti hai

  fetch('/api/graph')
    .then(function (r) { return r.json(); })
    .then(function (d) {
      nodes = d.nodes || [];
      links = d.links || [];

      if (!nodes.length) { emptyMsg.hidden = false; return; }

      // Pehle canvas naap lo. Ye upar hona zaroori hai - warna W aur H
      // abhi 0 hote hain aur saare gole (0,0) pe yaani screen ke bahar
      // top-left corner me chale jate hain.
      resize();

      // sabko ek circle me spread kar do, physics baaki sambhal legi
      nodes.forEach(function (n, i) {
        var ang = (i / nodes.length) * Math.PI * 2;
        var spread = Math.min(W, H) * 0.28;
        n.x = W / 2 + Math.cos(ang) * spread + (Math.random() - 0.5) * 30;
        n.y = H / 2 + Math.sin(ang) * spread + (Math.random() - 0.5) * 30;
        n.vx = 0; n.vy = 0;
        n.r = Math.min(26, 8 + Math.sqrt(n.words || 1) * 0.7);   // bada post = bada gola
      });

      // links me slug hai, unhe actual node object se badal do
      var byId = {};
      nodes.forEach(function (n) { byId[n.id] = n; });
      links = links.filter(function (l) {
        l.s = byId[l.source]; l.t = byId[l.target];
        return l.s && l.t;
      });

      requestAnimationFrame(tick);
    });

  function resize() {
    var rect = canvas.parentElement.getBoundingClientRect();
    W = rect.width; H = rect.height;

    // retina screens pe blur na ho isliye DPR se multiply karte hain
    var dpr = window.devicePixelRatio || 1;
    canvas.width = W * dpr;
    canvas.height = H * dpr;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  }
  window.addEventListener('resize', function () { resize(); alpha = 0.3; });

  function physics() {
    if (alpha < 0.005) return;   // sab settle ho gaya, ab CPU jalane ka matlab nahi

    // 1. dhakka - har jodi ke beech. O(n²) hai, par 100 posts pe bhi
    //    ye 5000 calculations hain, browser ko farak nahi padta.
    for (var i = 0; i < nodes.length; i++) {
      for (var j = i + 1; j < nodes.length; j++) {
        var a = nodes[i], b = nodes[j];
        var dx = b.x - a.x, dy = b.y - a.y;
        var d2 = dx * dx + dy * dy || 0.01;
        var d = Math.sqrt(d2);

        // bahut paas hai toh zor se dhakelo
        var force = 2200 / d2;
        if (d < a.r + b.r + 14) force += 1.2;   // overlap bilkul mat hone do

        var fx = (dx / d) * force, fy = (dy / d) * force;
        a.vx -= fx; a.vy -= fy;
        b.vx += fx; b.vy += fy;
      }
    }

    // 2. spring - jude hue nodes ko paas lao
    links.forEach(function (l) {
      var dx = l.t.x - l.s.x, dy = l.t.y - l.s.y;
      var d = Math.sqrt(dx * dx + dy * dy) || 0.01;
      var target = 110;
      var f = (d - target) * 0.012;
      var fx = (dx / d) * f, fy = (dy / d) * f;
      l.s.vx += fx; l.s.vy += fy;
      l.t.vx -= fx; l.t.vy -= fy;
    });

    // 3. beech me kheencho + friction
    nodes.forEach(function (n) {
      if (n === drag) return;      // jo pakda hua hai use physics se chhoot
      n.vx += (W / 2 - n.x) * 0.0016;
      n.vy += (H / 2 - n.y) * 0.0016;
      n.vx *= 0.86;                // friction, warna hamesha hilte rahenge
      n.vy *= 0.86;
      n.x += n.vx * alpha;
      n.y += n.vy * alpha;
    });

    alpha *= 0.995;   // dheere dheere thanda
  }

  function draw() {
    var css = getComputedStyle(document.documentElement);
    var accent = css.getPropertyValue('--accent').trim() || '#c2410c';
    var line = css.getPropertyValue('--line').trim() || '#e6e1d8';
    var ink = css.getPropertyValue('--ink').trim() || '#1c1a17';
    var soft = css.getPropertyValue('--ink-soft').trim() || '#6b6459';

    ctx.clearRect(0, 0, W, H);

    // pehle lines, taki gole unke upar aayein
    ctx.strokeStyle = line;
    ctx.lineWidth = 1.5;
    links.forEach(function (l) {
      var lit = hover && (l.s === hover || l.t === hover);
      ctx.strokeStyle = lit ? accent : line;
      ctx.globalAlpha = lit ? 0.9 : 0.55;
      ctx.beginPath();
      ctx.moveTo(l.s.x, l.s.y);
      ctx.lineTo(l.t.x, l.t.y);
      ctx.stroke();
    });
    ctx.globalAlpha = 1;

    nodes.forEach(function (n) {
      var lit = n === hover;

      ctx.beginPath();
      ctx.arc(n.x, n.y, n.r, 0, Math.PI * 2);
      ctx.fillStyle = accent;
      // draft posts halke dikhte hain
      ctx.globalAlpha = n.draft ? 0.35 : (lit ? 1 : 0.82);
      ctx.fill();

      if (lit) {
        ctx.globalAlpha = 1;
        ctx.strokeStyle = accent;
        ctx.lineWidth = 2.5;
        ctx.beginPath();
        ctx.arc(n.x, n.y, n.r + 5, 0, Math.PI * 2);
        ctx.stroke();
      }

      // Naam dikhao. Bahut saare posts hain toh chhote golo ke naam chhupa dete hain
      // warna sab ek dusre ke upar chadh jate hain - unpe hover karo toh dikh jayega.
      // Par kam posts hain toh sabke naam dikhne chahiye, warna adhoora lagta hai.
      ctx.globalAlpha = 1;
      if (nodes.length <= 15 || n.r > 13 || lit) {
        ctx.fillStyle = lit ? ink : soft;
        ctx.font = (lit ? '600 ' : '') + '12px ui-sans-serif, system-ui, sans-serif';
        ctx.textAlign = 'center';
        var label = n.title.length > 22 ? n.title.slice(0, 21) + '…' : n.title;
        ctx.fillText(label, n.x, n.y + n.r + 14);
      }
    });
  }

  function tick() {
    physics();
    draw();
    requestAnimationFrame(tick);
  }

  /* ---------- mouse ---------- */
  function at(e) {
    var rect = canvas.getBoundingClientRect();
    var x = e.clientX - rect.left, y = e.clientY - rect.top;
    for (var i = 0; i < nodes.length; i++) {
      var n = nodes[i];
      var dx = n.x - x, dy = n.y - y;
      if (dx * dx + dy * dy < (n.r + 6) * (n.r + 6)) return n;
    }
    return null;
  }

  canvas.addEventListener('mousedown', function (e) {
    drag = at(e);
    if (drag) { drag.moved = false; alpha = Math.max(alpha, 0.5); }
  });

  canvas.addEventListener('mousemove', function (e) {
    var rect = canvas.getBoundingClientRect();
    if (drag) {
      drag.x = e.clientX - rect.left;
      drag.y = e.clientY - rect.top;
      drag.vx = 0; drag.vy = 0;
      drag.moved = true;
      alpha = Math.max(alpha, 0.4);   // hilao toh baaki bhi adjust ho jayein
    } else {
      hover = at(e);
      canvas.style.cursor = hover ? 'pointer' : 'grab';
    }
  });

  window.addEventListener('mouseup', function () {
    // pakad ke chhoda par hilaya nahi = ye click tha, post kholo
    if (drag && !drag.moved) location.href = '/p/' + drag.id;
    drag = null;
  });

  // phone/tablet ke liye - judge log phone pe hi kholenge
  canvas.addEventListener('touchstart', function (e) {
    var t = e.touches[0];
    drag = at({ clientX: t.clientX, clientY: t.clientY });
    if (drag) { drag.moved = false; alpha = Math.max(alpha, 0.5); }
  }, { passive: true });

  canvas.addEventListener('touchmove', function (e) {
    if (!drag) return;
    e.preventDefault();
    var t = e.touches[0];
    var rect = canvas.getBoundingClientRect();
    drag.x = t.clientX - rect.left;
    drag.y = t.clientY - rect.top;
    drag.moved = true;
    alpha = Math.max(alpha, 0.4);
  }, { passive: false });

  canvas.addEventListener('touchend', function () {
    if (drag && !drag.moved) location.href = '/p/' + drag.id;
    drag = null;
  });
})();
