import Cocoa
import CoreGraphics

// Quản lý quyền Screen Recording (TCC) + tự relaunch.
enum Permissions {
    /// Đã được cấp quyền quay màn hình chưa.
    static func hasAccess() -> Bool {
        CGPreflightScreenCaptureAccess()
    }

    /// Bật hộp thoại xin quyền của hệ thống.
    @discardableResult
    static func request() -> Bool {
        CGRequestScreenCaptureAccess()
    }

    /// Mở thẳng pane Screen Recording trong System Settings (đỡ phải đi tìm).
    static func openSettings() {
        if let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture") {
            NSWorkspace.shared.open(url)
        }
    }

    /// Tự khởi động lại bằng cách re-trigger chính URL gostudio:// sau khi đã thoát.
    /// Tiến trình /bin/sh được reparent về launchd nên sống tiếp sau khi app này terminate;
    /// `open` lúc đó tạo instance mới — instance mới có quyền (vì là process mới sau khi cấp).
    static func relaunch(originalURL: String) {
        let task = Process()
        task.launchPath = "/bin/sh"
        task.arguments = ["-c", "sleep 1; open '\(originalURL)'"]
        try? task.run()
    }
}
