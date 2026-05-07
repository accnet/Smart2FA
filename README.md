# Smart2FA

**Smart2FA** là một ứng dụng quản lý mã xác thực 2 lớp (TOTP) tối giản, bảo mật và có thể tự host. Ứng dụng được thiết kế để chạy nhanh, nhẹ và hỗ trợ đầy đủ các tính năng hiện đại như PWA để cài đặt lên điện thoại.

## ✨ Tính năng chính

- **Bảo mật tuyệt đối**: Dữ liệu được mã hóa bằng thuật toán **Argon2id + AES-GCM**. Vault chỉ được giải mã khi bạn nhập đúng bộ đôi **Phrase + Passcode**.
- **Quản lý theo Nhóm (Tabs)**: Tổ chức các tài khoản theo tab (Default, Work, Personal...). Hỗ trợ tạo nhóm mới và đổi tên nhóm trực tiếp bằng cách nhấn đúp.
- **Trải nghiệm mượt mà**:
  - Mã TOTP cập nhật trực tiếp qua HTMX.
  - Thanh đếm ngược thời gian thực (smooth progress bar).
  - Cảnh báo khi mã sắp hết hạn (dưới 7 giây).
- **Tối ưu Mobile**: Giao diện responsive, hỗ trợ thao tác chạm, modal dạng bottom-sheet.
- **PWA Ready**: Có thể cài đặt trực tiếp lên màn hình chính (Home Screen) của Android và iOS như một ứng dụng gốc.
- **Quản lý linh hoạt**: Thêm, sửa, xóa tài khoản dễ dàng. Tự động nhận diện link `otpauth://`.
- **Sao lưu & Khôi phục**: Xuất/Nhập dữ liệu mã hóa dưới dạng file JSON để sao lưu.

## 🛠 Công nghệ sử dụng

- **Backend**: Go (Golang)
- **Frontend**: HTML5, Vanilla CSS, JavaScript, [HTMX](https://htmx.org/)
- **Database**: SQLite3
- **Mã hóa**: `golang.org/x/crypto/argon2`

## 🚀 Khởi chạy nhanh

Dự án đi kèm với script `start.sh` để quản lý việc chạy ứng dụng:

1. **Cấp quyền thực thi (lần đầu)**:
   ```bash
   chmod +x start.sh
   ```

2. **Chạy chế độ Development (Auto-reload code)**:
   ```bash
   ./start.sh dev
   ```

3. **Chạy chế độ Production (Build binary)**:
   ```bash
   ./start.sh prod
   ```

Mặc định ứng dụng sẽ chạy tại: **http://localhost:8083**

## 📱 Cài đặt lên điện thoại (PWA)

- **Android**: Mở ứng dụng bằng Chrome, nút **Install** sẽ xuất hiện ở thanh công cụ phía trên.
- **iOS**: Mở ứng dụng bằng Safari, nhấn nút **Share** rồi chọn **"Add to Home Screen"**.

## 📂 Cấu trúc thư mục

- `cmd/`: Chứa các tool phụ trợ (debug, v.v.)
- `data/`: Chứa file database SQLite (`smart2fa.db`)
- `internal/`: Logic cốt lõi (auth, crypto, handlers, totp, vault)
- `static/`: File tĩnh (CSS, JS, Icons, Service Worker, Manifest)
- `templates/`: Giao diện HTML (HTMX partials)

---
*Smart2FA — Minimalist · Self-hosted · Private*
