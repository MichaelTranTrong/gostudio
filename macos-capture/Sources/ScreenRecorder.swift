import AVFoundation
import ScreenCaptureKit

enum RecorderError: Error { case noDisplay }

// Quay video màn hình bằng ScreenCaptureKit, ghi MP4 (H.264 + AAC) qua AVAssetWriter.
// Hiện quay màn hình chính, full-screen. region window/area cho video là TODO.
final class ScreenRecorder: NSObject, SCStreamOutput {
    private let outputURL: URL
    private let captureAudio: Bool
    private let queue = DispatchQueue(label: "gostudio.capture.sample")

    private var stream: SCStream?
    private var writer: AVAssetWriter?
    private var videoInput: AVAssetWriterInput?
    private var audioInput: AVAssetWriterInput?
    private var sessionStarted = false

    init(outputURL: URL, captureAudio: Bool) {
        self.outputURL = outputURL
        self.captureAudio = captureAudio
    }

    func start() async throws {
        let content = try await SCShareableContent.excludingDesktopWindows(false, onScreenWindowsOnly: false)
        guard let display = content.displays.first else { throw RecorderError.noDisplay }

        let filter = SCContentFilter(display: display, excludingWindows: [])

        let config = SCStreamConfiguration()
        config.width = display.width   // TODO: nhân scale factor cho màn hình Retina
        config.height = display.height
        config.minimumFrameInterval = CMTime(value: 1, timescale: 60)
        config.pixelFormat = kCVPixelFormatType_32BGRA
        config.capturesAudio = captureAudio
        config.queueDepth = 6

        try setupWriter(width: display.width, height: display.height)

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
