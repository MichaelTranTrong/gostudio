import Foundation

// Upload file capture lên Go Studio backend (multipart/form-data).
enum Uploader {
    static func upload(fileURL: URL, type: String, completion: @escaping (Bool, String) -> Void) {
        guard let endpoint = URL(string: Config.backendURL + "/api/capture/upload") else {
            completion(false, "URL backend không hợp lệ")
            return
        }

        var request = URLRequest(url: endpoint)
        request.httpMethod = "POST"
        let boundary = "Boundary-\(UUID().uuidString)"
        request.setValue("multipart/form-data; boundary=\(boundary)", forHTTPHeaderField: "Content-Type")
        if !Config.captureToken.isEmpty {
            request.setValue(Config.captureToken, forHTTPHeaderField: "X-Capture-Token")
        }

        var body = Data()

        // field: type
        body.appendString("--\(boundary)\r\n")
        body.appendString("Content-Disposition: form-data; name=\"type\"\r\n\r\n")
        body.appendString("\(type)\r\n")

        // field: file
        let fileData = (try? Data(contentsOf: fileURL)) ?? Data()
        let filename = fileURL.lastPathComponent
        body.appendString("--\(boundary)\r\n")
        body.appendString("Content-Disposition: form-data; name=\"file\"; filename=\"\(filename)\"\r\n")
        body.appendString("Content-Type: application/octet-stream\r\n\r\n")
        body.append(fileData)
        body.appendString("\r\n")
        body.appendString("--\(boundary)--\r\n")

        request.httpBody = body

        URLSession.shared.dataTask(with: request) { _, response, error in
            if let error = error {
                completion(false, error.localizedDescription)
                return
            }
            let code = (response as? HTTPURLResponse)?.statusCode ?? 0
            completion((200..<300).contains(code), "HTTP \(code)")
        }.resume()
    }
}

private extension Data {
    mutating func appendString(_ string: String) {
        if let data = string.data(using: .utf8) { append(data) }
    }
}
