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
    // Freundliche, frontal gezeichnete Keramikwanne mit viel Schaum.
    // Die Form lehnt sich an klassische Kinderbuch-Illustrationen an:
    // breiter Rollrand, bauchiger weisser Wannenkörper und kleine Füsse.
    const x0 = 30, x1 = 370;
    const rimY = 158, bottomY = 270;
    const mid = (x0 + x1) / 2;

    // Wannenkörper: oben breit, unten sanft gerundet und etwas schmaler.
    svg.appendChild(rc.path(
      `M ${x0 + 12} ${rimY + 10} ` +
      `C ${x0 + 24} ${rimY + 70}, ${x0 + 34} ${bottomY - 4}, ${x0 + 82} ${bottomY} ` +
      `L ${x1 - 82} ${bottomY} ` +
      `C ${x1 - 34} ${bottomY - 4}, ${x1 - 24} ${rimY + 70}, ${x1 - 12} ${rimY + 10} Z`,
      { roughness: 1.25, bowing: 0.8, seed: seed + 1, stroke: INK, strokeWidth: 3.2,
        fill: "#f6f8fa", fillStyle: "solid" }));

    // Dezenter Keramikschatten, damit die weisse Wanne plastischer wirkt.
    svg.appendChild(rc.path(
      `M ${x0 + 53} ${rimY + 29} ` +
      `C ${x0 + 69} ${bottomY - 24}, ${x0 + 90} ${bottomY - 15}, ${x0 + 118} ${bottomY - 12} ` +
      `L ${x1 - 92} ${bottomY - 12}`,
      { roughness: 1, seed: seed + 2, stroke: "#dce3e8", strokeWidth: 9 }));

    // Kleine geschwungene Füsse.
    [[x0 + 78, -1], [x1 - 78, 1]].forEach(([fx, dir], k) => {
      svg.appendChild(rc.path(
        `M ${fx - 12} ${bottomY - 2} Q ${fx - 9 * dir} ${bottomY + 12} ${fx - 18 * dir} ${bottomY + 24} ` +
        `Q ${fx} ${bottomY + 29} ${fx + 18 * dir} ${bottomY + 24} ` +
        `Q ${fx + 9 * dir} ${bottomY + 12} ${fx + 12} ${bottomY - 2} Z`,
        { roughness: 1.2, seed: seed + 5 + k, stroke: INK, strokeWidth: 2.7,
          fill: "#cbd5dc", fillStyle: "solid" }));
    });

    // Hoher, runder Wasserhahn links hinter der Wanne.
    svg.appendChild(rc.path(
      `M ${x0 + 22} ${rimY - 2} L ${x0 + 22} ${rimY - 70} ` +
      `C ${x0 + 22} ${rimY - 96}, ${x0 + 64} ${rimY - 96}, ${x0 + 64} ${rimY - 70} ` +
      `L ${x0 + 64} ${rimY - 55} L ${x0 + 78} ${rimY - 55}`,
      { roughness: 1.05, seed: seed + 9, stroke: "#9aa6ae", strokeWidth: 5 }));
    svg.appendChild(rc.line(x0 + 61, rimY - 51, x0 + 61, rimY - 39,
      { roughness: 0.8, seed: seed + 10, stroke: "#70c8ee", strokeWidth: 2.2 }));

    // Üppige Schaumkrone aus unterschiedlich grossen, überlappenden Blasen.
    const foam = [
      [x0 + 15, 150, 31], [x0 + 39, 143, 39], [x0 + 68, 150, 35],
      [x0 + 96, 140, 43], [x0 + 128, 149, 38], [x0 + 158, 137, 48],
      [x0 + 195, 147, 40], [x0 + 225, 136, 49], [x0 + 261, 148, 40],
      [x0 + 291, 139, 46], [x0 + 324, 149, 35]
    ];
    foam.forEach(([fx, fy, d], k) => {
      svg.appendChild(rc.circle(fx, fy, d,
        { roughness: 1.25, seed: seed + 800 + k, stroke: "#9ad8ef", strokeWidth: 2,
          fill: "#fff", fillStyle: "solid" }));
    });

    // Breiter Rollrand liegt vor Schaum und Wannenkörper.
    svg.appendChild(rc.rectangle(x0 - 4, rimY - 1, x1 - x0 + 8, 18,
      { roughness: 1.15, bowing: 0.7, seed: seed + 20, stroke: INK, strokeWidth: 2.8,
        fill: "#f9fbfc", fillStyle: "solid" }));
    svg.appendChild(rc.line(x0 + 18, rimY + 5, x1 - 18, rimY + 5,
      { roughness: 0.8, seed: seed + 21, stroke: "#fff", strokeWidth: 3 }));

    // Kleine gelbe Badeente zwischen den Schaumblasen.
    svg.appendChild(rc.ellipse(mid - 8, 143, 54, 31,
      { roughness: 1.15, seed: seed + 30, stroke: "#d69a00", strokeWidth: 2,
        fill: "#ffd51f", fillStyle: "solid" }));
    svg.appendChild(rc.circle(mid + 8, 124, 31,
      { roughness: 1.1, seed: seed + 31, stroke: "#d69a00", strokeWidth: 2,
        fill: "#ffd51f", fillStyle: "solid" }));
    svg.appendChild(rc.path(`M ${mid + 22} 124 L ${mid + 39} 130 L ${mid + 22} 136 Z`,
      { roughness: 0.9, seed: seed + 32, stroke: "#d66a1d", strokeWidth: 1.8,
        fill: "#ff8b2d", fillStyle: "solid" }));
    svg.appendChild(rc.circle(mid + 13, 119, 3.5,
      { roughness: 0.5, seed: seed + 33, stroke: INK, fill: INK, fillStyle: "solid" }));

    // Nummerierte Seifenblasen steigen von der rechten Schaumkante auf.
    const startX = x1 - 60;
    const stepX = (W - 26 - startX) / N;
    for (let i = 0; i < N; i++) {
      const cx = startX + stepX * (i + 0.5);
      const rr = Math.min(48, stepX * 0.50) * (1 + (i % 3) * 0.07);
      const cy = 112 - 16 * Math.sin(i * 1.15) - i * 4;
      svg.appendChild(rc.circle(cx, cy, rr * 2,
        { roughness: 1.4, bowing: 1.2, seed: seed + 70 + i * 3, stroke: col(i),
          strokeWidth: 3, fill: col(i), fillStyle: "hachure", hachureGap: 9,
          fillWeight: 1, hachureAngle: 25 + i * 19 }));
      svg.appendChild(rc.arc(cx - rr * 0.30, cy - rr * 0.34, rr * 0.7, rr * 0.6,
        Math.PI * 1.05, Math.PI * 1.65, false,
        { roughness: 0.8, seed: seed + 150 + i, stroke: "#fff", strokeWidth: 2.6 }));
      txt(svg, cx, cy + 18, nums[i], "n");
    }
    return svg;
  }

  if (t === "sterne") {
    const margin = 60, step = (W - margin - 150) / N;
    const R = Math.min(step * 0.56, 54);
    // Mond als Start: dicke Sichel mit Gesicht
    const mx = 72, my = 150, MR = 54;      // Aussenkreis
    const ix = mx + 32, iy = my - 14, IR = 47;  // ausgestanzter Kreis
    // Schnittpunkte der beiden Kreise
    const dx = ix - mx, dy = iy - my, d = Math.hypot(dx, dy);
    const a = (d * d + MR * MR - IR * IR) / (2 * d);
    const h = Math.sqrt(Math.max(0, MR * MR - a * a));
    const px = mx + a * dx / d, py = my + a * dy / d;
    const p1 = [px + h * dy / d, py - h * dx / d];
    const p2 = [px - h * dy / d, py + h * dx / d];
    svg.appendChild(rc.path(
      `M ${p1[0].toFixed(1)} ${p1[1].toFixed(1)} ` +
      `A ${MR} ${MR} 0 1 0 ${p2[0].toFixed(1)} ${p2[1].toFixed(1)} ` +
      `A ${IR} ${IR} 0 0 1 ${p1[0].toFixed(1)} ${p1[1].toFixed(1)} Z`,
      { roughness: 1.2, bowing: 1, seed: seed, stroke: "#e0a01e", strokeWidth: 3.2,
        fill: "#ffd76a", fillStyle: "solid" }));
    // Krater (auf dem breiten Teil der Sichel)
    [[mx - 34, my - 34, 12], [mx - 20, my + 42, 15], [mx - 38, my + 4, 18]]
      .forEach((c, k) => svg.appendChild(rc.circle(c[0], c[1], c[2],
        { roughness: 1.2, seed: seed + 20 + k, stroke: "#e0a01e", strokeWidth: 2,
          fill: "#f3c23f", fillStyle: "solid" })));
    for (let i = 0; i < N; i++) {
      const cx = 140 + step * (i + 0.5);
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
