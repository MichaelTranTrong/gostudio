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

### 23. v1.5.3 — Cắt cả file đang có sẵn trong app (từ lịch sử)
- Mỗi dòng lịch sử audio/video (không phải ảnh) có nút **✂ Cắt** → mở tab Cắt, nạp file qua `/api/preview/:id` vào player (server hỗ trợ range nên tua được), chọn đoạn như upload
- **Backend** (`trim.go`): `TrimMedia` nhận thêm field **`source_id`** (id job nguồn) → đọc thẳng `OutputFile` làm input, **không copy lại**; ext suy từ file nguồn
- Job cắt-từ-nguồn lưu `input_file` = **nhãn `[Cắt] <tên>`** (không phải path thật) → xóa job này **không đụng file nguồn**; `trimName()` ở frontend bỏ nhãn khi hiển thị
- Thêm `uniqueOutputPath` (base, base-2, base-3…) → cắt cùng file nhiều lần không ghi đè (xong luôn TODO trùng tên output)
- **Frontend**: state `trimSourceId`; chọn file mới thì bỏ nguồn-từ-lịch-sử; `loadTrimPlayerSrc(url, isVideo)` tách dùng chung; submit gửi `source_id` hoặc `file`
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.5.3

### 24. v1.5.4 — Đổi tên file trong lịch sử
- Mỗi dòng lịch sử có nút **✏ Đổi tên** cạnh nút xóa → `prompt()` nhập tên mới, gọi `PATCH /api/jobs/:id/rename`
- **Backend**: thêm cột `display_name` (migration kiểm tra `information_schema.columns` trước khi `ALTER TABLE` — MySQL 8.4 **không hỗ trợ** `ADD COLUMN IF NOT EXISTS`, xem mục Lỗi đã gặp); `models.RenameJob`; `serveOutput` dùng `display_name` (giữ nguyên đuôi file thật) cho `Content-Disposition` khi tải về/xem
- Chỉ đổi **tên hiển thị + tên khi tải về**, không đổi tên file thật trên disk (`input_file`/`output_file` giữ nguyên) → xóa job vẫn dọn đúng file
- **Frontend**: `jobMeta`/`loadHistory` ưu tiên `j.display_name` nếu đã đổi, không thì dùng tên suy ra như cũ
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.5.4

### 25. v1.6.0 — Ghép nối video/audio
- Tab mới **"Ghép nối"**: thêm nhiều video (hoặc nhiều audio) theo thứ tự → ghép thành 1 file; nguồn là **upload mới** và/hoặc **file có sẵn trong lịch sử** (nút **+ Ghép** trên mỗi dòng, giống ✂ Cắt)
- **Backend** (`internal/handlers/concat.go`): route `POST /api/concat/media`, nhận field `order` (JSON `[{type:"upload"}|{type:"source",id}]`) + files upload theo thứ tự; job type `video_concat`/`audio_concat`
- Video khác độ phân giải/fps: chuẩn hóa theo **file đầu tiên** (ffprobe đọc kích thước → `scale`+`pad`+`setsar`+`fps` từng input qua `filter_complex`) rồi `concat` → MP4 (H.264/AAC); audio: `aformat` 44.1kHz stereo rồi `concat`, giữ định dạng nguồn (tái dùng `trimEncodeArgs`)
- Chặn ghép lẫn video+audio; **validate toàn bộ trước khi lưu** file upload xuống disk (2 bước: resolve/kiểm tra hết → mới save) tránh rác file khi request bị từ chối giữa chừng
- `input_file` = nhãn `[Ghép] N file` → xóa job không đụng file nguồn; file upload tạm tự xóa sau khi ghép (defer trong `runConcat`)
- **Frontend**: danh sách có thứ tự (▲▼ đổi chỗ, ✕ bỏ), state `concatItems` (`upload`|`source`); badge **Ghép** / **Ghép audio** (`badge-concat`)
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.6.0

### 26. v1.6.1 — TTS: bổ sung 6 giọng đọc
- Tab Chữ → Tiếng giờ có **7 giọng**: Ngọc Huyền (mặc định) + 6 giọng từ repo model gốc `pnnbao-ump/VieNeu-TTS-0.3B-q4-gguf`: Vĩnh (nam Nam), Bình (nam Bắc), Tuyên (nam Bắc), Đoan (nữ Nam), Ly (nữ Bắc), Ngọc (nữ Bắc)
- Cơ chế: giọng là "preset voices" (voice codes + ref text) nạp từ `voices.json` trong repo model HF; bản fine-tune ngoc-huyen chỉ khai báo 1 giọng, voice codes dùng chung codec (neucodec) nên nạp lẫn được vào model đang chạy
- Chỉ sửa `vieneu/fix_stream.py` (patch 3): sau khi load model, gọi `tts._load_voices_from_repo(repo gốc)` để merge presets, giữ nguyên `_default_voice` = NgocHuyen; frontend không đổi (`/voices` tự trả danh sách mới)
- Tên giọng trong dropdown kèm **gợi ý thể loại** (vd "Ly (nữ Bắc — tin tức, sách nói, e-learning)"): map `voiceLabels` trong `internal/handlers/tts.go` (`GetVoices` ghi đè `name` theo `id`, giọng lạ giữ tên gốc) — nhãn là việc của app, không patch vào VieNeu
- Lưu ý phát hiện khi làm: **patch 1 của fix_stream.py (infer_stream→infer) không còn áp dụng được** — upstream đổi whitespace nên pattern không khớp (build in WARN), app hiện chạy `infer_stream`; TTS vẫn hoạt động bình thường
- Release: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.6.1

### 27. v1.7.0 — Đăng ký / Đăng nhập (dùng chung tài khoản Go Keep)
- **Feature flag `AUTH_ENABLED`** (mặc định tắt): **1 codebase duy nhất** cho cả 2 môi trường — bản **local** (auth tắt, chạy y như cũ, không cần đăng nhập) và bản **VPS** (auth bật, đủ chức năng tài khoản). Mọi cập nhật tính năng khác tự đồng bộ vì chung 1 repo; khác biệt chỉ nằm ở biến môi trường.
- **Dùng chung kho tài khoản với Go Keep:** bản VPS mở **kết nối MySQL thứ 2** tới DB `gokeep` (bảng `users`) qua `GOKEEP_DB_DSN` — tài khoản tạo ở Go Studio cũng là tài khoản Go Keep (chung bảng users). Go Studio + Go Keep deploy **cùng VPS**, nối qua docker network `gokeep_default` (container `gokeep_mysql`).
- **Backend:**
  - `internal/auth/` (package mới): JWT HS256 **tự cài bằng stdlib** (không thêm dependency), middleware `RequireAuth`, repo user (bcrypt, khớp schema Go Keep), gửi email xác thực qua `notify.zalink.io.vn` (giống Go Keep, link do Go Studio phát hành).
  - `internal/handlers/auth.go`: `register`, `login`, `verify-email`, `resend-verify`, `me`. Register tạo user trong `gokeep.users` (email_verified=0) + gửi email xác thực → JWT.
  - `main.go`: khi `AUTH_ENABLED` → đăng ký route `/api/auth/*` + bọc toàn bộ route media bằng `RequireAuth`; endpoint capture (native app) luôn để ngoài (dùng token riêng). Truyền cờ `authEnabled` vào template.
  - **JWT session RIÊNG của Go Studio** (JWT_SECRET riêng, không cần trùng Go Keep) — dùng chung *credentials*, không dùng chung *session*.
- **Tách dữ liệu theo user:** thêm cột `jobs.user_id` (nullable). Mọi thao tác job lọc/kiểm tra theo `user_id`; `userID=0` (local) = không lọc → hành vi cũ. User không thấy/không xóa được job của người khác (đã test: GET/DELETE job người khác → 404).
- **Frontend:** trang Đăng nhập/Đăng ký (overlay `#authOverlay`, ẩn `#appRoot` tới khi đăng nhập), banner "email chưa xác thực + gửi lại", nút Đăng xuất. Wrapper `window.fetch` tự gắn `Authorization: Bearer`; token lưu **localStorage + cookie** (cookie để thẻ `<img>/<video>/<a download>` gửi token được — header không gắn vào các thẻ này). Chỉ hiện khi `window.AUTH_ENABLED===true`; tắt thì giao diện cũ nguyên vẹn.
- **Deploy:** `docker-compose.yml` giữ nguyên cho local. Thêm **`docker-compose.vps.yml`** (override) set `AUTH_ENABLED`, `GOKEEP_DB_DSN`, `JWT_SECRET`, `NOTIFY_URL`, `APP_URL` + nối network `gokeep_default`. Chạy VPS: `docker compose -f docker-compose.yml -f docker-compose.vps.yml up -d --build`.
- **Lưu ý deploy:** đặt `APP_URL` đúng domain Go Studio (link xác thực email dựa vào nó); đổi `JWT_SECRET` thành chuỗi ngẫu nhiên dài.

### 28. v1.7.1 — Quên mật khẩu / Đặt lại mật khẩu (hoàn thiện auth)
- Bổ sung luồng còn thiếu so với Go Keep: **quên mật khẩu → nhận email → đặt mật khẩu mới**.
- **Backend:** route `POST /api/auth/forgot-password` (gửi email chứa link, luôn trả success để chống dò email) + `POST /api/auth/reset-password` (đổi mật khẩu qua token, hết hạn 1 giờ). Dùng cột `reset_token`/`reset_token_expires` sẵn có của bảng `users` Go Keep; gửi email qua `notify.zalink.io.vn` (`SendResetEmail`). Link reset trỏ về `APP_URL/?reset_token=...`.
- **Frontend:** thêm 2 panel trong overlay auth (Quên mật khẩu: nhập email; Đặt mật khẩu mới: mở từ link email qua `?reset_token=`), link "Quên mật khẩu?" ở trang đăng nhập. `boot()` phát hiện `?reset_token` → hiện panel đặt lại (không cần đăng nhập).
- **Tự đăng xuất khi token hết hạn:** wrapper `fetch` bắt response 401 từ `/api/*` (trừ `/api/auth/*`) → xóa token + hiện lại màn đăng nhập.
- Đã test cục bộ 8 case (forgot/reset/token sai/mật khẩu ngắn/đổi xong login mật khẩu cũ 401 + mới 200/chống dò email/token dùng 1 lần).

### 29. v1.7.2 — Ẩn quay/chụp màn hình khỏi bản online
- Chức năng quay video / chụp ảnh chỉ chạy được ở **local trên macOS** (qua native app `gostudio://`), vô nghĩa trên VPS → **ẩn hẳn khỏi bản online** (`AUTH_ENABLED=true`).
- **Frontend:** bọc nav link "Quay màn hình" + toàn bộ `#tab-capture` trong `{{ if not .authEnabled }}...{{ end }}` → bản online không có luôn trong HTML. Thêm `?.` cho chỗ chuyển tab và 2 nút `shotBtn`/`vidBtn` (tránh crash script khi DOM vắng mặt).
- **Backend:** route `POST /api/capture/upload` chỉ đăng ký khi `!authEnabled` (local); bản online → 404.
- Đã test 2 chế độ: online (không có markup capture, endpoint 404) / local (giữ nguyên, endpoint 400).

### 30. v1.7.3 — Sửa TTS lặp chữ (còn sót từ v1.2.1)
- Bản v1.2.1 đã sửa lỗi TTS đọc câu reference + lặp, nhưng **thỉnh thoảng vẫn lặp cụm từ** (ngẫu nhiên, không phải lần nào cũng). Truy ra **2 nguyên nhân** trong service `vieneu`:
  1. **App vẫn chạy nhánh `infer_stream`** (không phải `infer`) — Patch 1 cũ của `fix_stream.py` khớp nguyên khối, upstream đổi khoảng trắng nên **hỏng âm thầm** từ v1.6.1. Nhánh streaming GGUF ghép audio từ các cửa sổ chồng lấp (lookback/lookforward) → đôi khi **nhân đôi đoạn audio → nghe thành lặp chữ** (repeat_penalty không chạm tới được). Đây mới là thủ phạm chính.
  2. **Lệnh sinh GGUF không truyền `repeat_penalty`** — `self.backbone(prompt, ...)` ở `_infer_ggml`/`_infer_stream_ggml` thiếu tham số này (mức mặc định quá nhẹ). Lưu ý: `repetitive_penalty=1.2` đặt lúc load model (standard.py dòng ~112) **sai tên tham số** → llama-cpp bỏ qua, xưa nay vô tác dụng.
- **Cách sửa** (đều qua `vieneu/fix_stream.py`, áp lúc build):
  - **Patch 1 viết lại bằng regex** (bền với thay đổi khoảng trắng): chuyển hẳn app sang `tts.infer(text, voice=..., temperature=0.7, top_k=30)` (non-streaming, không có lỗi ghép cửa sổ) — hạ `temperature` 1.0→0.7, `top_k` 50→30 để giảm lấy mẫu chệch vào vòng lặp.
  - **Patch 4 mới:** thêm `repeat_penalty=1.3` vào 2 lệnh sinh GGUF (`_infer_ggml` + `_infer_stream_ggml`).
- **Kiểm chứng bằng thống kê** (vì lỗi ngẫu nhiên, nghe 1-2 lần không kết luận được): stress-test 1 đoạn 25 lần, đo độ dài audio mỗi lần — vòng lặp sẽ tạo clip dài đột biến. Kết quả: **25/25 OK, độ dài 12.99–13.29s, std 0.08s, 0 ca nghi-lặp, 0 ca trơ-header**. Bản cũ từng bắt được ca "trơ-header" (server trả mỗi header WAV, không có tiếng); bản mới không tái diễn.
- **Lưu ý build:** `fix_stream.py` chạy ở Dockerfile dòng 16, **trước** khi biên dịch `llama-cpp-python` (dòng 20) → đổi file này khiến rebuild biên dịch lại llama-cpp (vài phút). Rebuild bằng `docker compose up -d --build vieneu`.

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
| Migrate lỗi `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` — `Error 1064 syntax` | `IF NOT EXISTS` cho `ADD COLUMN` là cú pháp MariaDB, **MySQL 8.4 không hỗ trợ** dù tài liệu dễ gây nhầm | Kiểm tra cột đã tồn tại qua `information_schema.columns` trước, chỉ `ALTER TABLE` nếu chưa có |
| `<video>/<img>/<a download>` bị 401 khi AUTH bật | Các thẻ này không gắn được header `Authorization` | Middleware `RequireAuth` nhận token từ **cookie** `gostudio_token` ngoài header; frontend lưu token cả localStorage lẫn cookie |
| `{{ .authEnabled }}` render ra ` false ` (có khoảng trắng) | html/template thêm khoảng trắng khi escape giá trị trong ngữ cảnh `<script>` | Không sao — `window.AUTH_ENABLED =  false ;` vẫn là JS hợp lệ |
| TTS **thỉnh thoảng** lặp cụm từ (dù v1.2.1 đã sửa lặp cơ bản) | (1) App chạy nhánh `infer_stream` (Patch 1 khớp nguyên khối, hỏng âm thầm khi upstream đổi whitespace) → ghép cửa sổ chồng lấp nhân đôi audio; (2) lệnh sinh GGUF thiếu `repeat_penalty` (`repetitive_penalty=1.2` lúc load model sai tên → vô tác dụng) | v1.7.3: Patch 1 viết lại bằng **regex** chuyển sang `infer` non-streaming + `temperature=0.7`/`top_k=30`; thêm `repeat_penalty=1.3` vào lệnh sinh. Kiểm chứng bằng stress-test 25× đo độ dài (std 0.08s, 0 outlier) |

---

## Đang làm

- **📵 CHỦ TRƯƠNG HIỆN TẠI: chỉ phát triển LOCAL, KHÔNG deploy lên server nữa.** Các phiên bản sau vẫn làm/chạy trên máy local như cũ (auth tắt, `docker compose up -d`, port 2005). **Không upload lên VPS** cho tới khi có kinh phí nâng cấp server đủ mạnh. Bản online đã gỡ (xem dưới).
- **🛑 Đã GỠ bản online khỏi VPS (2026-07-25).** VPS (`103.232.122.136`, SSH cổng 2223, user `ubuntu`, thư mục cũ `~/gostudio`) quá nhỏ/yếu, không đủ chạy đầy đủ tính năng (đặc biệt `vieneu`/TTS: disk 90%, RAM căng) → lần chạy online xem như thất bại, đã tháo dỡ để server nhẹ hơn (disk 90%→74%, thu hồi ~4.6GB).
  - Đã làm khi gỡ: `docker-compose -f docker-compose.yml -f docker-compose.vps.yml down -v` (xóa 2 container + 4 volume riêng gostudio: mysql_data/uploads/outputs/vieneu_cache + network) → `docker rmi gostudio-app` → `docker builder prune -f` + `docker image prune -f` → `rm -rf ~/gostudio`. Backup secret ra `~/gostudio.env.bak` (còn giữ). Đã xóa proxy host `gostudio.zalink.io.vn` trong Nginx Proxy Manager.
  - **DNS: không đụng gì.** `zalink.io.vn` dùng **wildcard `*.zalink.io.vn`** (quản lý tại PA Vietnam — `ns1/ns2.pavietnam.vn`) → không có bản ghi `gostudio` riêng để xóa; xóa wildcard sẽ chết luôn `gokeep`. Sau khi bỏ proxy host, `gostudio.zalink.io.vn` không còn phục vụ app (NPM trả mặc định/404).
  - **Go Keep KHÔNG bị đụng:** `gokeep_mysql` (Up 3 tháng, healthy) + `gokeep_app` + network `gokeep_default` là external, không do compose gostudio quản lý → `down -v` không xóa. Bảng `users` dùng chung nằm trong DB Go Keep, còn nguyên.
  - **Code auth vẫn còn nguyên trong repo** (feature flag `AUTH_ENABLED`, `docker-compose.vps.yml`) → khi có server mạnh, re-deploy lại nhanh: dựng lại `.env` (hoặc dùng backup) + chạy lệnh compose VPS. TODO lúc đó: build `vieneu` (TTS) — trước bị chặn vì disk/RAM; NPM proxy cần `client_max_body_size 1024m;` (tab Advanced) cho upload video lớn.
- Repo public: https://github.com/MichaelTranTrong/gostudio
- Release mới nhất: https://github.com/MichaelTranTrong/gostudio/releases/tag/v1.7.3

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
| **1 codebase + feature flag `AUTH_ENABLED`** thay vì 2 phiên bản/nhánh | Cập nhật tính năng tự đồng bộ cả local lẫn VPS, không phải merge tay; chỉ khác biến môi trường |
| Dùng chung tài khoản Go Keep qua **kết nối MySQL thứ 2** (không proxy API) | Go Studio + Go Keep cùng VPS → nối thẳng `gokeep.users` sạch hơn, không phụ thuộc Go Keep web online |
| JWT **tự cài bằng stdlib** (HS256) | Tránh thêm dependency; JWT của Go Studio độc lập session với Go Keep |
| Token lưu **cookie + localStorage** | Cookie cho thẻ media/download (không gắn header được), localStorage cho fetch API |
| **Chỉ phát triển local, ngừng deploy VPS** (2026-07-25) | VPS hiện quá yếu, không kham nổi đầy đủ tính năng (nhất là TTS `vieneu`) → đã gỡ bản online, các version sau làm local như cũ; chờ nâng cấp server mới re-deploy (code auth/`docker-compose.vps.yml` vẫn giữ sẵn) |

---

## TODO tiếp theo

- [x] Thêm tính năng cắt video/audio (trim theo thời gian) — v1.5.0 (video) + v1.5.1 (audio) + v1.5.2 (player trực quan)
- [ ] Thêm tính năng chuyển đổi định dạng khác (MP4→WAV, MP4→AAC, ...)
- [ ] Thêm thanh tiến trình thực từ FFmpeg (parse stderr `-progress`)
- [ ] Giới hạn kích thước file upload (hiện tại không giới hạn)
- [x] Xử lý trùng tên file output (thêm suffix nếu file đã tồn tại) — `uniqueOutputPath` cho job cắt (v1.5.3); các job khác chưa áp dụng
- [ ] **Build `vieneu` (TTS) trên VPS** — tạm hoãn (cần dọn disk VPS trước; xem mục "Đang làm")
- [ ] Tự động xóa file upload/output sau N ngày
- [x] Thêm xác thực người dùng (login) — v1.7.0 (dùng chung tài khoản Go Keep, feature flag `AUTH_ENABLED`)
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
| v1.5.3 | Cắt cả file có sẵn trong app (nút ✂ Cắt ở lịch sử, `source_id`, chống trùng tên output) |
| v1.5.4 | Đổi tên file trong lịch sử (nút ✏, cột `display_name`, áp dụng cho tên tải về) |
| v1.6.0 | Ghép nối video/audio (tab Ghép nối, nút + Ghép ở lịch sử, `/api/concat/media`, chuẩn hóa độ phân giải) |
| v1.6.1 | TTS: bổ sung 6 giọng đọc từ repo model gốc (7 giọng: 3 nam + 4 nữ, Bắc/Nam) + gợi ý thể loại trong dropdown |
| v1.7.0 | Đăng ký/Đăng nhập dùng chung tài khoản Go Keep (feature flag `AUTH_ENABLED`, JWT, xác thực email, lịch sử job tách theo user) |
| v1.7.1 | Quên mật khẩu / Đặt lại mật khẩu qua email + tự đăng xuất khi token hết hạn |
| v1.7.2 | Ẩn quay/chụp màn hình khỏi bản online (chỉ dùng local trên macOS) |
| v1.7.3 | Sửa TTS thỉnh thoảng lặp chữ (chuyển sang `infer` non-streaming + `repeat_penalty` + hạ temperature) |
