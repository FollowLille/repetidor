(() => {
  const shell = document.querySelector('[data-board]');
  if (!shell) return;
  const viewport = shell.querySelector('[data-board-viewport]');
  const world = shell.querySelector('[data-board-world]');
  const label = shell.querySelector('[data-zoom-label]');
  let scale = 1, panX = 0, panY = 0, active = null;
  const render = () => { world.style.transform = `translate(${panX}px,${panY}px) scale(${scale})`; label.textContent = `${Math.round(scale*100)}%`; drawEdges(); };
  const drawEdges = () => shell.querySelectorAll('[data-edge-from]').forEach(line => { const a=shell.querySelector(`[data-node-id="${line.dataset.edgeFrom}"]`),b=shell.querySelector(`[data-node-id="${line.dataset.edgeTo}"]`);if(!a||!b)return;line.setAttribute('x1',a.offsetLeft+a.offsetWidth/2);line.setAttribute('y1',a.offsetTop+a.offsetHeight/2);line.setAttribute('x2',b.offsetLeft+b.offsetWidth/2);line.setAttribute('y2',b.offsetTop+b.offsetHeight/2); });
  const zoom = delta => { scale=Math.max(.35,Math.min(2,scale+delta));render(); };
  shell.querySelector('[data-zoom-in]').onclick=()=>zoom(.1); shell.querySelector('[data-zoom-out]').onclick=()=>zoom(-.1); shell.querySelector('[data-center-board]').onclick=()=>{scale=1;panX=0;panY=0;render();};
  viewport.addEventListener('wheel',e=>{if(!e.ctrlKey)return;e.preventDefault();zoom(e.deltaY<0?.1:-.1)},{passive:false});
  viewport.addEventListener('pointerdown',e=>{const node=e.target.closest('[data-node-id]');if(node){if(!e.target.closest('.board_node_handle'))return;active={kind:'node',node,startX:e.clientX,startY:e.clientY,x:node.offsetLeft,y:node.offsetTop};node.setPointerCapture(e.pointerId);}else{active={kind:'pan',startX:e.clientX,startY:e.clientY,x:panX,y:panY};viewport.setPointerCapture(e.pointerId);}});
  viewport.addEventListener('pointermove',e=>{if(!active)return;if(active.kind==='pan'){panX=active.x+e.clientX-active.startX;panY=active.y+e.clientY-active.startY;render();return;}active.node.style.left=`${active.x+(e.clientX-active.startX)/scale}px`;active.node.style.top=`${active.y+(e.clientY-active.startY)/scale}px`;drawEdges();});
  viewport.addEventListener('pointerup',async()=>{if(!active)return;const item=active;active=null;if(item.kind!=='node')return;const id=item.node.dataset.nodeId;await fetch(`${location.pathname}/nodes/${id}/move`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({x:item.node.offsetLeft,y:item.node.offsetTop})});});
  render();
})();
