import { RequestList } from "./components/RequestList";
import "./App.css";

function App() {
  return (
    <div className="app">
      <header>
        <h1>HTTP Echo Server</h1>
      </header>
      <main>
        <RequestList />
      </main>
    </div>
  );
}

export default App;
