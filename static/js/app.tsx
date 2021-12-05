import 'bootstrap/dist/css/bootstrap.min.css';
import '../css/style.css';
import { render } from "solid-js/web";
import Bar from './bar.jsx';
import Items from './items.jsx';


function App() {
  return (
    <div className="container">
      <div className="row">
        <div className="col-sm-10 col-lg-6 offset-lg-3 offset-sm-1 gy-4">
          <div className="position-fixed fixed-top bg-white">
            <Bar />
          </div>
          <Items />
        </div>
      </div>
    </div>
  )
}
render(() => <App />, document.getElementById("app"));
