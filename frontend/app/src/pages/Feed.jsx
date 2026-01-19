import { useState, useEffect, useRef } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import api, { API_BASE } from '../services/api';
import { authService } from '../services/authService';
import './Feed.css';

function ImageCarousel({ images, postTitle }) {
  const [currentImageIndex, setCurrentImageIndex] = useState(0);
  const containerRef = useRef(null);
  const touchStartX = useRef(0);
  const touchEndX = useRef(0);

  const nextImage = (e) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    setCurrentImageIndex((prev) => (prev + 1) % images.length);
  };

  const prevImage = (e) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    setCurrentImageIndex((prev) => (prev - 1 + images.length) % images.length);
  };

  const goToImage = (index, e) => {
    if (e) {
      e.preventDefault();
      e.stopPropagation();
    }
    setCurrentImageIndex(index);
  };

  const handleTouchStart = (e) => {
    touchStartX.current = e.touches[0].clientX;
  };

  const handleTouchMove = (e) => {
    e.stopPropagation(); // Prevent vertical scrolling when swiping horizontally
  };

  const handleTouchEnd = (e) => {
    touchEndX.current = e.changedTouches[0].clientX;
    const swipeDistance = touchStartX.current - touchEndX.current;
    const minSwipeDistance = 50;

    if (Math.abs(swipeDistance) > minSwipeDistance) {
      if (swipeDistance > 0) {
        // Swipe left - next image
        nextImage();
      } else {
        // Swipe right - prev image
        prevImage();
      }
    }
    e.stopPropagation();
  };

  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    const handleWheel = (e) => {
      e.preventDefault();
      e.stopPropagation();
      if (e.deltaX > 50) {
        nextImage();
      } else if (e.deltaX < -50) {
        prevImage();
      }
    };

    container.addEventListener('wheel', handleWheel, { passive: false });
    return () => container.removeEventListener('wheel', handleWheel);
  }, []);

  return (
    <div 
      className="image-carousel" 
      ref={containerRef}
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onClick={(e) => e.stopPropagation()}
    >
      <div 
        className="carousel-container"
        style={{ transform: `translateX(-${currentImageIndex * 100}%)` }}
      >
        {images.map((img, imgIdx) => (
          <img 
            key={imgIdx} 
            src={img.image_url || img.ImageURL || img.url} 
            alt={`${postTitle} ${imgIdx + 1}`}
            className="post-image"
            onError={(e) => {
              console.error('Failed to load image:', img);
              e.target.style.display = 'none';
            }}
            draggable={false}
          />
        ))}
      </div>
      {images.length > 1 && (
        <>
          <div className="carousel-indicators" onClick={(e) => e.stopPropagation()}>
            {images.map((_, idx) => (
              <button
                key={idx}
                className={`carousel-dot ${idx === currentImageIndex ? 'active' : ''}`}
                onClick={(e) => goToImage(idx, e)}
                aria-label={`Go to image ${idx + 1}`}
              />
            ))}
          </div>
          {currentImageIndex > 0 && (
            <button 
              className="carousel-nav carousel-nav-prev" 
              onClick={(e) => prevImage(e)}
            >
              ‹
            </button>
          )}
          {currentImageIndex < images.length - 1 && (
            <button 
              className="carousel-nav carousel-nav-next" 
              onClick={(e) => nextImage(e)}
            >
              ›
            </button>
          )}
        </>
      )}
    </div>
  );
}

function Feed() {
  const [posts, setPosts] = useState([]);
  const location = useLocation();
  // Try to get index from location.state first, then from localStorage, default to 0
  const savedIndexFromState = location.state?.currentIndex;
  const getInitialIndex = () => {
    if (savedIndexFromState !== undefined) {
      return savedIndexFromState;
    }
    const stored = localStorage.getItem('feed_current_index');
    return stored ? parseInt(stored, 10) : 0;
  };
  const [currentIndex, setCurrentIndex] = useState(getInitialIndex);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState('');
  const [offset, setOffset] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const containerRef = useRef(null);
  const touchStartY = useRef(0);
  const touchEndY = useRef(0);
  const wheelTimeout = useRef(null);
  const navigate = useNavigate();

  useEffect(() => {
    loadFeed();
  }, []);

  useEffect(() => {
    // Save current index to localStorage whenever it changes
    localStorage.setItem('feed_current_index', currentIndex.toString());
  }, [currentIndex]);

  useEffect(() => {
    // Auto-scroll to current post
    if (containerRef.current && posts.length > 0) {
      // Make sure currentIndex is within bounds
      const validIndex = Math.min(currentIndex, posts.length - 1);
      if (validIndex !== currentIndex) {
        setCurrentIndex(validIndex);
        return;
      }
      
      const postElement = containerRef.current.children[currentIndex];
      if (postElement) {
        postElement.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
      }
      
      // Increment view count when post becomes visible
      if (posts[currentIndex] && posts[currentIndex].id) {
        api.post(`${API_BASE.interaction}/interactions/posts/${posts[currentIndex].id}/view`).catch(err => {
          console.warn('Failed to increment view:', err);
        });
      }
    }
  }, [currentIndex, posts]);

  useEffect(() => {
    // Keyboard navigation
    const handleKeyDown = (e) => {
      if (e.key === 'ArrowDown' || e.key === 'PageDown') {
        e.preventDefault();
        nextPost();
      } else if (e.key === 'ArrowUp' || e.key === 'PageUp') {
        e.preventDefault();
        prevPost();
      } else if (e.key === 'Escape') {
        navigate('/profile');
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [currentIndex, posts.length, navigate]);

  const loadFeed = async (loadOffset = 0, append = false) => {
    // Check if user is authenticated
    if (!authService.isAuthenticated()) {
      navigate('/login', { replace: true });
      return;
    }

    if (append) {
      setLoadingMore(true);
    } else {
      setLoading(true);
    }
    setError('');
    try {
      const response = await api.get(`${API_BASE.feed}/feed?limit=100&offset=${loadOffset}`);
      const postsData = response.data.posts || [];
      
      // If no new posts and we're appending, reset to beginning
      if (postsData.length === 0 && append) {
        setOffset(0);
        setHasMore(true);
        // Start from beginning
        loadFeed(0, false);
        return;
      }

      // Use posts data from feed - it should already have likes_count and is_liked from API
      const postsWithLikes = postsData.map((post) => ({
        ...post,
        likes_count: post.likes_count || 0,
        is_liked: post.is_liked || false
      }));

      if (append) {
        setPosts(prevPosts => {
          const currentLength = prevPosts.length;
          // Filter out duplicates based on post ID
          const existingIds = new Set(prevPosts.map(p => p.id));
          const uniqueNewPosts = postsWithLikes.filter(p => !existingIds.has(p.id));
          const newPosts = [...prevPosts, ...uniqueNewPosts];
          // Move to first new post if we were at the last post
          if (uniqueNewPosts.length > 0 && currentIndex === currentLength - 1) {
            setTimeout(() => setCurrentIndex(currentLength), 0);
          }
          return newPosts;
        });
        setOffset(prevOffset => prevOffset + postsWithLikes.length);
      } else {
        setPosts(postsWithLikes);
        setOffset(postsWithLikes.length);
      }
      
      // If we got less than requested, there are no more posts
      setHasMore(postsWithLikes.length === 100);
    } catch (err) {
      console.error('Failed to load feed:', err);
      if (err.response?.status === 401) {
        // Unauthorized - token is invalid, redirect to login
        authService.logout();
        navigate('/login', { replace: true });
        return;
      }
      setError(err.response?.data?.error || 'Ошибка загрузки ленты. Попробуйте обновить страницу.');
    } finally {
      setLoading(false);
      setLoadingMore(false);
    }
  };

  const handleLike = async (e, postId, index) => {
    e.stopPropagation();
    e.preventDefault();
    try {
      const response = await api.post(`${API_BASE.interaction}/interactions/posts/${postId}/like`);
      const liked = response.data?.liked ?? false;
      
      setPosts(prevPosts => {
        if (index >= prevPosts.length) return prevPosts;
        const newPosts = [...prevPosts];
        const post = newPosts[index];
        const currentLikesCount = parseInt(post.likes_count) || 0;
        newPosts[index] = {
          ...post,
          is_liked: liked,
          likes_count: liked ? (currentLikesCount + 1) : Math.max(0, currentLikesCount - 1)
        };
        return newPosts;
      });
    } catch (err) {
      console.error('Failed to like post:', err);
    }
  };

  const handleDonate = async (e, postId, creatorId) => {
    e.stopPropagation();
    e.preventDefault();
    if (!creatorId) {
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

  const nextPost = () => {
    if (posts.length === 0) return;
    
    // If we're at the last post, try to load more
    if (currentIndex === posts.length - 1) {
      if (hasMore && !loadingMore) {
        // Load next batch - currentIndex will be updated in loadFeed
        loadFeed(offset, true);
        return;
      } else if (!hasMore && !loadingMore) {
        // No more posts, start from beginning
        setCurrentIndex(0);
        return;
      }
      // If loading, don't change index yet
      return;
    }
    
    setCurrentIndex((prevIndex) => prevIndex + 1);
  };

  const prevPost = () => {
    if (posts.length === 0) return;
    
    // If we're at the first post, don't do anything (no circular scroll)
    if (currentIndex === 0) {
      return;
    }
    
    // Normal prev: go to previous post
    setCurrentIndex((prevIndex) => prevIndex - 1);
  };

  const handleTouchStart = (e) => {
    touchStartY.current = e.touches[0].clientY;
  };

  const handleTouchMove = (e) => {
    touchEndY.current = e.touches[0].clientY;
  };

  const handleTouchEnd = () => {
    const diff = touchStartY.current - touchEndY.current;
    const minSwipeDistance = 50;

    if (Math.abs(diff) > minSwipeDistance) {
      if (diff > 0) {
        // Swipe up - next post
        nextPost();
      } else {
        // Swipe down - previous post
        prevPost();
      }
    }
  };

  const handleWheel = (e) => {
    e.preventDefault();
    
    // Debounce wheel events to prevent rapid scrolling
    if (wheelTimeout.current) {
      return;
    }
    
    // Set timeout to prevent rapid scrolling (300ms)
    wheelTimeout.current = setTimeout(() => {
      wheelTimeout.current = null;
    }, 300);
    
    // Use threshold to prevent too sensitive scrolling
    const threshold = 50;
    if (Math.abs(e.deltaY) < threshold) {
      return;
    }
    
    if (e.deltaY > 0) {
      nextPost();
    } else {
      prevPost();
    }
  };

  const handleLogout = () => {
    authService.logout();
    navigate('/login');
  };

  if (loading && posts.length === 0) return <div className="feed-loading">Загрузка...</div>;
  if (error) {
    return (
      <div className="feed-error">
        <p>{error}</p>
        <button onClick={loadFeed} className="btn-retry">Попробовать снова</button>
      </div>
    );
  }
  if (posts.length === 0) {
    return (
      <div className="feed-empty">
        <p>Лента пуста</p>
        <p style={{ fontSize: '14px', color: '#666', marginTop: '10px' }}>
          Подпишитесь на авторов или создайте свой первый пост!
        </p>
        <div style={{ marginTop: '20px' }}>
          <button onClick={() => navigate('/create-post')} className="btn-create-post">Создать пост</button>
          <button onClick={loadFeed} className="btn-refresh" style={{ marginLeft: '10px' }}>Обновить</button>
        </div>
      </div>
    );
  }

  const currentPost = posts[currentIndex];

  return (
    <div 
      className="feed-tiktok"
      onTouchStart={handleTouchStart}
      onTouchMove={handleTouchMove}
      onTouchEnd={handleTouchEnd}
      onWheel={handleWheel}
    >
      <div className="feed-header">
        <div className="feed-header-content">
          <div className="feed-logo">Lick Scroll</div>
          <div className="feed-header-actions">
            <button onClick={() => navigate('/create-post')} className="btn-create">+</button>
            <button onClick={() => navigate('/profile')} className="btn-profile">👤</button>
          </div>
        </div>
      </div>
      <div className="feed-container" ref={containerRef}>
        {posts.map((post, index) => (
          <div 
            key={post.id} 
            className={`feed-post ${index === currentIndex ? 'active' : ''}`}
          >
            <div className="feed-post-content">
              <div 
                className="post-media-section"
                onClick={() => navigate(`/post/${post.id}`, { state: { currentIndex: index } })}
                style={{ cursor: 'pointer' }}
              >
                <div className="post-media">
                  {post.type === 'video' && post.media_url && (
                    <video 
                      src={post.media_url} 
                      controls
                      autoPlay={index === currentIndex}
                      loop
                      playsInline
                      muted={index !== currentIndex}
                      className="post-video"
                      onLoadedMetadata={(e) => {
                        if (index === currentIndex) {
                          e.target.play().catch(err => console.warn('Failed to autoplay video:', err));
                        }
                      }}
                    />
                  )}
                  {post.type === 'photo' && post.images && post.images.length > 0 && (
                    <ImageCarousel images={post.images} postTitle={post.title || 'Post'} />
                  )}
                  {post.type === 'photo' && (!post.images || post.images.length === 0) && post.media_url && (
                    <img 
                      src={post.media_url} 
                      alt={post.title || 'Post'}
                      className="post-image"
                      onError={(e) => {
                        console.error('Failed to load image:', post.media_url);
                        e.target.style.display = 'none';
                      }}
                    />
                  )}
                </div>
              </div>
              <div className="post-info-section">
                <div className="post-info-header">
                  <div 
                    className="post-author" 
                    onClick={() => post.creator_id && navigate(`/user/${post.creator_id}`)}
                    style={{ cursor: 'pointer' }}
                  >
                    <div className="author-avatar">
                      {post.creator_avatar ? (
                        <img src={post.creator_avatar} alt="Creator" />
                      ) : (
                        '👤'
                      )}
                    </div>
                    <div className="author-details">
                      <div className="author-username">{post.creator_username || 'Creator'}</div>
                      {post.category && (
                        <div className="author-category">#{post.category}</div>
                      )}
                    </div>
                  </div>
                </div>
                <div className="post-info-content">
                  {post.description && (
                    <p className="post-description">{post.description}</p>
                  )}
                  {!post.description && post.title && (
                    <p className="post-description">{post.title}</p>
                  )}
                </div>
                <div className="post-actions" onClick={(e) => e.stopPropagation()}>
                  <button 
                    className={`action-btn like-btn ${post.is_liked ? 'liked' : ''}`}
                    onClick={(e) => handleLike(e, post.id, index)}
                  >
                    ❤️ {post.likes_count || 0}
                  </button>
                  <button 
                    className="action-btn donate-btn"
                    onClick={(e) => handleDonate(e, post.id, post.creator_id)}
                  >
                    💰 Донат
                  </button>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
      <button onClick={handleLogout} className="btn-logout-bottom">Выйти</button>
    </div>
  );
}

export default Feed;
