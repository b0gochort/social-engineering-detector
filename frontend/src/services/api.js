import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Add token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle 401 errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// Decode JWT token to get user info
const decodeToken = (token) => {
  try {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    )
    return JSON.parse(jsonPayload)
  } catch (error) {
    console.error('Failed to decode token:', error)
    return null
  }
}

export const authService = {
  login: async (username, password) => {
    const response = await api.post('/api/auth/login', { username, password })
    const token = response.data.token
    const decoded = decodeToken(token)

    return {
      token: token,
      user: {
        name: decoded?.Username || username,
        email: username,
        role: decoded?.Role || 'user'
      }
    }
  },

  register: async (username, password) => {
    const response = await api.post('/api/auth/register', { username, password })
    // Backend returns { username, id }
    // Auto-login after registration
    const loginResponse = await api.post('/api/auth/login', { username, password })
    const token = loginResponse.data.token
    const decoded = decodeToken(token)

    return {
      token: token,
      user: {
        name: decoded?.Username || username,
        email: username,
        role: decoded?.Role || 'user'
      }
    }
  },
}

export const incidentService = {
  getAll: async (params) => {
    const response = await api.get('/api/events', { params })
    return response.data
  },

  getById: async (id) => {
    const response = await api.get(`/api/events/${id}`)
    return response.data
  },

  getStats: async () => {
    const response = await api.get('/api/analytics/dashboard')
    return response.data
  },
}

export const chatService = {
  getAll: async () => {
    const response = await api.get('/api/chats')
    return response.data
  },

  getById: async (id) => {
    const response = await api.get(`/api/chats/${id}`)
    return response.data
  },

  create: async (data) => {
    const response = await api.post('/api/chats', data)
    return response.data
  },

  update: async (id, data) => {
    const response = await api.put(`/api/chats/${id}`, data)
    return response.data
  },
}

export const analyticsService = {
  getOverview: async () => {
    const response = await api.get('/api/analytics/dashboard')
    return response.data
  },

  getThreatTrends: async (params) => {
    const response = await api.get('/api/analytics/trends', { params })
    return response.data
  },
}

export default api
