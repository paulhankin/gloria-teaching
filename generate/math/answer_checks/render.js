const SVG_NS = "http://www.w3.org/2000/svg";
const INK = "#1f3550";
const COLORS = ["#e8548c", "#2f9fd0", "#5eb85e", "#e0a01e", "#ef7a3d"];

function node(tag, attrs = {}) {
  const n = document.createElementNS(SVG_NS, tag);
  for (const [key, value] of Object.entries(attrs)) n.setAttribute(key, value);
  return n;
}

function label(svg, x, y, value, cls = "label") {
  const t = node("text", {x, y, class: cls, "text-anchor": "middle"});
  t.textContent = value;
  svg.appendChild(t);
}

function options(seed, stroke, fill) {
  const out = {seed, roughness: 1.2, bowing: 1.1, stroke: stroke || INK, strokeWidth: 3};
  if (fill) Object.assign(out, {fill, fillStyle: "solid"});
  return out;
}

function baseSVG() {
  const svg = node("svg", {xmlns: SVG_NS, viewBox: "0 0 900 190", preserveAspectRatio: "xMidYMid meet"});
  const style = node("style");
  style.textContent = ".label{font:700 25px 'Caveat',cursive;fill:#1f3550}.big{font:700 34px 'Caveat',cursive;fill:#e8548c}";
  svg.appendChild(style);
  return svg;
}

function drawPrice(svg, rc, spec, seed) {
  const xs = [260, 640];
  xs.forEach((x, i) => {
    const color = COLORS[i];
    if (spec.kind === "fruit") {
      const fruitX = [x - 38, x, x + 38];
      fruitX.forEach((fx, k) => {
        const pear = i === 1;
        if (pear) {
          svg.appendChild(rc.path(`M ${fx} 52 C ${fx-18} 70 ${fx-14} 88 ${fx-27} 104 C ${fx-45} 133 ${fx-24} 157 ${fx} 157 C ${fx+24} 157 ${fx+45} 133 ${fx+27} 104 C ${fx+14} 88 ${fx+18} 70 ${fx} 52 Z`, options(seed + i * 20 + k, "#668126", "#dbe97a")));
        } else {
          svg.appendChild(rc.circle(fx, 112, 78, options(seed + i * 20 + k, "#ae3544", "#f4838f")));
        }
        svg.appendChild(rc.line(fx, 70, fx + 5, 49, options(seed + 80 + i * 5 + k, "#76513a")));
      });
    } else {
      const widths = i === 0 ? [100, 88, 76] : [78, 72];
      widths.forEach((w, k) => svg.appendChild(rc.rectangle(x - w / 2, 128 - k * 36, w, 31,
        options(seed + i * 20 + k, color, i === 0 ? "#bfeaf7" : "#ffe08a"))));
    }
    label(svg, x, 184, spec.labels[i]);
    label(svg, x, 37, "? Fr.", "big");
  });
  svg.appendChild(rc.path("M 420 104 Q 450 75 480 104", options(seed + 90, "#8a94a6")));
  label(svg, 450, 135, "+", "big");
}

function drawCards(svg, rc, spec, seed) {
  const step = 160;
  const start = 130;
  spec.labels.forEach((name, i) => {
    const x = start + i * step;
    const color = COLORS[i];
    if (spec.kind === "house") {
      svg.appendChild(rc.polygon([[x - 50, 78], [x, 35], [x + 50, 78]], options(seed + i * 4, color, "#fff")));
      svg.appendChild(rc.rectangle(x - 43, 78, 86, 61, options(seed + i * 4 + 1, color, "#f7fbff")));
      svg.appendChild(rc.circle(x, 101, 25, options(seed + i * 4 + 2, color, "#fff")));
    } else if (spec.kind === "race") {
      svg.appendChild(rc.circle(x, 79, 34, options(seed + i * 4, color, "#fff")));
      svg.appendChild(rc.line(x, 96, x, 130, options(seed + i * 4 + 1, color)));
      svg.appendChild(rc.line(x, 107, x - 24, 119, options(seed + i * 4 + 2, color)));
      svg.appendChild(rc.line(x, 107, x + 24, 119, options(seed + i * 4 + 3, color)));
      svg.appendChild(rc.line(x, 130, x - 20, 151, options(seed + i * 4 + 4, color)));
      svg.appendChild(rc.line(x, 130, x + 20, 151, options(seed + i * 4 + 5, color)));
    } else {
      svg.appendChild(rc.rectangle(x - 50, 48, 100, 105, options(seed + i * 4, color, color)));
      svg.appendChild(rc.rectangle(x - 35, 64, 70, 70, options(seed + i * 4 + 1, "#ffffff", "#ffffff")));
      svg.appendChild(rc.rectangle(x - 22, 76, 44, 44, options(seed + i * 4 + 2, color, "#fff")));
    }
    label(svg, x, 181, name);
    label(svg, x, 29, "?", "big");
  });
  svg.appendChild(rc.line(45, 160, 855, 160, options(seed + 99, "#cbd5df")));
}

document.querySelectorAll("[data-picture]").forEach((host, index) => {
  const spec = CHECK_PICTURES[host.dataset.picture];
  if (!spec) return;
  const svg = baseSVG();
  const rc = rough.svg(svg);
  if (spec.kind === "fruit" || spec.kind === "books") drawPrice(svg, rc, spec, 100 + index * 100);
  else drawCards(svg, rc, spec, 100 + index * 100);
  host.appendChild(svg);
});
