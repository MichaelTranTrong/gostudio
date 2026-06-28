import Foundation

enum CaptureMode: String { case screenshot, video }
enum CaptureRegion: String { case full, window, area }
enum CaptureAudio: String { case none, system, mic, both }

// Phân tích URL gostudio://capture?mode=...&region=...&audio=...
struct CaptureRequest {
    let mode: CaptureMode
    let region: CaptureRegion
    let audio: CaptureAudio
    let rawURL: String

    init?(url: URL) {
        guard url.scheme == "gostudio" else { return nil }

        let items = URLComponents(url: url, resolvingAgainstBaseURL: false)?.queryItems ?? []
        func value(_ key: String) -> String? { items.first { $0.name == key }?.value }

        guard let mode = CaptureMode(rawValue: value("mode") ?? "") else { return nil }
        self.mode = mode
        self.region = CaptureRegion(rawValue: value("region") ?? "full") ?? .full
        self.audio = CaptureAudio(rawValue: value("audio") ?? "none") ?? .none
        self.rawURL = url.absoluteString
    }

    /// Hiện chỉ thu được âm thanh hệ thống (ScreenCaptureKit). mic/both → TODO.
    var capturesAudio: Bool { audio == .system || audio == .both }
}
