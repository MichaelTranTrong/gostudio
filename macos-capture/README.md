# GoStudio Capture (native macOS)

App native dùng-một-lần: web Go Studio kích hoạt qua URL scheme `gostudio://`, app quay/chụp
màn hình rồi upload về backend và tự thoát. Không chạy ngầm.

Xem thiết kế tổng thể: [../docs/screen-capture-design.md](../docs/screen-capture-design.md)

## Yêu cầu
- macOS 13+ (đã test mục tiêu: Ventura 13.4.1) — ScreenCaptureKit + capturesAudio cần macOS 13.
- Xcode hoặc Command Line Tools (`xcode-select --install`) để có `swiftc` + SDK.

## Build & cài
```bash
cd macos-capture
./build.sh
# Lần đầu chạy để đăng ký scheme + cấp quyền:
open ./GoStudioCapture.app
```
Lần đầu, macOS sẽ hỏi quyền **Screen Recording**: vào System Settings (app tự mở đúng pane),
bật toggle cho **GoStudio Capture**, rồi để app tự khởi động lại (hoặc bấm "Thử lại").

> Mở bằng Xcode: kéo thư mục `Sources/` + `Info.plist` vào một macOS App target, hoặc dùng
> `build.sh` để build nhanh từ terminal.

## Cách hoạt động
1. Web bấm "Chụp ảnh" / "Quay video" → mở `gostudio://capture?mode=...&region=...&audio=...`.
2. macOS bật app → app kiểm tra quyền → chụp (`screencapture`) hoặc quay (ScreenCaptureKit).
3. Quay video: hiện 1 nút **⏹ Dừng & Lưu**; bấm để kết thúc.
4. App upload file về `POST /api/capture/upload` của Go Studio → job hiện trong lịch sử.
5. App **tự thoát**.

### Tham số URL
| Tham số | Giá trị | Ghi chú |
|---|---|---|
| `mode` | `screenshot` \| `video` | bắt buộc |
| `region` | `full` \| `window` \| `area` | ảnh: đủ cả 3; video: hiện chỉ full (TODO) |
| `audio` | `none` \| `system` \| `mic` \| `both` | video: hiện hỗ trợ `none`/`system` (mic/both TODO) |

## Cấu hình
Sửa trong `Sources/Config.swift`:
- `backendURL` — mặc định `http://localhost:2005`.
- `captureToken` — khớp `CAPTURE_TOKEN` ở backend nếu bật.

## Ký app — NÊN làm ngay để quyền không bị hỏi lại
`build.sh` ký **ad-hoc** nếu chưa có cert: mỗi lần build đổi identity → quyền Screen Recording
đã cấp bị coi là của identity cũ → **app báo "chưa cấp" dù Settings hiện đã cấp** (lỗi kẹt).

Khắc phục bằng self-signed cert (miễn phí, không cần Apple account):
```bash
./make-cert.sh   # tạo cert "GoStudio Dev" (có thể hỏi mật khẩu đăng nhập)
./build.sh       # tự ký bằng cert → identity ổn định
```
Nếu trước đó đã build ad-hoc, xóa trạng thái quyền cũ rồi cấp lại một lần:
```bash
tccutil reset ScreenCapture com.gostudio.capture
```
Từ đó quyền **dính vĩnh viễn** qua mọi lần build. Developer account $99 chỉ cần nếu sau này
phân phối cho máy Mac khác.

## Hạn chế hiện tại (TODO)
- Video chỉ quay màn hình chính, full-screen (chưa theo `region` window/area).
- Audio video: mới `system`; `mic`/`both` chưa trộn micro.
- Chưa nhân scale factor cho màn hình Retina (video có thể nhỏ hơn độ phân giải thật).
