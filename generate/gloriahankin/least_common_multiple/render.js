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
  const result = {seed, roughness: 1.15, bowing: 1, stroke, strokeWidth: width};
  if (fill) Object.assign(result, {fill, fillStyle: "solid"});
  return result;
}

function baseSVG(viewBox) {
  const svg = node("svg", {xmlns: SVG_NS, viewBox, preserveAspectRatio: "xMidYMid meet"});
  const style = node("style");
  style.textContent = `
    .big{font:700 38px 'Caveat',cursive;fill:#1f3550}
    .number{font:700 23px 'Caveat',cursive;fill:#65778a}
    .note{font:700 25px 'Caveat',cursive;fill:#e8548c}
    .small{font:700 19px 'Caveat',cursive;fill:#65778a}
  `;
  svg.appendChild(style);
  return svg;
}

function jumps() {
  const svg = baseSVG("0 0 900 210");
  const rc = rough.svg(svg);
  const startX = 55;
  const endX = 700;
  const lineY = 125;
  svg.appendChild(rc.line(startX, lineY, endX, lineY, options(1, INK, null, 4)));
  for (let value = 0; value <= 24; value += 4) {
    const x = startX + value / 24 * (endX - startX);
    svg.appendChild(rc.line(x, lineY - 15, x, lineY + 15, options(10 + value, INK, null, 2)));
    label(svg, x, lineY + 42, String(value), "number");
    if (value < 24) {
      const nextX = startX + (value + 4) / 24 * (endX - startX);
      svg.appendChild(rc.path(`M ${x + 7} ${lineY - 18} Q ${(x + nextX) / 2} 35 ${nextX - 7} ${lineY - 18}`,
        options(50 + value, BLUE, null, 3)));
      label(svg, (x + nextX) / 2, 65, "+ 4", "small");
    }
  }
  svg.appendChild(rc.circle(800, 87, 100, options(90, GREEN, "#dff3df")));
  label(svg, 800, 98, "+ 4", "big");
  label(svg, 800, 165, "immer gleich weit", "note");
  label(svg, 800, 193, "springen", "note");
  return svg;
}

function meeting() {
  const svg = baseSVG("0 0 900 230");
  const rc = rough.svg(svg);
  label(svg, 450, 30, "Wo treffen sie sich?", "note");
  const startX = 55;
  const endX = 845;
  const yBlue = 85;
  const yPink = 155;
  svg.appendChild(rc.line(startX, yBlue, endX, yBlue, options(1, BLUE, null, 4)));
  svg.appendChild(rc.line(startX, yPink, endX, yPink, options(2, PINK, null, 4)));
  for (let n = 0; n <= 24; n += 2) {
    const x = startX + n / 24 * (endX - startX);
    svg.appendChild(rc.line(x, yBlue - 8, x, yBlue + 8, options(10 + n, BLUE, null, 2)));
    svg.appendChild(rc.line(x, yPink - 8, x, yPink + 8, options(40 + n, PINK, null, 2)));
    if (n % 6 === 0) label(svg, x, yPink + 34, String(n), "number");
  }
  for (const n of [0, 6, 12, 18, 24]) {
    const x = startX + n / 24 * (endX - startX);
    svg.appendChild(rc.circle(x, yBlue, 17, options(80 + n, BLUE, "#ccecf8")));
  }
  for (const n of [0, 8, 16, 24]) {
    const x = startX + n / 24 * (endX - startX);
    svg.appendChild(rc.circle(x, yPink, 17, options(120 + n, PINK, "#f8c5d8")));
  }
  svg.appendChild(rc.line(endX, yBlue + 15, endX, yPink - 15, options(170, GOLD, null, 5)));
  label(svg, endX - 16, 128, "24", "big", "end");
  label(svg, 450, 220, "6er- und 8er-Sprünge treffen sich bei 24.", "small");
  return svg;
}

document.querySelectorAll("[data-figure]").forEach((host) => {
  if (host.dataset.figure === "jumps") host.appendChild(jumps());
  if (host.dataset.figure === "meeting") host.appendChild(meeting());
});
