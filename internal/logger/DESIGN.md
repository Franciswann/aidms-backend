# 可擴展日誌管理系統 — 設計文檔

## 目標

提供一套不依賴特定儲存方式、可分級過濾、非同步寫入、可後續擴展的日誌系統，並整合進主服務作為 HTTP request logging middleware。

## 核心介面

```
LogEntry   — 一筆日誌的資料（介面，不是 struct）：Level() / Message() / Timestamp() / Fields()
LogWriter  — 寫入行為：Write(entry LogEntry) error
LogReader  — 讀取/清理行為：Read(level, filter) ([]LogEntry, error)、Clear(before time.Time) error
LogHandler — 擴展點：Handle(entry LogEntry)
LogManager — 整合上述四者，是系統唯一的對外入口
```

呼叫者（例如 `LoggingMiddleware`）只認識 `LogManager` 提供的方法（`WriteLog`/`ReadLogs`/`ClearLogs`/`RegisterLogHandler`），不直接接觸任何具體的 Writer/Reader/Handler 實作——這跟主服務一路使用的依賴反轉（DIP）是同一套原則。

### 為什麼 `LogEntry` 是介面而不是 struct

`LogWriter`/`LogReader`/`LogHandler` 三者都只依賴這個介面的方法，不依賴某個具體的內部欄位佈局，未來如果想換一種 entry 的內部表示方式（例如池化重用的版本，見下方「Zero-Allocation」章節），三個介面的實作完全不用改。

### 為什麼 `Read`/`Clear` 放在同一個 `LogReader` 介面

這兩個操作都是「對已經存在的紀錄做非新增的操作」，且都需要直接碰底層儲存（檔案/資料庫），由同一個介面、通常也由同一個具體實作（例如 `FileLogStore`）負責，方便這個實作對同一份資料來源做好併發保護（見下方）。

## 可插拔儲存

提供兩個同時滿足 `LogWriter` + `LogReader` 的實作：

- **`FileLogStore`**：寫進一個 JSON Lines 格式的檔案（見下方「結構化日誌」），`Read`/`Clear` 都讀同一份檔案，用同一個 `sync.Mutex` 保護，避免「清理」跟「寫入」同時動到同一個檔案而互相破壞
- **`InMemoryLogStore`**：純記憶體的 slice，不碰磁碟，主要給測試用，但也是「可插拔」這個設計目標最直接的證明——`LogManager` 完全不需要知道，也不關心，現在接的是哪一個

`internal/logger/file_store.go` 的取捨：每次 `Write` 都重新開檔、寫入、關閉，而不是一直保持檔案開著。理由：`Clear` 需要整個重寫檔案內容，如果 `Write` 持續握著同一個檔案描述符不放，跟 `Clear` 的重寫動作之間容易產生競爭或行為不一致。多付出每次開關檔案的成本，換取邏輯單純、好驗證——這個成本可以接受，因為 `LogManager` 本身已經把所有寫入序列化到單一背景 goroutine，寫入頻率不是這裡的效能瓶頸。

## 分級過濾

`LogLevel`（`debug`/`info`/`warn`/`error`）配合一個簡單的嚴重度排序表。`LogManager.WriteLog` 在把 entry 真正配置出來、送進 channel **之前**，就先比對 `level.meets(minLevel)`——低於設定的最低等級的日誌，連物件都不會被建立，直接丟棄，不會浪費任何資源在它們身上。

## 非同步寫入

`LogManager` 內部維護一個 buffered channel（`entries chan LogEntry`）+ 一個獨立的背景 goroutine（`run()`）。`WriteLog` 只是把 entry 塞進 channel 就立即返回，真正呼叫 `LogWriter.Write`（可能牽涉磁碟 I/O）的動作，全部發生在那個背景 goroutine 裡，不會拖慢呼叫端（例如：不會拖慢 HTTP request 的回應時間）。

**為什麼用單一 channel + 單一消費者，而不是每筆日誌開一個 goroutine**：日誌的核心用途是按時間順序重建系統行為，如果多個 goroutine 各自獨立寫入同一個儲存後端，落地順序會跟實際呼叫 `WriteLog` 的順序不一致，破壞日誌作為時間軸敘事的價值。單一消費者天然保證 FIFO，不需要額外加鎖。

`LogManager.Close()` 關閉 channel 並等待背景 goroutine 把佇列裡剩下的全部寫完才返回——這個方法是設計給之後的 Graceful Shutdown 呼叫的，跟 `ContainerService.Wait()`（等待背景非同步建立容器的工作做完）是同一個用途：服務關閉前，不該把還沒寫完的日誌或還沒做完的工作直接拋棄。

## 可擴展：`LogHandler`

`RegisterLogHandler` 讓任何實作 `Handle(entry LogEntry)` 的型別都能「訂閱」每一筆日誌，而不需要修改 `LogManager` 或任何 Writer/Reader。範例（`example_test.go` 裡的 `alertOnError`）：統計 ERROR 等級的數量並印出警示，模擬「異常告警」這類擴展功能。未來如果要做「遠程日誌聚合」，只需要再實作一個 `LogHandler`，把 entry 轉送到遠端服務，完全不影響既有程式碼。

## 結構化日誌（已實作）

`FileLogStore` 把每筆 entry 序列化成一行獨立的 JSON（JSON Lines / NDJSON 格式），而不是人類可讀的純文字。好處：每一行都是獨立合法的 JSON，可以直接餵給 `jq`、或未來接 Elasticsearch / Loki 之類的日誌索引系統，不需要寫自訂的文字解析器。`Fields()` 提供任意的結構化鍵值資料（見 `LoggingMiddleware`：method/path/status/latency/client_ip 都是獨立欄位，不是塞進一句話的文字訊息裡）。

## Zero-Allocation / Low GC Pressure（設計方向，未實作）

這次優先把時間放在核心介面設計、測試、文檔上，沒有實際做這塊的優化，但設計上會怎麼做：

- **`sync.Pool` 重用 `LogEntry` 物件**：目前每次 `WriteLog` 都用 `NewLogEntry` 配置一個新的 struct；高頻率寫入情境下，可以從一個 `sync.Pool` 借用已配置好的 entry、寫完後歸還，減少 GC 壓力
- **避免 `Fields()` 用 `map[string]interface{}`**：map 本身配置成本較高，且 `interface{}` 會讓數值型別逃逸到 heap；效能敏感的版本可以改用一組固定的 key-value 陣列（slice of struct），或針對常見欄位（method/path/status）開固定欄位，避免用 map
- **`FileLogStore` 改用持續開啟的 buffered writer**：目前每次寫入都開關檔案，雖然簡化了跟 `Clear` 的協調，但確實有額外的 syscall 成本；真正在意效能的版本，可以讓 `Write` 跟 `Clear` 共用一個檔案鎖、`Write` 持續用 `bufio.Writer` 累積後定期 flush，而不是每筆都重新開檔

## 跟主服務的整合

`internal/middleware/logging_middleware.go` 的 `LoggingMiddleware(manager)` 是一個標準的 Gin middleware：記錄每個 request 的 method/path/status/latency/client IP，狀態碼 `>=500` 記成 `error`、`>=400` 記成 `warn`、其餘 `info`。在 `main.go` 裡：

```go
logStore, _ := logger.NewFileLogStore(cfg.LogFilePath)
logManager := logger.NewLogManager(logStore, logStore, logger.LogLevel(cfg.LogMinLevel))
r.Use(middleware.LoggingMiddleware(logManager))
```

這展示了這套日誌系統不是一個孤立的練習題，而是真的能力裝進主服務裡，取代/補強 Gin 預設的 access log。

## 測試與範例

- `manager_test.go`：分級過濾、寫入順序保證、`ClearLogs`、`RegisterLogHandler`、`FileLogStore` 跟 `InMemoryLogStore` 行為一致性，皆通過 `-race`
- `example_test.go`：`Example()` 是 Go 標準的「可執行文檔」——`go test` 會真的執行它並比對輸出是否與註解裡的 `// Output:` 一致，同時也是 `go doc` 會展示給其他開發者看的官方使用範例
