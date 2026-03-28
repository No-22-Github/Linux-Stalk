import { Routes, Route } from 'react-router-dom';
import { Layout } from '@/components/Layout';
import { Overview } from '@/pages/Overview';
import { Devices } from '@/pages/Devices';
import { DeviceDetail } from '@/pages/DeviceDetail';

function App() {
  return (
    <Layout>
      <Routes>
        <Route path="/" element={<Overview />} />
        <Route path="/devices" element={<Devices />} />
        <Route path="/devices/:deviceId" element={<DeviceDetail />} />
      </Routes>
    </Layout>
  );
}

export default App;
