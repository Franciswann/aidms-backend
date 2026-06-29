# 架構與設計決策

## Clean Architecture

採用 Clean Architecture 四層架構，依賴方向僅能由外向內，內層不得 import 任何第三方套件：

```
Frameworks & Drivers     cmd/api/main.go, Gin, GORM, Docker SDK, swaggo
        ↓
Interface Adapters       internal/handler/, internal/repository/, internal/docker/, internal/middleware/
        ↓
Use Cases                internal/usecase/{container,user,file,job}/, internal/logger/
        ↓
Domain                   internal/domain/entity/, internal/domain/repository/
```

最內層（Domain）只放純粹的資料結構跟介面定義，完全不依賴 GORM、Gin、Docker SDK 等任何第三方套件；外層可以自由替換，內層完全不受影響。

## 關鍵設計決策

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
| 檔案上傳：實際存檔路徑用 `{FileStoragePath}/{userID}/{file 自己的 UUID}`，原始檔名只當中繼資料存進 DB | 從根本避免 path traversal（路徑遍歷）攻擊——不管使用者把檔名取成什麼字串，都不會影響實際寫入磁碟的路徑，不需要靠黑名單擋特殊符號；per-user 資料夾也方便日後依使用者做整批清理 |
| `FileService.Upload` 簽名只收 `io.Reader` + 純量參數，不收 `*multipart.FileHeader` | Use Case 層不該知道「這次上傳是透過 HTTP multipart」這件事，由 Handler（Interface Adapters 層）負責把 HTTP 特有的型別轉換成跟來源無關的普通參數 |
| `ContainerService.CreateAsync` 重用既有的同步 `Create`，包一層 goroutine | 長耗時的容器建立不用讓 HTTP request 卡住等待；背景工作直接呼叫已經寫好、已經測過的 `Create`，不重複實作一份建立邏輯 |
| goroutine 拿 `entity.Job` 的「副本」而非跟呼叫者共用同一個指標 | `go test -race` 抓到的真實 data race：原本同一個 `*entity.Job` 指標同時被「回傳給呼叫者」跟「背景 goroutine 持續寫入」兩邊共用，沒有同步機制；修法是讓 goroutine 操作自己的副本，呼叫者拿到的指標之後不會再被修改 |
| Graceful Shutdown 的關閉順序：HTTP server → `ContainerService.Wait()` → `LogManager.Close()` | 後兩者都是「背景仍可能在寫東西」的元件；如果 log 系統先關，HTTP shutdown 或容器背景工作還在跑時呼叫 `WriteLog` 會對已關閉的 channel 寫入而 panic，所以一定要最後關 |
| Repository 集成測試用 `tx := db.Begin()` + `defer tx.Rollback()`，不用 `t.Cleanup` 手動刪資料 | `*gorm.DB` 跟 `db.Begin()` 回的 transaction 是同一個型別，Repository 不需要為了測試改任何 production code；只要不呼叫 `Commit()`，測試裡的所有寫入保證不會真正留在資料庫，比手動對應每筆 insert 的清理動作更不容易出錯 |
| `ContainerHandler` 依賴 `handler` package 自己定義的 `containerUsecase` 介面，而非 `*container.ContainerService` 具體型別 | 跟 Repository／`ContainerRuntime` 同一套 DIP：消費端（Handler）只定義自己用得到的方法，測試時可以注入 mock，不需要真的連 Docker daemon 或資料庫；`*container.ContainerService` 本來就實作了這些方法，`main.go` 的組裝完全不用改 |

關於 `internal/logger` 套件自己的設計理由（可插拔儲存、分級過濾、非同步寫入、擴展點），見 [`internal/logger/DESIGN.md`](../internal/logger/DESIGN.md)。
