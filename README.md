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
└── docker-compose.yml             # 本機 PostgreSQL
```

集成測試跟被測程式碼放在同一個套件目錄下，檔名以 `_integration_test.go` 結尾並加上 `//go:build integration`（例如 `internal/repository/user_repository_integration_test.go`），預設 `go test ./...` 不會執行，需連線真實 PostgreSQL 時才用 `go test -tags=integration ./...` 執行。

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
| `domainrepo.ContainerRuntime` 介面（Domain 層）+ `internal/docker`（實作） | 跟 Repository 同一套 DIP 模式：Use Case 只認識「建立/啟動/停止/移除容器」這個介面，不直接依賴 Docker SDK，方便測試（mock）跟未來換成 K8s |
| `ContainerService.Create` 的補償回滾（compensating action） | Docker 建立容器跟存 DB 是兩個獨立系統，無法用單一 transaction 保證原子性；DB 寫入失敗時主動呼叫 `Remove` 清掉孤兒容器，盡量避免留下沒有紀錄、但持續佔資源的容器 |
| Start/Stop/Delete 前先做擁有權檢查（`container.UserID == userID`） | `userID` 從 JWT 解出來，但 container 是用 ID 查的；不檢查擁有權的話，使用者 A 只要知道某個 container 的 ID 就能操作使用者 B 的容器 |
| JWT 驗證強制限定 HMAC 演算法 | 防範 "alg confusion" 攻擊——不能讓 token 自己宣告的簽名演算法決定驗證方式，否則攻擊者可能用 `alg: none` 或非對稱演算法繞過驗證 |
| 檔案上傳：實際存檔路徑用 `{FileStoragePath}/{userID}/{file 自己的 UUID}`，原始檔名只當中繼資料存進 DB | 從根本避免 path traversal（路徑遍歷）攻擊——不管使用者把檔名取成什麼字串，都不會影響實際寫入磁碟的路徑，不需要靠黑名單擋特殊符號；per-user 資料夾則對應 PDF 要求、也方便日後依使用者做整批清理 |
| `FileService.Upload` 簽名只收 `io.Reader` + 純量參數，不收 `*multipart.FileHeader` | Use Case 層不該知道「這次上傳是透過 HTTP multipart」這件事，由 Handler（Interface Adapters 層）負責把 HTTP 特有的型別轉換成跟來源無關的普通參數 |
| `ContainerService.CreateAsync` 重用既有的同步 `Create`，包一層 goroutine | 長耗時的容器建立不用讓 HTTP request 卡住等待；背景工作直接呼叫已經寫好、已經測過的 `Create`，不重複實作一份建立邏輯 |
| goroutine 拿 `entity.Job` 的「副本」而非跟呼叫者共用同一個指標 | `go test -race` 抓到的真實 data race：原本同一個 `*entity.Job` 指標同時被「回傳給呼叫者」跟「背景 goroutine 持續寫入」兩邊共用，沒有同步機制；修法是讓 goroutine 操作自己的副本，呼叫者拿到的指標之後不會再被修改 |
| Graceful Shutdown 的關閉順序：HTTP server → `ContainerService.Wait()` → `LogManager.Close()` | 後兩者都是「背景仍可能在寫東西」的元件；如果 log 系統先關，HTTP shutdown 或容器背景工作還在跑時呼叫 `WriteLog` 會對已關閉的 channel 寫入而 panic，所以一定要最後關 |
| Repository 集成測試用 `tx := db.Begin()` + `defer tx.Rollback()`，不用 `t.Cleanup` 手動刪資料 | `*gorm.DB` 跟 `db.Begin()` 回的 transaction 是同一個型別，Repository 不需要為了測試改任何 production code；只要不呼叫 `Commit()`，測試裡的所有寫入保證不會真正留在資料庫，比手動對應每筆 insert 的清理動作更不容易出錯 |
| `ContainerHandler` 依賴 `handler` package 自己定義的 `containerUsecase` 介面，而非 `*container.ContainerService` 具體型別 | 跟 Repository／`ContainerRuntime` 同一套 DIP：消費端（Handler）只定義自己用得到的方法，測試時可以注入 mock，不需要真的連 Docker daemon 或資料庫；`*container.ContainerService` 本來就實作了這些方法，`main.go` 的組裝完全不用改 |

---

## 安裝與啟動

### 前置需求
- Go 1.25.6+
- Docker / Docker Compose（用於啟動本機 PostgreSQL，以及容器功能本身需要連線 Docker daemon）

### 步驟

```bash
# 1. 啟動 PostgreSQL
docker-compose up -d

# 2. 安裝相依套件
go mod download

# 3. 建一個 .env（範例值，JWT_SECRET 必填，其餘有預設值）
cat > .env <<EOF
JWT_SECRET=please-change-me
EOF

# 4. 啟動服務
go run ./cmd/api

# 5. 執行單元測試
go test ./...

# 6. 執行集成測試（需要 PostgreSQL 已啟動）
go test -tags=integration ./...
```

啟動後可以打開 `http://localhost:8080/swagger/index.html` 看互動式 API 文檔，或 `http://localhost:8080/health` 確認服務存活。

---

## API 設計

完整、可互動的版本請看 `/swagger/index.html`；這裡列路徑速覽：

```
POST   /api/v1/auth/register
POST   /api/v1/auth/login

GET    /api/v1/containers
POST   /api/v1/containers              # 非同步：立刻回 202 + Job，背景建立容器
GET    /api/v1/containers/{id}
POST   /api/v1/containers/{id}/start
POST   /api/v1/containers/{id}/stop
DELETE /api/v1/containers/{id}

GET    /api/v1/files
POST   /api/v1/files                   # multipart/form-data，欄位名稱 "file"
DELETE /api/v1/files/{id}

GET    /api/v1/jobs/{id}               # 查詢非同步任務狀態：pending/running/success/failed
```

除了 `/auth/*`，其餘都需要在 `Authorization: Bearer {token}` header 帶上登入取得的 JWT。

---

## 開發進度

### 已完成
- [x] Domain 層：entity（User / Container / File / Job）
- [x] Domain 層：Repository 介面（含 `ErrNotFound` sentinel error，避免 GORM 錯誤洩漏到 Use Case 層）
- [x] Interface Adapters 層：GORM Model（`TableName` / `ToDomain` / `XxxFromDomain`）
- [x] Interface Adapters 層：Repository 實作（皆有 compile-time interface check）
- [x] `configs/config.go`：DB 連線設定、環境變數讀取、`JWT_SECRET` fail-fast
- [x] `cmd/api/main.go`：DI 組裝、GORM AutoMigrate、Gin server 啟動
- [x] User 垂直切片：`UserService`（bcrypt 密碼雜湊、JWT 簽發）+ `UserHandler` + `/api/v1/auth/register`、`/api/v1/auth/login`（已驗證端到端，含單元測試）
- [x] JWT Auth Middleware：驗證 token、防 alg-confusion、注入 `userID` 到 context
- [x] Container 垂直切片：`domainrepo.ContainerRuntime` 介面 + `internal/docker`（Docker SDK 實作）+ `ContainerService`（含建立失敗的補償回滾、擁有權檢查）+ `ContainerHandler` + `/api/v1/containers/*`（已驗證端到端對真實 Docker daemon，含單元測試）
- [x] File 垂直切片：`FileService`（per-user 資料夾、UUID 檔名避免 path traversal、上傳失敗補償回滾）+ `FileHandler`（拆解 multipart upload）+ `/api/v1/files/*`（已驗證端到端，含單元測試）
- [x] Swagger API 文檔（swaggo）：Auth / Container / File / Job 全部 endpoint 都有 annotation，UI 在 `/swagger/index.html`
- [x] **[進階] 非同步任務處理**：`ContainerService.CreateAsync` 立刻回傳 Job（`202`），背景 goroutine 跑實際建立流程，狀態 `pending → running → success/failed`；`GET /jobs/{id}` 查詢進度；`sync.WaitGroup` 追蹤 in-flight 工作，為 Graceful Shutdown 預留掛勾（已驗證端到端，含單元測試，`-race` 乾淨）
- [x] **Task 2：可擴展日誌管理系統**（`internal/logger/`）：`LogEntry`/`LogWriter`/`LogReader`/`LogHandler` 四個介面 + `LogManager`；可插拔儲存（`FileLogStore` JSON Lines / `InMemoryLogStore`）、分級過濾、channel + 單一背景 goroutine 的非同步寫入（保證順序）、`RegisterLogHandler` 擴展點、結構化 JSON 輸出（[進階] 已實作，Zero-Allocation [進階] 設計方向寫在文檔未實作）。整合為 Task 1 的 `LoggingMiddleware`，已驗證端到端、含單元測試與可執行範例。設計文檔見 [`internal/logger/DESIGN.md`](internal/logger/DESIGN.md)
- [x] **[加分] Graceful Shutdown**：`http.Server` + 監聽 `SIGINT`/`SIGTERM`，關閉順序為 `srv.Shutdown`（等現有 request 做完，10 秒逾時）→ `ContainerService.Wait()`（等背景非同步建立容器的工作做完）→ `LogManager.Close()`（最後才關，避免對已關閉的 channel 寫入而 panic）。已驗證：故意在容器非同步建立中送出關閉訊號，程式等它真正跑完（Job 狀態 `success`、容器確實建立）才退出
- [x] **Repository 集成測試**：User / Container / File / Job 四個 repository 對真實 PostgreSQL 驗證 `Save`/`FindByXxx`/`Update`/`Delete` 與 `ErrNotFound` 轉換。檔名以 `_integration_test.go` 結尾並加 `//go:build integration`，預設 `go test ./...` 不會跑；每個測試包在一個 DB transaction 裡，結束時 `Rollback`，不會在資料庫留下任何測試資料，也不需要手動清理。執行：`go test -tags=integration ./internal/repository/...`
- [x] **Handler 層測試（代表範例）**：`ContainerHandler` 透過 `httptest` + `gin.TestMode` + 自定義 `containerUsecase` 介面（讓 `ContainerHandler` 依賴介面而非 `*container.ContainerService` 具體型別，跟 Repository/ContainerRuntime 同一套 DIP 模式）注入 mock，涵蓋成功路徑、400（binding 失敗）、404/403/500（`handleServiceError` 的三個分支）。其餘 3 個 Handler 因邏輯結構雷同，目前未逐一補測試

### 待完成
- [ ] 並發控制（Concurrency Control，進階，時間允許才做）