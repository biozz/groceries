export default function() {
  return (
    <div className="container">
      <div className="row">
        <div className="col-sm-10 col-lg-6 offset-lg-3 offset-sm-1 gy-1">
          <div className="input-group">
            <button className="btn btn-success">+</button>
            <div class="input-group-text">
              <input className="form-check-input mt-0" type="checkbox" />
            </div>
            <div class="input-group-text">
              <input className="form-check-input mt-0" type="checkbox" />
            </div>
            <input className="form-control form-control-sm search-box" type="text" />
            <button className="btn btn-secondary">X</button>
          </div>
        </div>
      </div>
    </div>
  )
}
