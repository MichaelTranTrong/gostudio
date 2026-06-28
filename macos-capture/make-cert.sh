#!/usr/bin/env bash
# Tạo self-signed code-signing certificate "GoStudio Dev" trong login keychain.
# Mục đích: cho app một identity ỔN ĐỊNH để quyền Screen Recording (TCC) không bị
# hỏi lại mỗi lần build (khác hẳn ad-hoc đổi identity liên tục).
#
# MIỄN PHÍ — không cần Apple Developer account. Chạy MỘT lần.
# Có thể bị macOS hỏi mật khẩu đăng nhập / "Allow" khi import — đó là bình thường.
set -euo pipefail

CN="GoStudio Dev"
KEYCHAIN="$HOME/Library/Keychains/login.keychain-db"

if security find-certificate -c "$CN" "$KEYCHAIN" >/dev/null 2>&1; then
    echo "✓ Cert '$CN' đã tồn tại — không cần tạo lại."
    exit 0
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "→ Tạo khóa + chứng chỉ tự ký (codeSigning)…"
openssl req -x509 -newkey rsa:2048 -nodes -days 3650 \
    -keyout "$TMP/key.pem" -out "$TMP/cert.pem" \
    -subj "/CN=$CN" \
    -addext "extendedKeyUsage=critical,codeSigning" \
    -addext "basicConstraints=critical,CA:false" \
    -addext "keyUsage=critical,digitalSignature"

# -legacy: bắt buộc với OpenSSL 3 để PKCS12 dùng thuật toán SHA1/3DES mà
# công cụ `security` của macOS đọc được (mặc định OpenSSL 3 không tương thích).
P12_PASS="gostudio"
openssl pkcs12 -export -legacy -inkey "$TMP/key.pem" -in "$TMP/cert.pem" \
    -out "$TMP/id.p12" -passout "pass:$P12_PASS" -name "$CN"

echo "→ Import vào login keychain (cho phép codesign dùng)…"
security import "$TMP/id.p12" -k "$KEYCHAIN" -P "$P12_PASS" -A -T /usr/bin/codesign

echo
echo "✅ Đã tạo identity '$CN'."
echo "   Giờ chạy lại ./build.sh — nó sẽ tự ký bằng cert này thay vì ad-hoc."
echo "   Lần cấp quyền Screen Recording tiếp theo sẽ là LẦN CUỐI (quyền dính vĩnh viễn)."
