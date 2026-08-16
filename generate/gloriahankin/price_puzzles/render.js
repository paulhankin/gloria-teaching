const SVGNS = "http://www.w3.org/2000/svg";
const INK = "#1f3550";
const PINK = "#e8548c";
const BLUE = "#2f9fd0";
const GOLD = "#e0a01e";
const GREEN = "#5eb85e";
const BASE = 238;              // ground line
const TOP = 40;                // top edge of the objects

function el(tag, attrs = {}) {
  const node = document.createElementNS(SVGNS, tag);
  for (const [k, v] of Object.entries(attrs)) node.setAttribute(k, v);
  return node;
}
function text(svg, x, y, value, cls) {
  const t = el("text", {x, y, class: cls, "text-anchor": "middle"});
  t.textContent = value;
  svg.appendChild(t);
}
// Outline + solid pastel fill (stays readable, also in black-and-white print)
function opts(seed, stroke, fill, sw) {
  const o = {seed, roughness: 1.1, bowing: 1.0, stroke: stroke || INK, strokeWidth: sw || 3};
  if (fill) Object.assign(o, {fill, fillStyle: "solid"});
  return o;
}
function line(svg, rc, x1, y1, x2, y2, o) { svg.appendChild(rc.line(x1, y1, x2, y2, o)); }

function labelUnder(svg, x, name) {
  text(svg, x, 296, name, "label");
}
function ground(svg, rc, seed) {
  svg.appendChild(rc.line(50, BASE, 850, BASE,
    {seed, roughness: 1.6, stroke: "#c9d3df", strokeWidth: 3}));
}

/* ---------- 1 plant / pot ---------- */
function drawPot(svg, rc, x, seed, tall) {
  const top = tall ? 128 : 128;
  svg.appendChild(rc.path(
    `M ${x-62} ${top} L ${x+62} ${top} L ${x+44} ${BASE} L ${x-44} ${BASE} Z`,
    opts(seed, "#a9542c", "#f0a878")));
  svg.appendChild(rc.rectangle(x - 70, top - 20, 140, 22, opts(seed + 1, "#a9542c", "#f7c1a1")));
}
// The plant grows IN its pot: unlike the other puzzles both objects are drawn
// together as one picture, with leader lines pointing at the two parts.
function drawPottedPlant(svg, rc, x, seed) {
  const potTop = 128;
  // plant: stem rooted inside the pot, leaves and a shoot above the rim
  line(svg, rc, x, potTop + 10, x, 48, opts(seed, "#3f8f3f"));
  const leaves = [[-2, 104, -60, 82], [2, 78, 60, 56], [-2, 52, -54, 32]];
  leaves.forEach((L, i) => {
    const [sx, sy, ex, ey] = [x + L[0], L[1], x + L[2], L[3]];
    svg.appendChild(rc.path(
      `M ${sx} ${sy} Q ${(sx + ex) / 2} ${ey - 26} ${ex} ${ey} Q ${(sx + ex) / 2} ${ey + 18} ${sx} ${sy} Z`,
      opts(seed + i + 1, "#3f8f3f", "#a8dc87")));
  });
  svg.appendChild(rc.path(`M ${x} 50 Q ${x - 22} 30 ${x} 20 Q ${x + 22} 30 ${x} 50 Z`,
    opts(seed + 9, "#3f8f3f", "#c6ecab")));
  // pot around the stem
  drawPot(svg, rc, x, seed + 20);
  // soil peeking out of the pot
  svg.appendChild(rc.path(
    `M ${x - 62} ${potTop} Q ${x} ${potTop + 16} ${x + 62} ${potTop} L ${x + 62} ${potTop + 12} ` +
    `Q ${x} ${potTop + 26} ${x - 62} ${potTop + 12} Z`,
    opts(seed + 40, "#7a5230", "#a07a4a")));
}
// Leader line from a label to the part of the picture it names.
function leader(svg, rc, x1, y1, x2, y2, seed) {
  line(svg, rc, x1, y1, x2, y2, opts(seed, "#8a94a6", null, 2));
  svg.appendChild(rc.circle(x2, y2, 9, opts(seed + 1, "#8a94a6", "#8a94a6", 1.5)));
}

/* ---------- 2 apple / pear ---------- */
function drawApple(svg, rc, x, seed) {
  const t = 96;
  svg.appendChild(rc.path(
    `M ${x} ${t+16} C ${x-24} ${t-10} ${x-78} ${t+2} ${x-78} ${t+58}` +
    ` C ${x-78} ${t+112} ${x-30} ${t+142} ${x} ${t+134}` +
    ` C ${x+30} ${t+142} ${x+78} ${t+112} ${x+78} ${t+58}` +
    ` C ${x+78} ${t+2} ${x+24} ${t-10} ${x} ${t+16} Z`,
    opts(seed, "#b93443", "#f4838f")));
  line(svg, rc, x, t + 14, x + 6, t - 26, opts(seed + 1, "#76513a"));
  svg.appendChild(rc.path(
    `M ${x+6} ${t-22} Q ${x+40} ${t-46} ${x+52} ${t-14} Q ${x+26} ${t-2} ${x+6} ${t-22} Z`,
    opts(seed + 2, "#3f8f3f", "#a8dc87")));
}
function drawPear(svg, rc, x, seed) {
  svg.appendChild(rc.path(
    `M ${x} 74 C ${x-30} 92 ${x-34} 124 ${x-42} 146` +
    ` C ${x-56} 176 ${x-70} 196 ${x-62} ${BASE-24}` +
    ` C ${x-54} ${BASE+6} ${x+54} ${BASE+6} ${x+62} ${BASE-24}` +
    ` C ${x+70} 196 ${x+56} 176 ${x+42} 146` +
    ` C ${x+34} 124 ${x+30} 92 ${x} 74 Z`,
    opts(seed, "#8da52f", "#dbe97a")));
  line(svg, rc, x, 76, x + 6, 44, opts(seed + 1, "#76513a"));
  svg.appendChild(rc.path(
    `M ${x+6} 48 Q ${x+38} 26 ${x+50} 56 Q ${x+24} 68 ${x+6} 48 Z`,
    opts(seed + 2, "#3f8f3f", "#a8dc87")));
}

/* ---------- 3 book / bookmark ---------- */
function drawBook(svg, rc, x, seed) {
  const b = BASE, t = b - 150;
  // front cover, slightly in perspective
  svg.appendChild(rc.path(`M ${x-64} ${t+10} L ${x+52} ${t} L ${x+52} ${b-10} L ${x-64} ${b} Z`,
    opts(seed, "#1c7fa8", "#57bbe0")));
  // page edge on the right (leaves)
  svg.appendChild(rc.path(`M ${x+52} ${t} L ${x+78} ${t+12} L ${x+78} ${b+2} L ${x+52} ${b-10} Z`,
    opts(seed + 1, "#9fb0c4", "#ffffff")));
  for (let i = 1; i <= 4; i++)
    line(svg, rc, x + 54, t + 12 + i * 28, x + 78, t + 24 + i * 28,
      opts(seed + 20 + i, "#c3cedc", null, 1.8));
  // title frame on the cover
  svg.appendChild(rc.rectangle(x - 48, t + 34, 84, 62,
    {...opts(seed + 3, "#ffffff", null, 2.5), roughness: 1.4}));
  line(svg, rc, x - 38, t + 54, x + 24, t + 50, opts(seed + 4, "#ffffff", null, 3));
  line(svg, rc, x - 38, t + 74, x + 10, t + 71, opts(seed + 5, "#ffffff", null, 3));
  // ribbon bookmark
  line(svg, rc, x + 20, t + 4, x + 24, t + 26, opts(seed + 6, GOLD, null, 3));
}
function drawBookmark(svg, rc, x, seed) {
  const t = 76, b = BASE;
  svg.appendChild(rc.polygon(
    [[x-34, t], [x+34, t], [x+34, b], [x, b-28], [x-34, b]],
    opts(seed, "#c23a6d", "#f7a8c4")));
  svg.appendChild(rc.circle(x, t + 40, 30, opts(seed + 1, GOLD, "#ffe08a")));
  line(svg, rc, x - 20, t + 76, x + 20, t + 76, opts(seed + 2, "#c23a6d", null, 2.5));
  line(svg, rc, x - 20, t + 96, x + 20, t + 96, opts(seed + 3, "#c23a6d", null, 2.5));
  // cord on top
  svg.appendChild(rc.path(`M ${x} ${t} C ${x-14} ${t-30} ${x+18} ${t-34} ${x+4} ${t-6}`,
    opts(seed + 4, GOLD, null, 2.5)));
}

/* ---------- 4 sandwich / drink ---------- */
function drawSandwich(svg, rc, x, seed) {
  const b = BASE, W = 92;
  // bottom slice of bread
  svg.appendChild(rc.path(`M ${x-W} ${b-8} L ${x-W} ${b-34} L ${x+W} ${b-34} L ${x+W} ${b-8} Z`,
    opts(seed, "#a97a2e", "#f7dfae")));
  // lettuce leaf (wavy)
  svg.appendChild(rc.path(
    `M ${x-W-6} ${b-34} Q ${x-W/2} ${b-52} ${x-16} ${b-36} Q ${x+34} ${b-56} ${x+W+6} ${b-38}` +
    ` L ${x+W+6} ${b-52} Q ${x+20} ${b-72} ${x-30} ${b-54} Q ${x-W/2} ${b-70} ${x-W-6} ${b-52} Z`,
    opts(seed + 1, "#3f8f3f", "#a8dc87")));
  // tomato / cheese
  svg.appendChild(rc.path(`M ${x-W} ${b-54} L ${x+W} ${b-58} L ${x+W} ${b-76} L ${x-W} ${b-72} Z`,
    opts(seed + 2, "#c23a6d", "#f7a8c4")));
  // top slice of bread (flat, like the bottom one)
  svg.appendChild(rc.path(`M ${x-W} ${b-74} L ${x-W} ${b-104} L ${x+W} ${b-104} L ${x+W} ${b-74} Z`,
    opts(seed + 3, "#a97a2e", "#f2d296")));
}
function drawDrink(svg, rc, x, seed) {
  const t = 96, b = BASE;
  svg.appendChild(rc.path(`M ${x-52} ${t} L ${x+52} ${t} L ${x+38} ${b} L ${x-38} ${b} Z`,
    opts(seed, "#1c7fa8", "#bfeaf7")));
  // lid
  svg.appendChild(rc.rectangle(x - 60, t - 16, 120, 18, opts(seed + 1, "#1c7fa8", "#86d1ec")));
  // straw
  svg.appendChild(rc.path(`M ${x+12} ${t-14} L ${x+34} ${t-62} L ${x+60} ${t-70}`,
    opts(seed + 2, PINK, null, 5)));
  // stripes on the cup
  line(svg, rc, x - 30, t + 42, x + 30, t + 42, opts(seed + 3, "#1c7fa8", null, 2.5));
  line(svg, rc, x - 26, t + 74, x + 26, t + 74, opts(seed + 4, "#1c7fa8", null, 2.5));
}

/* ---------- 5 hat / scarf ---------- */
function drawHat(svg, rc, x, seed) {
  const b = BASE;
  svg.appendChild(rc.path(`M ${x-74} ${b-38} C ${x-74} ${b-150} ${x+74} ${b-150} ${x+74} ${b-38} Z`,
    opts(seed, "#c23a6d", "#f7a8c4")));
  svg.appendChild(rc.rectangle(x - 86, b - 44, 172, 44, opts(seed + 1, "#c23a6d", "#ef87ad")));
  for (let i = 0; i < 5; i++)
    line(svg, rc, x - 66 + i * 33, b - 40, x - 66 + i * 33, b - 4,
      opts(seed + 5 + i, "#c23a6d", null, 2));
  svg.appendChild(rc.circle(x, b - 158, 46, opts(seed + 2, "#d79b12", "#ffe08a")));
}
function drawScarf(svg, rc, x, seed) {
  const t = 62, b = BASE;
  const stripes = (x0, y0, x1, y1, w, s2) => {
    for (let i = 1; i <= 3; i++) {
      const f = i / 4;
      line(svg, rc, x0 + (x1 - x0) * f - w / 2, y0 + (y1 - y0) * f,
        x0 + (x1 - x0) * f + w / 2, y0 + (y1 - y0) * f + 4, opts(s2 + i, "#1c7fa8", null, 2.5));
    }
  };
  // loop at the top
  svg.appendChild(rc.path(
    `M ${x-58} ${t+34} C ${x-58} ${t-16} ${x+58} ${t-16} ${x+58} ${t+34}` +
    ` L ${x+22} ${t+34} C ${x+22} ${t+12} ${x-22} ${t+12} ${x-22} ${t+34} Z`,
    opts(seed, "#1c7fa8", "#8fd6ee")));
  // left end
  svg.appendChild(rc.path(`M ${x-58} ${t+30} L ${x-22} ${t+30} L ${x-14} ${b-18} L ${x-56} ${b-18} Z`,
    opts(seed + 1, "#1c7fa8", "#8fd6ee")));
  // right end (shorter)
  svg.appendChild(rc.path(`M ${x+22} ${t+30} L ${x+58} ${t+30} L ${x+62} ${b-52} L ${x+18} ${b-52} Z`,
    opts(seed + 2, "#1c7fa8", "#bfeaf7")));
  stripes(x - 40, t + 40, x - 36, b - 22, 40, seed + 10);
  stripes(x + 40, t + 40, x + 42, b - 56, 40, seed + 20);
  // fringes
  for (let i = 0; i < 4; i++) {
    line(svg, rc, x - 50 + i * 12, b - 18, x - 53 + i * 12, b, opts(seed + 30 + i, "#1c7fa8", null, 2.5));
    line(svg, rc, x + 24 + i * 12, b - 52, x + 21 + i * 12, b - 34, opts(seed + 40 + i, "#1c7fa8", null, 2.5));
  }
}

/* ---------- 6 ball / skipping rope ---------- */
function drawBall(svg, rc, x, seed) {
  const cy = BASE - 78, r = 78;
  svg.appendChild(rc.circle(x, cy, r * 2, opts(seed, "#1c7fa8", "#bfeaf7")));
  // football pattern
  svg.appendChild(rc.polygon([[x, cy-34], [x+32, cy-11], [x+20, cy+27], [x-20, cy+27], [x-32, cy-11]],
    opts(seed + 1, "#1c7fa8", "#2f9fd0")));
  const pts = [[0,-34],[32,-11],[20,27],[-20,27],[-32,-11]];
  pts.forEach((p, i) => line(svg, rc, x + p[0], cy + p[1], x + p[0] * 2.2, cy + p[1] * 2.2,
    opts(seed + 10 + i, "#1c7fa8", null, 2.5)));
}
function drawRope(svg, rc, x, seed) {
  // rope as an arc between the two handles
  svg.appendChild(rc.path(
    `M ${x-72} 96 C ${x-150} 170 ${x-70} ${BASE+4} ${x} ${BASE-6}` +
    ` C ${x+70} ${BASE+4} ${x+150} 170 ${x+72} 96`,
    {seed, roughness: 1.1, stroke: "#c23a6d", strokeWidth: 6}));
  [[-72, 96], [72, 96]].forEach((p, i) => {
    svg.appendChild(rc.path(
      `M ${x+p[0]-13} ${p[1]} L ${x+p[0]+13} ${p[1]} L ${x+p[0]+11} ${p[1]-62} L ${x+p[0]-11} ${p[1]-62} Z`,
      opts(seed + 1 + i, "#7a5230", "#d9a35f")));
    svg.appendChild(rc.line(x + p[0] - 14, p[1] - 62, x + p[0] + 14, p[1] - 62,
      opts(seed + 5 + i, "#7a5230", null, 3)));
  });
}

/* ---------- 7 cake / bread ---------- */
function drawCake(svg, rc, x, seed) {
  const b = BASE;
  // plate
  svg.appendChild(rc.ellipse(x, b - 4, 216, 20, opts(seed, "#9fb0c4", "#eef3f8")));
  // two cake layers
  svg.appendChild(rc.rectangle(x - 84, b - 56, 168, 52, opts(seed + 1, "#a96939", "#e8b576")));
  svg.appendChild(rc.rectangle(x - 84, b - 100, 168, 46, opts(seed + 2, "#a96939", "#f0cb96")));
  // icing with drips
  svg.appendChild(rc.path(
    `M ${x-88} ${b-100} L ${x-88} ${b-124} L ${x+88} ${b-124} L ${x+88} ${b-100}` +
    ` Q ${x+66} ${b-84} ${x+44} ${b-100} T ${x} ${b-100} T ${x-44} ${b-100} T ${x-88} ${b-100} Z`,
    opts(seed + 3, "#c23a6d", "#f7a8c4")));
  // candle
  svg.appendChild(rc.rectangle(x - 8, b - 168, 16, 44, opts(seed + 4, "#1c7fa8", "#bfeaf7")));
  svg.appendChild(rc.ellipse(x, b - 176, 18, 26, opts(seed + 5, "#d79b12", "#ffe08a")));
}
function drawBread(svg, rc, x, seed) {
  const b = BASE;
  svg.appendChild(rc.path(
    `M ${x-100} ${b} C ${x-116} ${b-70} ${x-84} ${b-134} ${x-30} ${b-136}` +
    ` C ${x+40} ${b-140} ${x+112} ${b-96} ${x+100} ${b} Z`,
    opts(seed, "#8f5a26", "#e9bd72")));
  [-52, -12, 28, 66].forEach((dx, i) =>
    svg.appendChild(rc.path(`M ${x+dx-6} ${b-116} q 18 12 26 34`,
      opts(seed + i + 1, "#8f5a26", null, 3))));
}

/* ---------- 8 bike helmet / bike light ---------- */
function drawHelmet(svg, rc, x, seed) {
  const b = BASE - 4, cy = b;
  // visor at the front right
  svg.appendChild(rc.path(`M ${x+44} ${cy-42} C ${x+92} ${cy-40} ${x+104} ${cy-16} ${x+96} ${cy-2}` +
    ` C ${x+80} ${cy-14} ${x+62} ${cy-20} ${x+44} ${cy-22} Z`,
    opts(seed + 5, "#8c2450", "#ef87ad")));
  // helmet shell (dome)
  svg.appendChild(rc.path(
    `M ${x-90} ${cy-16} C ${x-92} ${cy-124} ${x+70} ${cy-136} ${x+86} ${cy-34}` +
    ` C ${x+40} ${cy-14} ${x-40} ${cy-8} ${x-90} ${cy-16} Z`,
    opts(seed, "#c23a6d", "#f7a8c4")));
  // vent slits
  [[-56, -76, -24, -90], [-16, -92, 16, -102], [24, -90, 54, -88]].forEach((s2, i) =>
    line(svg, rc, x + s2[0], cy + s2[1], x + s2[2], cy + s2[3], opts(seed + 1 + i, "#ffffff", null, 8)));
  // chin strap
  line(svg, rc, x - 66, cy - 14, x - 44, cy + 4, opts(seed + 8, INK, null, 3));
  line(svg, rc, x + 42, cy - 20, x + 16, cy + 4, opts(seed + 9, INK, null, 3));
  line(svg, rc, x - 44, cy + 4, x + 16, cy + 4, opts(seed + 10, INK, null, 3));
}
function drawLight(svg, rc, x, seed) {
  const cy = BASE - 104, R = 58;
  // mount
  svg.appendChild(rc.path(`M ${x-14} ${cy+R-6} L ${x+14} ${cy+R-6} L ${x+20} ${BASE} L ${x-20} ${BASE} Z`,
    opts(seed + 3, "#4a5568", "#aab4c4")));
  // housing
  svg.appendChild(rc.circle(x, cy, R * 2, opts(seed, "#4a5568", "#c9d3df")));
  // reflector + glowing light
  svg.appendChild(rc.circle(x, cy, R * 1.4, opts(seed + 1, "#d79b12", "#ffe08a")));
  svg.appendChild(rc.circle(x, cy, R * 0.6, opts(seed + 2, "#d79b12", "#fffbe4")));
  // light beams
  for (let i = 0; i < 7; i++) {
    const ang = -Math.PI * 0.78 + i * (Math.PI * 1.06 / 6);
    const c = Math.cos(ang), s2 = Math.sin(ang);
    line(svg, rc, x + c * (R + 14), cy + s2 * (R + 14), x + c * (R + 44), cy + s2 * (R + 44),
      opts(seed + 10 + i, "#d79b12", null, 3));
  }
}

function buildPicture(spec, seed) {
  const svg = el("svg", {xmlns: SVGNS, viewBox: "40 16 820 292", preserveAspectRatio: "xMidYMid meet"});
  const style = el("style");
  style.textContent =
    `.label{font:700 30px 'Caveat',cursive;fill:${INK}}` +
    `.price{font:700 30px 'Caveat',cursive;fill:${PINK}}` +
    `.total{font:700 34px 'Caveat',cursive;fill:${INK}}`;
  svg.appendChild(style);
  const rc = rough.svg(svg);
  ground(svg, rc, seed);
  const x1 = 240, x2 = 660;
  if (spec.kind === "plant") {
    // Special case: plant and pot belong together, so they share one drawing.
    const cx = (x1 + x2) / 2;
    // Only the middle of the canvas is used, so crop the viewBox to it.
    svg.setAttribute("viewBox", `${cx - 250} 16 500 292`);
    drawPottedPlant(svg, rc, cx, seed + 1);
    leader(svg, rc, cx - 152, 60, cx - 56, 78, seed + 60);
    text(svg, cx - 196, 66, spec.names[0], "label");
    leader(svg, rc, cx + 152, 190, cx + 44, 190, seed + 70);
    text(svg, cx + 196, 196, spec.names[1], "label");
    return svg;
  }
  const drawers = {
    fruit: [() => drawApple(svg,rc,x1,seed+1), () => drawPear(svg,rc,x2,seed+10)],
    reading: [() => drawBook(svg,rc,x1,seed+1), () => drawBookmark(svg,rc,x2,seed+10)],
    snack: [() => drawSandwich(svg,rc,x1,seed+1), () => drawDrink(svg,rc,x2,seed+10)],
    winter: [() => drawHat(svg,rc,x1,seed+1), () => drawScarf(svg,rc,x2,seed+10)],
    play: [() => drawBall(svg,rc,x1,seed+1), () => drawRope(svg,rc,x2,seed+10)],
    bakery: [() => drawCake(svg,rc,x1,seed+1), () => drawBread(svg,rc,x2,seed+10)],
    bike: [() => drawHelmet(svg,rc,x1,seed+1), () => drawLight(svg,rc,x2,seed+10)]
  };
  drawers[spec.kind][0](); drawers[spec.kind][1]();
  labelUnder(svg, x1, spec.names[0]);
  labelUnder(svg, x2, spec.names[1]);
  return svg;
}

for (const host of document.querySelectorAll("[data-price]")) {
  const key = host.dataset.price;
  host.appendChild(buildPicture(PRICES[key], 100 + Number(key.slice(1)) * 97));
}
