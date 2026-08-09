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
  const W = 900, H = 300;
  const svg = mk("svg", { xmlns: SVGNS, viewBox: `0 0 ${W} ${H}`,
                          preserveAspectRatio: "xMidYMid meet" });
  const st = mk("style", {});
  st.textContent = `
    .n{font:700 34px 'Caveat',cursive;fill:#1f3550;
       paint-order:stroke;stroke:#fff;stroke-width:5px;stroke-linejoin:round}
    .nw{font:700 34px 'Caveat',cursive;fill:#1f3550}
  `;
  svg.appendChild(st);
  const rc = rough.svg(svg);
  const t = spec.typ;
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
      txt(svg, x + wW / 2, baseY - wH / 2 + 12, nums[i], "n");
    }
    return svg;
  }

  if (t === "waescheleine") {
    const margin = 46, step = (W - 2 * margin) / N;
    const sw = Math.min(step * 0.74, 78), sh = 150;
    const y0 = 66, sag = 26;
    const lineY = (x) => y0 + sag * Math.sin(Math.PI * (x - 10) / (W - 20));
    let d = `M 10 ${y0}`;
    for (let x = 30; x <= W - 10; x += 20) d += ` L ${x} ${lineY(x)}`;
    svg.appendChild(rc.path(d, { roughness: 1.4, seed: seed, stroke: "#8a94a6", strokeWidth: 3.5 }));
    for (let i = 0; i < N; i++) {
      const cx = margin + step * (i + 0.5), ly = lineY(cx);
      svg.appendChild(rc.line(cx, ly, cx, ly + 14,
        { roughness: 1.2, seed: seed + i, stroke: INK, strokeWidth: 2.2 }));
      svg.appendChild(rc.rectangle(cx - sw / 2, ly + 14, sw, sh,
        { roughness: 1.6, bowing: 1.6, seed: seed + 60 + i, stroke: col(i),
          strokeWidth: 3, fill: col(i), fillStyle: "hachure", hachureGap: 10,
          fillWeight: 1, hachureAngle: 35 + i * 17 }));
      txt(svg, cx, ly + 14 + sh / 2 + 12, nums[i], "n");
    }
    return svg;
  }

  if (t === "sterne") {
    const margin = 60, step = (W - 2 * margin) / N;
    const R = Math.min(step * 0.56, 54);
    // Mond als Start
    svg.appendChild(rc.path(
      "M 44 118 A 34 34 0 1 0 44 182 A 26 26 0 1 1 44 118 Z",
      { roughness: 1.3, seed: seed, stroke: "#e0a01e", strokeWidth: 3,
        fill: "#e0a01e", fillStyle: "hachure", hachureGap: 7, fillWeight: 1 }));
    for (let i = 0; i < N; i++) {
      const cx = margin + step * (i + 0.5);
      const cy = 150 + (i % 2 ? 34 : -30);
      svg.appendChild(rc.polygon(starPts(cx, cy, R, R * 0.44),
        { roughness: 1.4, bowing: 1.2, seed: seed + 30 + i, stroke: col(i),
          strokeWidth: 3, fill: col(i), fillStyle: "hachure", hachureGap: 8,
          fillWeight: 1, hachureAngle: 30 + i * 19 }));
      txt(svg, cx, cy + 12, nums[i], "n");
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
    txt(svg, x, y + 12, nums[i], "n");
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
