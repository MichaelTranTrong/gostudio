#!/usr/bin/env bash
# Build GoStudio Capture.app từ mã nguồn Swift (không cần mở Xcode GUI).
# Yêu cầu: Xcode hoặc Command Line Tools (có swiftc + SDK macOS 13+).
set -euo pipefail
cd "$(dirname "$0")"

APP="GoStudioCapture.app"
BIN="GoStudioCapture"
ARCH="$(uname -m)"
TARGET="${ARCH}-apple-macos13.0"

echo "→ Biên dịch ($TARGET)…"
swiftc -O \
    -target "$TARGET" \
    -framework Cocoa \
    -framework ScreenCaptureKit \
    -framework AVFoundation \
    -o "$BIN" \
    Sources/*.swift

echo "→ Đóng gói $APP…"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS"
mv "$BIN" "$APP/Contents/MacOS/$BIN"
cp Info.plist "$APP/Contents/Info.plist"

# Ưu tiên self-signed cert "GoStudio Dev" (chạy ./make-cert.sh để tạo) → identity ổn định,
# quyền Screen Recording không bị hỏi lại. Nếu chưa có thì ký ad-hoc (đổi identity mỗi build).
SIGN_ID="GoStudio Dev"
if security find-certificate -c "$SIGN_ID" >/dev/null 2>&1; then
    echo "→ Ký bằng cert '$SIGN_ID' (identity ổn định)…"
    codesign --force --deep --sign "$SIGN_ID" "$APP"
else
    echo "→ Ký ad-hoc (chưa có cert — chạy ./make-cert.sh để quyền dính vĩnh viễn)…"
    codesign --force --deep --sign - "$APP"
fi

echo "→ Đăng ký URL scheme với Launch Services…"
LSREGISTER="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"
"$LSREGISTER" -f "$PWD/$APP" || true

echo
echo "✅ Xong: $PWD/$APP"
echo "   Lần đầu chạy:  open '$PWD/$APP'   (để cấp quyền Screen Recording)"
echo "   Sau đó web có thể kích hoạt qua gostudio://capture?..."
