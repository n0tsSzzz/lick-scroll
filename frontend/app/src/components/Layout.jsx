import { Link, useNavigate } from 'react-router-dom';
import { useState, useEffect } from 'react';
import api, { API_BASE } from '../services/api';
import { authService } from '../services/authService';
import { websocketService } from '../services/websocketService';
import './Layout.css';

function Layout({ children, user, onLogout }) {
  const navigate = useNavigate();
  const [notificationCount, setNotificationCount] = useState(0);

  const handleLogout = () => {
    authService.logout();
    websocketService.disconnect();
    onLogout();
    navigate('/login');
  };

  const loadNotificationCount = async () => {
    try {
      const response = await api.get(`${API_BASE.notification}/notifications?limit=1`);
      setNotificationCount(response.data.total || 0);
    } catch (err) {
      console.error('Failed to load notification count:', err);
    }
  };

  useEffect(() => {
    loadNotificationCount();
    
    // Subscribe to WebSocket for real-time notification count updates
    if (authService.isAuthenticated()) {
      websocketService.connect();
      const unsubscribe = websocketService.onNotification(() => {
        // Increment count when new notification arrives
        setNotificationCount(prev => prev + 1);
        // Also reload count to ensure accuracy
        loadNotificationCount();
      });
      
      // Also poll every 30 seconds as backup
      const interval = setInterval(loadNotificationCount, 30000);
      
      return () => {
        unsubscribe();
        clearInterval(interval);
      };
    }
  }, []);

  useEffect(() => {
    // Check token validity periodically
    const checkAuth = async () => {
      if (authService.isAuthenticated()) {
        const isValid = await authService.validateToken();
        if (!isValid) {
          handleLogout();
        }
      }
    };
    const authInterval = setInterval(checkAuth, 60000); // Check every minute
    return () => clearInterval(authInterval);
  }, []);

  return (
    <div className="layout">
      <header className="header">
        <div className="container">
          <Link to="/" className="logo">
            Lick Scroll
          </Link>
          <nav className="nav">
            <Link to="/">Лента</Link>
            <Link to="/create-post">Создать пост</Link>
            <Link to="/notifications" className="nav-notifications">
              🔔
              {notificationCount > 0 && (
                <span className="notification-badge">{notificationCount > 99 ? '99+' : notificationCount}</span>
              )}
            </Link>
            <Link to="/profile">Профиль</Link>
            <Link to="/analytics">Аналитика</Link>
            <button onClick={handleLogout} className="btn-logout">
              Выйти
            </button>
          </nav>
        </div>
      </header>
      <main className="main">{children}</main>
    </div>
  );
}

export default Layout;
