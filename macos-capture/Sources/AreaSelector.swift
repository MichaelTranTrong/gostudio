import Cocoa

// Overlay chọn vùng kèm KÍNH LÚP phóng to pixel (vì macOS không có sẵn).
// Trả về CGRect theo ĐIỂM, gốc TOP-LEFT màn hình chính (khớp `screencapture -R`
// và crop FFmpeg), hoặc nil nếu hủy (Esc / vùng quá nhỏ).
enum AreaSelector {
    private static var window: NSWindow?

    static func present(completion: @escaping (CGRect?) -> Void) {
        guard let screen = NSScreen.main,
              let image = CGDisplayCreateImage(CGMainDisplayID()) else {
            completion(nil)
            return
        }

        let win = OverlayWindow(contentRect: screen.frame, styleMask: .borderless,
                                backing: .buffered, defer: false)
        win.level = .screenSaver
        win.isOpaque = false
        win.backgroundColor = .clear
        win.hasShadow = false
        win.acceptsMouseMovedEvents = true

        let view = SelectionView(frame: NSRect(origin: .zero, size: screen.frame.size),
                                 image: image,
                                 screenHeight: screen.frame.height)
        view.onFinish = { rect in
            win.orderOut(nil)
            AreaSelector.window = nil
            completion(rect)
        }
        win.contentView = view
        window = win
        win.makeKeyAndOrderFront(nil)
        win.makeFirstResponder(view)
        NSApp.activate(ignoringOtherApps: true)
    }
}

// Borderless window phải override để nhận phím (Esc) và chuột.
private final class OverlayWindow: NSWindow {
    override var canBecomeKey: Bool { true }
    override var canBecomeMain: Bool { true }
}

private final class SelectionView: NSView {
    var onFinish: ((CGRect?) -> Void)?

    private let nsImage: NSImage
    private let screenHeight: CGFloat
    private var start: NSPoint?
    private var current: NSPoint?

    init(frame: NSRect, image: CGImage, screenHeight: CGFloat) {
        self.nsImage = NSImage(cgImage: image, size: frame.size)
        self.screenHeight = screenHeight
        super.init(frame: frame)
    }
    required init?(coder: NSCoder) { fatalError() }

    override var acceptsFirstResponder: Bool { true }

    override func viewDidMoveToWindow() {
        addTrackingArea(NSTrackingArea(rect: bounds,
                                       options: [.activeAlways, .mouseMoved, .inVisibleRect],
                                       owner: self, userInfo: nil))
        NSCursor.crosshair.set()
    }

    // MARK: - Chuột

    override func mouseDown(with event: NSEvent) {
        start = convert(event.locationInWindow, from: nil)
        current = start
        needsDisplay = true
    }
    override func mouseDragged(with event: NSEvent) {
        current = convert(event.locationInWindow, from: nil)
        needsDisplay = true
    }
    override func mouseMoved(with event: NSEvent) {
        current = convert(event.locationInWindow, from: nil)
        needsDisplay = true
    }
    override func mouseUp(with event: NSEvent) {
        if let r = selectionRect(), r.width >= 4, r.height >= 4 {
            // view gốc bottom-left → đổi sang top-left; kích thước chẵn cho H.264.
            let w = (Int(r.width) / 2) * 2
            let h = (Int(r.height) / 2) * 2
            let topLeft = CGRect(x: Int(r.minX), y: Int(screenHeight - r.maxY), width: w, height: h)
            onFinish?(topLeft)
        } else {
            onFinish?(nil)
        }
    }
    override func keyDown(with event: NSEvent) {
        if event.keyCode == 53 { onFinish?(nil) } // Esc
    }

    private func selectionRect() -> CGRect? {
        guard let s = start, let c = current else { return nil }
        return CGRect(x: min(s.x, c.x), y: min(s.y, c.y),
                      width: abs(s.x - c.x), height: abs(s.y - c.y))
    }

    // MARK: - Vẽ

    override func draw(_ dirtyRect: NSRect) {
        nsImage.draw(in: bounds)                       // nền: ảnh chụp tĩnh
        NSColor(white: 0, alpha: 0.35).setFill()       // phủ mờ
        bounds.fill()

        if let r = selectionRect() {                   // vùng chọn: rõ + viền
            nsImage.draw(in: r, from: r, operation: .copy, fraction: 1)
            NSColor.white.setStroke()
            let p = NSBezierPath(rect: r); p.lineWidth = 1; p.stroke()
        }

        if let c = current { drawLoupe(at: c) }
    }

    private func drawLoupe(at p: NSPoint) {
        let radius: CGFloat = 88     // kính lúp lớn hơn
        let sample: CGFloat = 26     // góc nhìn rộng hơn (thấy nhiều vùng hơn), phóng ≈ 6.8x
        let offset: CGFloat = 105    // khoảng cách kính lúp so với con trỏ

        var center = NSPoint(x: p.x + offset, y: p.y + offset)
        if center.x + radius > bounds.maxX { center.x = p.x - offset }
        if center.y + radius > bounds.maxY { center.y = p.y - offset }
        let loupe = NSRect(x: center.x - radius, y: center.y - radius, width: radius * 2, height: radius * 2)

        // Phóng to vùng quanh con trỏ vào trong vòng tròn (pixel sắc, không nội suy).
        NSGraphicsContext.saveGraphicsState()
        NSBezierPath(ovalIn: loupe).addClip()
        NSGraphicsContext.current?.imageInterpolation = .none
        let from = NSRect(x: p.x - sample / 2, y: p.y - sample / 2, width: sample, height: sample)
        nsImage.draw(in: loupe, from: from, operation: .copy, fraction: 1)
        NSGraphicsContext.restoreGraphicsState()

        NSColor.white.setStroke()
        let border = NSBezierPath(ovalIn: loupe); border.lineWidth = 2; border.stroke()

        // Crosshair giữa kính lúp.
        NSColor(white: 1, alpha: 0.6).setStroke()
        let cross = NSBezierPath()
        cross.move(to: NSPoint(x: loupe.minX, y: center.y)); cross.line(to: NSPoint(x: loupe.maxX, y: center.y))
        cross.move(to: NSPoint(x: center.x, y: loupe.minY)); cross.line(to: NSPoint(x: center.x, y: loupe.maxY))
        cross.lineWidth = 1; cross.stroke()

        // Tọa độ pixel (top-left).
        let text = "\(Int(p.x)), \(Int(screenHeight - p.y))" as NSString
        let attrs: [NSAttributedString.Key: Any] = [
            .foregroundColor: NSColor.white,
            .font: NSFont.monospacedDigitSystemFont(ofSize: 11, weight: .medium),
        ]
        let size = text.size(withAttributes: attrs)
        let bg = NSRect(x: center.x - size.width / 2 - 4, y: loupe.minY - size.height - 6,
                        width: size.width + 8, height: size.height + 4)
        NSColor(white: 0, alpha: 0.6).setFill(); NSBezierPath(rect: bg).fill()
        text.draw(at: NSPoint(x: bg.minX + 4, y: bg.minY + 2), withAttributes: attrs)
    }
}
