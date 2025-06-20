## Hướng dẫn cài đặt và sử dụng

Phần này sẽ hướng dẫn bạn từng bước để thiết lập môi trường, khởi chạy mạng lưới blockchain và thực hiện các giao dịch cơ bản.

### Yêu cầu

Để chạy dự án này, bạn cần cài đặt:
- Go (phiên bản 1.18 trở lên)
- Docker và Docker Compose

### Các bước thực hiện

#### Bước 1: Cài đặt ban đầu (chỉ làm một lần)

Đây là các bước để chuẩn bị dữ liệu và cấu hình cần thiết trước khi khởi chạy mạng lưới.

1.  **Tạo ví cho người dùng:**

    Mở terminal và chạy lệnh sau để tạo ví cho người dùng `alice` và `bob`. Thông tin ví (khóa riêng, khóa công khai, địa chỉ) sẽ được lưu trong thư mục `wallets/`.
    ```bash
    go run cmd/create_user/create_user.go --name alice
    go run cmd/create_user/create_user.go --name bob
    ```
    Hãy sao chép lại 2 địa chỉ (`address`) được tạo ra để sử dụng ở bước tiếp theo. [cite: v-t-cong/blockchain-go/blockchain-go-5844c98f503beec27df26cd035575a78ca363bac/cmd/create_user/create_user.go]

2.  **Cấu hình khối nguyên thủy (Genesis Block):**

    Khối nguyên thủy là khối đầu tiên của chuỗi, nơi chúng ta cấp một lượng tiền ban đầu cho các tài khoản. Mở file `genesis.json` và dán các địa chỉ bạn vừa sao chép vào, đồng thời cấp cho họ một số dư.

    *Ví dụ nội dung file `genesis.json`:*
    ```json
    {
      "alloc": {
        "<địa_chỉ_của_alice>": { "balance": 1000000.0 },
        "<địa_chỉ_của_bob>": { "balance": 500000.0 }
      }
    }
    ```

3.  **Tạo dữ liệu Genesis:**

    Chạy lệnh sau để đọc file `genesis.json` và tạo ra file dữ liệu `genesis.dat`. File này sẽ được các node sử dụng khi khởi động lần đầu.
    ```bash
    go run cmd/build_genesis/main.go
    ```

#### Bước 2: Khởi chạy mạng lưới

Sử dụng Docker Compose để build và chạy 4 node (1 leader, 3 follower) cùng một lúc. Lệnh này sẽ thiết lập một mạng lưới ảo để các node có thể giao tiếp với nhau. [cite: v-t-cong/blockchain-go/blockchain-go-5844c98f503beec27df26cd035575a78ca363bac/docker-compose.yml]
```bash
docker-compose up --build
```
#### Bước 3: thực hiện gửi dịch bên trong blockchain
chạy lệnh sau `go run cmd/client/sendtx.go` để thực hiện giao dịch chuyển tiền giữa các tài khoản

để có thể kiểm tra giao dịch đã gửi thành công hay chưa bạn có có thể kiểm tra log ở phía docker hoặc kiểm tra bằng địa chỉ wallet bằng câu lệnh sau.
```bash
go run cmd/getbalance/maingo --address <địa_chỉ_bạn_muốn_kiểm_tra>
```
bạn có thể lấy địa chỉ wallet bằng cách mở thư mục wallets.
