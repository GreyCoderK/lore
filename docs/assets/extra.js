// Force Mermaid readability after rendering
document.addEventListener('DOMContentLoaded', function() {
  // Wait for mermaid to render
  setTimeout(fixMermaid, 1500);
  setTimeout(fixMermaid, 3000);

  // Also observe for dynamic theme changes
  const observer = new MutationObserver(function() {
    setTimeout(fixMermaid, 500);
  });
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-md-color-scheme']
  });
});

function fixMermaid() {
  const isDark = document.documentElement.getAttribute('data-md-color-scheme') === 'slate';
  const textColor = isDark ? '#cdd6f4' : '#1a1a1a';
  const nodeFill = isDark ? 'rgba(129,140,248,0.12)' : 'rgba(79,70,229,0.1)';
  const nodeStroke = isDark ? '#818cf8' : '#4f46e5';
  const edgeStroke = isDark ? '#818cf8' : '#4f46e5';
  const bgColor = isDark ? '#1e1e2e' : '#ffffff';

  document.querySelectorAll('.mermaid').forEach(function(diagram) {
    diagram.style.background = bgColor;
    diagram.style.padding = '1.5rem';
    diagram.style.borderRadius = '8px';

    // Fix all text elements
    diagram.querySelectorAll('text, tspan').forEach(function(el) {
      el.style.fill = textColor;
      el.style.fontSize = '14px';
      el.style.fontFamily = 'Instrument Sans, system-ui, sans-serif';
      el.setAttribute('fill', textColor);
      el.setAttribute('font-size', '14');
    });

    // Fix foreignObject content (nodeLabels)
    diagram.querySelectorAll('foreignObject div, foreignObject span, foreignObject p, .nodeLabel, .label').forEach(function(el) {
      el.style.color = textColor;
      el.style.fontSize = '14px';
      el.style.fontFamily = 'Instrument Sans, system-ui, sans-serif';
      el.style.lineHeight = '1.4';
    });

    // Fix edgeLabels
    diagram.querySelectorAll('.edgeLabel, .edgeLabel rect, .edgeLabel span').forEach(function(el) {
      el.style.color = textColor;
      el.style.fill = textColor;
      if (el.tagName === 'rect' || el.classList.contains('edgeLabel')) {
        el.style.fill = bgColor;
        el.setAttribute('fill', bgColor);
      }
    });

    // Fix node shapes
    diagram.querySelectorAll('.node rect, .node circle, .node polygon, .node path, .basic.label-container').forEach(function(el) {
      el.style.fill = nodeFill;
      el.style.stroke = nodeStroke;
      el.style.strokeWidth = '1.5px';
      el.setAttribute('fill', nodeFill);
      el.setAttribute('stroke', nodeStroke);
    });

    // Fix edges/arrows
    diagram.querySelectorAll('.edgePath path, .flowchart-link').forEach(function(el) {
      el.style.stroke = edgeStroke;
      el.style.strokeWidth = '1.5px';
      el.setAttribute('stroke', edgeStroke);
    });

    // Fix arrowheads
    diagram.querySelectorAll('marker path').forEach(function(el) {
      el.style.fill = edgeStroke;
      el.setAttribute('fill', edgeStroke);
    });
  });
}
