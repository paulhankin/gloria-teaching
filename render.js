
const SVGNS="http://www.w3.org/2000/svg";
const PAL={pink:"#e8548c", blue:"#2f9fd0", yellow:"#e0a01e"};
const INK="#1f3550";

function mk(tag,attrs){const e=document.createElementNS(SVGNS,tag);
  for(const k in attrs) e.setAttribute(k,attrs[k]); return e;}

function txt(svg,x,y,s,cls){
  const t=mk("text",{x:x,y:y,"text-anchor":"middle",class:cls});
  t.textContent=s; svg.appendChild(t); return t;}

function build(spec,seed){
  const three = spec.type==="3";
  const W=400, H = three?400:(spec.box?272:248);
  const svg=mk("svg",{xmlns:SVGNS,viewBox:`0 0 ${W} ${H}`,preserveAspectRatio:"xMidYMid meet"});
  const st=mk("style",{});
  st.textContent=`
    .lbl{font:700 21px 'Caveat',cursive;fill:#1f3550}
    .z,.zo{font:700 31px 'Caveat',cursive;fill:#1f3550;
      paint-order:stroke;stroke:#fff;stroke-width:6px;stroke-linejoin:round}
    .q{font:700 35px 'Caveat',cursive;fill:#c62828;
      paint-order:stroke;stroke:#fff;stroke-width:6px;stroke-linejoin:round}
    .uni{font:400 17px 'Caveat',cursive;fill:#8a94a6}
  `;
  svg.appendChild(st);
  const rc=rough.svg(svg);
  const o=(s)=>({roughness:1.9,bowing:1.6,seed:seed+s,stroke:INK,strokeWidth:2.2});

  if(spec.box){
    svg.appendChild(rc.rectangle(6,6,W-12,H-12,
      {roughness:2.4,bowing:2.5,seed:seed+9,stroke:"#8a94a6",strokeWidth:1.8}));
  }

  const circles = three
    ? [[150,152,97],[252,152,97],[201,242,97]]
    : [[148,spec.box?142:132,96],[252,spec.box?142:132,96]];

  circles.forEach((c,i)=>{
    const col=PAL[spec.colors[i]];
    svg.appendChild(rc.circle(c[0],c[1],c[2]*2,{
      roughness:1.7,bowing:1.4,seed:seed+i*7+1,
      stroke:col, strokeWidth:3.2,
      fill:col, fillStyle:"hachure", fillWeight:1.0,
      hachureGap:11, hachureAngle:[41,-38,88][i]}));
  });

  // labels
  const L=spec.labels;
  if(three){
    txt(svg,78,34,L[0],"lbl"); txt(svg,326,34,L[1],"lbl"); txt(svg,201,382,L[2],"lbl");
  } else {
    const y=spec.box?38:28;
    txt(svg,92,y,L[0],"lbl"); txt(svg,312,y,L[1],"lbl");
  }

  const v=spec.values, P = three
    ? {A:[102,126],B:[300,126],C:[201,334],AB:[201,118],AC:[148,234],BC:[254,234],
       ABC:[201,192],out:[352,H-42]}
    : (spec.box
       ? {A:[104,150],AB:[200,150],B:[298,150],out:[352,H-32]}
       : {A:[104,140],AB:[200,140],B:[298,140]});
  for(const k in P){
    if(v[k]===undefined) continue;
    const cls = v[k]==="?" ? "q" : (k==="out"?"zo":"z");
    txt(svg,P[k][0],P[k][1],v[k],cls);
  }
  if(spec.box) txt(svg,0,0,"",""); // noop
  if(spec.box){
    const t=mk("text",{x:20,y:H-14,class:"uni"}); t.textContent=spec.box; svg.appendChild(t);
  }
  return svg;
}


for (const el of document.querySelectorAll('[data-diagram]')) {
  const key = el.getAttribute('data-diagram');
  const spec = DIAGRAMS[key];
  if (!spec) continue;
  el.appendChild(build(spec, key.charCodeAt(1) * 137 + 11));
}
