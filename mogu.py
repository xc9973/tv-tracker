import sqlite3
import datetime
import time
import tmdb_api
import requests 
import sys
import os  # 新增：用于锁定文件路径

# ================= 核心配置 =================
TG_BOT_TOKEN = "8189108565:AAEiIjvBtaFcpWtaLb0sHFpLkh97XbrOxXo" 
TG_CHAT_ID = "-1002530823476"

# ================= 🔧 路径自动修正 (关键修改) =================
# 获取脚本当前所在的文件夹路径
BASE_DIR = os.path.dirname(os.path.abspath(__file__))

# 强制将数据库和日志文件指定在该路径下
DB_FILE = os.path.join(BASE_DIR, "local_schedule.db")
LOG_FILE = os.path.join(BASE_DIR, "run_log.txt") # 运行日志
REPORT_FILE = os.path.join(BASE_DIR, "今日更新清单.txt")

# 确保能引用到同目录下的 tmdb_api
sys.path.append(BASE_DIR)
# =========================================================

def write_log(msg):
    """写日志，方便排查定时任务是否运行"""
    now = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    try:
        with open(LOG_FILE, "a", encoding="utf-8") as f:
            f.write(f"[{now}] {msg}\n")
    except: pass

def get_beijing_now():
    utc_now = datetime.datetime.now(datetime.timezone.utc)
    return utc_now + datetime.timedelta(hours=8)

def send_telegram_message(text):
    if not TG_BOT_TOKEN: return
    url = f"https://api.telegram.org/bot{TG_BOT_TOKEN}/sendMessage"
    payload = { "chat_id": TG_CHAT_ID, "text": text, "parse_mode": "HTML" }
    try:
        requests.post(url, json=payload, timeout=10)
    except Exception as e:
        write_log(f"❌ 发送失败: {e}")

def test_telegram():
    print("\n🔔 正在发送测试消息...")
    bj_time = get_beijing_now().strftime("%Y-%m-%d %H:%M:%S")
    msg = f"🔔 <b>Telegram 通知测试</b>\n\n北京时间: {bj_time}\n(服务器时间已校准为 UTC+8)"
    send_telegram_message(msg)

def init_db():
    conn = sqlite3.connect(DB_FILE)
    cursor = conn.cursor()
    cursor.execute('''CREATE TABLE IF NOT EXISTS shows (tmdb_id INTEGER PRIMARY KEY, name TEXT, total_seasons INTEGER)''')
    cursor.execute('''CREATE TABLE IF NOT EXISTS episodes (tmdb_id INTEGER, season INTEGER, episode INTEGER, title TEXT, overview TEXT, air_date TEXT, UNIQUE(tmdb_id, season, episode))''')
    
    cols = [('resource_time', "TEXT DEFAULT '待定'"), ('status', "TEXT DEFAULT 'Unknown'"), ('next_air_date', "TEXT DEFAULT '待定'")]
    for col, definition in cols:
        try: cursor.execute(f"ALTER TABLE shows ADD COLUMN {col} {definition}")
        except: pass
    conn.commit()
    conn.close()

def sync_show_to_local(tmdb_id, show_name):
    print(f"🔄 同步: {show_name}")
    details = tmdb_api.get_show_details(tmdb_id)
    if not details: return
    latest = details.get('number_of_seasons', 1)
    
    conn = sqlite3.connect(DB_FILE)
    conn.execute("UPDATE shows SET total_seasons = ? WHERE tmdb_id = ?", (latest, tmdb_id))
    
    eps = tmdb_api.get_season_episodes(tmdb_id, latest)
    if eps:
        for ep in eps:
            try:
                conn.execute('INSERT OR REPLACE INTO episodes (tmdb_id, season, episode, title, overview, air_date) VALUES (?, ?, ?, ?, ?, ?)', 
                             (tmdb_id, ep['season'], ep['episode'], ep['title'], ep['overview'], ep['air_date']))
            except: pass
    conn.commit()
    conn.close()

def subscribe_show():
    print("\n➕ 添加订阅")
    id_input = input("请输入 TMDB ID: ").strip()
    if not id_input.isdigit(): return
    tmdb_id = int(id_input)
    
    details = tmdb_api.get_show_details(tmdb_id)
    if not details: return
    
    name = details.get('name', 'Unknown')
    country = details.get('origin_country', ['Unknown'])[0] if details.get('origin_country') else 'Unknown'
    
    status_map = {'Returning Series': '连载中', 'Ended': '已完结', 'Canceled': '已取消', 'Pilot': '试播集', 'In Production': '制作中'}
    status = status_map.get(details.get('status'), details.get('status', 'Unknown'))
    
    nxt = details.get('next_episode_to_air')
    last = details.get('last_episode_to_air')
    next_date = nxt['air_date'] if nxt else (last['air_date'] if last else '待定')
    if status == '已完结': next_date = '已完结'
    
    auto_time = "18:00" if country in ['US','GB','CA'] else ("20:00" if country in ['CN','TW'] else ("23:00" if country in ['JP','KR'] else "待定"))
    
    print(f"✅ 识别到: 《{name}》 (状态: {status})")
    
    conn = sqlite3.connect(DB_FILE)
    try:
        conn.execute('''INSERT INTO shows (tmdb_id, name, resource_time, status, next_air_date) VALUES (?,?,?,?,?)
            ON CONFLICT(tmdb_id) DO UPDATE SET resource_time=excluded.resource_time, status=excluded.status, next_air_date=excluded.next_air_date''',
            (tmdb_id, name, auto_time, status, next_date))
    except Exception as e: print(f"❌ Error: {e}")
    conn.commit()
    conn.close()
    sync_show_to_local(tmdb_id, name)

def generate_local_report():
    write_log("开始执行日报检查...")
    today_str = get_beijing_now().strftime('%Y-%m-%d')
    
    conn = sqlite3.connect(DB_FILE)
    conn.row_factory = sqlite3.Row
    cursor = conn.cursor()
    cursor.execute('''
        SELECT s.name, s.resource_time, e.season, e.episode, e.title
        FROM episodes e JOIN shows s ON e.tmdb_id = s.tmdb_id
        WHERE e.air_date = ? ORDER BY s.resource_time ASC
    ''', (today_str,))
    rows = cursor.fetchall()
    conn.close()

    msg_lines = [f"📅 <b>{today_str} 追剧日报</b>", "="*20]
    if not rows:
        msg_lines.append("🍵 今天无更新。")
        write_log(f"检查完成：日期 {today_str} 无更新")
    else:
        for row in rows:
            line = f"⏰ <code>[{row['resource_time']}]</code> <b>{row['name']}</b>\n   S{row['season']}E{row['episode']}"
            if row['title']: line += f" - {row['title']}"
            msg_lines.append(line)
            msg_lines.append("-" * 20)
        write_log(f"检查完成：发现 {len(rows)} 个更新，准备发送")

    send_telegram_message("\n".join(msg_lines))

def refresh_all_shows():
    conn = sqlite3.connect(DB_FILE)
    cursor = conn.cursor()
    cursor.execute("SELECT tmdb_id, name FROM shows")
    shows = cursor.fetchall()
    conn.close()
    for row in shows: sync_show_to_local(row[0], row[1])

def main():
    init_db()
    
    # --- 关键修改：处理 auto 参数 ---
    if len(sys.argv) > 1 and sys.argv[1] == "auto":
        try:
            generate_local_report()
        except Exception as e:
            write_log(f"严重错误: {e}")
            print(f"Error: {e}")
        return
    # -----------------------------

    while True:
        print(f"\n📂 数据文件: {DB_FILE}")
        print("1. ➕ 订阅")
        print("2. 🚀 发送日报")
        print("3. 🔄 刷新缓存")
        print("4. 🔔 测试通知")
        print("5. 👋 退出")
        c = input("选: ")
        if c == '1': subscribe_show()
        elif c == '2': generate_local_report()
        elif c == '3': refresh_all_shows()
        elif c == '4': test_telegram()
        elif c == '5': break

if __name__ == "__main__":
    main()
