import { useEffect, useState } from 'react';
import {
  getDashboard,
  completeTask,
  postponeTask,
  type DashboardData,
  type Task,
} from '../services/api';
import './Dashboard.css';

const emptyData: DashboardData = {
  update_tasks: [],
  organize_tasks: [],
};

export default function Dashboard() {
  const [data, setData] = useState<DashboardData>(emptyData);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [completingId, setCompletingId] = useState<number | null>(null);
  const [postponingId, setPostponingId] = useState<number | null>(null);

  const loadDashboard = async () => {
    try {
      setLoading(true);
      setError(null);
      const result = await getDashboard();
      setData(result);
    } catch (err) {
      console.error(err);
      setError('加载任务数据失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDashboard();
  }, []);

  const showSuccess = (msg: string) => {
    setSuccess(msg);
    setTimeout(() => setSuccess(null), 3000);
  };

  const handleComplete = async (task: Task) => {
    try {
      setCompletingId(task.id);
      setError(null);
      await completeTask(task.id);
      showSuccess(`已完成：${task.tv_show_name}`);
      await loadDashboard();
    } catch (err) {
      console.error(err);
      setError('标记任务失败，请稍后重试');
    } finally {
      setCompletingId(null);
    }
  };

  const handlePostpone = async (task: Task) => {
    try {
      setPostponingId(task.id);
      setError(null);
      await postponeTask(task.id);
      showSuccess(`已推迟到明天：${task.tv_show_name}`);
      await loadDashboard();
    } catch (err) {
      console.error(err);
      setError('推迟任务失败，请稍后重试');
    } finally {
      setPostponingId(null);
    }
  };

  const renderTasks = (tasks: Task[], typeLabel: string, emptyText: string) => (
    <section className="task-section">
      <h3>{typeLabel} ({tasks.length})</h3>
      {tasks.length === 0 ? (
        <p className="empty-message">{emptyText}</p>
      ) : (
        <div className="task-list">
          {tasks.map((task) => (
            <div
              key={task.id}
              className={`task-item ${task.task_type === 'ORGANIZE' ? 'organize' : ''}`}
            >
              <div className="task-content">
                <span className="task-time">{task.resource_time || '未知'}</span>
                <span className="task-name">{task.tv_show_name}</span>
                <span className="task-desc">{task.description}</span>
              </div>
              <div className="task-actions">
                <button
                  className="btn btn-warning"
                  onClick={() => handlePostpone(task)}
                  disabled={postponingId === task.id || completingId === task.id}
                >
                  {postponingId === task.id ? '推迟中...' : '⏭ 推迟'}
                </button>
                <button
                  className="btn btn-success"
                  onClick={() => handleComplete(task)}
                  disabled={completingId === task.id || postponingId === task.id}
                >
                  {completingId === task.id ? '处理中...' : '✓ 完成'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );

  if (loading) {
    return <div className="dashboard loading">加载中...</div>;
  }

  const totalTasks = data.update_tasks.length + data.organize_tasks.length;

  return (
    <div className="dashboard">
      <div className="dashboard-header">
        <div>
          <h2>任务管理</h2>
          <p className="task-summary">未完成任务：{totalTasks} 条</p>
        </div>
        <button className="btn" onClick={loadDashboard} disabled={loading}>
          刷新
        </button>
      </div>

      {error && <div className="error-message">{error}</div>}
      {success && <div className="success-message">{success}</div>}

      {renderTasks(data.update_tasks, '今日更新', '暂无更新任务')}
      {renderTasks(data.organize_tasks, '待整理归档', '暂无整理任务')}
    </div>
  );
}
