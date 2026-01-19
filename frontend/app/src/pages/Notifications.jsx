import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api, { API_BASE } from '../services/api';
import { websocketService } from '../services/websocketService';
import { authService } from '../services/authService';
import './Notifications.css';

function Notifications() {
  const [notifications, setNotifications] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  useEffect(() => {
    loadNotifications();
    
    // Subscribe to WebSocket notifications
    if (authService.isAuthenticated()) {
      websocketService.connect();
      const unsubscribe = websocketService.onNotification((notification) => {
        // Add new notification to the top of the list, avoiding duplicates
        setNotifications(prev => {
          // Check if notification already exists (by type, post_id, and timestamp)
          const exists = prev.some(n => {
            const sameType = n.type === notification.type;
            const samePostId = n.data?.post_id === notification.data?.post_id;
            const sameTime = n.created_at === notification.created_at;
            return sameType && samePostId && sameTime;
          });
          
          if (exists) {
            console.log('Notification already exists, skipping:', notification);
            return prev;
          }
          
          console.log('Adding new notification via WebSocket:', notification);
          return [notification, ...prev];
        });
      });
      
      return () => {
        unsubscribe();
      };
    }
  }, []);

  const loadNotifications = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await api.get(`${API_BASE.notification}/notifications`);
      setNotifications(response.data.notifications || []);
    } catch (err) {
      console.error('Failed to load notifications:', err);
      if (err.response?.status === 401) {
        navigate('/login');
        return;
      }
      setError('Не удалось загрузить уведомления');
    } finally {
      setLoading(false);
    }
  };

  const formatTime = (timestamp) => {
    if (!timestamp) return '';
    const date = new Date(timestamp);
    const now = new Date();
    const diff = now - date;
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);

    if (minutes < 1) return 'только что';
    if (minutes < 60) return `${minutes} мин. назад`;
    if (hours < 24) return `${hours} ч. назад`;
    if (days < 7) return `${days} дн. назад`;
    return date.toLocaleDateString('ru-RU');
  };

  const handleNotificationClick = async (notification) => {
    // Handle navigation based on notification type
    if (notification.type === 'new_post' || notification.type === 'like') {
      const postId = notification.data?.post_id;
      if (postId) {
        // Remove from local state immediately for better UX
        setNotifications(prev => prev.filter(n => 
          !(n.data?.post_id === postId && n.type === notification.type)
        ));
        
        // Delete notification on server
        try {
          await api.delete(`${API_BASE.notification}/notifications/${postId}`);
        } catch (err) {
          console.error('Failed to delete notification:', err);
          // If deletion failed, reload notifications to get correct state
          loadNotifications();
        }
        
        navigate(`/post/${postId}`);
      }
    }
    // subscription notifications are not clickable
  };

  if (loading) {
    return <div className="notifications-loading">Загрузка...</div>;
  }

  if (error) {
    return <div className="notifications-error">{error}</div>;
  }

  return (
    <div className="notifications">
      <div className="notifications-header">
        <h1>Уведомления</h1>
        <button onClick={loadNotifications} className="btn-refresh">
          Обновить
        </button>
      </div>
      {notifications.length === 0 ? (
        <div className="no-notifications">
          <p>У вас пока нет уведомлений</p>
        </div>
      ) : (
        <div className="notifications-list">
          {notifications.map((notification, index) => {
            // Create unique key from notification data
            const notificationKey = notification.data?.post_id 
              ? `${notification.type}-${notification.data.post_id}-${notification.created_at}`
              : `${notification.type}-${notification.user_id}-${notification.created_at}-${index}`;
            
            return (
              <div
                key={notificationKey}
                className={`notification-item ${(notification.type === 'new_post' || notification.type === 'like') && notification.data?.post_id ? 'clickable' : ''}`}
                onClick={() => handleNotificationClick(notification)}
              >
                <div className="notification-icon">
                  {notification.type === 'new_post' ? '📝' : 
                   notification.type === 'like' ? '❤️' :
                   notification.type === 'subscription' ? '👤' : '🔔'}
                </div>
                <div className="notification-content">
                  <div className="notification-title">{notification.title}</div>
                  <div className="notification-message">{notification.message}</div>
                  {notification.created_at && (
                    <div className="notification-time">
                      {formatTime(notification.created_at)}
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

export default Notifications;
