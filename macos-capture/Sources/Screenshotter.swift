import Foundation

// Chụp ảnh màn hình qua công cụ hệ thống /usr/sbin/screencapture.
enum Screenshotter {
    static func capture(region: CaptureRegion, to url: URL) -> Bool {
        // -x: không phát âm thanh chụp. -i: chế độ tương tác.
        // (region=area dùng overlay kính lúp riêng → captureRect, không qua đây.)
        var args: [String]
        switch region {
        case .full:   args = ["-x"]              // toàn màn hình chính, chụp ngay
        case .window: args = ["-i", "-w", "-x"]  // tương tác, chỉ chọn cửa sổ
        case .area:   args = ["-i", "-x"]        // (dự phòng) tương tác, kéo chọn vùng
        }
        args.append(url.path)
        return run(args, to: url)
    }

    // Chụp đúng một vùng (điểm, gốc top-left) không tương tác — dùng sau overlay kính lúp.
    static func captureRect(_ rect: CGRect, to url: URL) -> Bool {
        let r = "\(Int(rect.minX)),\(Int(rect.minY)),\(Int(rect.width)),\(Int(rect.height))"
        return run(["-R", r, "-x", url.path], to: url)
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
