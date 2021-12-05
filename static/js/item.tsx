export default function(props) {
  return (
    <li className="list-group-item list-group-item-action"
      classList={{'list-group-item-light': props.item.is_checked}}
    >
      <input type="checkbox" className="form-check-input float-start me-1 h5" checked={props.item.is_checked}/>
      <div className="overflow-auto item-name">
        <span>{props.item.name}</span>
      </div>
      <div class="item-actions">
        <button className="btn btn-link link-secondary">✏️</button>
      </div>
    </li>
  )
}

