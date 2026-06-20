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

---

## Đang làm

- Ổn định v1.1.1
- Repo public: https://github.com/MichaelTranTrong/gostudio
- Release mới nhất: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.1.1

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

---

## Versions

| Version | Nội dung |
|---|---|
| v1.0.0 | Chuyển MP4 → MP3, Docker, MySQL, giao diện web dark theme |
| v1.1.0 | Xóa từng job, xóa toàn bộ lịch sử + reset ID |
| v1.1.1 | Xóa file vật lý (MP4 + MP3) khi xóa job, tránh chiếm dung lượng disk |
