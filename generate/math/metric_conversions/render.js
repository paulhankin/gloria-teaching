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
    .factor{font:700 18px 'Caveat',cursive;fill:#65778a}
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

function drawConnector(svg, rc, from, to, factor, seed) {
  const fromRight = from.x + from.w / 2;
  const toLeft = to.x - to.w / 2;
  const forwardY1 = from.y - 14;
  const forwardY2 = to.y - 24;
  const reverseY1 = to.y - 4;
  const reverseY2 = from.y + 8;
  const middleX = (fromRight + toLeft) / 2;

  // Both arrows end exactly at a box edge. Their arrowheads point back into
  // the open gap, so neither the line nor the head crosses a box.
  drawArrow(svg, rc, fromRight, forwardY1, toLeft, forwardY2, seed, GREEN);
  drawArrow(svg, rc, toLeft, reverseY1, fromRight, reverseY2, seed + 10, PINK);
  label(svg, middleX, (forwardY1 + forwardY2) / 2 - 10, `· ${factor}`, "factor");
  label(svg, middleX, (reverseY1 + reverseY2) / 2 + 18, `: ${factor}`, "factor");
}

function ladder() {
  const svg = baseSVG("0 0 1000 220");
  const rc = rough.svg(svg);
  const units = [
    {name: "km", x: 90, y: 42, w: 150, color: "#bfe3cd"},
    {name: "m", x: 300, y: 82, w: 150, color: "#bfeaf7"},
    {name: "dm", x: 510, y: 122, w: 150, color: "#ffe6a6"},
    {name: "cm", x: 720, y: 162, w: 150, color: "#f8c5d8"},
    {name: "mm", x: 900, y: 202, w: 100, color: "#ddd0f2"},
  ];
  units.forEach((unit, index) => {
    svg.appendChild(rc.rectangle(unit.x - unit.w / 2, unit.y - 36, unit.w, 48,
      {...options(20 + index, INK, unit.color), roughness: 0.8}));
    label(svg, unit.x, unit.y, unit.name, "unit");
  });
  drawConnector(svg, rc, units[0], units[1], 1000, 60);
  for (let i = 1; i < units.length - 1; i++) {
    drawConnector(svg, rc, units[i], units[i + 1], 10, 90 + i * 30);
  }
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
  if (kind === "compare") host.appendChild(compare());
});
