import Foundation

// Chụp ảnh màn hình qua công cụ hệ thống /usr/sbin/screencapture.
enum Screenshotter {
    static func capture(region: CaptureRegion, to url: URL) -> Bool {
        // -x: không phát âm thanh chụp. -i: chế độ tương tác (bắt buộc để chọn).
        var args: [String]
        switch region {
        case .full:   args = ["-x"]              // toàn màn hình chính, chụp ngay
        case .window: args = ["-i", "-w", "-x"]  // tương tác, chỉ chọn cửa sổ
        case .area:   args = ["-i", "-x"]        // tương tác, kéo chọn vùng
        }
        args.append(url.path)

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
