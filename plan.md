# Smart2FA — Implementation Plan & Context

## Project Overview
Web app quản lý mã 2FA tối giản bằng Golang, ưu tiên tính tiện dụng (UX), tốc độ và khả năng tự host (self-hosted).

**Mục tiêu:**
- Self-host, single binary.
- Không cần tài khoản truyền thống; unlock bằng bộ đôi `phrase` + `passcode`.
- Dữ liệu mã hóa hoàn toàn, đảm bảo tính riêng tư.
- Hỗ trợ hàng triệu vault trên một instance duy nhất.

---

## Technology Stack

### Backend
- **Language**: Golang (Standard Library `net/http`)
- **Web**: `http.NewServeMux` (Go 1.22+ routing)
- **Database**: SQLite3 (với các tối ưu hóa WAL mode)
- **TOTP**: `github.com/pquerna/otp/totp`

### Frontend
- **HTML**: Server-side templates (`html/template`)
- **Updates**: [HTMX](https://htmx.org/) (Partial rendering)
- **Styling**: Modern Vanilla CSS (Custom properties, Grid/Flexbox)
- **Client Logic**: Vanilla JS for timers and PWA handling.

### Security
- **KDF**: Argon2id
- **Encryption**: AES-256-GCM

---

## Core Security Model
Vault được xác định duy nhất bởi: `SHA256(phrase + ":" + passcode)`.
- Không lưu mật khẩu/tên đăng nhập.
- Key mã hóa được derive qua Argon2id từ phrase+passcode.
- Nếu nhập phrase/passcode mới → Một vault rỗng mới được tạo tự động.

---

## Database Schema
- **vaults**: Lưu thông tin quản lý (ID, vault_hash, salt, counts).
- **vault_blobs**: Lưu dữ liệu đã mã hóa (JSON blob).

**Architecture Decision**: Lưu toàn bộ tài khoản trong 1 vault thành 1 encrypted JSON blob duy nhất để tối ưu tốc độ đọc/ghi và scale SQLite.

---

## Vault Data Structure
Dữ liệu lưu trữ bên trong blob:
```json
[
  {
    "name": "Google",
    "secret": "N6ZLGKOD...",
    "group": "Personal"
  }
]
```

---

## Key Features & UX

### 1. Live TOTP Display
- **HTMX Polling**: Reload partial `/partial/codes` mỗi 5 giây.
- **Smooth Timer**: JavaScript client-side chạy mỗi giây để cập nhật thanh tiến trình mượt mà (không đợi server).
- **Format**: Mã 6 số hiển thị dạng `XXX XXX` dễ đọc.
- **Expiring State**: Thanh tiến trình và mã chuyển sang màu đỏ + hiệu ứng pulse khi còn dưới 7 giây.

### 2. Tab Groups Management
- **Horizontal Bar**: Thanh tab cuộn ngang trên mobile.
- **Inline Creation**: Nhấn "+" tạo tab mới, nhập tên ngay trên tab.
- **Double-click Rename**: Đổi tên group nhanh chóng bằng cách nhấn đúp.
- **Badge Count**: Hiển thị số lượng account trong mỗi group.

### 3. Vault Entry Management
- **Add**: Hỗ trợ dán link `otpauth://` hoặc nhập tay. Tự động strip khoảng trắng trong secret.
- **Edit**: Cập nhật Name, Group hoặc Secret (tùy chọn).
- **Delete**: Xóa entry kèm confirm.

### 4. PWA (Progressive Web App)
- **Installable**: Hỗ trợ cài lên Android (Install button) và iOS (Add to Home Screen banner).
- **Standalone**: Chạy không có thanh trình duyệt, giao diện như app native.
- **Offline Cache**: Service Worker cache các file tĩnh (CSS/JS/Icons) để load tức thì.

### 5. Mobile Optimization
- **Responsive Layout**: Chuyển đổi card hiển thị từ 1 hàng sang 2 hàng trên màn hình nhỏ.
- **Bottom Sheets**: Modal trượt lên từ phía dưới trên mobile.
- **iOS Zoom Fix**: `font-size: 16px` cho inputs ngăn trình duyệt tự zoom.

---

## Infrastructure & Maintenance
- **Startup**: Script `start.sh` hỗ trợ 2 chế độ `dev` (go run) và `prod` (go build).
- **Port**: Chạy mặc định trên cổng **8083**.
- **Backup**: Hệ thống Export/Import file `.votp` mã hóa.

---

## Future Goals
- Hỗ trợ thêm các loại mã 2FA khác (8 chữ số, Steam, v.v.).
- Thêm tính năng Search nhanh trong vault.
- Tùy chọn Dark/Light mode tự động theo hệ thống.
