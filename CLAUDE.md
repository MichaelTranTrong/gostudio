# Go Studio

## Dự án: Go Studio

Ứng dụng web chỉnh sửa âm thanh và video, chạy trên Docker, port 2005.
Repo: https://github.com/MichaelTranTrong/gostudio

---

## Stack

- **Backend:** Go (Gin framework)
- **Database:** MySQL 8.4
- **Xử lý media:** FFmpeg (chạy trong container Alpine)
- **Frontend:** HTML + CSS + Vanilla JS (dark theme)
- **Infrastructure:** Docker + Docker Compose

---

## Đã làm

### 1. Khởi tạo dự án
```bash
cd /Users/michaeltran/Documents/Studio/gostudio
go mod init gostudio
go get github.com/gin-gonic/gin@latest
go get github.com/go-sql-driver/mysql@latest
```

### 2. Cấu trúc thư mục
```
gostudio/
├── Dockerfile
├── docker-compose.yml
├── .dockerignore
├── .gitignore
├── main.go
├── go.mod / go.sum
├── internal/
│   ├── database/db.go       ← Kết nối MySQL, auto-migrate bảng jobs
│   ├── models/job.go        ← CRUD job (pending/processing/done/failed)
│   └── handlers/convert.go  ← Upload MP4, chạy FFmpeg async, download MP3
├── web/
│   ├── templates/index.html
│   └── static/
│       ├── css/style.css
│       └── js/app.js
├── uploads/                 ← Docker volume
└── outputs/                 ← Docker volume
```

### 3. Tính năng: Chuyển MP4 → MP3
- Upload file MP4 qua giao diện web (kéo thả hoặc chọn file)
- FFmpeg trích xuất audio chạy bất đồng bộ (goroutine)
- Poll trạng thái job mỗi 1.5 giây
- Tải file MP3 về khi hoàn thành
- Lịch sử tất cả job hiển thị trong bảng, tự refresh mỗi 8 giây

### 4. Bảng database: `jobs`
```sql
CREATE TABLE jobs (
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    type        VARCHAR(50)  NOT NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'pending',
    input_file  VARCHAR(512) NOT NULL,
    output_file VARCHAR(512),
    error_msg   TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```
Status flow: `pending` → `processing` → `done` | `failed`

### 5. Docker setup
```yaml
# docker-compose.yml
services:
  app:   port 2005, depends on db healthy
  db:    MySQL 8.4, healthcheck mysqladmin ping
volumes:
  mysql_data / uploads_data / outputs_data
```
Credentials MySQL: user=`gostudio`, pass=`gostudio123`, db=`gostudio`

### 6. Các lệnh Docker thường dùng
```bash
docker compose up -d            # Khởi động
docker compose up -d --build    # Build lại sau khi sửa code
docker compose down             # Dừng
docker compose down -v          # Dừng và xóa toàn bộ volume (reset DB)
docker compose logs -f app      # Xem log app
docker compose ps               # Kiểm tra trạng thái container
```

### 7. Đưa lên GitHub
```bash
gh auth login                   # Chọn GitHub.com → HTTPS → web browser
git init
git add .
git commit -m "Initial release: Go Studio v1.0.0"
gh repo create gostudio --public --description "..." --push --source .
git tag v1.0.0
git push origin v1.0.0
gh release create v1.0.0 --title "Go Studio v1.0.0" --notes "..."
```

---

## Đã làm (tiếp theo)

### 8. v1.1.0 — Xóa lịch sử
- Nút **✕** trên mỗi dòng → xóa từng job (`DELETE /api/jobs/:id`)
- Nút **Xóa toàn bộ** → xóa hết + reset `AUTO_INCREMENT` về 1 (`DELETE /api/jobs`)
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.1.0

### 11. v1.1.1 — Xóa file vật lý khi xóa job
- Khi xóa job (từng job hoặc toàn bộ), tự động xóa luôn file MP4 trong `uploads/` và file MP3 trong `outputs/`
- Trước đây chỉ xóa record DB, file vẫn còn trong Docker volume gây chiếm dung lượng disk
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.1.1

### 12. v1.2.0 — Chữ → Tiếng (TTS) với VieNeu-TTS
- Tab mới "Chữ → Tiếng": nhập text, chọn giọng, tạo MP3 (offline)
- Thêm service Python `vieneu` (FastAPI, port 8001 nội bộ) vào Docker Compose
- Go gọi VieNeu qua HTTP, tự chia text ≤2800 ký tự, ghép audio bằng FFmpeg
- Lịch sử dùng chung, badge phân biệt MP4 / TTS
- Model: `pnnbao-ump/VieNeu-TTS-0.3B-ngoc-huyen-gguf-Q4_0` (GGUF, llama-cpp CPU)
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.2.0

### 13. v1.2.1 — Sửa lỗi TTS đọc câu reference + lặp lại
- Audio TTS đọc cả câu mẫu giọng ("...tính chiến đấu, tính định hướng") và lặp nội dung
- Nguyên nhân: VieNeu chọn sai `use_chat_format` cho bản fine-tune 0.3B
- Fix qua `vieneu/fix_stream.py` (chạy lúc build image), xem mục Lỗi đã gặp
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.2.1

### 14. v1.3.0 — Quay màn hình & chụp ảnh macOS (native app)
- Web kích hoạt app native macOS qua URL scheme `gostudio://capture?mode=...`, app quay/chụp rồi upload về Go Studio (tái dùng job pipeline + lịch sử)
- **Backend** (`internal/handlers/capture.go`): `POST /api/capture/upload` nhận ảnh/video, job type `screenshot`/`screen_record`, `.mov`→`.mp4`, download content-type theo đuôi file
- **Native app** (`macos-capture/`): Swift dùng-một-lần — ScreenCaptureKit (video + system audio) + `screencapture` (ảnh). Panel điều khiển, xin quyền khi bấm Bắt đầu + tự relaunch. Quay video: ẩn UI, dừng bằng nút ⏹ trên thanh menu (giống macOS)
- Ký **self-signed cert** (`make-cert.sh`) để quyền Screen Recording dính vĩnh viễn qua các lần build
- **Web**: tab "Quay màn hình" phát `gostudio://`, badge lịch sử Ảnh/Quay
- Thiết kế chi tiết: [docs/screen-capture-design.md](docs/screen-capture-design.md)
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.3.0

### 9. Lệnh release GitHub
```bash
git add .
git commit -m "message"
git tag v1.x.x
git push origin main
git push origin v1.x.x
gh release create v1.x.x --title "Go Studio v1.x.x" --notes "..."
```

### 10. Lỗi đã gặp và cách xử lý
| Lỗi | Nguyên nhân | Cách xử lý |
|---|---|---|
| Dialog chọn file mở 2 lần | `<label for="fileInput">` + `dropZone click` cùng trigger | Đổi `<label>` thành `<span>` |
| Tên file tải về có dấu `+` thay dấu cách | Gin `FileAttachment` không encode đúng RFC 5987 | Set thủ công header `Content-Disposition` với `filename*=UTF-8''...` |
| Tên file upload có dấu `+` | Browser encode space thành `+` trong multipart | Dùng `url.QueryUnescape` trước khi lưu |
| Container DB không start — `file exists` | containerd bị kẹt state cũ | `docker compose down && docker rm -f gostudio-db gostudio-app && docker compose up -d` |
| VieNeu crash khi khởi động — `ModuleNotFoundError: trafilatura` / `llama_cpp` | Minimal install thiếu dependency | Cài thêm `trafilatura` + `llama-cpp-python` (cần `cmake build-essential g++` để compile) trong Dockerfile |
| VieNeu báo `No file found ... VieNeu-TTS-v2-Q4-K-M.gguf` | Model trên HF đổi tên file GGUF | `sed` đổi filename trong `src/vieneu/standard.py` thành `VieNeu-TTS-0.3B-ngoc-huyen-Q4_0.gguf` |
| Audio TTS đọc cả câu reference + lặp lại | `use_chat_format` chỉ bật cho repo v1; bản fine-tune 0.3B dùng sai định dạng prompt → đọc ref text + lặp (308 tokens/6s thay vì 74/1.5s) | Đổi heuristic thành `"VieNeu-TTS" in backbone_repo`; dùng `infer()` thay `infer_stream()`. Patch qua `vieneu/fix_stream.py` |
| Native app "đã cấp quyền nhưng vẫn báo chưa" / toggle Settings cũ không tự tắt | Ký **ad-hoc** đổi identity mỗi build → quyền TCC gắn identity cũ, bản mới không khớp | Tạo **self-signed cert** ổn định (`make-cert.sh`, OpenSSL 3 cần `-legacy` cho PKCS12); `tccutil reset ScreenCapture com.gostudio.capture` |
| Quay video không ra file (lịch sử trống) | Cast `SCStreamFrameInfo` status thất bại → loại sạch mọi frame → file rỗng → không upload | Bắt đầu writer session ngay frame đầu; chỉ bỏ frame khi đọc được status và ≠ `.complete` |
| Lần bấm quay thứ 2 vô tình dừng bản quay 1 | App single-shot nhưng instance cũ còn chạy, URL mới route về nó; nút Bắt đầu rơi vào state `.recording` | Bỏ qua URL mới khi đang quay/lưu; đóng panel = thoát app; nút ⏹ menu bar luôn dừng được |
| Phím tắt toàn cục (Carbon `RegisterEventHotKey`) không kích hoạt | Đăng ký `status=0` nhưng sự kiện không tới app helper kiểu này (xác nhận qua log: handler không chạy) | Bỏ phím tắt; dùng nút ⏹ trên thanh menu (đúng cách macOS dừng quay) |

---

## Đang làm

- Ổn định v1.3.0
- Repo public: https://github.com/MichaelTranTrong/gostudio
- Release mới nhất: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.3.0

---

## Quyết định quan trọng

| Quyết định | Lý do |
|---|---|
| Dùng **Gin** thay vì net/http thuần | Routing tiện hơn, middleware sẵn có, phổ biến trong Go web |
| Dùng **FFmpeg** qua `exec.Command` thay vì thư viện Go | FFmpeg là chuẩn công nghiệp, hỗ trợ mọi codec, không cần CGO |
| FFmpeg chạy **bất đồng bộ** (goroutine) | File video lớn xử lý lâu, không thể block HTTP request |
| **Docker Compose** với healthcheck | Đảm bảo MySQL ready trước khi app khởi động, tránh lỗi connection |
| Dùng **Alpine** cho runtime image | Image nhỏ (~120MB với FFmpeg) thay vì Ubuntu (~800MB) |
| `filename*=UTF-8''...` (RFC 5987) cho download | Gin's `FileAttachment` encode sai tên file có dấu cách/tiếng Việt thành `+` |
| `url.QueryUnescape` cho tên file upload | Browser gửi tên file có thể bị encode `+` thay cho dấu cách trong multipart form |
| Xóa `<label for="fileInput">` | Label + dropZone click cùng trigger fileInput → dialog mở 2 lần |

---

## TODO tiếp theo

- [ ] Thêm tính năng cắt audio/video (trim theo thời gian)
- [ ] Thêm tính năng chuyển đổi định dạng khác (MP4→WAV, MP4→AAC, ...)
- [ ] Thêm thanh tiến trình thực từ FFmpeg (parse stderr `-progress`)
- [ ] Giới hạn kích thước file upload (hiện tại không giới hạn)
- [ ] Xử lý trùng tên file output (thêm suffix nếu file đã tồn tại)
- [ ] Tự động xóa file upload/output sau N ngày
- [ ] Thêm xác thực người dùng (login)
- [ ] Quay video macOS: hỗ trợ region cửa sổ/vùng, thu micro, nhân scale Retina (hiện full-screen + system audio)

---

## Versions

| Version | Nội dung |
|---|---|
| v1.0.0 | Chuyển MP4 → MP3, Docker, MySQL, giao diện web dark theme |
| v1.1.0 | Xóa từng job, xóa toàn bộ lịch sử + reset ID |
| v1.1.1 | Xóa file vật lý (MP4 + MP3) khi xóa job, tránh chiếm dung lượng disk |
| v1.2.0 | Chữ → Tiếng (TTS) với VieNeu-TTS, service Python riêng trong Docker |
| v1.2.1 | Sửa lỗi TTS đọc câu reference + lặp lại (use_chat_format) |
| v1.3.0 | Quay màn hình & chụp ảnh macOS qua native app (ScreenCaptureKit, scheme `gostudio://`) |
