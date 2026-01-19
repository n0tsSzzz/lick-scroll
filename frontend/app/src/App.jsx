import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { useState, useEffect } from 'react';
import { authService } from './services/authService';
import { websocketService } from './services/websocketService';
import Login from './pages/Login';
import Register from './pages/Register';
import Feed from './pages/Feed';
import Profile from './pages/Profile';
import UserProfile from './pages/UserProfile';
import Post from './pages/Post';
import CreatePost from './pages/CreatePost';
import Analytics from './pages/Analytics';
import Notifications from './pages/Notifications';
import Layout from './components/Layout';
import './styles/index.css';

function ProtectedRoute({ children }) {
  const location = useLocation();
  
  if (!authService.isAuthenticated()) {
    // Сохраняем путь, куда пользователь пытался попасть, чтобы после логина вернуть его туда
    return <Navigate to={`/login?redirect=${encodeURIComponent(location.pathname)}`} replace />;
  }
  return children;
}

function App() {
  const [user, setUser] = useState(null);
  const [isCheckingAuth, setIsCheckingAuth] = useState(true);

  useEffect(() => {
    // Проверяем валидность токена при старте
    const checkAuth = async () => {
      if (authService.isAuthenticated()) {
        const isValid = await authService.validateToken();
        if (isValid) {
          const currentUser = authService.getCurrentUser();
          setUser(currentUser);
          // Connect WebSocket for notifications
          websocketService.connect();
        } else {
          // Token is invalid - clear user and let ProtectedRoute handle redirect
          setUser(null);
          authService.logout();
          websocketService.disconnect();
        }
      } else {
        websocketService.disconnect();
      }
      setIsCheckingAuth(false);
    };
    checkAuth();
    
    // Cleanup WebSocket on unmount
    return () => {
      if (!authService.isAuthenticated()) {
        websocketService.disconnect();
      }
    };
  }, []);

  if (isCheckingAuth) {
    return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', color: '#fff' }}>Загрузка...</div>;
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route 
          path="/login" 
          element={<Login onLogin={setUser} />} 
        />
        <Route 
          path="/register" 
          element={<Register onRegister={setUser} />} 
        />
        <Route
          path="/"
          element={
            <ProtectedRoute>
              <Feed />
            </ProtectedRoute>
          }
        />
        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <Layout user={user} onLogout={() => setUser(null)}>
                <Profile user={user} />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/user/:userId"
          element={
            <ProtectedRoute>
              <Layout user={user} onLogout={() => setUser(null)}>
                <UserProfile />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/create-post"
          element={
            <ProtectedRoute>
              <Layout user={user} onLogout={() => setUser(null)}>
                <CreatePost />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/analytics"
          element={
            <ProtectedRoute>
              <Layout user={user} onLogout={() => setUser(null)}>
                <Analytics />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/notifications"
          element={
            <ProtectedRoute>
              <Layout user={user} onLogout={() => setUser(null)}>
                <Notifications />
              </Layout>
            </ProtectedRoute>
          }
        />
        <Route
          path="/post/:postId"
          element={
            <ProtectedRoute>
              <Layout user={user} onLogout={() => setUser(null)}>
                <Post />
              </Layout>
            </ProtectedRoute>
          }
        />
        {/* Catch-all route - перенаправляем на /login для неавторизованных */}
        <Route
          path="*"
          element={
            authService.isAuthenticated() ? (
              <Navigate to="/" replace />
            ) : (
              <Navigate to="/login" replace />
            )
          }
        />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
