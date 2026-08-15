(() => {
  const shell = document.querySelector('[data-board]');
  if (!shell) return;
  const viewport = shell.querySelector('[data-board-viewport]');
  const world = shell.querySelector('[data-board-world]');
  const label = shell.querySelector('[data-zoom-label]');
  const connectButton = shell.querySelector('[data-connect-toggle]');
  const dialog = shell.querySelector('[data-edit-dialog]');
  const editForm = shell.querySelector('[data-edit-form]');
  const drawings = shell.querySelector('[data-board-drawings]');
  const undoButton = shell.querySelector('[data-undo-drawing]'), redoButton = shell.querySelector('[data-redo-drawing]');
  let scale = 1, panX = 0, panY = 0, active = null, connectMode = false, connectFrom = null, editingNode = null, boardMode = 'cursor', strokeColor = 'amber';
  const undoStack = [], redoStack = [];

  const strokeHex = {amber: '#ffbd78', violet: '#9a7cff', mint: '#69d8c3', rose: '#f283aa', white: '#f4f0fa'};
  const svgElement = name => document.createElementNS('http://www.w3.org/2000/svg', name);
  const pointAt = e => { const rect = viewport.getBoundingClientRect(); return {x: (e.clientX - rect.left - panX) / scale, y: (e.clientY - rect.top - panY) / scale}; };
  const pathData = points => points.map((point, index) => `${index ? 'L' : 'M'}${point.x.toFixed(1)},${point.y.toFixed(1)}`).join(' ');
  const renderStroke = (group, kind, points, color, width) => {
    const marker = color.startsWith('marker_'), actualColor = marker ? color.slice(7) : color;
    const shape = svgElement(kind === 'arrow' ? 'line' : 'path');
    shape.classList.add('board_stroke'); if (marker) shape.classList.add('board_marker_stroke'); shape.setAttribute('stroke', strokeHex[actualColor] || strokeHex.amber); shape.setAttribute('stroke-width', width); shape.setAttribute('fill', 'none'); shape.setAttribute('stroke-linecap', 'round'); shape.setAttribute('stroke-linejoin', 'round');
    if (marker) shape.setAttribute('stroke-opacity', '.32');
    if (kind === 'arrow') { const first = points[0], last = points[points.length - 1]; shape.setAttribute('x1', first.x); shape.setAttribute('y1', first.y); shape.setAttribute('x2', last.x); shape.setAttribute('y2', last.y); shape.setAttribute('marker-end', 'url(#board-arrow)'); }
    else shape.setAttribute('d', pathData(points));
    group.append(shape); return shape;
  };
  drawings.querySelectorAll('[data-stroke-id]').forEach(group => { try { renderStroke(group, group.dataset.strokeKind, JSON.parse(group.dataset.strokePoints), group.dataset.strokeColor, group.dataset.strokeWidth); } catch (_) {} });

  const drawEdges = () => shell.querySelectorAll('[data-edge-from]').forEach(line => {
    const a = shell.querySelector(`[data-node-id="${line.dataset.edgeFrom}"]`);
    const b = shell.querySelector(`[data-node-id="${line.dataset.edgeTo}"]`);
    if (!a || !b) return;
    line.setAttribute('x1', a.offsetLeft + a.offsetWidth / 2); line.setAttribute('y1', a.offsetTop + a.offsetHeight / 2);
    line.setAttribute('x2', b.offsetLeft + b.offsetWidth / 2); line.setAttribute('y2', b.offsetTop + b.offsetHeight / 2);
  });
  const render = () => { world.style.transform = `translate(${panX}px,${panY}px) scale(${scale})`; label.textContent = `${Math.round(scale * 100)}%`; drawEdges(); };
  const zoom = delta => { scale = Math.max(.35, Math.min(2, scale + delta)); render(); };
  const postJSON = (url, body) => fetch(url, {method: 'POST', headers: {'Content-Type': 'application/json'}, body: JSON.stringify(body)});
  const clearConnection = () => { connectFrom?.classList.remove('is_connect_source'); connectFrom = null; };
  const updateHistory = () => { undoButton.disabled = !undoStack.length; redoButton.disabled = !redoStack.length; };
  const saveStroke = async payload => {
    const response = await postJSON(`${location.pathname}/strokes`, payload); if (!response.ok) return null;
    const result = await response.json(), group = svgElement('g'); group.dataset.strokeId = result.id; group.dataset.strokeKind = payload.kind; group.dataset.strokePoints = JSON.stringify(payload.points); group.dataset.strokeColor = payload.color; group.dataset.strokeWidth = payload.width; drawings.append(group); renderStroke(group, payload.kind, payload.points, payload.color, payload.width); return {id: result.id, group, payload};
  };

  shell.querySelector('[data-collapse-creator]').addEventListener('click', () => shell.querySelector('[data-creator-panel]').classList.toggle('is_collapsed'));
  undoButton.addEventListener('click', async () => { const item = undoStack.pop(); if (!item) return; const response = await postJSON(`${location.pathname}/strokes/${item.id}/delete`, {}); if (response.ok) { item.group.remove(); redoStack.push(item); } else undoStack.push(item); updateHistory(); });
  redoButton.addEventListener('click', async () => { const previous = redoStack.pop(); if (!previous) return; const item = await saveStroke(previous.payload); if (item) undoStack.push(item); else redoStack.push(previous); updateHistory(); });

  shell.querySelectorAll('[data-board-mode]').forEach(button => button.addEventListener('click', () => {
    boardMode = button.dataset.boardMode; connectMode = false; clearConnection(); connectButton.classList.remove('is_active'); shell.classList.remove('is_connecting');
    shell.querySelectorAll('[data-board-mode]').forEach(item => item.classList.toggle('is_active', item === button));
    shell.dataset.mode = boardMode;
  }));
  shell.querySelectorAll('[data-stroke-color]').forEach(button => button.addEventListener('click', () => { strokeColor = button.dataset.strokeColor; shell.querySelectorAll('[data-stroke-color]').forEach(item => item.classList.toggle('is_active', item === button)); }));

  shell.querySelectorAll('[data-tool-tab]').forEach(tab => tab.addEventListener('click', () => {
    shell.querySelectorAll('[data-tool-tab]').forEach(item => item.classList.toggle('is_active', item === tab));
    shell.querySelectorAll('[data-tool-panel]').forEach(panel => panel.classList.toggle('is_active', panel.dataset.toolPanel === tab.dataset.toolTab));
  }));

  shell.querySelector('[data-zoom-in]').onclick = () => zoom(.1);
  shell.querySelector('[data-zoom-out]').onclick = () => zoom(-.1);
  shell.querySelector('[data-center-board]').onclick = () => { scale = 1; panX = 0; panY = 0; render(); };
  connectButton.onclick = () => { connectMode = !connectMode; boardMode = 'cursor'; shell.dataset.mode = 'cursor'; shell.querySelectorAll('[data-board-mode]').forEach(item => item.classList.toggle('is_active', item.dataset.boardMode === 'cursor')); connectButton.classList.toggle('is_active', connectMode); shell.classList.toggle('is_connecting', connectMode); clearConnection(); };
  viewport.addEventListener('wheel', e => { if (!e.ctrlKey) return; e.preventDefault(); zoom(e.deltaY < 0 ? .1 : -.1); }, {passive: false});

  viewport.addEventListener('click', async e => {
    const edit = e.target.closest('[data-edit-node]');
    if (edit) {
      editingNode = edit.closest('[data-node-id]');
      editForm.elements.title.value = editingNode.dataset.nodeTitle;
      editForm.elements.content.value = editingNode.dataset.nodeContent;
      editForm.elements.color.value = editingNode.dataset.nodeColor;
      editForm.elements.text_color.value = editingNode.dataset.nodeTextColor || 'white';
      dialog.showModal(); return;
    }
    if (boardMode === 'eraser') {
      const stroke = e.target.closest('[data-stroke-id]');
      if (!stroke) return;
      const response = await postJSON(`${location.pathname}/strokes/${stroke.dataset.strokeId}/delete`, {});
      if (response.ok) stroke.remove();
      return;
    }
    if (!connectMode || e.target.closest('button,form,input,textarea,select,audio')) return;
    const node = e.target.closest('[data-node-id]');
    if (!node) return;
    if (!connectFrom) { connectFrom = node; node.classList.add('is_connect_source'); return; }
    if (connectFrom === node) { clearConnection(); return; }
    const body = new URLSearchParams({from_node_id: connectFrom.dataset.nodeId, to_node_id: node.dataset.nodeId});
    const response = await fetch(`${location.pathname}/edges`, {method: 'POST', body});
    if (response.ok) location.reload();
  });

  editForm.addEventListener('submit', async e => {
    if (e.submitter?.value === 'cancel' || !editingNode) return;
    e.preventDefault();
    const response = await postJSON(`${location.pathname}/nodes/${editingNode.dataset.nodeId}/edit`, {title: editForm.elements.title.value, content: editForm.elements.content.value, color: editForm.elements.color.value, textColor: editForm.elements.text_color.value});
    if (response.ok) location.reload();
  });

  viewport.addEventListener('pointerdown', e => {
    if (connectMode || (e.target.closest('button,form,input,audio') && !e.target.closest('[data-resize-node]'))) return;
    if (boardMode === 'pen' || boardMode === 'marker' || boardMode === 'arrow') {
      const point = pointAt(e), group = svgElement('g'); drawings.append(group);
      const points = [point, point], storedKind = boardMode === 'marker' ? 'pen' : boardMode, storedColor = boardMode === 'marker' ? `marker_${strokeColor}` : strokeColor, width = boardMode === 'marker' ? 18 : boardMode === 'arrow' ? 3 : 3.5;
      const shape = renderStroke(group, storedKind, points, storedColor, width);
      active = {kind: 'draw', drawKind: storedKind, group, shape, points, color: storedColor, width};
      viewport.setPointerCapture(e.pointerId); return;
    }
    if (boardMode === 'eraser') return;
    const node = e.target.closest('[data-node-id]');
    if (node) {
      if (e.target.closest('[data-resize-node]')) active = {kind: 'resize', node, startX: e.clientX, startY: e.clientY, width: node.offsetWidth, height: node.offsetHeight};
      else if (e.target.closest('.board_node_handle')) active = {kind: 'node', node, startX: e.clientX, startY: e.clientY, x: node.offsetLeft, y: node.offsetTop};
      else return;
      node.setPointerCapture(e.pointerId);
    } else { active = {kind: 'pan', startX: e.clientX, startY: e.clientY, x: panX, y: panY}; viewport.setPointerCapture(e.pointerId); }
  });
  viewport.addEventListener('pointermove', e => {
    if (!active) return;
    if (active.kind === 'pan') { panX = active.x + e.clientX - active.startX; panY = active.y + e.clientY - active.startY; render(); return; }
    if (active.kind === 'draw') { const point = pointAt(e); if (active.drawKind === 'arrow') active.points[1] = point; else active.points.push(point); if (active.drawKind === 'arrow') { active.shape.setAttribute('x2', point.x); active.shape.setAttribute('y2', point.y); } else active.shape.setAttribute('d', pathData(active.points)); return; }
    if (active.kind === 'resize') { active.node.style.width = `${Math.max(180, active.width + (e.clientX - active.startX) / scale)}px`; active.node.style.height = `${Math.max(100, active.height + (e.clientY - active.startY) / scale)}px`; drawEdges(); return; }
    active.node.style.left = `${active.x + (e.clientX - active.startX) / scale}px`; active.node.style.top = `${active.y + (e.clientY - active.startY) / scale}px`; drawEdges();
  });
  viewport.addEventListener('pointerup', async () => {
    if (!active) return;
    const item = active; active = null;
    if (item.kind === 'draw') {
      const payload = {kind: item.drawKind, points: item.points, color: item.color, width: item.width};
      const response = await postJSON(`${location.pathname}/strokes`, payload);
      if (!response.ok) { item.group.remove(); return; }
      const result = await response.json(); item.group.dataset.strokeId = result.id; item.group.dataset.strokeKind = item.drawKind; item.group.dataset.strokePoints = JSON.stringify(item.points); item.group.dataset.strokeColor = item.color; item.group.dataset.strokeWidth = item.width;
      undoStack.push({id: result.id, group: item.group, payload}); redoStack.length = 0; updateHistory();
      return;
    }
    const id = item.node?.dataset.nodeId;
    if (item.kind === 'node') await postJSON(`${location.pathname}/nodes/${id}/move`, {x: item.node.offsetLeft, y: item.node.offsetTop});
    if (item.kind === 'resize') await postJSON(`${location.pathname}/nodes/${id}/resize`, {width: item.node.offsetWidth, height: item.node.offsetHeight});
  });
  render();
})();
