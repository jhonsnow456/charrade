import { Route, Routes } from 'react-router-dom';
import Home from './pages/Home';
import Start from './pages/Start';
import Room from './pages/Room';
import Score from './pages/Score';

function App() {
  return (
    <Routes>
      <Route path="/" element={<Home />} />
      <Route path="/start" element={<Start />} />
      <Route path="/room/:roomId" element={<Room />} />
      <Route path="/room/:roomId/score" element={<Score />} />
    </Routes>
  );
}

export default App;
