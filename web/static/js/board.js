(() => {
  const shell = document.querySelector('[data-board]');
  if (!shell) return;
  const viewport = shell.querySelector('[data-board-viewport]');
  const world = shell.querySelector('[data-board-world]');
  const label = shell.querySelector('[data-zoom-label]');
  const connectButton = shell.querySelector('[data-connect-toggle]');
  const dialog = shell.querySelector('[data-edit-dialog]');
  const editForm = shell.querySelector('[data-edit-form]');
  let scale = 1, panX = 0, panY = 0, active = null, connectMode = false, connectFrom = null, editingNode = null;

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

  shell.querySelectorAll('[data-tool-tab]').forEach(tab => tab.addEventListener('click', () => {
    shell.querySelectorAll('[data-tool-tab]').forEach(item => item.classList.toggle('is_active', item === tab));
    shell.querySelectorAll('[data-tool-panel]').forEach(panel => panel.classList.toggle('is_active', panel.dataset.toolPanel === tab.dataset.toolTab));
  }));

  shell.querySelector('[data-zoom-in]').onclick = () => zoom(.1);
  shell.querySelector('[data-zoom-out]').onclick = () => zoom(-.1);
  shell.querySelector('[data-center-board]').onclick = () => { scale = 1; panX = 0; panY = 0; render(); };
  connectButton.onclick = () => { connectMode = !connectMode; connectButton.classList.toggle('is_active', connectMode); shell.classList.toggle('is_connecting', connectMode); clearConnection(); };
  viewport.addEventListener('wheel', e => { if (!e.ctrlKey) return; e.preventDefault(); zoom(e.deltaY < 0 ? .1 : -.1); }, {passive: false});

  viewport.addEventListener('click', async e => {
    const edit = e.target.closest('[data-edit-node]');
    if (edit) {
      editingNode = edit.closest('[data-node-id]');
      editForm.elements.title.value = editingNode.dataset.nodeTitle;
      editForm.elements.content.value = editingNode.dataset.nodeContent;
      editForm.elements.color.value = editingNode.dataset.nodeColor;
      dialog.showModal(); return;
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
    const response = await postJSON(`${location.pathname}/nodes/${editingNode.dataset.nodeId}/edit`, {title: editForm.elements.title.value, content: editForm.elements.content.value, color: editForm.elements.color.value});
    if (response.ok) location.reload();
  });

  viewport.addEventListener('pointerdown', e => {
    if (connectMode || (e.target.closest('button,form,input,audio') && !e.target.closest('[data-resize-node]'))) return;
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
    if (active.kind === 'resize') { active.node.style.width = `${Math.max(180, active.width + (e.clientX - active.startX) / scale)}px`; active.node.style.height = `${Math.max(100, active.height + (e.clientY - active.startY) / scale)}px`; drawEdges(); return; }
    active.node.style.left = `${active.x + (e.clientX - active.startX) / scale}px`; active.node.style.top = `${active.y + (e.clientY - active.startY) / scale}px`; drawEdges();
  });
  viewport.addEventListener('pointerup', async () => {
    if (!active) return;
    const item = active; active = null;
    const id = item.node?.dataset.nodeId;
    if (item.kind === 'node') await postJSON(`${location.pathname}/nodes/${id}/move`, {x: item.node.offsetLeft, y: item.node.offsetTop});
    if (item.kind === 'resize') await postJSON(`${location.pathname}/nodes/${id}/resize`, {width: item.node.offsetWidth, height: item.node.offsetHeight});
  });
  render();
})();
