Here is your full **14-step checklist to build QT-1** — a high-performance, AI-safe gateway with a configurable UI. This version includes both the **Go backend (middleware proxy)** and a **React + sqllite-based frontend dashboard**.

---

# 📘 Project Checklist: Build **QT-1** — Configurable AI Middleware with UI

---

## 🧠 Overview

**QT-1** is a standalone, high-performance middleware written in Go that acts as a configurable proxy in front of any AI agent endpoint. It includes an **admin dashboard** (React + Tailwind) to configure moderation, relevance scoring, kill switch behavior, endpoint redirection, and logging.

---

## ✅ 14-Step Project Plan

---

### 1. 🧱 Project Initialization (Monorepo)

* [ ] Create a folder structure:

  ```
  qt1/
  ├── backend/      # Go middleware
  ├── frontend/     # React dashboard
  ├── config/       # Shared config + schema
  ├── .env
  └── README.md
  ```
* [ ] Initialize Go module: `go mod init github.com/yourname/qt1/backend`
* [ ] Initialize React app (Vite or Next.js): `npx create-react-app qt1/frontend --template typescript`
* [ ] Set up Git + .gitignore

---

### 2. ⚙️ Config Loader (Go)

* [ ] `config/config.go`: define a struct for middleware config
* [ ] Support YAML loading and `.env` support (OpenAI keys, ports, etc.)
* [ ] Validate fields like `target_url`, thresholds, log paths
* [ ] Add live reload support (watch file or REST update)

---

### 3. 🌐 HTTP Proxy Middleware (Go)

* [ ] Launch HTTP server on `:8080`
* [ ] Create `/chat` endpoint:

  * [ ] Accept JSON: `{ user_id, session_id, message }`
  * [ ] Run through kill switch → moderation → relevance → logging
* [ ] Forward request to `config.TargetURL` using HTTP client
* [ ] Return downstream response to client

---

### 4. 🧼 Input Moderation (Go)

* [ ] `filter.go`: Build a basic regex word filter (violence, self-harm, NSFW, etc.)
* [ ] Add optional OpenAI moderation API support
* [ ] Use `.env` for `OPENAI_API_KEY`
* [ ] Block unsafe requests with custom response

---

### 5. 🧠 Relevance Filtering (Go)

* [ ] Accept optional embedding vector for domain scope
* [ ] Compute cosine similarity between incoming message and domain goal
* [ ] Block/allow based on `config.relevance_threshold`
* [ ] Use OpenAI embeddings or local model endpoint

---

### 6. 🛑 Kill Switch System (Go)

* [ ] JSON file or DB table: list of `blocked_users` and `blocked_sessions`
* [ ] Check on every incoming request
* [ ] Add REST API (`/api/block`, `/api/unblock`) for frontend to manage

---

### 7. 🪵 Logging System (Go)

* [ ] Log to file: `chatlogs.txt`
* [ ] Include: timestamp, user ID, session ID, moderation result, relevance score, final action
* [ ] Optional: support SQLite

---

### 8. 🧪 Admin API Endpoints (Go)

* [ ] `/api/config` (GET, PUT)
* [ ] `/api/logs` (GET with filters)
* [ ] `/api/status` (health ping)
* [ ] `/api/users` (if sqllite used)
* [ ] Secure endpoints using JWT if possible with SQLLite

---

### 9. 🧰 Frontend UI Setup (React + Tailwind)

* [ ] Create base layout with navigation (Dashboard, Logs, Config, Users)
* [ ] Install Tailwind CSS
* [ ] Add global context for `auth`, `settings`, and `session`

---

### 11. 🧮 Dashboard Views (React)

* [ ] **Config Panel**

  * [ ] Toggle moderation, relevance
  * [ ] Set thresholds, keywords
  * [ ] Change proxy endpoint
* [ ] **Log Viewer**

  * [ ] Table of logs
  * [ ] Filter by user, status
* [ ] **Kill Switch Control**

  * [ ] View all blocked sessions/users
  * [ ] Toggle block status
* [ ] **Health Status Page**

  * [ ] Ping Go API and show service status

---

### 12. 🔄 Backend–Frontend Sync

* [ ] React frontend hits Go API to update settings
* [ ] Go middleware reads updated settings (poll or websocket or auto-reload file)
* [ ] Use optimistic UI updates with config file response

---

### 13. 🐳 Dockerize Everything

* [ ] `Dockerfile` for Go backend
* [ ] `Dockerfile` for React frontend (or `serve` static files)
* [ ] `docker-compose.yml`:

  ```yaml
  services:
    qt1-backend:
      build: ./backend
      ports: [8080:8080]
    qt1-frontend:
      build: ./frontend
      ports: [3000:3000]
  ```
* [ ] Add environment variables to `.env` and pass into containers

---

### 14. 📦 Polish & Launch

* [ ] Add CI (GitHub Actions or Railway for deploy)
* [ ] Build production frontend with `vite build` or `next build`
* [ ] Set up NGINX or Cloudflare proxy (optional)
* [ ] Finalize `README.md` with:

  * Usage instructions
  * Endpoint details
  * Security recommendations
* [ ] Roadmap (e.g. WebSocket support, metrics, multiple agents)

---

Would you like this output as a downloadable `.md` file or to begin generating the code scaffolding for `backend/` and `frontend/`?
