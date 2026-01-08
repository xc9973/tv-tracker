import { useState } from 'react';
import { searchTV, subscribe } from '../services/api';
import type { SearchResult } from '../services/api';
import './Search.css';

export default function Search() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [subscribing, setSubscribing] = useState<number | null>(null);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;

    try {
      setLoading(true);
      setError(null);
      const data = await searchTV(query);
      setResults(data);
      if (data.length === 0) {
        setMessage('未找到相关剧集');
      } else {
        setMessage(null);
      }
    } catch (err) {
      setError('搜索失败');
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleSubscribe = async (tmdbId: number) => {
    try {
      setSubscribing(tmdbId);
      setError(null);
      await subscribe(tmdbId);
      setMessage('订阅成功！');
      setTimeout(() => setMessage(null), 3000);
    } catch (err: unknown) {
      if (err && typeof err === 'object' && 'response' in err) {
        const axiosErr = err as { response?: { status?: number } };
        if (axiosErr.response?.status === 409) {
          setError('该剧集已订阅');
        } else {
          setError('订阅失败');
        }
      } else {
        setError('订阅失败');
      }
      console.error(err);
    } finally {
      setSubscribing(null);
    }
  };

  return (
    <div className="search-page">
      <h2>搜索剧集</h2>
      <form className="search-form" onSubmit={handleSearch}>
        <input
          type="text"
          className="search-input"
          placeholder="输入剧名搜索..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <button type="submit" className="btn btn-primary" disabled={loading}>
          {loading ? '搜索中...' : '🔍 搜索'}
        </button>
      </form>
      {error && <div className="error-message">{error}</div>}
      {message && <div className="info-message">{message}</div>}
      
      {results.length > 0 && (
        <div className="search-results">
          <div className="results-header">搜索结果 ({results.length})</div>
          {results.map((show) => (
            <div key={show.id} className="show-item">
              <div className="show-main">
                <span className="show-name">{show.name}</span>
                <span className="show-meta">
                  首播：{show.first_air_date || '未知'}
                  {show.origin_country && show.origin_country.length > 0 && (
                    <> · 地区：{show.origin_country.join(', ')}</>
                  )}
                </span>
              </div>
              <button
                className="btn btn-success"
                onClick={() => handleSubscribe(show.id)}
                disabled={subscribing === show.id}
              >
                {subscribing === show.id ? '订阅中...' : '+ 订阅'}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
