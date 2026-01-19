import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import api, { API_BASE } from '../services/api';
import { authService } from '../services/authService';
import './Profile.css';

function Profile({ user: initialUser }) {
  const [user, setUser] = useState(initialUser);
  const [wallet, setWallet] = useState(null);
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [topupAmount, setTopupAmount] = useState('');
  const [topupLoading, setTopupLoading] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    loadWallet();
    loadUser();
  }, []);

  useEffect(() => {
    if (user?.id) {
      loadMyPosts();
    }
  }, [user?.id]);

  const loadUser = async () => {
    try {
      const response = await api.get(`${API_BASE.auth}/me`);
      setUser(response.data);
    } catch (err) {
      console.error('Ошибка загрузки пользователя:', err);
    }
  };

  const loadWallet = async () => {
    setLoading(true);
    try {
      const response = await api.get(`${API_BASE.wallet}/wallet`);
      setWallet(response.data);
    } catch (err) {
      console.error('Ошибка загрузки кошелька:', err);
    } finally {
      setLoading(false);
    }
  };

  const loadMyPosts = async () => {
    if (!user?.id) return;
    try {
      const response = await api.get(`${API_BASE.post}/posts/creator/${user.id}`);
      setPosts(response.data.posts || []);
    } catch (err) {
      console.error('Ошибка загрузки постов:', err);
    }
  };

  const handleAvatarUpload = async (e) => {
    const file = e.target.files[0];
    if (!file) return;

    if (!file.type.startsWith('image/')) {
      alert('Пожалуйста, выберите файл изображения');
      return;
    }

    const formData = new FormData();
    formData.append('avatar', file);

    setUploading(true);
    try {
      const response = await api.post(`${API_BASE.auth}/avatar`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      if (response.data && response.data.id) {
        setUser(response.data);
        // Update user in localStorage
        localStorage.setItem('currentUser', JSON.stringify(response.data));
      } else {
        throw new Error('Invalid response from server');
      }
    } catch (err) {
      console.error('Ошибка загрузки аватарки:', err);
      if (err.response?.status !== 200) {
        alert(err.response?.data?.error || 'Ошибка загрузки аватарки');
      }
    } finally {
      setUploading(false);
    }
  };

  const handleTopUp = async (e) => {
    e.preventDefault();
    const amount = parseInt(topupAmount);
    if (isNaN(amount) || amount <= 0) {
      alert('Введите корректную сумму');
      return;
    }

    setTopupLoading(true);
    try {
      const response = await api.post(`${API_BASE.wallet}/wallet/topup`, { amount });
      setWallet(response.data);
      setTopupAmount('');
      alert(`Кошелек пополнен на ${amount}`);
    } catch (err) {
      console.error('Ошибка пополнения кошелька:', err);
      alert(err.response?.data?.error || 'Ошибка пополнения кошелька');
    } finally {
      setTopupLoading(false);
    }
  };

  return (
    <div className="profile">
      <h1>Профиль</h1>
      <div className="profile-card">
        <h2>Информация</h2>
        <div className="avatar-section">
          <div className="avatar-preview">
            {user?.avatar_url ? (
              <img src={user.avatar_url} alt="Avatar" />
            ) : (
              <div className="avatar-placeholder">👤</div>
            )}
          </div>
          <label className="avatar-upload-btn">
            {uploading ? 'Загрузка...' : 'Загрузить аватарку'}
            <input
              type="file"
              accept="image/*"
              onChange={handleAvatarUpload}
              style={{ display: 'none' }}
              disabled={uploading}
            />
          </label>
        </div>
        <p><strong>Email:</strong> {user?.email}</p>
        <p><strong>Username:</strong> {user?.username}</p>
      </div>
      {wallet && (
        <div className="profile-card">
          <h2>Кошелёк</h2>
          <p className="wallet-balance"><strong>Баланс:</strong> {wallet.balance} ₽</p>
          <form onSubmit={handleTopUp} className="topup-form">
            <input
              type="number"
              min="1"
              placeholder="Сумма пополнения"
              value={topupAmount}
              onChange={(e) => setTopupAmount(e.target.value)}
              className="topup-input"
              disabled={topupLoading}
            />
            <button 
              type="submit" 
              className="topup-btn"
              disabled={topupLoading || !topupAmount}
            >
              {topupLoading ? 'Пополнение...' : 'Пополнить'}
            </button>
          </form>
        </div>
      )}
      <div className="profile-card">
        <h2>Мои посты ({posts.length})</h2>
        {posts.length === 0 ? (
          <div className="no-posts">У вас пока нет постов</div>
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
  );
}

export default Profile;
