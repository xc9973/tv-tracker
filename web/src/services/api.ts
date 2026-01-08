import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
});

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
