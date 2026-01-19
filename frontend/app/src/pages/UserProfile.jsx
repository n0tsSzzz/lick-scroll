import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api, { API_BASE } from '../services/api';
import { authService } from '../services/authService';
import './UserProfile.css';

function UserProfile() {
  const { userId } = useParams();
  const navigate = useNavigate();
  const [user, setUser] = useState(null);
  const [posts, setPosts] = useState([]);
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [notificationsEnabled, setNotificationsEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const currentUser = authService.getCurrentUser();

  useEffect(() => {
    if (!authService.isAuthenticated()) {
      navigate('/login');
      return;
    }
    setLoading(true);
    setError('');
    Promise.all([
      loadUserProfile(),
      loadUserPosts(),
      checkSubscription(),
      checkNotificationSettings()
    ]).finally(() => {
      setLoading(false);
    });
  }, [userId]);

  const loadUserProfile = async () => {
    try {
      const response = await api.get(`${API_BASE.auth}/user/${userId}`);
      setUser(response.data);
    } catch (err) {
      console.error('Failed to load user profile:', err);
      setError('Не удалось загрузить профиль пользователя');
    }
  };

  const loadUserPosts = async () => {
    try {
      const response = await api.get(`${API_BASE.post}/posts/creator/${userId}`);
      setPosts(response.data.posts || []);
    } catch (err) {
      console.error('Failed to load user posts:', err);
    }
  };

  const checkSubscription = async () => {
    if (!currentUser || !userId || currentUser.id === userId) return;
    try {
      const response = await api.get(`${API_BASE.auth}/users/${currentUser.id}/subscriptions/${userId}/status`);
      setIsSubscribed(response.data.subscribed || false);
    } catch (err) {
      console.error('Failed to check subscription:', err);
      setIsSubscribed(false);
    }
  };

  const checkNotificationSettings = async () => {
    if (!currentUser || !userId || currentUser.id === userId) {
      setNotificationsEnabled(false);
      return;
    }
    try {
      const response = await api.get(`${API_BASE.notification}/notifications/settings/${userId}`);
      // API returns enabled: true/false, default is true if not set
      // But we need to check: if subscription exists, default enabled is true
      // If subscription doesn't exist, notifications should be false
      const enabled = response.data?.enabled === true;
      setNotificationsEnabled(enabled);
      console.log('Notification settings loaded:', { userId, enabled, response: response.data });
    } catch (err) {
      console.error('Failed to check notification settings:', err);
      // If 404 or error, assume notifications are disabled
      setNotificationsEnabled(false);
    }
  };

  const handleSubscribe = async () => {
    if (!currentUser) return;
    try {
      await api.post(`${API_BASE.auth}/users/${currentUser.id}/subscriptions/${userId}`);
      setIsSubscribed(true);
      // Process notification queue after subscribing
      try {
        await api.post(`${API_BASE.notification}/notifications/process-queue`);
      } catch (err) {
        console.warn('Failed to process notification queue:', err);
      }
    } catch (err) {
      console.error('Failed to subscribe:', err);
      alert(err.response?.data?.error || 'Ошибка подписки');
    }
  };

  const handleToggleNotifications = async () => {
    if (!isSubscribed) {
      alert('Сначала подпишитесь на пользователя');
      return;
    }
    try {
      if (notificationsEnabled) {
        await api.delete(`${API_BASE.notification}/notifications/settings/${userId}`);
        setNotificationsEnabled(false);
      } else {
        await api.post(`${API_BASE.notification}/notifications/settings/${userId}`);
        setNotificationsEnabled(true);
        // Process notification queue after enabling
        try {
          await api.post(`${API_BASE.notification}/notifications/process-queue`);
        } catch (err) {
          console.warn('Failed to process notification queue:', err);
        }
      }
    } catch (err) {
      console.error('Failed to toggle notifications:', err);
      alert(err.response?.data?.error || 'Ошибка изменения настроек уведомлений');
    }
  };

  const handleUnsubscribe = async () => {
    if (!currentUser) return;
    try {
      await api.delete(`${API_BASE.auth}/users/${currentUser.id}/subscriptions/${userId}`);
      setIsSubscribed(false);
      // Also disable notifications when unsubscribing
      setNotificationsEnabled(false);
      try {
        await api.delete(`${API_BASE.notification}/notifications/settings/${userId}`);
      } catch (err) {
        console.warn('Failed to disable notifications:', err);
      }
    } catch (err) {
      console.error('Failed to unsubscribe:', err);
      alert(err.response?.data?.error || 'Ошибка отписки');
    }
  };

  if (loading) {
    return <div className="user-profile-loading">Загрузка...</div>;
  }

  if (error) {
    return <div className="user-profile-error">{error}</div>;
  }

  if (!user) {
    return <div className="user-profile-error">Пользователь не найден</div>;
  }

  const isOwnProfile = currentUser && currentUser.id === userId;

  return (
    <div className="user-profile">
      <div className="user-profile-header">
        <button onClick={() => navigate(-1)} className="btn-back">← Назад</button>
      </div>
      <div className="user-profile-content">
        <div className="user-info">
          <div className="user-avatar-large">
            {user.avatar_url ? (
              <img src={user.avatar_url} alt={user.username} />
            ) : (
              <div className="avatar-placeholder-large">👤</div>
            )}
          </div>
          <div className="user-details">
            <h1>{user.username || 'Creator'}</h1>
            <p className="user-email">{user.email}</p>
            {!isOwnProfile && (
              <div className="user-actions">
                <button
                  className={`btn-subscribe ${isSubscribed ? 'subscribed' : ''}`}
                  onClick={isSubscribed ? handleUnsubscribe : handleSubscribe}
                >
                  {isSubscribed ? '✓ Подписан' : '+ Подписаться'}
                </button>
                {isSubscribed && (
                  <button
                    className={`btn-notifications ${notificationsEnabled ? 'enabled' : ''}`}
                    onClick={handleToggleNotifications}
                    title={notificationsEnabled ? 'Отключить уведомления от этого пользователя' : 'Включить уведомления от этого пользователя'}
                  >
                    {notificationsEnabled ? '🔔 Уведомления включены' : '🔕 Получить уведомления'}
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
        <div className="user-posts-section">
          <h2>Посты ({posts.length})</h2>
          {posts.length === 0 ? (
            <div className="no-posts">У пользователя пока нет постов</div>
          ) : (
            <div className="posts-grid">
              {posts.map((post) => (
                <div
                  key={post.id}
                  className="post-card"
                  onClick={() => navigate(`/post/${post.id}`)}
                >
                  {post.type === 'video' && post.media_url ? (
                    <video src={post.media_url} className="post-thumbnail" />
                  ) : post.images && post.images.length > 0 ? (
                    <img
                      src={post.images[0].image_url || post.images[0].ImageURL || post.images[0].url}
                      alt={post.title}
                      className="post-thumbnail"
                    />
                  ) : post.media_url ? (
                    <img src={post.media_url} alt={post.title} className="post-thumbnail" />
                  ) : (
                    <div className="post-placeholder">Нет изображения</div>
                  )}
                  <div className="post-card-info">
                    <div className="post-likes">❤️ {post.likes_count || 0}</div>
                    <div className="post-views">👁️ {post.views || 0}</div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

export default UserProfile;
