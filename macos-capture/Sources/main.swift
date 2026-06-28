import Cocoa

// GoStudio Capture — app native macOS dùng-một-lần.
// Được web app kích hoạt qua URL scheme gostudio://capture?...
// Quay/chụp màn hình → upload về Go Studio → tự thoát.
//
// .accessory: không icon Dock, không cửa sổ thường trú (khớp LSUIElement).

let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.setActivationPolicy(.accessory)
app.run()
