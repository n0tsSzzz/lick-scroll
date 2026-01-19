import { useState, useEffect } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import api, { API_BASE } from '../services/api';
import { authService } from '../services/authService';
import './Post.css';

function Post() {
  const { postId } = useParams();
  const navigate = useNavigate();
  const location = useLocation();
  const savedIndex = location.state?.currentIndex;
  const [post, setPost] = useState(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!authService.isAuthenticated()) {
      navigate('/login');
      return;
    }
    loadPost();
    incrementView();
    // Delete notification for this post when viewing
    deleteNotification();
  }, [postId]);

  const deleteNotification = async () => {
    if (!postId) return;
    try {
      await api.delete(`${API_BASE.notification}/notifications/${postId}`);
    } catch (err) {
      // Ignore errors - notification might not exist
      console.warn('Failed to delete notification:', err);
    }
  };

  const incrementView = async () => {
    if (!postId) return;
    try {
      await api.post(`${API_BASE.interaction}/interactions/posts/${postId}/view`);
    } catch (err) {
      console.warn('Failed to increment view:', err);
    }
  };

  const loadPost = async () => {
    setLoading(true);
    setError('');
    try {
      const response = await api.get(`${API_BASE.post}/posts/${postId}`);
      setPost(response.data);
    } catch (err) {
      console.error('Failed to load post:', err);
      if (err.response?.status === 404) {
        setError('Пост не найден');
      } else if (err.response?.status === 401) {
        navigate('/login');
        return;
      } else {
        setError('Не удалось загрузить пост');
      }
    } finally {
      setLoading(false);
    }
  };

  const handleLike = async (e) => {
    e.stopPropagation();
    e.preventDefault();
    try {
      await api.post(`${API_BASE.interaction}/interactions/posts/${postId}/like`);
      const response = await api.get(`${API_BASE.post}/posts/${postId}`);
      setPost(response.data);
    } catch (err) {
      console.error('Failed to like post:', err);
    }
  };

  const handleDonate = async () => {
    if (!post?.creator_id) {
      alert('Не удалось определить автора поста');
      return;
    }
    const amount = prompt('Введите сумму доната (минимум 1):');
    if (!amount) return;
    const donateAmount = parseInt(amount, 10);
    if (isNaN(donateAmount) || donateAmount < 1) {
      alert('Введите корректную сумму (минимум 1)');
      return;
    }
    try {
      await api.post(`${API_BASE.wallet}/wallet/donate/${postId}`, { amount: donateAmount });
      alert(`Успешно отправлено ${donateAmount} монет автору поста!`);
    } catch (err) {
      console.error('Failed to donate:', err);
      if (err.response?.status === 400) {
        alert(err.response.data.error || 'Ошибка при отправке доната');
      } else if (err.response?.status === 401) {
        alert('Необходимо войти в систему');
        navigate('/login');
      } else {
        alert('Не удалось отправить донат');
      }
    }
  };

  const handleBack = () => {
    if (savedIndex !== undefined) {
      navigate('/', { state: { currentIndex: savedIndex } });
    } else {
      navigate(-1);
    }
  };

  if (loading) {
    return <div className="post-loading">Загрузка...</div>;
  }

  if (error) {
    return (
      <div className="post-error">
        <p>{error}</p>
        <button onClick={handleBack}>Вернуться в ленту</button>
      </div>
    );
  }

  if (!post) {
    return null;
  }

  return (
    <div className="post-page">
      <div className="post-header">
        <button onClick={handleBack} className="btn-back">← Назад</button>
        {post.creator_id && (
          <button 
            onClick={() => navigate(`/user/${post.creator_id}`)}
            className="btn-creator"
          >
            {post.creator_username || 'Creator'}
          </button>
        )}
      </div>
      <div className="post-content">
        <div className="post-media-full">
          {post.type === 'video' && post.media_url && (
            <video src={post.media_url} controls autoPlay className="post-video-full" />
          )}
          {post.type === 'photo' && post.images && post.images.length > 0 && (
            <div className="post-images-full">
              {post.images.map((img, idx) => (
                <img
                  key={idx}
                  src={img.image_url || img.ImageURL || img.url}
                  alt={`${post.title || 'Post'} ${idx + 1}`}
                  className="post-image-full"
                />
              ))}
            </div>
          )}
          {post.type === 'photo' && (!post.images || post.images.length === 0) && post.media_url && (
            <img src={post.media_url} alt={post.title || 'Post'} className="post-image-full" />
          )}
        </div>
        <div className="post-details">
          <h1>{post.title || 'Без названия'}</h1>
          {post.description && <p className="post-description">{post.description}</p>}
          {post.category && <p className="post-category">#{post.category}</p>}
          <div className="post-stats">
            <div className="post-stat">
              <button onClick={(e) => handleLike(e)} className="btn-like">
                ❤️ {post.likes_count || 0}
              </button>
            </div>
            <div className="post-stat">👁️ {post.views || 0}</div>
            <div className="post-stat">
              <button onClick={handleDonate} className="btn-donate">
                💰 Донат
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default Post;
