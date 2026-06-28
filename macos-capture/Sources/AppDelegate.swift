import Cocoa

final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    private enum PanelState { case idle, needPermission, recording, saving }

    private var handledURL = false
    private var request: CaptureRequest?
    private var recorder: ScreenRecorder?
    private var state: PanelState = .idle
    private var statusItem: NSStatusItem?

    private var controlWindow: NSWindow?
    private var primaryButton: NSButton?
    private var statusLabel: NSTextField?

    // Nếu app chạy mà không kèm URL (vd lần đầu để đăng ký scheme) → thoát sau 3s.
    func applicationDidFinishLaunching(_ notification: Notification) {
        DispatchQueue.main.asyncAfter(deadline: .now() + 3) { [weak self] in
            if self?.handledURL == false { self?.quit() }
        }
    }

    // AppKit gọi khi mở qua gostudio://...
    // Luôn hiện panel — KHÔNG kiểm tra/xin quyền lúc khởi động.
    func application(_ application: NSApplication, open urls: [URL]) {
        // Chỉ bỏ qua URL mới khi ĐANG quay/lưu (tránh cướp tiến trình).
        // Các trạng thái khác (panel đang chờ) → mở lại với yêu cầu mới.
        if state == .recording || state == .saving { return }

        guard let url = urls.first, let request = CaptureRequest(url: url) else {
            quit()
            return
        }
        handledURL = true
        self.request = request

        // Dọn panel cũ (nếu có) rồi hiện panel cho yêu cầu mới.
        controlWindow?.orderOut(nil)
        controlWindow = nil
        state = .idle
        showControlPanel(for: request)
    }

    // MARK: - Panel điều khiển

    private func showControlPanel(for request: CaptureRequest) {
        let window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 360, height: 160),
                              styleMask: [.titled, .closable], backing: .buffered, defer: false)
        window.title = "GoStudio Capture"
        window.level = .floating
        window.isReleasedWhenClosed = false
        window.delegate = self
        window.center()

        let status = NSTextField(wrappingLabelWithString: describe(request))
        status.frame = NSRect(x: 20, y: 70, width: 320, height: 70)
        statusLabel = status

        let primary = NSButton(title: request.mode == .video ? "🎥 Bắt đầu quay" : "📷 Chụp ngay",
                               target: self, action: #selector(primaryTapped))
        primary.frame = NSRect(x: 20, y: 18, width: 220, height: 40)
        primary.bezelStyle = .rounded
        primary.keyEquivalent = "\r"
        primaryButton = primary

        let cancel = NSButton(title: "Hủy", target: self, action: #selector(cancelTapped))
        cancel.frame = NSRect(x: 250, y: 18, width: 90, height: 40)
        cancel.bezelStyle = .rounded

        window.contentView?.addSubview(status)
        window.contentView?.addSubview(primary)
        window.contentView?.addSubview(cancel)
        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        controlWindow = window
    }

    private func describe(_ request: CaptureRequest) -> String {
        let region: String = {
            switch request.region {
            case .full: return "toàn màn hình"
            case .window: return "cửa sổ"
            case .area: return "chọn vùng"
            }
        }()
        if request.mode == .video {
            let audio = request.capturesAudio ? "có âm thanh hệ thống" : "không âm thanh"
            return "Quay video — \(region), \(audio).\n" +
                   "Bấm để bắt đầu. App ẩn đi; dừng bằng nút ⏹ trên thanh menu (góc trên phải)."
        }
        return "Chụp ảnh — \(region).\nBấm để chụp."
    }

    // MARK: - Nút chính (xin quyền tại đây nếu chưa có)

    @objc private func primaryTapped() {
        guard let request = request else { quit(); return }

        switch state {
        case .needPermission:
            // Người dùng đã cấp quyền trong Settings → mở lại bằng tiến trình mới để nhận quyền.
            Permissions.relaunch(originalURL: request.rawURL)
            quit()

        case .recording:
            stopRecording()

        case .idle:
            if Permissions.hasAccess() {
                beginCapture(request)
            } else {
                enterNeedPermission()
            }

        case .saving:
            break // đang lưu, bỏ qua thao tác
        }
    }

    private func enterNeedPermission() {
        Permissions.request()
        Permissions.openSettings()
        state = .needPermission
        statusLabel?.stringValue =
            "Hãy bật quyền “Quay màn hình” cho GoStudio Capture trong System Settings vừa mở, " +
            "rồi bấm “Mở lại”."
        primaryButton?.title = "Mở lại sau khi cấp quyền"
    }

    private func beginCapture(_ request: CaptureRequest) {
        switch request.mode {
        case .screenshot:
            controlWindow?.orderOut(nil) // ẩn panel để không lọt vào ảnh
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                self.doScreenshot(request)
            }
        case .video:
            state = .recording
            startRecording(request)
            // Ẩn giao diện app khỏi màn hình đang quay…
            controlWindow?.orderOut(nil)
            // …và hiện nút ⏹ trên thanh menu để dừng (giống macOS).
            showStopStatusItem()
        }
    }

    private func showStopStatusItem() {
        let item = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        item.button?.title = "⏹ Dừng quay"
        item.button?.target = self
        item.button?.action = #selector(stopTapped)
        statusItem = item
    }

    @objc private func stopTapped() {
        stopRecording()
    }

    @objc private func cancelTapped() {
        quit()
    }

    // Đóng cửa sổ panel (nút đỏ) khi chưa quay → thoát app, tránh để lại tiến trình kẹt.
    func windowWillClose(_ notification: Notification) {
        if state == .idle || state == .needPermission {
            quit()
        }
    }

    // MARK: - Chụp ảnh

    private func doScreenshot(_ request: CaptureRequest) {
        let output = Config.tempFile(prefix: "screenshot", ext: "png")
        if Screenshotter.capture(region: request.region, to: output) {
            uploadAndQuit(fileURL: output, type: "screenshot")
        } else {
            quit()
        }
    }

    // MARK: - Quay video

    private func startRecording(_ request: CaptureRequest) {
        let output = Config.tempFile(prefix: "recording", ext: "mp4")
        let recorder = ScreenRecorder(outputURL: output, captureAudio: request.capturesAudio)
        self.recorder = recorder
        Task {
            do {
                try await recorder.start()
            } catch {
                await MainActor.run { self.quit() }
            }
        }
    }

    private func stopRecording() {
        guard state == .recording else { return } // chống dừng trùng
        state = .saving
        if let item = statusItem {
            NSStatusBar.system.removeStatusItem(item)
            statusItem = nil
        }
        Task {
            let url = await recorder?.stop()
            await MainActor.run {
                if let url = url {
                    self.uploadAndQuit(fileURL: url, type: "screen_record")
                } else {
                    self.quit()
                }
            }
        }
    }

    // MARK: - Upload + thoát

    private func uploadAndQuit(fileURL: URL, type: String) {
        Uploader.upload(fileURL: fileURL, type: type) { _, _ in
            DispatchQueue.main.async {
                try? FileManager.default.removeItem(at: fileURL)
                self.quit()
            }
        }
    }

    private func quit() {
        NSApp.terminate(nil)
    }
}
