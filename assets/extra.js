// Force Mermaid to always render in dark theme (like code blocks)
document.addEventListener('DOMContentLoaded', function() {
  setTimeout(fixMermaid, 1000);
  setTimeout(fixMermaid, 2000);
  setTimeout(fixMermaid, 4000);

  new MutationObserver(function() {
    setTimeout(fixMermaid, 500);
  }).observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-md-color-scheme']
  });
});

function fixMermaid() {
  var bg = '#1e1e2e';
  var text = '#cdd6f4';
  var nodeFill = 'rgba(129,140,248,0.12)';
  var nodeStroke = '#818cf8';

  document.querySelectorAll('.mermaid').forEach(function(d) {
    d.style.cssText = 'background:#1e1e2e!important;padding:1.5rem;border-radius:8px;border:1px solid #313244!important';

    d.querySelectorAll('text, tspan').forEach(function(el) {
      el.setAttribute('fill', text);
      el.setAttribute('font-size', '14');
      el.style.cssText = 'fill:' + text + '!important;font-size:14px!important;font-family:Instrument Sans,system-ui,sans-serif!important';
    });

    d.querySelectorAll('foreignObject div, foreignObject span, foreignObject p, .nodeLabel, .label, .edgeLabel span').forEach(function(el) {
      el.style.cssText = 'color:' + text + '!important;font-size:14px!important;font-family:Instrument Sans,system-ui,sans-serif!important;line-height:1.4!important';
    });

    d.querySelectorAll('.edgeLabel, .edgeLabel rect').forEach(function(el) {
      el.setAttribute('fill', bg);
      el.style.fill = bg;
    });

    d.querySelectorAll('.node rect, .node circle, .node polygon, .node path, .basic.label-container, .flowchart-label rect').forEach(function(el) {
      el.setAttribute('fill', nodeFill);
      el.setAttribute('stroke', nodeStroke);
      el.setAttribute('stroke-width', '1.5');
    });

    d.querySelectorAll('.edgePath path, .flowchart-link').forEach(function(el) {
      el.setAttribute('stroke', nodeStroke);
      el.setAttribute('stroke-width', '1.5');
    });

    d.querySelectorAll('marker path').forEach(function(el) {
      el.setAttribute('fill', nodeStroke);
    });
  });
}
