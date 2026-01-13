import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { ThemeProvider } from './contexts/ThemeContext'
import Layout from './components/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import Dashboard from './pages/Dashboard'
import Incidents from './pages/Incidents'
import Chats from './pages/Chats'
import Analytics from './pages/Analytics'
import ModelTester from './pages/ModelTester'
import Settings from './pages/Settings'
import CollectorSettings from './pages/CollectorSettings'
import PrivateRoute from './components/PrivateRoute'

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <Router>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />

          <Route element={<PrivateRoute />}>
            <Route element={<Layout />}>
              <Route path="/" element={<Navigate to="/dashboard" replace />} />
              <Route path="/dashboard" element={<Dashboard />} />
              <Route path="/incidents" element={<Incidents />} />
              <Route path="/chats" element={<Chats />} />
              <Route path="/analytics" element={<Analytics />} />
              <Route path="/model-tester" element={<ModelTester />} />
              <Route path="/collector" element={<CollectorSettings />} />
              <Route path="/settings" element={<Settings />} />
            </Route>
          </Route>
        </Routes>
        </Router>
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
