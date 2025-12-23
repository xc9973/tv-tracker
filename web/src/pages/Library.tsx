import { useState, useEffect } from 'react';
import { getLibrary } from '../services/api';
import type { TVShow } from '../services/api';
import './Library.css';

export default function Library() {
  const [shows, setShows] = useState<TVShow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchLibrary = async () => {
      try {
        setLoading(true);
        setError(null);
        const data = await getLibrary();
        setShows(data);
      } catch (err) {
        setError('获取片库失败');
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetchLibrary();
  }, []);

  const getStatusBadge = (status: string, isArchived: boolean) => {
    if (isArchived) {
      return <span className="status-badge archived">已归档</span>;
    }
    switch (status) {
      case 'Returning Series':
        return <span className="status-badge returning">连载中</span>;
      case 'Ended':
        return <span className="status-badge ended">已完结</span>;
      case 'Canceled':
        return <span className="status-badge canceled">已取消</span>;
      default:
        return <span className="status-badge unknown">{status}</span>;
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="library-page">
      <h2>我的片库 ({shows.length})</h2>
      {error && <div className="error-message">{error}</div>}
      {shows.length === 0 ? (
        <p className="empty-message">暂无订阅剧集，去搜索页面添加吧！</p>
      ) : (
        <div className="show-grid">
          {shows.map((show) => (
            <div key={show.id} className={`library-card ${show.is_archived ? 'archived' : ''}`}>
              <div className="library-card-header">
                <h3 className="library-show-name">{show.name}</h3>
                {getStatusBadge(show.status, show.is_archived)}
              </div>
              <div className="library-card-info">
                <p><span className="label">季数：</span>{show.total_seasons}</p>
                <p><span className="label">地区：</span>{show.origin_country || '未知'}</p>
                <p><span className="label">资源时间：</span>{show.resource_time}</p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
