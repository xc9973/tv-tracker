import { useState, useEffect } from 'react';
import { getLibrary, updateResourceTime } from '../services/api';
import type { TVShow } from '../services/api';
import './Library.css';

export default function Library() {
  const [shows, setShows] = useState<TVShow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [resourceTimeInput, setResourceTimeInput] = useState('');
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

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

  useEffect(() => {
    fetchLibrary();
  }, []);

  const getStatusText = (status: string, isArchived: boolean) => {
    if (isArchived) return '已归档';
    switch (status) {
      case 'Returning Series':
        return '连载中';
      case 'Ended':
        return '已完结';
      case 'Canceled':
        return '已取消';
      default:
        return status;
    }
  };

  const handleEdit = (show: TVShow) => {
    setEditingId(show.id);
    setResourceTimeInput(show.resource_time || '');
    setMessage(null);
  };

  const handleCancel = () => {
    setEditingId(null);
    setResourceTimeInput('');
  };

  const handleSave = async (showId: number) => {
    const trimmed = resourceTimeInput.trim();
    if (!trimmed) {
      setError('请输入资源时间');
      return;
    }
    try {
      setSaving(true);
      setError(null);
      await updateResourceTime(showId, trimmed);
      setMessage('资源时间已更新');
      setTimeout(() => setMessage(null), 3000);
      setEditingId(null);
      setResourceTimeInput('');
      await fetchLibrary();
    } catch (err) {
      console.error(err);
      setError('更新资源时间失败');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  return (
    <div className="library-page">
      <div className="library-header">
        <h2>我的片库 ({shows.length})</h2>
        <button className="btn" onClick={fetchLibrary} disabled={loading}>
          刷新
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}
      {message && <div className="success-message">{message}</div>}

      {shows.length === 0 ? (
        <p className="empty-message">暂无订阅剧集，去搜索页面添加吧！</p>
      ) : (
        <div className="library-list">
          {shows.map((show) => (
            <div key={show.id} className={`library-item ${show.is_archived ? 'archived' : ''}`}>
              <div className="library-main">
                <span className="show-name">{show.name}</span>
                <span className="show-info">
                  {getStatusText(show.status, show.is_archived)} · {show.total_seasons} 季
                  {show.origin_country && <> · {show.origin_country}</>}
                </span>
              </div>
              
              <div className="resource-time-section">
                {editingId === show.id ? (
                  <div className="edit-mode">
                    <input
                      type="text"
                      value={resourceTimeInput}
                      onChange={(e) => setResourceTimeInput(e.target.value)}
                      className="time-input"
                      placeholder="例如 22:00"
                      disabled={saving}
                    />
                    <button
                      className="btn btn-success"
                      onClick={() => handleSave(show.id)}
                      disabled={saving}
                    >
                      {saving ? '保存中...' : '保存'}
                    </button>
                    <button className="btn" onClick={handleCancel} disabled={saving}>
                      取消
                    </button>
                  </div>
                ) : (
                  <div className="view-mode">
                    <span className="time-label">资源时间：</span>
                    <span className="time-value">{show.resource_time || '待定'}</span>
                    {!show.is_archived && (
                      <button className="btn-link" onClick={() => handleEdit(show)}>
                        编辑
                      </button>
                    )}
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
