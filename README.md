# Blockchain-Go

Hệ thống blockchain mô phỏng việc chuyển tiền giữa hai người.

## Hướng dẫn chạy hệ thống

### Bước 1: Build các node
Trước tiên, bạn cần build lại các node bằng Docker:

```bash
docker-compose build --no-cache

### Bước 2: Khởi chạy hệ thống

```bash
docker-compose up

### Bước 3: gửi dao dịch
```bash
go run cmd/client/sendtx.go


### Bước 4: Kiểm tra giao dịch
 mở logs của docker để xem kiểm tra giao dịch