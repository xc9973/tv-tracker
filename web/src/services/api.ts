import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
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

export interface SyncResult {
  update_tasks: number;
  organize_tasks: number;
  errors: number;
}

export const getDashboard = async (): Promise<DashboardData> => {
  const response = await api.get<DashboardData>('/dashboard');
  return response.data;
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

export const sync = async (): Promise<SyncResult> => {
  const response = await api.post<{ result: SyncResult }>('/sync');
  return response.data.result;
};

export const completeTask = async (taskId: number): Promise<void> => {
  await api.post(`/tasks/${taskId}/complete`);
};

export const getLibrary = async (): Promise<TVShow[]> => {
  const response = await api.get<{ shows: TVShow[] }>('/library');
  return response.data.shows;
};

export const sendReport = async (): Promise<void> => {
  await api.post('/report');
};

export const TMDB_IMAGE_BASE = 'https://image.tmdb.org/t/p/w200';
