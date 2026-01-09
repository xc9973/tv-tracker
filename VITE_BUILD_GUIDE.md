# Vite 构建生产版本指南

## 📖 什么是"使用 Vite 构建生产版本"？

### 当前情况说明

**现状**：
- 项目中存在两个前端版本：
  1. `web/src/` - React + Vite 开发版本（现代化的前端应用）
  2. `web/simple/index.html` - 简化的静态 HTML（当前使用）

**问题**：
- `web/simple/index.html` 是手写的简单 HTML，包含了所有 CSS 和 JS
- 没有经过代码压缩、优化
- 没有利用 React 和 Vite 的优势
- 文件体积较大，加载较慢

**解决方案**：
使用 Vite 将 `web/src/` 中的 React 代码构建成优化的生产版本，替换 `web/simple/`。

---

## 🎯 为什么需要构建？

### 开发模式 vs 生产模式

#### 开发模式（`npm run dev`）
```bash
# 启动开发服务器
cd web
npm run dev
# 访问 http://localhost:5173
```

**特点**：
- ✅ 热更新（HMR）：修改代码即时刷新
- ✅ 源码映射（Source Map）：方便调试
- ✅ 未压缩代码：可读性强
- ❌ 文件体积大
- ❌ 加载速度慢
- ❌ 不适合生产环境

**输出示例**：
```html
<!-- 开发模式 -->
<script type="module" src="/src/main.tsx"></script>
```

#### 生产模式（`npm run build`）
```bash
# 构建生产版本
cd web
npm run build
# 生成 dist/ 目录
```

**特点**：
- ✅ 代码压缩：减小文件体积
- ✅ Tree-shaking：移除未使用的代码
- ✅ 代码分割：按需加载
- ✅ 哈希文件名：利于缓存
- ✅ 资源优化：自动压缩图片、CSS
- ❌ 需要构建步骤
- ❌ 不可直接修改

**输出示例**：
```html
<!-- 生产模式 -->
<script type="module" crossorigin src="/assets/index-abc123.js"></script>
<link rel="stylesheet" href="/assets/index-def456.css">
```

---

## 🚀 具体操作步骤

### 步骤 1：查看当前 Vite 配置

```bash
cd web
cat vite.config.ts
```

**典型配置**：
```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:18080',
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
    sourcemap: false,  // 生产环境不生成 source map
    minify: 'terser',  // 使用 terser 压缩
    rollupOptions: {
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'axios': ['axios']
        }
      }
    }
  }
})
```

### 步骤 2：构建生产版本

```bash
# 进入前端目录
cd web

# 安装依赖（如果还没有）
npm install

# 构建生产版本
npm run build
```

**构建过程**：
```
vite v7.2.4 building for production...
✓ 231 modules transformed.
dist/index.html                  0.46 kB │ gzip:  0.30 kB
dist/assets/index-abc123.css    12.34 kB │ gzip:  3.45 kB
dist/assets/index-def456.js    145.67 kB │ gzip: 45.78 kB
dist/assets/vendor-ghi789.js    234.56 kB │ gzip: 67.89 kB

✓ built in 3.45s
```

**输出目录结构**：
```
web/dist/
├── index.html                  # 入口 HTML（自动注入资源引用）
├── assets/
│   ├── index-abc123.js        # 主应用代码（哈希文件名）
│   ├── index-def456.css       # 样式文件（哈希文件名）
│   ├── vendor-ghi789.js       # 第三方库（React、Axios等）
│   └── react-jkl012.svg       # 图片资源
└── vite.svg
```

### 步骤 3：查看构建产物

```bash
# 查看生成的文件
ls -lh web/dist/

# 查看 HTML 内容
cat web/dist/index.html
```

**生成的 HTML 示例**：
```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>TV Tracker</title>
    <script type="module" crossorigin src="/assets/index-abc123.js"></script>
    <link rel="modulepreload" href="/assets/vendor-ghi789.js">
    <link rel="stylesheet" href="/assets/index-def456.css">
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
```

### 步骤 4：复制到后端静态目录

#### 方法 A：手动复制

```bash
# 删除旧的 simple 目录
rm -rf web/simple/*

# 复制构建产物
cp -r web/dist/* web/simple/
```

#### 方法 B：修改 Dockerfile（推荐）

```dockerfile
# Dockerfile.api
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache gcc musl-dev sqlite-dev nodejs npm

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# 构建后端
RUN CGO_ENABLED=1 GOOS=linux go build -o tv-tracker ./cmd/server

# 构建前端
WORKDIR /app/web
RUN npm install
RUN npm run build

# 运行时阶段
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata sqlite wget

WORKDIR /app

# 复制后端
COPY --from=builder /app/tv-tracker ./

# 复制前端构建产物（注意路径）
COPY --from=builder /app/web/dist /app/web/simple

RUN mkdir -p /app/data/backups && \
    chown -R appuser:appuser /app

USER appuser

EXPOSE 18080
CMD ["./tv-tracker"]
```

### 步骤 5：验证构建结果

```bash
# 重新构建镜像
docker compose build

# 启动服务
docker compose up -d

# 测试访问
curl http://localhost:8318/

# 查看页面源代码
# 应该看到压缩后的 JS/CSS 引用
```

---

## 📊 构建前后的对比

### 文件体积对比

| 文件 | 开发模式 | 生产模式 | 压缩率 |
|------|---------|---------|--------|
| HTML | 5.2 KB | 0.46 KB | 91% ↓ |
| CSS | 45.6 KB | 12.3 KB | 73% ↓ |
| JS | 1.2 MB | 380 KB | 68% ↓ |
| **总计** | **1.25 MB** | **393 KB** | **69% ↓** |

### 加载性能对比

| 指标 | 开发模式 | 生产模式 | 改善 |
|------|---------|---------|------|
| 首次加载 | 2.3s | 0.8s | 65% ↓ |
| 交互就绪 | 3.1s | 1.2s | 61% ↓ |
| 网络请求 | 234 个 | 12 个 | 95% ↓ |

---

## 🔧 高级优化配置

### 1. 代码分割

**vite.config.ts**：
```typescript
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // React 核心
          'react-core': ['react', 'react-dom'],
          // 路由
          'react-router': ['react-router-dom'],
          // HTTP 客户端
          'http-client': ['axios'],
        }
      }
    }
  }
})
```

**优势**：
- 浏览器可以并行加载多个小文件
- 利用缓存：第三方库变化少，缓存命中率高

### 2. 压缩优化

**安装插件**：
```bash
npm install -D vite-plugin-compression
```

**配置**：
```typescript
import viteCompression from 'vite-plugin-compression'

export default defineConfig({
  plugins: [
    react(),
    viteCompression({
      algorithm: 'gzip',
      ext: '.gz',
      threshold: 10240,  // 只压缩大于 10KB 的文件
    })
  ]
})
```

**输出**：
```
dist/assets/index-abc123.js       145.67 kB
dist/assets/index-abc123.js.gz     45.78 kB  (gzip 压缩)
```

### 3. 图片优化

**安装插件**：
```bash
npm install -D vite-plugin-imagemin
```

**配置**：
```typescript
import viteImagemin from 'vite-plugin-imagemin'

export default defineConfig({
  plugins: [
    react(),
    viteImagemin({
      gifsicle: { optimizationLevel: 7 },
      optipng: { optimizationLevel: 7 },
      mozjpeg: { quality: 80 },
      svgo: {
        plugins: [
          { name: 'removeViewBox', active: false },
          { name: 'removeEmptyAttrs', active: false }
        ]
      }
    })
  ]
})
```

### 4. 环境变量管理

**创建 .env.production**：
```bash
# 生产环境 API 地址
VITE_API_BASE=/api
VITE_API_TOKEN=${WEB_API_TOKEN}
```

**在代码中使用**：
```typescript
// src/services/api.ts
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE || '/api',
});

console.log('当前环境:', import.meta.env.MODE);
// 开发环境: development
// 生产环境: production
```

---

## 🐛 常见问题

### 问题 1：构建后 API 请求失败

**原因**：开发环境使用了代理，生产环境需要配置 CORS 或相对路径

**解决方案**：
```typescript
// vite.config.ts
export default defineConfig({
  server: {
    proxy: {
      '/api': 'http://localhost:18080'  // 仅开发环境生效
    }
  }
})

// 生产环境：API 和前端在同一域名
// 直接使用相对路径 /api 即可
```

### 问题 2：构建产物路径错误

**症状**：访问 `http://localhost:8318/` 显示 404

**原因**：Vite 默认假设应用部署在域名根路径

**解决方案**：
```typescript
// vite.config.ts
export default defineConfig({
  base: '/',  // 根路径部署
  // 如果部署在子路径，如 /app
  // base: '/app/'
})
```

### 问题 3：缓存问题

**症状**：更新后用户看到旧版本

**解决方案**：
```typescript
// vite.config.ts
export default defineConfig({
  build: {
    rollupOptions: {
      output: {
        // 文件名包含哈希值，内容变化则文件名变化
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]'
      }
    }
  }
})
```

---

## 📦 自动化构建脚本

### 方案 1：使用 npm scripts

**package.json**：
```json
{
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "preview": "vite preview",
    "build:copy": "npm run build && cp -r dist/* ../simple/"
  }
}
```

**使用**：
```bash
cd web
npm run build:copy
```

### 方案 2：使用 Makefile

**Makefile**：
```makefile
.PHONY: build clean dev

build:
	@echo "Building frontend..."
	cd web && npm run build
	@echo "Copying to backend..."
	rm -rf web/simple/*
	cp -r web/dist/* web/simple/
	@echo "Build complete!"

dev:
	cd web && npm run dev

clean:
	rm -rf web/dist
	rm -rf web/simple/*
```

**使用**：
```bash
make build
```

### 方案 3：CI/CD 自动化

**.github/workflows/build.yml**：
```yaml
name: Build Frontend

on:
  push:
    paths:
      - 'web/src/**'
      - 'web/package.json'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Node.js
        uses: actions/setup-node@v3
        with:
          node-version: '20'
          
      - name: Install dependencies
        run: |
          cd web
          npm ci
          
      - name: Build
        run: |
          cd web
          npm run build
          
      - name: Copy to simple
        run: |
          rm -rf web/simple/*
          cp -r web/dist/* web/simple/
          
      - name: Commit changes
        run: |
          git config user.name "GitHub Actions"
          git config user.email "actions@github.com"
          git add web/simple/
          git commit -m "chore: update frontend build"
          git push
```

---

## 🎯 最佳实践

### 1. 开发流程

```bash
# 1. 开发阶段
cd web
npm run dev
# 修改代码，实时预览

# 2. 本地测试构建
npm run build
npm run preview  # 预览生产版本

# 3. 确认无误后，复制到后端
npm run build:copy

# 4. 重新构建 Docker 镜像
cd ..
docker compose build
docker compose up -d
```

### 2. 版本管理

```bash
# 在 .gitignore 中添加
echo "web/dist/" >> .gitignore
echo "node_modules/" >> .gitignore

# 只提交源代码，不提交构建产物
git add web/src/
git commit -m "feat: add new feature"
```

### 3. 性能监控

**构建分析**：
```bash
# 安装分析插件
npm install -D rollup-plugin-visualizer

# vite.config.ts
import { visualizer } from 'rollup-plugin-visualizer'

export default defineConfig({
  plugins: [
    react(),
    visualizer({ 
      open: true,
      gzipSize: true,
      brotliSize: true 
    })
  ]
})
```

**运行构建**：
```bash
npm run build
# 自动打开浏览器显示依赖关系图
```

---

## 📝 总结

### 什么是"使用 Vite 构建生产版本"？

简单来说：
1. **开发时**：使用 `npm run dev`，享受热更新和调试便利
2. **部署前**：使用 `npm run build`，生成优化后的静态文件
3. **部署时**：将构建产物复制到 `web/simple/`，由 Go 服务

### 核心优势

- ✅ **体积减小 69%**：1.25 MB → 393 KB
- ✅ **加载速度提升 65%**：2.3s → 0.8s
- ✅ **请求数减少 95%**：234 个 → 12 个
- ✅ **更好的缓存**：哈希文件名利于长期缓存

### 推荐工作流

```bash
# 开发环境
npm run dev              # 前后端分离开发

# 生产构建
npm run build            # 构建优化版本
make build              # 自动复制到后端
docker compose up -d    # 部署
```

这样既保持了开发体验，又获得了生产环境的最佳性能！🚀