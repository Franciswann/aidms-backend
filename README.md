# AIDMS Backend

容器管理系統後端服務。

提供使用者管理、檔案上傳、Docker 容器操作（建立/啟動/停止/刪除）的 RESTful API，並包含一套可擴展的日誌管理系統。

> 本專案目前仍在開發中，下方「開發進度」章節會持續更新實際完成狀態。

---

## 技術棧

| 類別 | 選用 |
|---|---|
| 語言 | Go 1.25.6 |
| Web 框架 | Gin |
| ORM | GORM |
| 資料庫 | PostgreSQL 16 |
| 容器操作 | Docker SDK for Go |
| API 文檔 | swaggo/swag |
| 測試 | testify |

---

## 架構設計

採用 Clean Architecture 四層架構，依賴方向僅能由外向內，內層不得 import 任何第三方套件。

```
Frameworks & Drivers     cmd/api/main.go, Gin, GORM, Docker SDK, swaggo
        ↓
Interface Adapters       internal/handler/, internal/repository/, internal/docker/, internal/middleware/
        ↓
Use Cases                internal/usecase/{container,user,file,job}/, internal/logger/
        ↓
Domain                   internal/domain/entity/, internal/domain/repository/
```

### 目錄結構

```
.
├── cmd/api/                       # 程式進入點，負責 DI 組裝與 server 啟動
├── configs/                       # 設定檔與環境變數讀取
├── internal/
│   ├── domain/
│   │   ├── entity/                # Domain 實體（User, Container, File, Job）
│   │   └── repository/            # Repository 介面（由 Use Case 定義需求）
│   ├── repository/                # Repository 介面的 GORM 實作 + Model
│   ├── usecase/                   # 商業邏輯層
│   ├── docker/                    # Docker SDK 封裝
│   ├── handler/                   # Gin HTTP Handler
│   ├── middleware/                # 日誌、認證、錯誤處理中間件
│   └── logger/                    # 可擴展日誌管理系統（Task 2）
├── pkg/apperror/                  # 共用錯誤定義
├── test/                          # 集成測試
└── docker-compose.yml             # 本機 PostgreSQL
```

### 關鍵設計決策

| 決策 | 理由 |
|---|---|
| Clean Architecture | 各層職責分明，內層不依賴外層，方便替換外部工具（例如資料庫、容器引擎） |
| Domain entity 與 GORM model 分開 | 避免 Domain 層被第三方套件的 tag／型別污染，維持零依賴，方便單獨測試 |
| Repository 介面放在 Domain 層，實作放在 Interface Adapters 層 | Use Case 定義「需要什麼」，外層決定「怎麼做」，符合依賴反轉原則（DIP），未來可無痛切換 PostgreSQL/SQLite |
| 使用 UUID 而非自增 ID | 避免 ID 可被列舉猜測（安全性），且在分散式環境下不會有衝突 |
| `ContainerStatus`／`JobStatus` 自定義型別 | 限制合法值集合，編譯期就能防止寫入無效狀態，比起裸字串更安全 |
| 容器同時保留系統內部 UUID 與 Docker 容器的 `DockerID` | 對外 API 只暴露系統 UUID，對內才用 `DockerID` 呼叫 Docker SDK；未來若改用 Kubernetes，只需替換內部對應邏輯，API 合約不受影響 |
| Compile-time interface check（`var _ Interface = (*Struct)(nil)`） | 讓 Repository 實作是否完整符合介面在編譯期就能發現，不必等到執行期才出錯 |

---

## 安裝與啟動

### 前置需求
- Go 1.25.6+
- Docker / Docker Compose（用於啟動本機 PostgreSQL）

### 步驟

```bash
# 1. 啟動 PostgreSQL
docker-compose up -d

# 2. 安裝相依套件
go mod download

# 3. 建置確認
go build ./...

# 4. 執行測試
go test ./...
```

> `cmd/api/main.go` 尚未實作，目前無法以 `go run` 啟動完整 API 服務，詳見下方開發進度。

---

## API 設計（規劃中）

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login

GET    /api/v1/containers
POST   /api/v1/containers
GET    /api/v1/containers/:id
POST   /api/v1/containers/:id/start
POST   /api/v1/containers/:id/stop
DELETE /api/v1/containers/:id

GET    /api/v1/files
POST   /api/v1/files
DELETE /api/v1/files/:id

GET    /api/v1/jobs/:id
```

---

## 開發進度

### 已完成
- [x] Domain 層：entity（User / Container / File / Job）
- [x] Domain 層：Repository 介面
- [x] Interface Adapters 層：GORM Model（`TableName` / `ToDomain` / `XxxFromDomain`）
- [x] Interface Adapters 層：Repository 實作（皆有 compile-time interface check）

### 待完成
- [ ] `configs/config.go`：DB 連線設定、環境變數讀取
- [ ] `cmd/api/main.go`：DI 組裝、GORM AutoMigrate、Gin server 啟動
- [ ] Use Case 層：UserService、ContainerService、FileService、JobService
- [ ] Docker SDK Adapter（`internal/docker/`）
- [ ] Gin HTTP Handlers（`internal/handler/`）
- [ ] 日誌管理系統（`internal/logger/`，Task 2，整合為 Task 1 的 logging middleware）
- [ ] JWT Auth Middleware
- [ ] 檔案上傳功能
- [ ] 單元測試與集成測試
- [ ] Swagger API 文檔
- [ ] 非同步任務處理（Async Job，進階）
- [ ] 並發控制（Concurrency Control，進階）
- [ ] Graceful Shutdown（加分項）