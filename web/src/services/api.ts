import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
});

// Request interceptor to add auth token
api.interceptors.request.use((config) => {
  const token = import.meta.env.VITE_API_TOKEN;
  if (token) {
    if (!config.headers) {
      config.headers = {} as typeof config.headers;
    }
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response interceptor for unified error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    let errorMessage = '请求失败，请稍后重试';
    
    if (error.response) {
      // Server responded with error status
      const status = error.response.status;
      const data = error.response.data;
      
      switch (status) {
        case 400:
          errorMessage = data?.error || '请求参数错误';
          break;
        case 401:
          errorMessage = '未授权，请检查访问令牌';
          break;
        case 403:
          errorMessage = '禁止访问';
          break;
        case 404:
          errorMessage = '请求的资源不存在';
          break;
        case 409:
          errorMessage = data?.error || '资源冲突';
          break;
        case 500:
          errorMessage = '服务器错误，请稍后重试';
          break;
        default:
          errorMessage = data?.error || `请求失败 (${status})`;
      }
    } else if (error.request) {
      // Request made but no response
      errorMessage = '网络连接失败，请检查网络';
    } else {
      // Error in request setup
      errorMessage = error.message || '请求配置错误';
    }
    
    console.error('API Error:', errorMessage, error);
    
    // Create user-friendly error
    const enhancedError = new Error(errorMessage);
    (enhancedError as any).originalError = error;
    
    return Promise.reject(enhancedError);
  }
);

export interface Task {
  id: number;
  tv_show_id: number;
  tv_show_name: string;
  resource_time: string;
  task_type: 'UPDATE' | 'ORGANIZE';
  description: string;
  is_completed: boolean;
  created_at: string;
}

export interface DashboardData {
  update_tasks: Task[];
  organize_tasks: Task[];
}

export interface SearchResult {
  id: number;
  name: string;
  poster_path: string;
  first_air_date: string;
  origin_country: string[];
}

export interface TVShow {
  id: number;
  tmdb_id: number;
  name: string;
  total_seasons: number;
  status: string;
  origin_country: string;
  resource_time: string;
  is_archived: boolean;
  created_at: string;
  updated_at: string;
}

export interface Episode {
  id: number;
  tmdb_id: number;
  season: number;
  episode: number;
  title: string;
  overview: string;
  air_date: string;
}

export interface TodayEpisode {
  episode: Episode;
  show_name: string;
  resource_time: string;
  show_id: number;
}

export const getDashboard = async (): Promise<DashboardData> => {
  const response = await api.get<DashboardData>('/dashboard');
  return response.data;
};

export const getTodayEpisodes = async (): Promise<TodayEpisode[]> => {
  const response = await api.get<{ episodes: TodayEpisode[] }>('/today');
  return response.data.episodes;
};

export const searchTV = async (query: string): Promise<SearchResult[]> => {
  const response = await api.get<{ results: SearchResult[] }>('/search', {
    params: { q: query },
  });
  return response.data.results;
};

export const subscribe = async (tmdbId: number): Promise<TVShow> => {
  const response = await api.post<{ show: TVShow }>('/subscribe', {
    tmdb_id: tmdbId,
  });
  return response.data.show;
};

export const completeTask = async (taskId: number): Promise<void> => {
  await api.post(`/tasks/${taskId}/complete`);
};

export const postponeTask = async (taskId: number): Promise<void> => {
  await api.post(`/tasks/${taskId}/postpone`);
};

export const getLibrary = async (): Promise<TVShow[]> => {
  const response = await api.get<{ shows: TVShow[] }>('/library');
  return response.data.shows;
};

export const updateResourceTime = async (id: number, resourceTime: string): Promise<TVShow> => {
  const response = await api.put<{ show: TVShow }>(`/shows/${id}/resource-time`, {
    resource_time: resourceTime,
  });
  return response.data.show;
};
