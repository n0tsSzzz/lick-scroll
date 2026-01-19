import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import api, { API_BASE } from '../services/api';
import './CreatePost.css';

function CreatePost() {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [type, setType] = useState('photo');
  const [category, setCategory] = useState('');
  const [files, setFiles] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleFileChange = (e) => {
    const selectedFiles = Array.from(e.target.files);
    
    if (selectedFiles.length === 0) {
      setFiles([]);
      return;
    }
    
    // Auto-detect type based on first file
    const firstFile = selectedFiles[0];
    const isVideo = firstFile.type.startsWith('video/');
    
    // If video, only allow one file
    if (isVideo) {
      if (selectedFiles.length > 1) {
        setError('Для видео можно выбрать только один файл');
        setFiles([firstFile]);
      } else {
        setFiles([firstFile]);
        setError('');
      }
      setType('video');
    } else {
      // For images, limit to 10
      if (selectedFiles.length > 10) {
        setError('Максимум 10 фотографий в одном посте');
        setFiles(selectedFiles.slice(0, 10));
      } else {
        setFiles(selectedFiles);
        setError('');
      }
      setType('photo');
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const formData = new FormData();
      formData.append('title', title);
      formData.append('description', description);
      formData.append('type', type);
      formData.append('category', category);

      if (type === 'photo') {
        files.forEach((file) => {
          formData.append('images', file);
        });
      } else {
        if (files[0]) {
          formData.append('media', files[0]);
        }
      }

      await api.post(`${API_BASE.post}/posts`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' }
      });

      navigate('/');
    } catch (err) {
      setError(err.response?.data?.error || 'Ошибка создания поста');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="create-post">
      <h1>Создать пост</h1>
      <form onSubmit={handleSubmit} className="post-form">
        <div className="form-group">
          <label>Заголовок</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            required
            placeholder="Название поста"
          />
        </div>
        <div className="form-group">
          <label>Описание</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows="4"
            placeholder="Описание поста"
          />
        </div>
        <div className="form-group">
          <label>Тип (определяется автоматически)</label>
          <input
            type="text"
            value={type === 'photo' ? 'Фото' : 'Видео'}
            disabled
            style={{ opacity: 0.6, cursor: 'not-allowed' }}
          />
        </div>
        <div className="form-group">
          <label>Категория</label>
          <input
            type="text"
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            placeholder="fetish, cosplay, etc."
          />
        </div>
        <div className="form-group">
          <label>{type === 'photo' ? `Изображения (макс. 10, выбрано: ${files.length})` : 'Видео'}</label>
          <input
            type="file"
            onChange={handleFileChange}
            multiple={true}
            accept="image/*,video/*"
            required
          />
        </div>
        {error && <div className="error-message">{error}</div>}
        <button type="submit" disabled={loading} className="btn-primary">
          {loading ? 'Создание...' : 'Создать пост'}
        </button>
      </form>
    </div>
  );
}

export default CreatePost;
