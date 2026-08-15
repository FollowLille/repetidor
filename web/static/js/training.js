document.addEventListener('DOMContentLoaded', function () {
  const form = document.querySelector('[data-answer-mode="build"], [data-answer-mode="anagram"]');
  if (!form) return;

  const output = form.querySelector('[data-build-reply]');
  const answer = form.querySelector('.build_answer');
  const tiles = Array.from(form.querySelectorAll('.letter_tile'));
  const backspace = form.querySelector('[data-build-backspace]');
  const clear = form.querySelector('[data-build-clear]');
  let value = [];

  function render() {
    output.value = value.join('');
    answer.innerHTML = '';
    value.forEach(function (letter) {
      const cell = document.createElement('span');
      cell.className = 'answer_cell filled';
      cell.textContent = letter;
      answer.appendChild(cell);
    });
    for (let i = value.length; i < tiles.length; i += 1) {
      const cell = document.createElement('span');
      cell.className = 'answer_cell';
      answer.appendChild(cell);
    }
  }

  tiles.forEach(function (tile) {
    tile.addEventListener('click', function () {
      if (tile.disabled) return;
      value.push(tile.dataset.letter || tile.textContent);
      tile.disabled = true;
      render();
    });
  });

  backspace && backspace.addEventListener('click', function () {
    const removed = value.pop();
    if (removed) {
      const tile = tiles.slice().reverse().find(function (item) {
        return item.disabled && (item.dataset.letter || item.textContent) === removed;
      });
      if (tile) tile.disabled = false;
    }
    render();
  });

  clear && clear.addEventListener('click', function () {
    value = [];
    tiles.forEach(function (tile) { tile.disabled = false; });
    render();
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Backspace') {
      event.preventDefault();
      backspace && backspace.click();
      return;
    }
    const key = event.key.toLowerCase();
    const tile = tiles.find(function (item) {
      return !item.disabled && (item.dataset.letter || item.textContent).toLowerCase() === key;
    });
    if (tile) tile.click();
  });

  render();
});
