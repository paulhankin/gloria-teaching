const SVGNS = "http://www.w3.org/2000/svg";
const INK = "#1f3550";
const PINK = "#e8548c";
const BLUE = "#2f9fd0";
const GOLD = "#e0a01e";
const GREEN = "#5eb85e";

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
function opts(seed, stroke, fill) {
  const o = {seed, roughness: 1.25, bowing: 1.1, stroke: stroke || INK, strokeWidth: 3};
  if (fill) Object.assign(o, {fill, fillStyle: "hachure", hachureGap: 8, fillWeight: 1.2});
  return o;
}
function priceTag(svg, rc, x, y, seed) {
  svg.appendChild(rc.rectangle(x - 43, y - 25, 86, 50, {
    ...opts(seed, PINK, "#fff7fa"), roughness: 0.9, fillStyle: "solid"
  }));
  text(svg, x, y + 10, "? Fr.", "price");
}
function ground(svg, rc, seed) {
  svg.appendChild(rc.line(55, 245, 845, 245, {seed, roughness: 1.8, stroke: "#d8dfe8", strokeWidth: 3}));
}

function drawPlant(svg, rc, x, seed) {
  svg.appendChild(rc.line(x, 165, x, 76, opts(seed, GREEN)));
  [[-42,112,-8,126], [40,92,7,110], [-34,65,-5,85], [31,50,4,73]].forEach((p, i) =>
    svg.appendChild(rc.ellipse(x + p[0] / 2, p[1], Math.abs(p[0]), 34,
      opts(seed + i + 1, GREEN, "#a8dc87"))));
}
function drawPot(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-63} 155 L ${x+63} 155 L ${x+46} 235 L ${x-45} 235 Z`,
    opts(seed, "#b85f38", "#e69264")));
  svg.appendChild(rc.rectangle(x - 72, 145, 144, 24, opts(seed + 1, "#b85f38", "#f0a078")));
}
function drawApple(svg, rc, x, seed) {
  svg.appendChild(rc.circle(x, 153, 130, opts(seed, "#b93443", "#ef6472")));
  svg.appendChild(rc.line(x + 2, 90, x + 12, 56, opts(seed + 1, "#76513a")));
  svg.appendChild(rc.ellipse(x + 35, 63, 48, 25, opts(seed + 2, GREEN, "#8dcc69")));
}
function drawPear(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x} 73 C ${x-16} 105 ${x-70} 128 ${x-61} 188 C ${x-53} 246 ${x+54} 246 ${x+62} 188 C ${x+70} 130 ${x+17} 106 ${x} 73 Z`,
    opts(seed, "#8da52f", "#c8df59")));
  svg.appendChild(rc.line(x, 77, x + 10, 46, opts(seed + 1, "#76513a")));
  svg.appendChild(rc.ellipse(x + 30, 53, 43, 22, opts(seed + 2, GREEN, "#8dcc69")));
}
function drawBook(svg, rc, x, seed) {
  svg.appendChild(rc.rectangle(x - 80, 75, 160, 150, opts(seed, BLUE, "#86d1ec")));
  svg.appendChild(rc.line(x - 55, 75, x - 55, 225, opts(seed + 1, BLUE)));
  svg.appendChild(rc.line(x - 28, 112, x + 48, 112, opts(seed + 2, "#fff")));
  svg.appendChild(rc.line(x - 28, 132, x + 35, 132, opts(seed + 3, "#fff")));
}
function drawBookmark(svg, rc, x, seed) {
  svg.appendChild(rc.polygon([[x-30,55],[x+30,55],[x+30,224],[x,197],[x-30,224]],
    opts(seed, PINK, "#f49abb")));
  svg.appendChild(rc.circle(x, 102, 28, opts(seed + 1, GOLD, "#ffd76a")));
}
function drawSandwich(svg, rc, x, seed) {
  svg.appendChild(rc.polygon([[x-105,190],[x,69],[x+105,190]], opts(seed, "#b87b31", "#efc779")));
  svg.appendChild(rc.line(x - 79, 174, x + 78, 174, {seed:seed+1, roughness:1.4, stroke:GREEN, strokeWidth:10}));
  svg.appendChild(rc.line(x - 64, 158, x + 64, 158, {seed:seed+2, roughness:1.2, stroke:PINK, strokeWidth:9}));
}
function drawDrink(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-58} 82 L ${x+58} 82 L ${x+43} 224 L ${x-42} 224 Z`, opts(seed, BLUE, "#b8e7f5")));
  svg.appendChild(rc.line(x + 14, 82, x + 48, 37, opts(seed + 1, PINK)));
  svg.appendChild(rc.ellipse(x, 82, 118, 26, opts(seed + 2, BLUE, "#eafaff")));
}
function drawHat(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-75} 177 C ${x-70} 87 ${x+70} 87 ${x+75} 177 Z`, opts(seed, PINK, "#f49abb")));
  svg.appendChild(rc.rectangle(x - 88, 169, 176, 43, opts(seed + 1, PINK, "#ef87ad")));
  svg.appendChild(rc.circle(x, 80, 48, opts(seed + 2, GOLD, "#ffd76a")));
}
function drawScarf(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-72} 56 C ${x-13} 101 ${x+28} 127 ${x+59} 207 L ${x+9} 229 C ${x-13} 150 ${x-55} 123 ${x-104} 91 Z`, opts(seed, BLUE, "#75c7e5")));
  for (let i = 0; i < 4; i++) svg.appendChild(rc.line(x + 10 + i * 14, 225, x + 5 + i * 14, 246, opts(seed + 5 + i, BLUE)));
}
function drawBall(svg, rc, x, seed) {
  svg.appendChild(rc.circle(x, 150, 155, opts(seed, BLUE, "#8ed4ed")));
  svg.appendChild(rc.arc(x, 150, 150, 68, 0, Math.PI, false, opts(seed + 1, PINK)));
  svg.appendChild(rc.arc(x, 150, 70, 150, Math.PI / 2, Math.PI * 1.5, false, opts(seed + 2, GOLD)));
}
function drawRope(svg, rc, x, seed) {
  svg.appendChild(rc.ellipse(x, 151, 157, 172, opts(seed, GREEN)));
  svg.appendChild(rc.line(x - 77, 161, x - 102, 221, {seed:seed+1, roughness:1.2, stroke:PINK, strokeWidth:10}));
  svg.appendChild(rc.line(x + 77, 161, x + 102, 221, {seed:seed+2, roughness:1.2, stroke:PINK, strokeWidth:10}));
}
function drawCake(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-96} 104 L ${x+93} 104 L ${x+72} 221 L ${x-73} 221 Z`, opts(seed, "#a96939", "#e8b576")));
  svg.appendChild(rc.path(`M ${x-96} 104 Q ${x-55} 72 ${x-16} 101 T ${x+55} 99 T ${x+93} 104`, opts(seed+1, PINK, "#f49abb")));
  svg.appendChild(rc.circle(x + 8, 69, 27, opts(seed + 2, "#b93443", "#ef6472")));
}
function drawBread(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-99} 204 L ${x-85} 112 C ${x-62} 53 ${x+60} 53 ${x+87} 112 L ${x+99} 204 Z`, opts(seed, "#a96b31", "#e9bd72")));
  [-42,0,42].forEach((dx, i) => svg.appendChild(rc.line(x + dx - 13, 100, x + dx + 7, 135, opts(seed + i + 1, "#fff1c4"))));
}
function drawHelmet(svg, rc, x, seed) {
  svg.appendChild(rc.path(`M ${x-98} 166 C ${x-93} 67 ${x+84} 48 ${x+102} 157 L ${x+33} 157 L ${x+6} 211 L ${x-45} 211 L ${x-57} 166 Z`, opts(seed, PINK, "#f38fb4")));
  svg.appendChild(rc.line(x - 8, 75, x + 10, 155, opts(seed + 1, "#fff")));
  svg.appendChild(rc.line(x - 65, 105, x + 68, 132, opts(seed + 2, "#fff")));
}
function drawLight(svg, rc, x, seed) {
  svg.appendChild(rc.circle(x, 135, 125, opts(seed, GOLD, "#ffd76a")));
  svg.appendChild(rc.circle(x, 135, 80, { ...opts(seed + 1, "#d69200", "#fff7c7"), fillStyle:"solid" }));
  svg.appendChild(rc.rectangle(x - 27, 195, 54, 35, opts(seed + 2, INK, "#8a94a6")));
  [[-80,-43],[-88,0],[-70,50],[80,-43],[88,0],[70,50]].forEach((p,i) =>
    svg.appendChild(rc.line(x+p[0],135+p[1],x+p[0]*1.22,135+p[1]*1.22,opts(seed+10+i,GOLD))));
}

function buildPicture(spec, seed) {
  const svg = el("svg", {xmlns: SVGNS, viewBox: "0 0 900 300", preserveAspectRatio: "xMidYMid meet"});
  const style = el("style");
  style.textContent = `.label{font:700 28px 'Caveat',cursive;fill:${INK}} .price{font:700 28px 'Caveat',cursive;fill:${PINK}} .total{font:700 31px 'Caveat',cursive;fill:${INK}}`;
  svg.appendChild(style);
  const rc = rough.svg(svg);
  ground(svg, rc, seed);
  const x1 = 260, x2 = 640;
  const drawers = {
    pflanze: [() => { drawPlant(svg,rc,x1,seed+1); drawPot(svg,rc,x1,seed+10); }, () => drawPot(svg,rc,x2,seed+20)],
    obst: [() => drawApple(svg,rc,x1,seed+1), () => drawPear(svg,rc,x2,seed+10)],
    lesen: [() => drawBook(svg,rc,x1,seed+1), () => drawBookmark(svg,rc,x2,seed+10)],
    "znüni": [() => drawSandwich(svg,rc,x1,seed+1), () => drawDrink(svg,rc,x2,seed+10)],
    winter: [() => drawHat(svg,rc,x1,seed+1), () => drawScarf(svg,rc,x2,seed+10)],
    spiel: [() => drawBall(svg,rc,x1,seed+1), () => drawRope(svg,rc,x2,seed+10)],
    baeckerei: [() => drawCake(svg,rc,x1,seed+1), () => drawBread(svg,rc,x2,seed+10)],
    velo: [() => drawHelmet(svg,rc,x1,seed+1), () => drawLight(svg,rc,x2,seed+10)]
  };
  drawers[spec.typ][0](); drawers[spec.typ][1]();
  priceTag(svg, rc, x1, 270, seed + 80);
  priceTag(svg, rc, x2, 270, seed + 81);
  text(svg, x1, 298, spec.namen[0], "label");
  text(svg, x2, 298, spec.namen[1], "label");
  text(svg, 450, 114, "+", "total");
  svg.appendChild(rc.rectangle(385, 137, 130, 49, {...opts(seed+90, GREEN, "#f4fff9"), fillStyle:"solid"}));
  text(svg, 450, 170, `= ${spec.summe} Fr.`, "total");
  return svg;
}

for (const host of document.querySelectorAll("[data-preis]")) {
  const key = host.dataset.preis;
  host.appendChild(buildPicture(PREISE[key], 100 + Number(key.slice(1)) * 97));
}
