import Foundation
import CoreGraphics

// Chụp ảnh màn hình qua công cụ hệ thống /usr/sbin/screencapture.
enum Screenshotter {
    // -C: hiện con trỏ (chỉ dùng được ở chế độ KHÔNG tương tác). Mặc định screencapture ẩn con trỏ.
    static func capture(region: CaptureRegion, hideCursor: Bool, to url: URL) -> Bool {
        // -x: không phát âm thanh chụp. -i: chế độ tương tác.
        // (region=area dùng overlay kính lúp riêng → captureRect, không qua đây.)
        var args: [String]
        switch region {
        case .full:   args = hideCursor ? ["-x"] : ["-C", "-x"]  // full: bật/tắt con trỏ được
        case .window: args = ["-i", "-w", "-x"]                  // (dự phòng — app dùng captureWindow)
        case .area:   args = ["-i", "-x"]                        // (dự phòng)
        }
        args.append(url.path)
        return run(args, to: url)
    }

    // Chụp một cửa sổ theo windowID (-l), bỏ bóng đổ (-o). Không tương tác → không bị
    // overlay trong suốt phủ màn hình bắt nhầm như chế độ click-chọn `-i -w`.
    static func captureWindow(id: CGWindowID, hideCursor: Bool, to url: URL) -> Bool {
        var args = ["-l\(id)", "-o", "-x"]
        if !hideCursor { args.insert("-C", at: 0) }
        args.append(url.path)
        return run(args, to: url)
    }

    // Chụp đúng một vùng (điểm, gốc top-left) không tương tác — dùng sau overlay kính lúp.
    static func captureRect(_ rect: CGRect, hideCursor: Bool, to url: URL) -> Bool {
        let r = "\(Int(rect.minX)),\(Int(rect.minY)),\(Int(rect.width)),\(Int(rect.height))"
        var args = ["-R", r, "-x"]
        if !hideCursor { args.insert("-C", at: 0) }
        args.append(url.path)
        return run(args, to: url)
    }

    private static func run(_ args: [String], to url: URL) -> Bool {
        let task = Process()
        task.launchPath = "/usr/sbin/screencapture"
        task.arguments = args
        do {
            try task.run()
            task.waitUntilExit()
        } catch {
            return false
        }
        return task.terminationStatus == 0 && FileManager.default.fileExists(atPath: url.path)
    }
}
