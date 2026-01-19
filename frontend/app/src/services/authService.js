import api, { API_BASE } from './api';

export const authService = {
  async register(email, username, password) {
    const response = await api.post(`${API_BASE.auth}/register`, {
      email,
      username,
      password
    });
    if (response.data.token) {
      localStorage.setItem('authToken', response.data.token);
      localStorage.setItem('currentUser', JSON.stringify(response.data.user));
    }
    return response.data;
  },

  async login(email, password) {
    const response = await api.post(`${API_BASE.auth}/login`, {
      email,
      password
    });
    if (response.data.token) {
      localStorage.setItem('authToken', response.data.token);
      localStorage.setItem('currentUser', JSON.stringify(response.data.user));
    }
    return response.data;
  },

  logout() {
    localStorage.removeItem('authToken');
    localStorage.removeItem('currentUser');
  },

  getCurrentUser() {
    const userStr = localStorage.getItem('currentUser');
    return userStr ? JSON.parse(userStr) : null;
  },

  isAuthenticated() {
    return !!localStorage.getItem('authToken');
  },

  async validateToken() {
    const token = localStorage.getItem('authToken');
    if (!token) {
      return false;
    }
    try {
      const response = await api.get(`${API_BASE.auth}/me`);
      if (response.data && response.data.id) {
        localStorage.setItem('currentUser', JSON.stringify(response.data));
        return true;
      }
      // Invalid response - clear token
      this.logout();
      return false;
    } catch (err) {
      // If 401 (Unauthorized) or 404 (User not found) - token is invalid
      if (err.response?.status === 401 || err.response?.status === 404) {
        this.logout();
      }
      return false;
    }
  }
};
