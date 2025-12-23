import requests

API_KEY = "d7a4995d12efa9c242e823548c45ce9e"
BASE_URL = "https://api.themoviedb.org/3"

def search_tv(query):
    """搜索剧集"""
    url = f"{BASE_URL}/search/tv"
    params = {"api_key": API_KEY, "query": query, "language": "zh-CN"}
    try:
        res = requests.get(url, params=params, timeout=5)
        data = res.json()
        if data.get('results'):
            return data['results'][0]
    except:
        pass
    return None

def get_show_details(tmdb_id):
    """获取剧集的基础详情（主要是为了知道它一共有几季）"""
    url = f"{BASE_URL}/tv/{tmdb_id}"
    params = {"api_key": API_KEY, "language": "zh-CN"}
    try:
        res = requests.get(url, params=params)
        if res.status_code == 200:
            return res.json()
    except:
        pass
    return None

def get_season_episodes(tmdb_id, season_number):
    """
    一次性获取某一季的所有集数信息
    """
    url = f"{BASE_URL}/tv/{tmdb_id}/season/{season_number}"
    params = {"api_key": API_KEY, "language": "zh-CN"}
    
    episodes_list = []
    try:
        res = requests.get(url, params=params)
        if res.status_code == 200:
            data = res.json()
            # 遍历这一季的每一集
            for ep in data.get('episodes', []):
                if ep.get('air_date'): # 只有定档的才存
                    episodes_list.append({
                        "season": ep['season_number'],
                        "episode": ep['episode_number'],
                        "title": ep['name'],
                        "overview": ep['overview'],
                        "air_date": ep['air_date']
                    })
    except:
        pass
    return episodes_list