import AVFoundation
import ScreenCaptureKit

enum RecorderError: Error { case noDisplay }

// Quay video bằng ScreenCaptureKit, ghi MP4 (H.264 + AAC) qua AVAssetWriter.
// Quay toàn màn hình chính, hoặc một cửa sổ cụ thể nếu truyền `window`.
final class ScreenRecorder: NSObject, SCStreamOutput {
    private let outputURL: URL
    private let captureAudio: Bool
    private let window: SCWindow?
    private let hideCursor: Bool
    private let queue = DispatchQueue(label: "gostudio.capture.sample")

    private var stream: SCStream?
    private var writer: AVAssetWriter?
    private var videoInput: AVAssetWriterInput?
    private var audioInput: AVAssetWriterInput?
    private var sessionStarted = false

    init(outputURL: URL, captureAudio: Bool, window: SCWindow? = nil, hideCursor: Bool = false) {
        self.outputURL = outputURL
        self.captureAudio = captureAudio
        self.window = window
        self.hideCursor = hideCursor
    }

    func start() async throws {
        let filter: SCContentFilter
        let width: Int
        let height: Int

        if let window = window {
            // Quay đúng một cửa sổ.
            filter = SCContentFilter(desktopIndependentWindow: window)
            width = max(2, Int(window.frame.width)) & ~1   // H.264 cần kích thước chẵn
            height = max(2, Int(window.frame.height)) & ~1
        } else {
            // Quay toàn màn hình chính.
            let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: false)
            guard let display = content.displays.first else { throw RecorderError.noDisplay }
            filter = SCContentFilter(display: display, excludingWindows: [])
            width = display.width    // TODO: nhân scale factor cho màn hình Retina
            height = display.height
        }

        let config = SCStreamConfiguration()
        config.width = width
        config.height = height
        config.minimumFrameInterval = CMTime(value: 1, timescale: 60)
        config.pixelFormat = kCVPixelFormatType_32BGRA
        config.capturesAudio = captureAudio
        config.showsCursor = !hideCursor   // có tác dụng từ macOS 14+; Ventura vẫn hiện con trỏ
        config.queueDepth = 6

        try setupWriter(width: width, height: height)

        let stream = SCStream(filter: filter, configuration: config, delegate: nil)
        try stream.addStreamOutput(self, type: .screen, sampleHandlerQueue: queue)
        if captureAudio {
            try stream.addStreamOutput(self, type: .audio, sampleHandlerQueue: queue)
        }
        self.stream = stream
        try await stream.startCapture()
    }

    func stop() async -> URL? {
        try? await stream?.stopCapture()
        videoInput?.markAsFinished()
        audioInput?.markAsFinished()
        await writer?.finishWriting()
        return writer?.status == .completed ? outputURL : nil
    }

    // MARK: - Setup

    private func setupWriter(width: Int, height: Int) throws {
        let writer = try AVAssetWriter(outputURL: outputURL, fileType: .mp4)

        let videoSettings: [String: Any] = [
            AVVideoCodecKey: AVVideoCodecType.h264,
            AVVideoWidthKey: width,
            AVVideoHeightKey: height,
        ]
        let vInput = AVAssetWriterInput(mediaType: .video, outputSettings: videoSettings)
        vInput.expectsMediaDataInRealTime = true
        if writer.canAdd(vInput) { writer.add(vInput) }
        self.videoInput = vInput

        if captureAudio {
            let audioSettings: [String: Any] = [
                AVFormatIDKey: kAudioFormatMPEG4AAC,
                AVNumberOfChannelsKey: 2,
                AVSampleRateKey: 48000,
            ]
            let aInput = AVAssetWriterInput(mediaType: .audio, outputSettings: audioSettings)
            aInput.expectsMediaDataInRealTime = true
            if writer.canAdd(aInput) { writer.add(aInput) }
            self.audioInput = aInput
        }

        self.writer = writer
    }

    // MARK: - SCStreamOutput

    func stream(_ stream: SCStream, didOutputSampleBuffer sampleBuffer: CMSampleBuffer, of type: SCStreamOutputType) {
        guard CMSampleBufferDataIsReady(sampleBuffer), let writer = writer else { return }

        if type == .screen {
            // Bắt đầu session ở frame đầu tiên (kể cả khi không đọc được status).
            if writer.status == .unknown {
                writer.startWriting()
                writer.startSession(atSourceTime: CMSampleBufferGetPresentationTimeStamp(sampleBuffer))
                sessionStarted = true
            }
            // Chỉ bỏ qua nếu ĐỌC ĐƯỢC status và nó KHÔNG complete; đọc không ra thì cứ ghi.
            if let attachments = CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, createIfNecessary: false) as? [[SCStreamFrameInfo: Any]],
               let raw = attachments.first?[.status] as? Int,
               let status = SCFrameStatus(rawValue: raw),
               status != .complete {
                return
            }
            if writer.status == .writing, let input = videoInput, input.isReadyForMoreMediaData {
                input.append(sampleBuffer)
            }
        } else if type == .audio {
            guard sessionStarted, writer.status == .writing,
                  let input = audioInput, input.isReadyForMoreMediaData else { return }
            input.append(sampleBuffer)
        }
    }
}
