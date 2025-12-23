import { BrowserRouter, Routes, Route } from 'react-router-dom';
import Layout from './components/Layout';
import Dashboard from './pages/Dashboard';
import Search from './pages/Search';
import Library from './pages/Library';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Layout />}>
          <Route index element={<Dashboard />} />
          <Route path="search" element={<Search />} />
          <Route path="library" element={<Library />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
