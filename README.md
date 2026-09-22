# GoCron (Lightweight Self-Hosted Easycron Alternative)

A high-performance, ultra-lightweight web cron service and scheduler built with Go. Designed as an internal, self-hosted replacement for services like Easycron to execute HTTP GET/POST jobs across multiple endpoints efficiently.

---

## ✨ Features

- **Ultra Low Footprint:** Operates on **< 30 MB RAM** with near-zero idle CPU usage.
- **Single Monolith Binary:** All HTML templates, static assets, and OpenAPI specs are bundled via `//go:embed`. Zero runtime external dependencies.
- **Easycron Parity & Flexibility:**
  - **Preset Intervals:** Every X minutes, hours, or days.
  - **Standard Cron Expression:** Full support for 5-field and 6-field Easycron formats.
  - **Manual Matrix Calendar Selector:** Flexible grid selector for Minutes (0-59), Hours (0-23), Days (1-31), Months (Jan-Dec), and Days of Week (Sun-Sat).
  - **Dynamic EPD Calculation:** Automatic real-time calculation of daily execution frequency (Executions Per Day).
  - **Executions Modal:** Interactive view displaying 10 past execution logs and 10 future predicted schedules using `cron.Schedule.Next()`.
- **Pure Go Embedded Database:** Zero-config SQLite database via `modernc.org/sqlite` (no CGO compiler required) configured with **WAL Mode (`PRAGMA journal_mode=WAL`)** for high concurrency.
- **Auto-Pruning:** Daily background worker automatically prunes execution logs older than 7 days.
- **Interactive API Documentation:** Embedded Scalar API reference available out-of-the-box at `/docs`.
- **Simple Session Auth:** Lightweight, secure 1-hour cookie-based admin login with custom `.env` credentials.

---

## 🛠 Tech Stack

- **Core Backend:** Go (Golang 1.22+)
- **HTTP Router:** Chi v5 (`github.com/go-chi/chi/v5`)
- **Scheduler Engine:** `github.com/robfig/cron/v3`
- **Database:** SQLite (`modernc.org/sqlite` - Pure Go, CGO-Free)
- **Frontend UI:** Go `html/template` + Tailwind CSS CDN + HTMX
- **API Documentation:** Scalar (`@scalar/api-reference`) + OpenAPI 3.0.3

---

## 🚀 Getting Started

Follow these steps to set up and run GoCron on your local machine or server.

### Prerequisites

- [Go](https://go.dev/dl/) (version 1.22 or higher)
- Git

### 1. Clone the Repository

```bash
git clone https://github.com/IrgiRamz/icx-cron.git
cd icx-cron
```

### 2. Environment Configuration

Copy the example environment file and customize your static credentials:

```bash
cp .env.example .env
```

Edit your `.env` file:

```env
PORT=8080
DB_PATH=cron.db
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123!
SESSION_SECRET=your-random-secret-key-here
API_KEY=iconix-cron-secret-key
MAX_WORKERS=30
LOG_RETENTION_DAYS=7
```

### 3. Install Dependencies

Download and tidy up required Go modules:

```bash
go mod tidy
```

### 4. Build the Executable

Compile the project into a compact, production-ready binary:

#### Linux / macOS:
```bash
go build -ldflags="-s -w" -o cron-service ./cmd/server
```

#### Windows (PowerShell / CMD):
```bash
go build -ldflags="-s -w" -o cron-service.exe ./cmd/server
```

### 5. Run the Application

#### Development Mode:
```bash
go run ./cmd/server
```

#### Production Mode:

##### Linux / macOS:
```bash
./cron-service
```

##### Windows:
```bash
./cron-service.exe
```

Once started, open your browser and navigate to:
- **Dashboard:** [http://localhost:8080](http://localhost:8080) (Log in using credentials set in `.env`)
- **API Documentation:** [http://localhost:8080/docs](http://localhost:8080/docs)

---

## 🐧 Production Deployment (Systemd on Linux VPS)

To keep GoCron running continuously in the background and auto-restart on server reboots:

1. Create a Systemd service file:

```bash
sudo nano /etc/systemd/system/gocron.service
```

2. Paste the following configuration (adjust paths and user accordingly):

```ini
[Unit]
Description=GoCron Scheduler Service
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/icx-cron
ExecStart=/opt/icx-cron/cron-service
Restart=always
RestartSec=5
EnvironmentFile=/opt/icx-cron/.env
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

3. Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable gocron
sudo systemctl start gocron
```

4. Check service status:

```bash
sudo systemctl status gocron
```

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).