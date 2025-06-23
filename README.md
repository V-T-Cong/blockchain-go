# Blockchain-Go: Mô phỏng Hệ thống Blockchain

Một dự án mô phỏng hệ thống blockchain được xây dựng bằng ngôn ngữ Go, tập trung vào việc mô tả luồng giao dịch, cơ chế đồng thuận đơn giản và tương tác giữa các node trong một mạng lưới phi tập trung.

## 1. Tổng quan dự án

Dự án này xây dựng một blockchain thu nhỏ từ con số không, cho phép người dùng tạo ví, gửi "tiền", và xem các giao dịch được xác nhận và thêm vào chuỗi thông qua một mạng lưới đa node. Mục tiêu chính là để học hỏi và trình bày các khái niệm cốt lõi của công nghệ blockchain một cách trực quan.

### Các tính năng chính

* **Mạng lưới Peer-to-Peer (P2P)**: Hệ thống được thiết lập để chạy với nhiều node (một leader và các follower) giao tiếp với nhau qua gRPC.
* **Cơ chế đồng thuận Leader/Follower**: Một node được chỉ định làm leader có vai trò tạo khối mới, trong khi các follower xác thực và bỏ phiếu cho khối đó.
* **Quản lý ví điện tử**: Cung cấp các công cụ để tạo, lưu trữ và nạp ví điện tử (dựa trên cặp khóa ECDSA) vào các file JSON.
* **Lưu trữ bất biến & Quản lý trạng thái**: Tách biệt rõ ràng giữa việc lưu trữ lịch sử giao dịch (Blockchain) và trạng thái số dư hiện tại (State), sử dụng LevelDB để đảm bảo tính bền vững.
* **Faucet (Vòi tiền)**: Cung cấp một công cụ để "nạp" thêm tiền cho bất kỳ tài khoản nào trong quá trình thử nghiệm mà không cần khởi động lại hệ thống.

---

## 2. Các quyết định kiến trúc & Công nghệ sử dụng

### Quyết định kiến trúc

1. **Mô hình Đồng thuận Leader/Follower**:
    * **Mô tả**: Thay vì các thuật toán phức tạp như Proof-of-Work, hệ thống sử dụng một mô hình tập trung hơn nơi một `leader` duy nhất chịu trách nhiệm tạo khối và đề xuất cho các `follower`. Các follower chỉ cần xác thực và bỏ phiếu.
    * **Lý do lựa chọn**: Mô hình này đơn giản để triển khai, giúp tập trung vào luồng logic chính (tạo giao dịch, xác thực, bỏ phiếu, commit) mà không bị sa đà vào sự phức tạp của các thuật toán đồng thuận phân tán thực thụ. Nó rất phù hợp cho mục đích mô phỏng và học tập.
    * **Khả năng chịu lỗi (Fault Tolerance)**: Để hệ thống tiếp tục hoạt động khi có một số node follower bị lỗi (tối đa `f` node lỗi), ngưỡng bỏ phiếu được thiết lập là `(N/2) + 1` (với `N` là tổng số node ban đầu). Leader sẽ tự động tính phiếu của chính mình, do đó nó chỉ cần nhận được `(N/2)` phiếu từ các follower. Điều này cho phép hệ thống có thể chịu được `f = (N/2) - 1` node follower bị lỗi mà vẫn đạt được đồng thuận.

2. **Tách biệt giữa Sổ cái Blockchain và Sổ phụ Trạng thái**:
    * **Mô tả**: Dữ liệu được lưu trong LevelDB được chia làm hai phần riêng biệt:
        * **Blockchain**: Chuỗi các khối chứa toàn bộ lịch sử giao dịch, không thể thay đổi.
        * **State**: Một bảng ánh xạ `địa chỉ -> số dư` đơn giản, lưu lại số dư hiện tại của tất cả các tài khoản. Bảng này sẽ được cập nhật mỗi khi một khối mới được commit.
    * **Lý do lựa chọn**: Đây là một quyết định kiến trúc quan trọng để tối ưu hiệu năng. Việc kiểm tra số dư của một tài khoản chỉ cần một lượt đọc duy nhất từ State thay vì phải quét lại toàn bộ lịch sử Blockchain, giúp hệ thống phản hồi nhanh hơn rất nhiều.

### Công nghệ sử dụng

* **Go (Golang)**: Được chọn vì hiệu năng cao, khả năng xử lý đồng thời (concurrency) mạnh mẽ thông qua goroutines, và hệ sinh thái thư viện phong phú, rất phù hợp để xây dựng các hệ thống mạng.
* **gRPC & Protocol Buffers**: Cung cấp một phương thức hiệu quả và có cấu trúc rõ ràng để các node giao tiếp với nhau. Việc định nghĩa API thông qua file `.proto` giúp đảm bảo tính nhất quán và dễ dàng mở rộng.
* **LevelDB**: Một thư viện lưu trữ key-value đơn giản, nhẹ và hiệu quả do Google phát triển. Nó rất phù-hợp cho việc lưu trữ dữ liệu blockchain và state trong một dự án mô phỏng mà không cần đến các hệ quản trị cơ sở dữ liệu phức tạp.
* **Docker & Docker Compose**: Công cụ không thể thiếu để mô phỏng một mạng lưới đa node trên một máy tính duy nhất. Nó giúp việc thiết lập, khởi chạy và quản lý môi trường trở nên vô cùng đơn giản và có thể tái tạo.

---

## 3. Hướng dẫn cài đặt và sử dụng

### Yêu cầu

* Go (phiên bản 1.18 trở lên)
* Docker và Docker Compose

### Bước 1: Thiết lập ban đầu (Chỉ làm một lần)

1. **Tạo các ví cần thiết:**
    Hệ thống cần ít nhất 3 ví: `alice`, `bob` để giao dịch và `faucet` để đóng vai trò kho bạc.

    ```bash
    # Chú ý cú pháp: có subcommand 'create-user'
    go run cmd/create_user/create_user.go create-user --name alice
    go run cmd/create_user/create_user.go create-user --name bob
    go run cmd/create_user/create_user.go create-user --name faucet
    ```

    Sao chép lại 3 địa chỉ (`address`) được tạo ra.

2. **Cấu hình Khối Nguyên Thủy (`genesis.json`):**
    Tạo môt file là genesis.json ngoài cùng của thư mục gốc
    Mở file `genesis.json`, dán các địa chỉ trên vào và cấp vốn ban đầu. Ví `faucet` nên có một số dư thật lớn.

    ```json
    {
      "alloc": {
        "<địa_chỉ_faucet>": { "balance": 1000000000.0 },
        "<địa_chỉ_alice>": { "balance": 10000.0 },
        "<địa_chỉ_bob>": { "balance": 5000.0 }
      }
    }
    ```

3. **Tạo Dữ liệu Genesis (`genesis.dat`):**

    ```bash
    go run cmd/build_genesis/main.go
    ```

### Bước 2: Khởi chạy mạng lưới

Sử dụng Docker Compose để build và chạy 4 node (1 leader, 3 follower).

```bash
docker-compose up --build
```

### Bước 3: tương tác với hệ thống

mở một terminal mới để thực hiện các lệnh sau.

1. **kiểm tra số dư ban đầu:**

Xác nhận rằng các tài khoản đã được cấp vốn đúng như trong genesis.json.

  ```bash
  go run cmd/getbalance/main.go --address <địa_chỉ_bạn_muốn_kiểm_tra>
  ```

2. **Gửi một giao dịch:**
  
  Chạy client để thực hiện một giao dịch từ alice đến bob.

  ```bash
  go run cmd/client/sendtx.go
  ```

Theo dõi log trên docker desktop hoặc terminal docker-compose để xem quá trình đồng thuận được diễn ra

3. **bạp thêm tiền bằng faucet**

  sử dụng câu lệnh bên dưới để nạp tiền và địa chỉ người nhận.

  ```bash
  go run cmd/faucet/main.go --to <ĐỊA_CHỈ_NHẬN> --amount <SỐ_TIỀN>
  ```

  sample

  ```bash
  go run cmd/faucet/main.go --to địa_chỉ_của_bob> --amount 500
  ```

## 4. Cấu trúc thu mục

├── cmd/                # Chứa code cho các chương trình thực thi (node, client, tools)
├── pkg/                # Chứa logic cốt lõi của hệ thống, có thể tái sử dụng
│   ├── blockchain/     # Định nghĩa cấu trúc Block, Transaction
│   ├── p2p_v2/         # Logic client/server gRPC và đồng thuận
│   ├── state/          # Logic quản lý số dư (State Database)
│   ├── storage/        # Logic trừu tượng hóa việc tương tác với LevelDB
│   └── wallet/         # Logic tạo ví, ký và xác thực giao dịch
├── proto/              # Chứa các file định nghĩa Protocol Buffers (.proto)
├── wallets/            # Nơi lưu trữ các file ví đã được tạo
├── docker-compose.yml  # File cấu hình để chạy mạng lưới đa node
├── genesis.json        # File cấu hình vốn ban đầu cho blockchain
└── README.md           # Tài liệu dự án
