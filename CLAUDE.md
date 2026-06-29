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

### 15. v1.3.1 — Chọn cửa sổ cho chụp ảnh & quay video
- **Chụp ảnh cửa sổ/vùng**: sửa bug thiếu cờ `-i` của `screencapture` (window=`-i -w`, area=`-i`) → giờ vào đúng chế độ click-chọn / kéo chọn
- **Quay video theo cửa sổ**: thêm dropdown chọn cửa sổ trên panel, quay đúng cửa sổ qua `SCContentFilter(desktopIndependentWindow:)`
- Hai cách chọn giữ khác nhau (ảnh: overlay click của macOS; video: dropdown) — ScreenCaptureKit không có UI chọn sẵn
- Chạy `screencapture` ở luồng nền (tránh block main run loop)
- Còn lại: video `region=area` (kéo vùng) chưa làm — tạm quay full
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.3.1

### 16. v1.3.2 — Chọn vùng với kính lúp phóng to (ảnh + video)
- macOS region selection chỉ hiện tọa độ, **không có kính lúp** → tự dựng overlay (`AreaSelector.swift`): ảnh chụp tĩnh làm nền, kéo chọn vùng, **vòng tròn kính lúp phóng to pixel** (interpolation `.none`) + tọa độ
- **Chụp ảnh vùng**: overlay → chụp đúng vùng qua `screencapture -R x,y,w,h`
- **Quay video vùng**: overlay lấy tọa độ → quay full màn hình → **backend cắt bằng FFmpeg** (`crop=w:h:x:y`), vì `SCStreamConfiguration.sourceRect` cần macOS 14 (máy đang Ventura 13)
- Backend: `capture.go` nhận `crop_x/y/w/h`, `processCaptureVideo` gộp transcode + crop
- Tọa độ thống nhất: video quay ở độ phân giải điểm (1440×900) = tọa độ overlay → crop khớp
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.3.2

### 17. v1.3.3 — Ẩn con trỏ (ảnh) + đếm ngược trước khi quay
- Checkbox **"Ẩn con trỏ chuột"** trên web → `cursor=hide`
- **Chụp ảnh**: ẩn con trỏ bằng cách bỏ cờ `-C` của `screencapture` (full/vùng); cửa sổ luôn ẩn (chế độ `-i` không nhận `-C`)
- **Quay video**: `SCStreamConfiguration.showsCursor=false` — nhưng **Ventura 13.4 không tôn trọng** (chỉ hiệu lực từ macOS 14), xem mục Lỗi đã gặp
- Thay thế: **đếm ngược 3-2-1** (`Countdown.swift`) trước khi quay video — overlay không chặn chuột, tắt trước khi quay → kịp đưa con trỏ ra chỗ khuất, không lọt vào video
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.3.3

### 18. v1.3.4 — Tách Chụp ảnh / Quay video thành 2 tab con
- Vì options đã khác nhau (ảnh: ẩn con trỏ; video: âm thanh + đếm ngược) → tách trong cùng tab "Quay màn hình" thành **2 tab con** (subtabs) Chụp ảnh / Quay video, không thêm tab trên header
- Chỉ sửa frontend (`index.html`, `app.js`, `style.css`); mỗi tab con chỉ hiện điều khiển liên quan
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.3.4

### 19. v1.4.0 — Xem ảnh/video/audio trực tiếp trong web
- Trước đây file kết quả chỉ **tải về** (`Content-Disposition: attachment`); giờ thêm nút **👁 Xem** mở/phát ngay trong trình duyệt
- **Backend** (`internal/handlers/convert.go`): tách `serveOutput(c, disposition)` dùng chung; `DownloadOutput`=`attachment`, thêm `PreviewOutput`=`inline`; route `GET /api/preview/:id`
- **Frontend**: modal `#previewModal` hiển thị `<img>`/`<video>`/`<audio>` tùy loại job (`mediaKind()` suy từ `type`), autoplay; đóng bằng ✕ / click nền / Esc (xóa `innerHTML` để dừng phát)
- Mỗi dòng lịch sử (`done`) có **👁 Xem** cạnh **⬇ Tải về**
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.4.0

### 20. v1.5.0 — Cắt video theo thời gian
- Tab mới **"Cắt video"**: upload video, nhập thời gian **Bắt đầu / Kết thúc**, FFmpeg cắt đoạn đó (xuất MP4), chạy async như các job khác
- **Backend** (`internal/handlers/trim.go`): job type `video_trim`, route `POST /api/trim/video`; `parseTimecode` nhận giây / `MM:SS` / `HH:MM:SS`; cắt bằng `-ss` (trước `-i`, seek nhanh) + `-t duration`, **re-encode** (`libx264`+`aac`+`faststart`) để đúng frame; kết thúc trống = cắt tới hết
- **Frontend**: drop-zone + 2 ô thời gian; dùng chung `pollJob`, lịch sử, nút **👁 Xem**/**⬇ Tải về**; badge **Cắt** (`badge-trim`), `mediaKind('video_trim')`=video
- Lưu `input_file` = đường dẫn thật → xóa job dọn được cả file gốc
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.5.0

### 21. v1.5.1 — Cắt cả audio (mở rộng tab Cắt)
- Tab đổi tên **"Cắt video/audio"**, nhận thêm mp3/wav/m4a/aac/flac/ogg; route đổi `POST /api/trim/video` → **`/api/trim/media`**, handler `TrimVideo`→`TrimMedia`
- Backend tự chọn codec/đuôi output theo nguồn (`trimEncodeArgs`): video→MP4 (H.264/AAC); mp3→libmp3lame, wav→pcm_s16le, flac→flac, ogg→libvorbis, m4a/aac→aac. Audio **giữ nguyên định dạng**
- Job type tách `video_trim` / `audio_trim` để preview đúng kiểu (`mediaKind('audio_trim')`=audio); badge **Cắt** / **Cắt audio**
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.5.1

### 22. v1.5.2 — Chọn thời gian bằng player trực quan
- Chọn file xong → hiện `<video>`/`<audio>` (objectURL, client-side) + **timeline 2 tay kéo** Bắt đầu/Kết thúc; kéo để đặt đoạn, vùng chọn tô màu, có **playhead** chạy theo media
- Nút **▶ Xem đoạn đã chọn**: tua tới start, play, tự dừng ở end (`previewStopAt` trong `timeupdate`); click trên track = tua media
- Đồng bộ **2 chiều** với 2 ô nhập tay (vẫn chỉnh chính xác được): tay kéo → `writeInputs` ghi `fmtTimecode` (2 số lẻ); gõ tay → `syncFromInputs` (`parseTimeJS`); end ở cuối = ô trống = cắt tới hết
- Chỉ sửa frontend (`index.html`, `app.js`, `style.css`); backend `/api/trim/media` giữ nguyên (vẫn đọc 2 ô start/end)
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.5.2

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
| Quay video không ẩn được con trỏ dù `showsCursor=false` | `SCStreamConfiguration.showsCursor` chỉ có tác dụng từ macOS 14; Ventura 13.4 bỏ qua (đã xác nhận qua log: showsCursor=false nhưng con trỏ vẫn hiện) | Giữ `showsCursor=false` (đúng chuẩn, tự chạy khi lên macOS 14+); thêm **đếm ngược 3-2-1** để người dùng tự giấu con trỏ |
| Native app "đã cấp quyền nhưng vẫn báo chưa" / toggle Settings cũ không tự tắt | Ký **ad-hoc** đổi identity mỗi build → quyền TCC gắn identity cũ, bản mới không khớp | Tạo **self-signed cert** ổn định (`make-cert.sh`, OpenSSL 3 cần `-legacy` cho PKCS12); `tccutil reset ScreenCapture com.gostudio.capture` |
| Quay video không ra file (lịch sử trống) | Cast `SCStreamFrameInfo` status thất bại → loại sạch mọi frame → file rỗng → không upload | Bắt đầu writer session ngay frame đầu; chỉ bỏ frame khi đọc được status và ≠ `.complete` |
| Lần bấm quay thứ 2 vô tình dừng bản quay 1 | App single-shot nhưng instance cũ còn chạy, URL mới route về nó; nút Bắt đầu rơi vào state `.recording` | Bỏ qua URL mới khi đang quay/lưu; đóng panel = thoát app; nút ⏹ menu bar luôn dừng được |
| Phím tắt toàn cục (Carbon `RegisterEventHotKey`) không kích hoạt | Đăng ký `status=0` nhưng sự kiện không tới app helper kiểu này (xác nhận qua log: handler không chạy) | Bỏ phím tắt; dùng nút ⏹ trên thanh menu (đúng cách macOS dừng quay) |

---

## Đang làm

- Ổn định v1.5.2
- Repo public: https://github.com/MichaelTranTrong/gostudio
- Release mới nhất: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.5.2

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

- [x] Thêm tính năng cắt video/audio (trim theo thời gian) — v1.5.0 (video) + v1.5.1 (audio) + v1.5.2 (player trực quan)
- [ ] Thêm tính năng chuyển đổi định dạng khác (MP4→WAV, MP4→AAC, ...)
- [ ] Thêm thanh tiến trình thực từ FFmpeg (parse stderr `-progress`)
- [ ] Giới hạn kích thước file upload (hiện tại không giới hạn)
- [ ] Xử lý trùng tên file output (thêm suffix nếu file đã tồn tại)
- [ ] Tự động xóa file upload/output sau N ngày
- [ ] Thêm xác thực người dùng (login)
- [ ] Quay video macOS: thu micro, nhân scale Retina (cửa sổ + vùng đã xong ở v1.3.1/v1.3.2)

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
| v1.3.1 | Chọn cửa sổ cho chụp ảnh (`-i -w`) & quay video (dropdown + `SCContentFilter`) |
| v1.3.2 | Chọn vùng với kính lúp phóng to (overlay tự dựng) cho ảnh + video, crop FFmpeg |
| v1.3.3 | Ẩn con trỏ khi chụp ảnh + đếm ngược 3-2-1 trước khi quay video |
| v1.3.4 | Tách Chụp ảnh / Quay video thành 2 tab con trong bảng điều khiển |
| v1.4.0 | Xem ảnh/video/audio trực tiếp trong web (nút 👁 Xem + modal, `/api/preview/:id`) |
| v1.5.0 | Cắt video theo thời gian (tab Cắt video, `/api/trim/video`, FFmpeg `-ss`/`-t`) |
| v1.5.1 | Cắt cả audio (mp3/wav/m4a/aac/flac/ogg), route đổi `/api/trim/media`, giữ định dạng nguồn |
| v1.5.2 | Chọn thời gian cắt bằng player trực quan (timeline 2 tay kéo, playhead, xem thử đoạn) |
