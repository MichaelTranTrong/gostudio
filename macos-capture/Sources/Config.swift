import Foundation

// Cấu hình tĩnh — sửa trực tiếp tại đây (app launch-by-scheme khó truyền env).
enum Config {
    /// Địa chỉ Go Studio backend.
    static let backendURL = "http://localhost:2005"

    /// Khớp với CAPTURE_TOKEN ở backend nếu bật. Để rỗng nếu backend không dùng token.
    static let captureToken = ""

    /// File tạm trong thư mục temp (xóa sau khi upload).
    static func tempFile(prefix: String, ext: String) -> URL {
        let name = "\(prefix)_\(Int(Date().timeIntervalSince1970)).\(ext)"
        return FileManager.default.temporaryDirectory.appendingPathComponent(name)
    }
}
