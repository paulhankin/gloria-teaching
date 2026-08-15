const SVG_NS = "http://www.w3.org/2000/svg";
const CLOCK_INK = "#1f3550";
const CLOCK_PINK = "#e8548c";

function clockNode(tag, attrs = {}) {
  const element = document.createElementNS(SVG_NS, tag);
  for (const [key, value] of Object.entries(attrs)) element.setAttribute(key, value);
  return element;
}

function clockOptions(seed, stroke = CLOCK_INK, width = 2.5) {
  return {seed, roughness: 0.75, bowing: 0.7, stroke, strokeWidth: width};
}

function pointOnClock(cx, cy, radius, units, total) {
  const angle = units / total * Math.PI * 2 - Math.PI / 2;
  return {x: cx + Math.cos(angle) * radius, y: cy + Math.sin(angle) * radius};
}

function drawClock(spec, blank, seed) {
  const svg = clockNode("svg", {xmlns: SVG_NS, viewBox: "0 0 220 220", preserveAspectRatio: "xMidYMid meet"});
  const style = clockNode("style");
  style.textContent = ".clock-number{font:700 19px 'Patrick Hand',cursive;fill:#1f3550}";
  svg.appendChild(style);
  const rc = rough.svg(svg);
  const cx = 110, cy = 110;

  svg.appendChild(rc.circle(cx, cy, 190, {...clockOptions(seed), fill: "#fffdf8", fillStyle: "solid"}));
  for (let i = 0; i < 60; i++) {
    const major = i % 5 === 0;
    const outer = pointOnClock(cx, cy, 84, i, 60);
    const inner = pointOnClock(cx, cy, major ? 76 : 80, i, 60);
    const tick = clockNode("line", {
      x1: inner.x, y1: inner.y, x2: outer.x, y2: outer.y,
      stroke: major ? CLOCK_INK : "#aeb9c8", "stroke-width": major ? 2.2 : 1.1,
      "stroke-linecap": "round"
    });
    svg.appendChild(tick);
  }
  for (let number = 1; number <= 12; number++) {
    const p = pointOnClock(cx, cy + 6, 61, number, 12);
    const label = clockNode("text", {x: p.x, y: p.y, class: "clock-number", "text-anchor": "middle", "dominant-baseline": "middle"});
    label.textContent = number;
    svg.appendChild(label);
  }

  if (!blank) {
    const minutePoint = pointOnClock(cx, cy, 67, spec.minute, 60);
    const hourUnits = (spec.hour % 12) + spec.minute / 60;
    const hourPoint = pointOnClock(cx, cy, 45, hourUnits, 12);
    svg.appendChild(rc.line(cx, cy, hourPoint.x, hourPoint.y, clockOptions(seed + 1, CLOCK_INK, 6)));
    svg.appendChild(rc.line(cx, cy, minutePoint.x, minutePoint.y, clockOptions(seed + 2, CLOCK_PINK, 4)));
  }
  svg.appendChild(rc.circle(cx, cy, 10, {...clockOptions(seed + 3, CLOCK_INK, 2), fill: CLOCK_INK, fillStyle: "solid"}));
  return svg;
}

document.querySelectorAll("[data-clock]").forEach((host, index) => {
  const spec = CLOCKS[host.dataset.clock];
  if (spec) host.appendChild(drawClock(spec, host.dataset.blank === "true", 100 + index * 7));
});
