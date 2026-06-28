import Cocoa
import ScreenCaptureKit

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
    private var windowPopup: NSPopUpButton?
    private var pickableWindows: [SCWindow] = []
    private var cropRect: CGRect?   // vùng cắt cho quay video theo vùng

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
        let needsWindowPicker = (request.mode == .video && request.region == .window)
        let height: CGFloat = needsWindowPicker ? 205 : 160

        let window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 360, height: height),
                              styleMask: [.titled, .closable], backing: .buffered, defer: false)
        window.title = "GoStudio Capture"
        window.level = .floating
        window.isReleasedWhenClosed = false
        window.delegate = self
        window.center()

        let status = NSTextField(wrappingLabelWithString: describe(request))
        status.frame = NSRect(x: 20, y: height - 90, width: 320, height: 70)
        statusLabel = status
        window.contentView?.addSubview(status)

        // Dropdown chọn cửa sổ (chỉ cho quay video theo cửa sổ).
        if needsWindowPicker {
            let popup = NSPopUpButton(frame: NSRect(x: 20, y: 66, width: 320, height: 26))
            popup.addItem(withTitle: "Đang tải danh sách cửa sổ…")
            popup.isEnabled = false
            windowPopup = popup
            window.contentView?.addSubview(popup)
            loadWindowsIntoPopup()
        }

        let primary = NSButton(title: request.mode == .video ? "🎥 Bắt đầu quay" : "📷 Chụp ngay",
                               target: self, action: #selector(primaryTapped))
        primary.frame = NSRect(x: 20, y: 18, width: 220, height: 40)
        primary.bezelStyle = .rounded
        primary.keyEquivalent = "\r"
        primaryButton = primary

        let cancel = NSButton(title: "Hủy", target: self, action: #selector(cancelTapped))
        cancel.frame = NSRect(x: 250, y: 18, width: 90, height: 40)
        cancel.bezelStyle = .rounded

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
            if request.region == .area {
                // Đợi panel biến mất rồi mở overlay kính lúp (chụp ảnh nền sạch).
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                    self.selectAreaThenScreenshot()
                }
            } else {
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                    self.doScreenshot(request)
                }
            }
        case .video:
            if request.region == .window {
                let idx = windowPopup?.indexOfSelectedItem ?? -1
                guard idx >= 0, idx < pickableWindows.count else {
                    statusLabel?.stringValue = "Chưa chọn được cửa sổ. Hãy chọn trong danh sách rồi bấm lại."
                    return
                }
                beginVideoRecording(window: pickableWindows[idx])
            } else if request.region == .area {
                controlWindow?.orderOut(nil)
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.3) {
                    self.selectAreaThenRecord()
                }
            } else {
                beginVideoRecording(window: nil) // toàn màn hình
            }
        }
    }

    // Chọn vùng (overlay kính lúp) → quay full màn hình → backend cắt vùng bằng FFmpeg.
    private func selectAreaThenRecord() {
        AreaSelector.present { rect in
            guard let rect = rect else { self.quit(); return }
            self.cropRect = rect
            self.beginVideoRecording(window: nil)
        }
    }

    private func beginVideoRecording(window: SCWindow?) {
        state = .recording
        startRecording(window: window)
        // Ẩn giao diện app khỏi khung hình…
        controlWindow?.orderOut(nil)
        // …và hiện nút ⏹ trên thanh menu để dừng (giống macOS).
        showStopStatusItem()
    }

    // MARK: - Nạp danh sách cửa sổ vào dropdown (region=window)

    private func loadWindowsIntoPopup() {
        Task {
            let content = try? await SCShareableContent.excludingDesktopWindows(true, onScreenWindowsOnly: true)
            let mine = Bundle.main.bundleIdentifier
            let windows = (content?.windows ?? []).filter {
                $0.isOnScreen
                    && ($0.title?.isEmpty == false)
                    && $0.owningApplication?.bundleIdentifier != mine
                    && $0.frame.width > 80 && $0.frame.height > 80
            }
            await MainActor.run {
                self.pickableWindows = windows
                guard let popup = self.windowPopup else { return }
                popup.removeAllItems()
                if windows.isEmpty {
                    popup.addItem(withTitle: "Không tìm thấy cửa sổ nào")
                    popup.isEnabled = false
                } else {
                    for w in windows {
                        let app = w.owningApplication?.applicationName ?? "?"
                        popup.addItem(withTitle: "\(app) — \(w.title ?? "")")
                    }
                    popup.isEnabled = true
                }
            }
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

    private func selectAreaThenScreenshot() {
        AreaSelector.present { rect in
            guard let rect = rect else { self.quit(); return } // hủy
            let output = Config.tempFile(prefix: "screenshot", ext: "png")
            DispatchQueue.global().async {
                let ok = Screenshotter.captureRect(rect, to: output)
                DispatchQueue.main.async {
                    if ok {
                        self.uploadAndQuit(fileURL: output, type: "screenshot")
                    } else {
                        self.quit()
                    }
                }
            }
        }
    }

    private func doScreenshot(_ request: CaptureRequest) {
        let output = Config.tempFile(prefix: "screenshot", ext: "png")
        // Chạy nền: screencapture tương tác (-i) sẽ chặn cho tới khi chọn xong,
        // không được block main run loop.
        DispatchQueue.global().async {
            let ok = Screenshotter.capture(region: request.region, to: output)
            DispatchQueue.main.async {
                if ok {
                    self.uploadAndQuit(fileURL: output, type: "screenshot")
                } else {
                    self.quit()
                }
            }
        }
    }

    // MARK: - Quay video

    private func startRecording(window: SCWindow?) {
        let output = Config.tempFile(prefix: "recording", ext: "mp4")
        let recorder = ScreenRecorder(outputURL: output,
                                      captureAudio: request?.capturesAudio ?? false,
                                      window: window)
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
        Uploader.upload(fileURL: fileURL, type: type, crop: cropRect) { _, _ in
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
