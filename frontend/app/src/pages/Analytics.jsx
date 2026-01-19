import { useState, useEffect } from 'react';
import api, { API_BASE } from '../services/api';
import './Analytics.css';

function Analytics() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    loadStats();
  }, []);

  const loadStats = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await api.get(`${API_BASE.analytics}/analytics/creator/stats`);
      setStats(response.data);
    } catch (err) {
      setError(err.response?.data?.error || 'Ошибка загрузки статистики');
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="loading">Загрузка...</div>;
  if (error) return <div className="error">{error}</div>;

  return (
    <div className="analytics">
      <h1>Аналитика</h1>
      {stats && (
        <div className="stats-grid">
          <div className="stat-card">
            <h3>Посты</h3>
            <p className="stat-value">{stats.total_posts}</p>
          </div>
          <div className="stat-card">
            <h3>Просмотры</h3>
            <p className="stat-value">{stats.total_views}</p>
          </div>
          <div className="stat-card">
            <h3>Лайки</h3>
            <p className="stat-value">{stats.total_likes}</p>
          </div>
          <div className="stat-card">
            <h3>Подписчики</h3>
            <p className="stat-value">{stats.total_subscribers}</p>
          </div>
          <div className="stat-card">
            <h3>Донаты</h3>
            <p className="stat-value">{stats.total_donations}</p>
          </div>
          <div className="stat-card">
            <h3>Доход</h3>
            <p className="stat-value">{stats.total_revenue}</p>
          </div>
        </div>
      )}
    </div>
  );
}

export default Analytics;
