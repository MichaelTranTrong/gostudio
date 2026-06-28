import Cocoa

// Overlay đếm ngược 3-2-1 trước khi quay video — để kịp đưa con trỏ ra chỗ khuất.
// Không chặn chuột (ignoresMouseEvents) nên vẫn di chuột được trong lúc đếm.
// Overlay tắt TRƯỚC khi quay bắt đầu nên không lọt vào video.
enum Countdown {
    private static var window: NSWindow?
    private static var timer: Timer?

    static func show(from seconds: Int, completion: @escaping () -> Void) {
        guard let screen = NSScreen.main else { completion(); return }

        let win = NSWindow(contentRect: screen.frame, styleMask: .borderless,
                           backing: .buffered, defer: false)
        win.level = .screenSaver
        win.isOpaque = false
        win.backgroundColor = .clear
        win.ignoresMouseEvents = true

        let container = NSView(frame: NSRect(origin: .zero, size: screen.frame.size))

        let size: CGFloat = 200
        let circle = NSView(frame: NSRect(x: (screen.frame.width - size) / 2,
                                          y: (screen.frame.height - size) / 2,
                                          width: size, height: size))
        circle.wantsLayer = true
        circle.layer?.backgroundColor = NSColor(white: 0, alpha: 0.55).cgColor
        circle.layer?.cornerRadius = size / 2

        let label = NSTextField(labelWithString: "\(seconds)")
        label.font = .systemFont(ofSize: 110, weight: .bold)
        label.textColor = .white
        label.alignment = .center
        label.frame = NSRect(x: 0, y: (size - 130) / 2, width: size, height: 130)
        circle.addSubview(label)
        container.addSubview(circle)

        let hint = NSTextField(labelWithString: "Đưa con trỏ ra chỗ khuất trước khi quay")
        hint.font = .systemFont(ofSize: 15, weight: .medium)
        hint.textColor = .white
        hint.alignment = .center
        hint.frame = NSRect(x: (screen.frame.width - 420) / 2, y: circle.frame.minY - 40,
                            width: 420, height: 22)
        container.addSubview(hint)

        win.contentView = container
        win.orderFrontRegardless()
        window = win

        var remaining = seconds
        timer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { t in
            remaining -= 1
            if remaining >= 1 {
                label.stringValue = "\(remaining)"
            } else {
                t.invalidate()
                Countdown.timer = nil
                win.orderOut(nil)
                Countdown.window = nil
                completion()
            }
        }
    }
}
