const SVG_NS = "http://www.w3.org/2000/svg";
const INK = "#1f3550";
const PINK = "#e8548c";
const BLUE = "#2f9fd0";
const GOLD = "#e0a01e";
const GREEN = "#5eb85e";

function node(tag, attrs = {}) {
  const element = document.createElementNS(SVG_NS, tag);
  for (const [key, value] of Object.entries(attrs)) element.setAttribute(key, value);
  return element;
}

function label(svg, x, y, value, className, anchor = "middle") {
  const element = node("text", {x, y, class: className, "text-anchor": anchor});
  element.textContent = value;
  svg.appendChild(element);
}

function options(seed, stroke = INK, fill = null, width = 3) {
  const result = {seed, roughness: 1.2, bowing: 1, stroke, strokeWidth: width};
  if (fill) Object.assign(result, {fill, fillStyle: "solid"});
  return result;
}

function baseSVG(viewBox) {
  const svg = node("svg", {xmlns: SVG_NS, viewBox, preserveAspectRatio: "xMidYMid meet"});
  const style = node("style");
  style.textContent = `
    .unit{font:700 38px 'Caveat',cursive;fill:#1f3550}
    .small{font:700 22px 'Caveat',cursive;fill:#65778a}
    .note{font:700 25px 'Caveat',cursive;fill:#e8548c}
  `;
  svg.appendChild(style);
  return svg;
}

function drawArrow(svg, rc, x1, y1, x2, y2, seed, color) {
  svg.appendChild(rc.line(x1, y1, x2, y2, options(seed, color, null, 3)));
  const angle = Math.atan2(y2 - y1, x2 - x1);
  const wing = 11;
  for (const delta of [-0.65, 0.65]) {
    svg.appendChild(rc.line(x2, y2,
      x2 - wing * Math.cos(angle + delta), y2 - wing * Math.sin(angle + delta),
      options(seed + 1 + (delta > 0 ? 1 : 0), color, null, 3)));
  }
}

function ladder() {
  const svg = baseSVG("0 0 1000 220");
  const rc = rough.svg(svg);
  const units = [
    {name: "km", x: 75, y: 44, w: 170, color: "#bfe3cd"},
    {name: "m", x: 285, y: 84, w: 170, color: "#bfeaf7"},
    {name: "dm", x: 495, y: 124, w: 170, color: "#ffe6a6"},
    {name: "cm", x: 705, y: 164, w: 170, color: "#f8c5d8"},
    {name: "mm", x: 875, y: 204, w: 100, color: "#ddd0f2"},
  ];
  units.forEach((unit, index) => {
    svg.appendChild(rc.rectangle(unit.x - unit.w / 2, unit.y - 36, unit.w, 48,
      options(20 + index, INK, unit.color)));
    label(svg, unit.x, unit.y, unit.name, "unit");
  });
  drawArrow(svg, rc, 175, 28, 355, 62, 60, GREEN);
  label(svg, 265, 25, "× 1000", "small");
  for (let i = 1; i < units.length - 1; i++) {
    drawArrow(svg, rc, units[i].x + 90, units[i].y + 4, units[i + 1].x - 90, units[i + 1].y + 34,
      70 + i * 5, GREEN);
    label(svg, (units[i].x + units[i + 1].x) / 2, units[i].y + 13, "× 10", "small");
  }
  drawArrow(svg, rc, 850, 216, 670, 184, 100, PINK);
  label(svg, 760, 217, "÷ 10", "small");
  drawArrow(svg, rc, 455, 114, 265, 78, 110, PINK);
  label(svg, 360, 118, "÷ 10", "small");
  drawArrow(svg, rc, 265, 70, 120, 42, 120, PINK);
  label(svg, 188, 78, "÷ 1000", "small");
  return svg;
}

function measure() {
  const svg = baseSVG("0 0 240 620");
  const rc = rough.svg(svg);
  // Ruler
  svg.appendChild(rc.rectangle(50, 30, 90, 520, options(1, GOLD, "#fff4c7")));
  for (let i = 0; i <= 20; i++) {
    const y = 50 + i * 24;
    const length = i % 10 === 0 ? 55 : (i % 5 === 0 ? 42 : 28);
    svg.appendChild(rc.line(50, y, 50 + length, y, options(10 + i, INK, null, i % 5 === 0 ? 2.5 : 1.5)));
  }
  label(svg, 95, 588, "messen", "note");
  // Pencil
  svg.appendChild(rc.polygon([[166, 92], [202, 92], [202, 475], [184, 528], [166, 475]],
    options(80, PINK, "#f8c5d8")));
  svg.appendChild(rc.line(166, 150, 202, 150, options(81, PINK)));
  label(svg, 184, 65, "?", "unit");
  return svg;
}

function compare() {
  const svg = baseSVG("0 0 350 300");
  const rc = rough.svg(svg);
  // Balance scale
  svg.appendChild(rc.line(175, 62, 175, 230, options(1, INK, null, 5)));
  svg.appendChild(rc.line(92, 235, 258, 235, options(2, INK, null, 5)));
  svg.appendChild(rc.line(65, 93, 285, 93, options(3, BLUE, null, 5)));
  svg.appendChild(rc.circle(175, 93, 18, options(4, INK, "#fff")));
  [[65, "<"], [285, ">"]].forEach(([x, sign], index) => {
    svg.appendChild(rc.line(x, 93, x - 42, 174, options(10 + index, INK)));
    svg.appendChild(rc.line(x, 93, x + 42, 174, options(12 + index, INK)));
    svg.appendChild(rc.path(`M ${x-48} 174 Q ${x} 202 ${x+48} 174`, options(14 + index, GOLD)));
    label(svg, x, 168, sign, "unit");
  });
  label(svg, 175, 285, "erst umwandeln, dann vergleichen", "small");
  return svg;
}

document.querySelectorAll("[data-figure]").forEach((host) => {
  const kind = host.dataset.figure;
  if (kind === "ladder") host.appendChild(ladder());
  if (kind === "measure") host.appendChild(measure());
  if (kind === "compare") host.appendChild(compare());
});
