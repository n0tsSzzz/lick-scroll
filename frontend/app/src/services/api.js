import axios from 'axios';

const API_BASE = {
  auth: 'http://localhost:8001/api/v1',
  post: 'http://localhost:8002/api/v1',
  feed: 'http://localhost:8003/api/v1',
  interaction: 'http://localhost:8007/api/v1',
  wallet: 'http://localhost:8005/api/v1',
  notification: 'http://localhost:8006/api/v1',
  analytics: 'http://localhost:8008/api/v1'
};

const api = axios.create({
  timeout: 10000,
});

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('authToken');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('authToken');
      localStorage.removeItem('currentUser');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export { API_BASE };
export default api;
