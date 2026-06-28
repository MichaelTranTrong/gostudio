# Thiết kế: Quay màn hình & Chụp ảnh trên macOS cho Go Studio

> Bản phác kiến trúc — chốt từ thảo luận. Mục tiêu: thêm khả năng **quay video màn hình** và
> **chụp ảnh màn hình** MacBook, trong khi vẫn giữ Go Studio (Docker) làm nơi **lưu trữ + xử lý**.
> Trạng thái: **chưa code**, đây là mốc tham chiếu trước khi bắt đầu.

---

## 1. Vấn đề gốc & lựa chọn

Go Studio chạy trong container Linux (Alpine) → **không nhìn thấy màn hình macOS**. Không thể
"bấm nút web → server quay màn hình" như cách FFmpeg trích MP3. Capture bắt buộc xảy ra ở
phía OS (native), không thể trong Docker.

Đã cân nhắc 2 hướng:

| Hướng | Kết luận |
|---|---|
| **A. Quay trong trình duyệt** (`getDisplayMedia` + `MediaRecorder`) | Phương án dự phòng. Zero-install, đa nền tảng, nhưng output `.webm`, system audio kém trên macOS, không hotkey/chụp ảnh ngon. |
| **B. Native macOS app (đã chọn)** | Chất lượng cao (ScreenCaptureKit), bắt được system audio, chụp ảnh đúng nghĩa. Đổi lại thêm codebase Swift, chỉ chạy macOS. |

**Quyết định: làm hướng B** theo mô hình hybrid — native chỉ lo *capture*, mọi thứ khác đẩy về
Go Studio.

---

## 2. Mô hình tổng thể

```
┌──────────────────────────┐                 ┌──────────────────────────┐
│  Web app (tab trình duyệt)│                 │  Go Studio (Docker)       │
│  localhost:2005           │                 │                          │
│                           │                 │                          │
│  bấm "Chụp/Quay" ─────────┼─ gostudio:// ──►│                          │
│                           │  (URL scheme)   │                          │
│                           │                 │                          │
│  poll /api/jobs (đã có) ◄─┼─────────────────┤  POST /api/upload        │
│  → hiện job mới           │                 │  → tạo job               │
└──────────────────────────┘                 │  → FFmpeg transcode      │
              ▲                               │  → lưu DB + outputs/     │
              │ gostudio://capture?...        │  → lịch sử chung         │
              ▼                               └──────────────────────────┘
┌──────────────────────────┐                            ▲
│  Native capture app       │   POST file (HTTP)         │
│  (dùng-một-lần)           │────────────────────────────┘
│                           │   http://localhost:2005
│  • ScreenCaptureKit (video)│
│  • screencapture (ảnh)    │
│  • bật khi web gọi        │
│  • xong → tự thoát        │
└──────────────────────────┘
```

**Vòng lặp tự khép qua hệ thống job/history sẵn có**: web chỉ làm 2 việc nó đã làm rồi —
mở 1 link và poll lịch sử. Native lo phần capture + upload.

---

## 3. Native capture app — nguyên tắc cốt lõi

### 3.1. Dùng-một-lần (ephemeral), KHÔNG chạy ngầm
- Không `LSUIElement` menu bar thường trú, không HTTP server nền.
- Vòng đời: **web gọi → app bật (cold start ~0.5–1s) → capture → upload → `NSApp.terminate`**.
- Khi không dùng: **không process nào tồn tại**.

```
        OFF (không process)
          │  gostudio://capture?...
          ▼
   macOS bật app
          │
   chụp ảnh / quay video
          │
   upload về localhost:2005
          │
   tự thoát ──► OFF
```

### 3.2. Panel điều khiển — không tự quay
Khi web gọi, app bật lên và **hiện panel điều khiển** (không tự chụp/quay ngay).
Người dùng bấm nút mới bắt đầu — chủ động về thời điểm.
- **Chụp ảnh** — `launch → panel "📷 Chụp ngay" → (ẩn panel) → capture → upload → quit`.
- **Quay video** — `launch → panel "🎥 Bắt đầu quay" → bấm → ẨN panel + hiện nút ⏹ trên thanh
  menu (giống macOS) → bấm ⏹ → dừng → upload → quit`. Panel ẩn để không lọt khung hình.
  (Đã thử phím tắt toàn cục qua Carbon `RegisterEventHotKey` nhưng sự kiện không tới được app
  helper kiểu này — bỏ; nút menu bar là cách dừng chuẩn, giống chính macOS.)
- **Re-entrancy:** app dùng-một-lần — sau khi đã nhận 1 URL thì bỏ qua URL mới, tránh lần kích
  hoạt sau cướp tiến trình đang quay (lỗi từng gặp: lần bấm 2 vô tình dừng bản quay 1).
- Có nút **Hủy** để thoát mà không capture.
- App chỉ hiển thị panel khi đang dùng — không chạy ngầm.

### 3.3. Công nghệ
- Video + system audio: **ScreenCaptureKit** (macOS 12.3+, Ventura có sẵn).
- Ảnh: `screencapture` CLI hoặc ScreenCaptureKit single-frame.
- Encode: H.264/HEVC phần cứng → MP4/MOV đưa thẳng về backend.

---

## 4. Kích hoạt từ web: URL scheme `gostudio://`

Native app đăng ký scheme trong `Info.plist` (`CFBundleURLTypes`). Web mở link:

```js
// Chụp ảnh cửa sổ
window.location.href = 'gostudio://capture?mode=screenshot&region=window';

// Quay video toàn màn hình kèm system audio
window.location.href = 'gostudio://capture?mode=video&audio=system';
```

### Bộ tham số (nháp)
| Tham số | Giá trị | Ý nghĩa |
|---|---|---|
| `mode` | `screenshot` \| `video` | Chụp ảnh hay quay video |
| `region` | `full` \| `window` \| `area` | Toàn màn hình / cửa sổ / chọn vùng |
| `audio` | `none` \| `system` \| `mic` \| `both` | Nguồn âm thanh (chỉ cho video) |
| `display` | `0`,`1`,… | Màn hình nào (đa màn hình) |
| `delay` | số giây | Hẹn giờ trước khi chụp |

→ Web quyết định "chụp gì", native lo "chụp thế nào".

### Lưu ý thực tế
1. **Lần đầu trình duyệt hỏi** "Cho phép trang này mở GoStudio?" — có checkbox "luôn cho phép".
2. **App phải cài & chạy ≥1 lần** để Launch Services biết scheme.
3. **Web khó biết app đã cài chưa** — nếu sau X giây không thấy job mới → gợi ý "Đã cài GoStudio Capture chưa?".

---

## 5. Quyền Screen Recording (TCC)

Screen Recording là quyền *nặng* hơn micro/camera: không bấm "Allow" inline được, phải vào
**System Settings → Privacy & Security → Screen Recording**, gạt toggle, rồi **relaunch app 1 lần**.

### Giảm ma sát
- **Deep-link mở thẳng pane** (đỡ phải đi tìm):
  ```swift
  let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture")!
  NSWorkspace.shared.open(url)
  ```
  (Vẫn phải tự gạt toggle — macOS không cho app tự cấp quyền cho mình.)

- **Tự relaunch sau khi cấp quyền** (để khỏi đóng-mở tay):
  ```swift
  if CGPreflightScreenCaptureAccess() {        // poll tới khi quyền = true
      let path = Bundle.main.bundlePath
      let task = Process()
      task.launchPath = "/usr/bin/open"
      task.arguments = ["-n", path]            // open -n: instance mới độc lập
      task.launch()
      NSApp.terminate(nil)
  }
  ```
  Mẹo: dùng `open` (do launchd chạy, tiến trình riêng) để bản mới sống độc lập khi app cũ thoát.

- macOS Ventura cũng tự hiện nút **"Quit & Reopen"** khi cấp quyền cho app đang chạy → bấm là xong.

### "Relaunch" làm rõ
- Chỉ là **1 lần đóng-mở ngay sau khi cấp quyền** (vì macOS đọc trạng thái quyền lúc khởi động).
- **Không** liên quan đến lúc dùng hằng ngày — ngày thường chỉ "launch → chụp → thoát".

---

## 6. Ký app & chứng chỉ — **GÁC LẠI, tính sau**

Không chặn việc bắt đầu. Tóm tắt để sau quyết:

| Kiểu ký | Cần gì | TCC nhớ quyền? |
|---|---|---|
| Ad-hoc (`codesign -s -`) | Không gì | ❌ Đổi identity mỗi build → cấp quyền lại mỗi lần |
| **Self-signed** (Keychain Access, free) | Không cần Apple ID | ✅ Cấp quyền 1 lần đời |
| Developer ID ($99/năm) | Apple Developer account | ✅ — chỉ cần khi **chia cho máy Mac khác** / App Store |

- Dev ban đầu: cứ **ad-hoc**, chấp nhận cấp quyền lại vài lần khi build.
- Khi dùng ổn định: tạo **self-signed cert** (10 phút) → cấp quyền 1 lần.
- **$99 chỉ đặt ra nếu sau này phân phối** — để sau.

---

## 7. Thay đổi phía Go Studio (backend)

Tái dùng tối đa pipeline job hiện có:

- **Endpoint nhận file:** dùng lại `POST /api/upload` (hoặc thêm `/api/capture/upload`) — nhận
  MP4/MOV/PNG từ native app.
- **Job type mới:** thêm `screen_record` / `screenshot` vào cột `type` của bảng `jobs`
  (cạnh `convert`, `tts`).
- **Transcode (tùy chọn):** nếu native gửi MOV → FFmpeg sang MP4; ảnh PNG có thể giữ nguyên.
- **Lịch sử:** badge phân biệt loại job (MP4 / TTS / Screen / Photo) — mở rộng UI hiện có.
- **Bảo mật:** endpoint chỉ nghe localhost; cân nhắc token đơn giản để chỉ native app local gọi được.

Không cần thay đổi lớn — chủ yếu là thêm `type` và badge.

---

## 8. Môi trường mục tiêu: macOS Ventura 13.4.1

Mọi viên gạch đều có sẵn, và **né được** phiền của Sequoia:

| Thành phần | Ventura 13.4.1 |
|---|---|
| ScreenCaptureKit | ✅ (từ 12.3) |
| `CGPreflight/RequestScreenCaptureAccess()` | ✅ (từ 11) |
| Nút "Quit & Reopen" khi cấp quyền | ✅ |
| Deep-link `?Privacy_ScreenCapture` | ✅ |
| Self-signed giữ TCC | ✅ |
| **Nhắc lại quyền định kỳ hàng tuần** | ❌ — chỉ Sequoia 15 mới có → **Ventura yên vĩnh viễn sau khi cấp 1 lần** |

---

## 9. Lộ trình triển khai (đề xuất)

1. ✅ **Backend:** job type `screenshot`/`screen_record`, endpoint `POST /api/capture/upload`,
   badge lịch sử, download content-type theo đuôi file. (`internal/handlers/capture.go`) — đã verify.
2. ✅ **Native app khung:** Swift app dùng-một-lần, scheme `gostudio://`, parse tham số, tự thoát.
   (`macos-capture/`) — compile sạch, đóng gói `.app`, scheme đã đăng ký.
3. ✅ **Chụp ảnh:** mode `screenshot` qua `screencapture` (full/window/area) → upload.
4. ✅ **Quay video:** mode `video` ScreenCaptureKit + nút Stop + system audio → upload.
   (Còn TODO: region window/area, mic, retina scale.)
5. ✅ **Quyền & relaunch:** `CGPreflight/Request`, deep-link Settings, tự relaunch.
6. ✅ **Web UI:** tab "Quay màn hình" gọi `gostudio://`, phát hiện "chưa cài app" sau 15s.
7. ⏸ **Ký app:** đang ad-hoc; self-signed cert gác lại tới khi dùng ổn định.

> Trạng thái: khung end-to-end đã dựng xong. Việc còn lại là **chạy thử thật trên máy**
> (cấp quyền lần đầu, bấm web → app quay/chụp → file về lịch sử) và tinh chỉnh các TODO.

---

## 10. Câu hỏi mở

- Native app: Swift thuần (Xcode) hay có thể wrap gọn hơn? (Swift + ScreenCaptureKit là chuẩn.)
- Có cần preview/điều khiển realtime từ web không? (Nếu cần → phải có local HTTP server → phá mô
  hình "không chạy ngầm". Hiện **không** làm.)
- Định dạng output chuẩn hóa: MP4 (H.264) cho video, PNG cho ảnh?
- Cơ chế bảo mật endpoint upload (token local)?
