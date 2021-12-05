import { createSignal, For } from 'solid-js';
import Item from './item.tsx';

export default function() {
  const [items, setItems] = createSignal([
    { "id": 1, "category": "test", "name": "asdf1", "is_checked": false },
    { "id": 1, "category": "test", "name": "asdf2", "is_checked": true },
    { "id": 1, "category": "test2", "name": "asdf3", "is_checked": false },
  ]);

  const showCategory = function(i) {
    if (i == 0) {
      return true
    }
    if (i > 0 && i < items().length) {
      if (items()[i].category !== items()[i - 1].category) {
        return true
      }
    }
    return false
  }

  return (
    <div className="mt-4">
      <ul className="list-group shadow-sm mt-1" >
        <For each={items()}>{(item, i) =>
          <>
            <Show when={showCategory(i())}><span>{item.category}</span></Show>
            <Item item={item} />
          </>
        }</For>
      </ul>
    </div>
  )
}
