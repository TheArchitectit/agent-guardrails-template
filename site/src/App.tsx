import { Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import { Home, Story, How, Install, Compatibility, Policies, Roadmap, Security, Community } from './pages/Pages'
export default function App() {
  return (<Routes><Route element={<Layout />}>
    <Route index element={<Home />} /><Route path="story" element={<Story />} />
    <Route path="how-it-works" element={<How />} /><Route path="install" element={<Install />} />
    <Route path="compatibility" element={<Compatibility />} /><Route path="policies" element={<Policies />} />
    <Route path="roadmap" element={<Roadmap />} /><Route path="security" element={<Security />} />
    <Route path="community" element={<Community />} /><Route path="*" element={<Story />} />
  </Route></Routes>)
}
