const SVG_NS = "http://www.w3.org/2000/svg";
const LINE_INK = "#1f3550";

function numberLineNode(tag, attrs = {}) {
  const element = document.createElementNS(SVG_NS, tag);
  for (const [key, value] of Object.entries(attrs)) element.setAttribute(key, value);
  return element;
}

function drawNumberLine(spec) {
  const width = 1240;
  const height = 145;
  const margin = 34;
  const axisY = 63;
  const step = (width - 2 * margin) / (spec.max - spec.min);
  const xFor = value => margin + (value - spec.min) * step;
  const svg = numberLineNode("svg", {
    xmlns: SVG_NS,
    viewBox: `0 0 ${width} ${height}`,
    preserveAspectRatio: "xMidYMid meet"
  });
  const style = numberLineNode("style");
  style.textContent = `
    .number-label { font: 700 25px 'Patrick Hand', cursive; fill: #1f3550; }
    .negative-label { fill: #e8548c; }
    .zero-label { font-size: 31px; fill: #2e8b57; }
  `;
  svg.appendChild(style);

  const rc = rough.svg(svg);
  svg.appendChild(rc.line(margin, axisY, width - margin, axisY, {
    seed: 28, roughness: 0.8, bowing: 0.45, stroke: LINE_INK, strokeWidth: 3
  }));

  svg.appendChild(rc.path(`M ${margin + 10} ${axisY - 9} L ${margin} ${axisY} L ${margin + 10} ${axisY + 9}`, {
    seed: 29, roughness: 0.7, stroke: LINE_INK, strokeWidth: 3
  }));
  svg.appendChild(rc.path(`M ${width - margin - 10} ${axisY - 9} L ${width - margin} ${axisY} L ${width - margin - 10} ${axisY + 9}`, {
    seed: 30, roughness: 0.7, stroke: LINE_INK, strokeWidth: 3
  }));

  for (let value = spec.min; value <= spec.max; value++) {
    const x = xFor(value);
    const isZero = value === 0;
    svg.appendChild(rc.line(x, axisY - (isZero ? 17 : 11), x, axisY + (isZero ? 17 : 11), {
      seed: 100 + value - spec.min,
      roughness: 0.65,
      bowing: 0.35,
      stroke: isZero ? "#2e8b57" : LINE_INK,
      strokeWidth: isZero ? 4.5 : 2.3
    }));
    const label = numberLineNode("text", {
      x,
      y: 112,
      "text-anchor": "middle",
      class: `number-label ${value < 0 ? "negative-label" : ""} ${isZero ? "zero-label" : ""}`
    });
    label.textContent = value;
    svg.appendChild(label);
  }
  return svg;
}

document.querySelectorAll("[data-number-line]").forEach(host => {
  host.appendChild(drawNumberLine(NUMBER_LINE));
});
