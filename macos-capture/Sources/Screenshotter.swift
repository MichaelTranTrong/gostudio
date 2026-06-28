import Foundation

// Chụp ảnh màn hình qua công cụ hệ thống /usr/sbin/screencapture.
enum Screenshotter {
    static func capture(region: CaptureRegion, to url: URL) -> Bool {
        var args = ["-x"] // -x: không phát âm thanh chụp
        switch region {
        case .full:   break          // toàn màn hình chính
        case .window: args.append("-w") // chọn cửa sổ (tương tác)
        case .area:   args.append("-i") // chọn vùng (tương tác)
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
