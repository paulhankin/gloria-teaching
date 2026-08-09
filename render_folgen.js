const SVGNS = "http://www.w3.org/2000/svg";
const PAL = ["#e8548c", "#2f9fd0", "#e0a01e", "#5eb85e", "#9b6fd0", "#ef7a3d"];
const INK = "#1f3550";

function mk(tag, attrs) {
  const e = document.createElementNS(SVGNS, tag);
  for (const k in attrs) e.setAttribute(k, attrs[k]);
  return e;
}
function txt(svg, x, y, s, cls) {
  const t = mk("text", { x: x, y: y, "text-anchor": "middle", class: cls });
  t.textContent = s; svg.appendChild(t); return t;
}
function eye(rc, svg, x, y, r) {
  svg.appendChild(rc.circle(x, y, r * 2, { roughness: 0.9, seed: 5, stroke: INK,
    strokeWidth: 1.6, fill: INK, fillStyle: "solid" }));
}

function starPts(cx, cy, R, r) {
  const p = [];
  for (let i = 0; i < 10; i++) {
    const a = -Math.PI / 2 + i * Math.PI / 5;
    const rad = i % 2 ? r : R;
    p.push([cx + rad * Math.cos(a), cy + rad * Math.sin(a)]);
  }
  return p;
}

function buildFolge(spec, seed) {
  const nums = spec.zahlen;
  const N = nums.length;
  const t = spec.typ;
  const W = (t === "schornstein" || t === "badewanne") ? 1450 : (t === "waescheleine" ? 1250 : 900), H = 300;
  const svg = mk("svg", { xmlns: SVGNS, viewBox: `0 0 ${W} ${H}`,
                          preserveAspectRatio: "xMidYMid meet" });
  const st = mk("style", {});
  st.textContent = `
    .n{font:700 52px 'Caveat',cursive;fill:#1f3550;
       paint-order:stroke;stroke:#fff;stroke-width:7px;stroke-linejoin:round}
    .nw{font:700 40px 'Caveat',cursive;fill:#1f3550}
  `;
  svg.appendChild(st);
  const rc = rough.svg(svg);
  const col = (i) => PAL[i % PAL.length];

  if (t === "zug") {
    const margin = 30, avail = W - 2 * margin - 150;
    const step = avail / N;
    const wW = Math.min(step * 0.86, 86), wH = 96, baseY = 200;
    // Gleis
    svg.appendChild(rc.line(12, baseY + 34, W - 12, baseY + 34,
      { roughness: 2.2, seed: seed, stroke: "#8a94a6", strokeWidth: 3 }));
    // Lok
    const lx = margin;
    svg.appendChild(rc.rectangle(lx, baseY - 90, 122, 90,
      { roughness: 1.5, seed: seed + 1, stroke: INK, strokeWidth: 3,
        fill: "#8a94a6", fillStyle: "hachure", hachureGap: 9, fillWeight: 1 }));
    svg.appendChild(rc.rectangle(lx + 16, baseY - 132, 34, 44,
      { roughness: 1.5, seed: seed + 2, stroke: INK, strokeWidth: 3 }));
    svg.appendChild(rc.rectangle(lx + 62, baseY - 74, 46, 40,
      { roughness: 1.4, seed: seed + 3, stroke: INK, strokeWidth: 2.4 }));
    [lx + 30, lx + 92].forEach((wx, k) =>
      svg.appendChild(rc.circle(wx, baseY + 14, 44,
        { roughness: 1.4, seed: seed + 10 + k, stroke: INK, strokeWidth: 3 })));
    for (let i = 0; i < N; i++) {
      const x = margin + 150 + i * step;
      svg.appendChild(rc.rectangle(x, baseY - wH, wW, wH,
        { roughness: 1.5, bowing: 1.4, seed: seed + 20 + i, stroke: col(i),
          strokeWidth: 3, fill: col(i), fillStyle: "hachure", hachureGap: 10,
          fillWeight: 1, hachureAngle: 40 + i * 13 }));
      [x + wW * 0.25, x + wW * 0.75].forEach((wx, k) =>
        svg.appendChild(rc.circle(wx, baseY + 10, 30,
          { roughness: 1.3, seed: seed + 40 + i * 2 + k, stroke: INK, strokeWidth: 2.4 })));
      txt(svg, x + wW / 2, baseY - wH / 2 + 18, nums[i], "n");
    }
    return svg;
  }

  if (t === "waescheleine") {
    const margin = 140, step = (W - margin - 30) / N;
    const sw = Math.min(step * 0.44, 56), sh = 176;
    const y0 = 60, sag = 22;
    const lineY = (x) => y0 + sag * Math.sin(Math.PI * (x - 10) / (W - 20));
    let d = `M 70 ${y0}`;
    for (let x = 90; x <= W - 10; x += 20) d += ` L ${x} ${lineY(x)}`;
    svg.appendChild(rc.path(d, { roughness: 1.4, seed: seed, stroke: "#8a94a6", strokeWidth: 3.5 }));
    // Startpfahl mit Vogel
    svg.appendChild(rc.line(70, y0 - 6, 70, y0 + 216,
      { roughness: 1.3, seed: seed + 900, stroke: "#8a5a35", strokeWidth: 7 }));
    svg.appendChild(rc.line(34, y0 + 30, 106, y0 + 30,
      { roughness: 1.3, seed: seed + 901, stroke: "#8a5a35", strokeWidth: 5 }));
    // Vogel auf dem Pfahl
    svg.appendChild(rc.ellipse(70, y0 - 34, 54, 40,
      { roughness: 1.3, seed: seed + 902, stroke: INK, strokeWidth: 2.6,
        fill: "#e8548c", fillStyle: "solid" }));
    svg.appendChild(rc.circle(52, y0 - 52, 30,
      { roughness: 1.2, seed: seed + 903, stroke: INK, strokeWidth: 2.4,
        fill: "#e8548c", fillStyle: "solid" }));
    svg.appendChild(rc.polygon([[38, y0 - 54], [18, y0 - 48], [38, y0 - 42]],
      { roughness: 1.0, seed: seed + 904, stroke: "#e0a01e", strokeWidth: 2,
        fill: "#e0a01e", fillStyle: "solid" }));
    eye(rc, svg, 48, y0 - 58, 3);
    for (let i = 0; i < N; i++) {
      const cx = margin + step * (i + 0.5), ly = lineY(cx);
      // Klammer
      svg.appendChild(rc.line(cx, ly - 6, cx, ly + 16,
        { roughness: 1.0, seed: seed + i, stroke: INK, strokeWidth: 2.6 }));
      // Socke: Schaft nach unten, Fuss nach rechts
      const top = ly + 14, w = sw, h = sh;
      const l = cx - w / 2, r2 = cx + w / 2;
      const ankle = top + h * 0.54;          // Beginn Ferse
      const footY = top + h;                 // Sohle
      const toe = r2 + w * 0.80;             // Zehenspitze
      const sockPath =
        `M ${l} ${top} L ${r2} ${top} L ${r2} ${ankle} ` +
        `Q ${toe} ${ankle + 4} ${toe} ${(ankle + footY) / 2} ` +
        `Q ${toe} ${footY} ${toe - w * 0.5} ${footY} ` +
        `L ${l} ${footY} Z`;
      const o1 = { roughness: 1.5, bowing: 1.3, seed: seed + 60 + i,
                   stroke: col(i), strokeWidth: 3 };
      svg.appendChild(rc.path(sockPath, Object.assign({}, o1,
        { fill: col(i), fillStyle: "hachure", hachureGap: 9, fillWeight: 1,
          hachureAngle: 35 + i * 17 })));
      // Bündchen
      svg.appendChild(rc.line(l, top + 16, r2, top + 16,
        { roughness: 1.2, seed: seed + 90 + i, stroke: col(i), strokeWidth: 2.4 }));
      // Zahl in den Fuss der Socke (dort ist mehr Platz)
      txt(svg, (l + toe) / 2 - 2, (ankle + footY) / 2 + 16, nums[i], "n");
    }
    return svg;
  }

  if (t === "schornstein") {
    // Haus mit Schornstein links, Rauchwolken steigen nach rechts oben
    const hx = 26, hy = 206, hw = 124, hh = 84;
    // Dach
    svg.appendChild(rc.polygon([[hx - 12, hy], [hx + hw / 2, hy - 48], [hx + hw + 12, hy]],
      { roughness: 1.5, seed: seed + 1, stroke: "#b5553d", strokeWidth: 3,
        fill: "#b5553d", fillStyle: "hachure", hachureGap: 8, fillWeight: 1 }));
    // Wand
    svg.appendChild(rc.rectangle(hx, hy, hw, hh,
      { roughness: 1.5, seed: seed + 2, stroke: INK, strokeWidth: 3,
        fill: "#e8d8b0", fillStyle: "hachure", hachureGap: 10, fillWeight: 1,
        hachureAngle: 80 }));
    // Fenster
    svg.appendChild(rc.rectangle(hx + 30, hy + 24, 38, 36,
      { roughness: 1.3, seed: seed + 3, stroke: INK, strokeWidth: 2.4 }));
    // Schornstein
    const sx = hx + hw * 0.66, sy = hy - 44;
    svg.appendChild(rc.rectangle(sx, sy - 40, 34, 56,
      { roughness: 1.4, seed: seed + 4, stroke: INK, strokeWidth: 3,
        fill: "#9a8f86", fillStyle: "hachure", hachureGap: 8, fillWeight: 1 }));
    // Wolken
    const startX = sx + 50, startY = sy - 34;
    const stepX = (W - 26 - startX) / N;
    for (let i = 0; i < N; i++) {
      const cx = startX + stepX * (i + 0.5);
      const rr = Math.min(44, stepX * 0.46) * (1 + i * 0.035);
      const cy = startY - 6 * Math.sin(i * 0.9) - i * 5;
      const o = { roughness: 1.7, bowing: 1.5, seed: seed + 70 + i * 5,
                  stroke: col(i), strokeWidth: 3,
                  fill: col(i), fillStyle: "hachure", hachureGap: 9,
                  fillWeight: 1, hachureAngle: 30 + i * 23 };
      // Wolke aus 4 ueberlappenden Kreisen (Umriss)
      const oo = Object.assign({}, o, { fillStyle: "hachure", hachureGap: 10 });
      svg.appendChild(rc.circle(cx - rr * 0.72, cy + rr * 0.26, rr * 1.10, oo));
      svg.appendChild(rc.circle(cx + rr * 0.74, cy + rr * 0.30, rr * 1.00,
        Object.assign({}, oo, { seed: seed + 71 + i * 5 })));
      svg.appendChild(rc.circle(cx - rr * 0.10, cy - rr * 0.34, rr * 1.30,
        Object.assign({}, oo, { seed: seed + 72 + i * 5 })));
      svg.appendChild(rc.circle(cx + rr * 0.30, cy + rr * 0.34, rr * 1.20,
        Object.assign({}, oo, { seed: seed + 73 + i * 5 })));
      txt(svg, cx, cy + 19, nums[i], "n");
    }
    return svg;
  }

  if (t === "badewanne") {
    // Freistehende Wanne (Klauenfuss) links, Seifenblasen steigen nach rechts
    const x0 = 34, x1 = 320;          // linke / rechte Aussenkante
    const yL = 112, yR = 158;         // Randhoehe hinten (links) / vorne (rechts)
    const yB = 262;                   // Wannenboden
    // Wasserspiegel (leicht schraeg wie der Rand)
    const wl = (x) => yL + 34 + (yR - yL) * (x - x0) / (x1 - x0);

    // Wannenkoerper (Silhouette)
    const body =
      `M ${x0} ${yL} ` +
      `C ${x0 - 10} ${yL + 66}, ${x0 + 4} ${yB - 34}, ${x0 + 52} ${yB} ` +
      `L ${x1 - 46} ${yB} ` +
      `C ${x1 + 2} ${yB - 30}, ${x1 + 8} ${yR + 54}, ${x1} ${yR} ` +
      `Q ${(x0 + x1) / 2} ${(yL + yR) / 2 + 16} ${x0} ${yL} Z`;
    svg.appendChild(rc.path(body,
      { roughness: 1.4, bowing: 1, seed: seed + 1, stroke: INK, strokeWidth: 3.2,
        fill: "#f2fbff", fillStyle: "solid" }));
    // Wasser (innen, unterhalb des Wasserspiegels)
    svg.appendChild(rc.path(
      `M ${x0 + 6} ${wl(x0) - 2} ` +
      `C ${x0 + 2} ${yB - 56}, ${x0 + 18} ${yB - 12}, ${x0 + 58} ${yB - 10} ` +
      `L ${x1 - 52} ${yB - 10} ` +
      `C ${x1 - 10} ${yB - 14}, ${x1 - 4} ${yR + 46}, ${x1 - 7} ${wl(x1) - 2} Z`,
      { roughness: 1.5, seed: seed + 3, stroke: "#7fd1f5", strokeWidth: 2,
        fill: "#b6e3f5", fillStyle: "hachure", hachureGap: 9, fillWeight: 1,
        hachureAngle: 12 }));
    // Rollrand (doppelte Linie entlang der Oeffnung)
    svg.appendChild(rc.path(
      `M ${x0 - 2} ${yL + 11} Q ${(x0 + x1) / 2} ${(yL + yR) / 2 + 27} ${x1 + 2} ${yR + 11}`,
      { roughness: 1.2, seed: seed + 4, stroke: INK, strokeWidth: 2.6 }));
    // Klauenfuesse
    [x0 + 62, x1 - 58].forEach((fx, k) => {
      svg.appendChild(rc.path(
        `M ${fx - 15} ${yB - 4} C ${fx - 13} ${yB + 16}, ${fx - 22} ${yB + 20}, ${fx - 20} ${yB + 30} ` +
        `L ${fx + 20} ${yB + 30} C ${fx + 22} ${yB + 20}, ${fx + 13} ${yB + 16}, ${fx + 15} ${yB - 4} Z`,
        { roughness: 1.3, seed: seed + 5 + k, stroke: INK, strokeWidth: 2.8,
          fill: "#dfe8f2", fillStyle: "solid" }));
    });
    // Wasserhahn hinten links
    svg.appendChild(rc.path(
      `M ${x0 + 6} ${yL - 6} L ${x0 + 6} ${yL - 62} Q ${x0 + 8} ${yL - 78} ${x0 + 30} ${yL - 78} ` +
      `L ${x0 + 52} ${yL - 78} L ${x0 + 52} ${yL - 60}`,
      { roughness: 1.2, seed: seed + 9, stroke: "#8a94a6", strokeWidth: 4.2 }));
    svg.appendChild(rc.circle(x0 + 6, yL - 72, 16,
      { roughness: 1.1, seed: seed + 10, stroke: "#8a94a6", strokeWidth: 3 }));
    // Schaumkrone auf dem Wasser
    for (let k = 0; k < 8; k++) {
      const sx = x0 + 14 + k * (x1 - x0 - 30) / 7;
      svg.appendChild(rc.circle(sx, wl(sx) - 8 - (k % 2) * 9, 40 - (k % 3) * 6,
        { roughness: 1.6, seed: seed + 800 + k, stroke: "#7fd1f5", strokeWidth: 2.4,
          fill: "#eaf7fd", fillStyle: "solid" }));
    }
    // Blasen
    const startX = x0 + (x1 - x0) * 0.62;
    const stepX = (W - 26 - startX) / N;
    for (let i = 0; i < N; i++) {
      const cx = startX + stepX * (i + 0.5);
      const rr = Math.min(48, stepX * 0.50) * (1 + (i % 3) * 0.07);
      const cy = 112 - 16 * Math.sin(i * 1.15) - i * 4;
      svg.appendChild(rc.circle(cx, cy, rr * 2,
        { roughness: 1.4, bowing: 1.2, seed: seed + 70 + i * 3, stroke: col(i),
          strokeWidth: 3, fill: col(i), fillStyle: "hachure", hachureGap: 9,
          fillWeight: 1, hachureAngle: 25 + i * 19 }));
      // Glanzpunkt
      svg.appendChild(rc.arc(cx - rr * 0.30, cy - rr * 0.34, rr * 0.7, rr * 0.6,
        Math.PI * 1.05, Math.PI * 1.65, false,
        { roughness: 0.8, seed: seed + 150 + i, stroke: "#fff", strokeWidth: 2.6 }));
      txt(svg, cx, cy + 18, nums[i], "n");
    }
    return svg;
  }

  if (t === "sterne") {
    const margin = 60, step = (W - 2 * margin) / N;
    const R = Math.min(step * 0.56, 54);
    // Mond als Start
    svg.appendChild(rc.path(
      "M 50 108 A 42 42 0 1 0 50 192 A 32 32 0 1 1 50 108 Z",
      { roughness: 1.3, seed: seed, stroke: "#e0a01e", strokeWidth: 3,
        fill: "#e0a01e", fillStyle: "hachure", hachureGap: 7, fillWeight: 1 }));
    for (let i = 0; i < N; i++) {
      const cx = margin + step * (i + 0.5);
      const cy = 150 + (i % 2 ? 34 : -30);
      svg.appendChild(rc.polygon(starPts(cx, cy, R, R * 0.44),
        { roughness: 1.4, bowing: 1.2, seed: seed + 30 + i, stroke: col(i),
          strokeWidth: 3, fill: col(i), fillStyle: "hachure", hachureGap: 8,
          fillWeight: 1, hachureAngle: 30 + i * 19 }));
      txt(svg, cx, cy + 18, nums[i], "n");
    }
    return svg;
  }

  // Kreis-Ketten: raupe / schlange / drache
  const margin = 54, step = (W - 2 * margin) / (N + 1);
  const r = Math.min(40, step * 0.54);
  const cy = (i) => 158 + 22 * Math.sin(i * 0.85);
  const cx = (i) => margin + step * (i + 0.5);
  const taper = (i) => (t === "schlange" ? r * (1 - 0.22 * i / N) : r);

  // Schwanz
  if (t === "schlange" || t === "drache") {
    const xe = cx(N) + r * 1.3, ye = cy(N);
    svg.appendChild(rc.line(cx(N), ye, xe + 28, ye - 26,
      { roughness: 1.6, seed: seed + 3, stroke: INK, strokeWidth: 3 }));
  }
  // Koerper
  for (let i = 0; i < N; i++) {
    const x = cx(i + 1), y = cy(i + 1), rr = taper(i);
    svg.appendChild(rc.circle(x, y, rr * 2,
      { roughness: 1.6, bowing: 1.3, seed: seed + 70 + i * 5, stroke: col(i),
        strokeWidth: 3.2, fill: col(i), fillStyle: "hachure", hachureGap: 9,
        fillWeight: 1, hachureAngle: 35 + i * 21 }));
    if (t === "raupe") {
      svg.appendChild(rc.line(x - 8, y + rr - 3, x - 14, y + rr + 20,
        { roughness: 1.4, seed: seed + 200 + i, stroke: INK, strokeWidth: 2.2 }));
      svg.appendChild(rc.line(x + 8, y + rr - 3, x + 14, y + rr + 20,
        { roughness: 1.4, seed: seed + 300 + i, stroke: INK, strokeWidth: 2.2 }));
    }
    if (t === "drache") {
      svg.appendChild(rc.polygon(
        [[x - 15, y - rr + 4], [x, y - rr - 26], [x + 15, y - rr + 4]],
        { roughness: 1.4, seed: seed + 400 + i, stroke: "#5eb85e", strokeWidth: 2.6,
          fill: "#5eb85e", fillStyle: "hachure", hachureGap: 7, fillWeight: 1 }));
    }
    txt(svg, x, y + 18, nums[i], "n");
  }
  // Kopf
  const hx = cx(0), hy = cy(0), hr = r * 1.22;
  svg.appendChild(rc.circle(hx, hy, hr * 2,
    { roughness: 1.5, bowing: 1.2, seed: seed + 2, stroke: INK, strokeWidth: 3.4,
      fill: spec.kopffarbe || "#ffd76a", fillStyle: "solid" }));
  eye(rc, svg, hx - hr * 0.34, hy - hr * 0.22, 5);
  eye(rc, svg, hx + hr * 0.30, hy - hr * 0.26, 5);
  svg.appendChild(rc.arc(hx, hy + hr * 0.10, hr * 0.9, hr * 0.7,
    0.25 * Math.PI, 0.75 * Math.PI, false,
    { roughness: 1.2, seed: seed + 4, stroke: INK, strokeWidth: 2.4 }));
  if (t === "raupe") {
    [[-0.5, -1], [0.45, -1]].forEach((d, k) => {
      const ax = hx + hr * d[0], ay = hy - hr * 0.86;
      svg.appendChild(rc.line(ax, ay, ax + d[0] * 20, ay - 34,
        { roughness: 1.3, seed: seed + 500 + k, stroke: INK, strokeWidth: 2.4 }));
      svg.appendChild(rc.circle(ax + d[0] * 22, ay - 40, 15,
        { roughness: 1.2, seed: seed + 510 + k, stroke: "#e8548c", strokeWidth: 2.4,
          fill: "#e8548c", fillStyle: "solid" }));
    });
  }
  if (t === "drache") {
    [[-0.55, -1], [0.4, -1]].forEach((d, k) =>
      svg.appendChild(rc.polygon(
        [[hx + hr * d[0] - 9, hy - hr * 0.72],
         [hx + hr * d[0] + d[0] * 12, hy - hr * 1.5],
         [hx + hr * d[0] + 9, hy - hr * 0.72]],
        { roughness: 1.3, seed: seed + 600 + k, stroke: INK, strokeWidth: 2.4,
          fill: "#5eb85e", fillStyle: "solid" })));
  }
  if (t === "schlange") {
    svg.appendChild(rc.line(hx - hr, hy + 6, hx - hr - 30, hy + 6,
      { roughness: 1.2, seed: seed + 700, stroke: "#d10000", strokeWidth: 2.4 }));
    svg.appendChild(rc.line(hx - hr - 30, hy + 6, hx - hr - 44, hy - 6,
      { roughness: 1.2, seed: seed + 701, stroke: "#d10000", strokeWidth: 2.2 }));
    svg.appendChild(rc.line(hx - hr - 30, hy + 6, hx - hr - 44, hy + 18,
      { roughness: 1.2, seed: seed + 702, stroke: "#d10000", strokeWidth: 2.2 }));
  }
  return svg;
}

for (const el of document.querySelectorAll('[data-folge]')) {
  const key = el.getAttribute('data-folge');
  const spec = FOLGEN[key];
  if (!spec) continue;
  const svg = buildFolge(spec, (key.charCodeAt(1) || 7) * 91 + 13);
  el.appendChild(svg);
  // viewBox eng an den Inhalt anpassen (kein Leerraum)
  try {
    const b = svg.getBBox(), p = 8;
    svg.setAttribute('viewBox',
      `${b.x - p} ${b.y - p} ${b.width + 2 * p} ${b.height + 2 * p}`);
  } catch (e) {}
}
