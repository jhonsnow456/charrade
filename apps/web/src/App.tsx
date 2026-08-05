import { Route, Routes } from 'react-router-dom';
import Home from './pages/Home';
import Start from './pages/Start';
import Room from './pages/Room';

function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/start" element={<Start />} />
      <Route path="/room/:roomId" element={<Room />} />
    </Routes>
  );
}

export default App;
