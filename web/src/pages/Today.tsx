import { useEffect, useState } from 'react';
import { getTodayEpisodes, type TodayEpisode } from '../services/api';
import './Today.css';

export default function Today() {
  const [episodes, setEpisodes] = useState<TodayEpisode[]>([]);
  const [loading, setLoading] = useState(true);

  const loadEpisodes = async () => {
    try {
      setLoading(true);
      const res = await getTodayEpisodes();
      setEpisodes(res);
    } catch (e) {
      console.error('getTodayEpisodes error', e);
      setEpisodes([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadEpisodes();
  }, []);

  const formatDate = () => {
    const today = new Date();
    const year = today.getFullYear();
    const month = String(today.getMonth() + 1).padStart(2, '0');
    const day = String(today.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };

  const getWeekDay = () => {
    const days = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
    return days[new Date().getDay()];
  };

  // 骨架屏加载状态
  if (loading) {
    return (
      <div className="today minimal">
        <div className="header-row">
          <div className="title">📺 今日更新</div>
          <div className="meta skeleton-text" style={{width: '200px', height: '20px'}}></div>
          <button type="button" className="refresh" disabled>刷新</button>
        </div>
        <ul className="simple-list">
          {[1, 2, 3, 4, 5].map((i) => (
            <li key={i} className="simple-item skeleton-item">
              <span className="skeleton-text" style={{width: '120px'}}></span>
              <span className="sep">·</span>
              <span className="skeleton-text" style={{width: '60px'}}></span>
              <span className="sep">·</span>
              <span className="skeleton-text" style={{width: '200px'}}></span>
              <span className="skeleton-text" style={{width: '60px'}}></span>
            </li>
          ))}
        </ul>
      </div>
    );
  }

  return (
    <div className="today minimal">
      <div className="header-row">
        <div className="title">📺 今日更新</div>
        <div className="meta">{formatDate()} {getWeekDay()} · {episodes.length} 条</div>
        <button type="button" className="refresh" onClick={loadEpisodes}>刷新</button>
      </div>

      <ul className="simple-list">
        {episodes.map((it) => (
          <li key={`${it.show_id}-${it.episode.season}-${it.episode.episode}`} className="simple-item">
            <span className="sname">{it.show_name}</span>
            <span className="sep">·</span>
            <span className="ep">S{String(it.episode.season).padStart(2, '0')}E{String(it.episode.episode).padStart(2, '0')}</span>
            {it.episode.title ? (
              <><span className="sep">·</span><span className="title">{it.episode.title}</span></>
            ) : null}
            <span className="time">[{it.resource_time}]</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
