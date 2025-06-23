# Blockchain-go: Mô phỏng hệ thống Blockchain

Dự án mô phỏng hệ thống blockchain được xây dựng bằng ngôn ngữ Go, tập trung vào việc mô tả luồng giao dịch, cơ chế đồng thuận đơn giản và tương tác giữa các node trong một mạng lưới phi tập trung.

## 1. Tổng quan dự án

Dự án này xây dựng một blockchain thu nhỏ từ con số không, cho phép người dùng tạo ví, gửi "tiền", và xem các giao dịch được xác nhận và thêm vào chuỗi thông qua một mạng lưới đa node. Mục tiêu chính là để học hỏi và trình bày các khái niệm cốt lõi của công nghệ blockchain một cách trực quan.

### Các tính năng chính

* **Mạng lưới Peer-to-Peer (P2P)**: Hệ thống được thiết lập để chạy với nhiều node (một leader và các follower) giao tiếp với nhau qua gRPC.
* **Cơ chế đồng thuận Leader/Follower**: Một node được chỉ định làm leader có vai trò tạo khối mới, trong khi các follower xác thực và bỏ phiếu cho khối đó.

--

## 2. Các quyết đinh kiến trúc & Công nghê sử dụng

### Quyết định kiến trúc

1. **Mô hình Đồng thuận Leader/Follower**:

* **Mô tả**: Thay vì các thuật toán phức tạp như Proof-of-Work, hệ thống sử dụng một mô hình tập trung hơn nơi một `leader` duy nhất chịu trách nhiệm tạo khối và đề xuất cho các `follower`. Các follower chỉ cần xác thực và bỏ phiếu.

* **Lý do lựa chọn**: Mô hình này đơn giản để triển khai, giúp tập trung vào luồng logic chính (tạo giao dịch, xác thực, bỏ phiếu, commit) mà không bị sa đà vào sự phức tạp của các thuật toán đồng thuận phân tán thực thụ. Nó rất phù hợp cho mục đích mô phỏng và học tập.
* **Khả năng chịu lỗi (Fault Tolerance)**: Để hệ thống tiếp tục hoạt động khi có một số node follower bị lỗi (tối đa `f` node lỗi), ngưỡng bỏ phiếu được thiết lập là `(N/2) + 1` (với `N` là tổng số node ban đầu). Leader sẽ tự động tính phiếu của chính mình, do đó nó chỉ cần nhận được `(N/2)` phiếu từ các follower. Điều này cho phép hệ thống có thể chịu được `f = (N/2) - 1` node follower bị lỗi mà vẫn đạt được đồng thuận.

2. **Sử dụng Merkle Patricia Trie để quản lý số dư**
 Để tóm gọn toàn bộ trạng thái của hệ thống(số dư của tất cả các tài khoản) vào một hash duy nhất

* **làm đặc trưng cho trạng thái** `StateRoot` hoạt động như một bằng chứng mật mã không thể giả mạo, đại diện cho toàn bộ dữ liệu tài khoản tại thời điểm một khối được tạo ra.

* **Giúp xác thực hiệu quả** Các node trong mạng có thể nhanh chóng kiểm tra xem chúng có đồng thuận về trạng thái hay không chỉ bằng cách so sánh một `StateRoot`duy nhất, thay vì phải so sánh số dư của từng tài khoản

### Công nghệ sử dụng

* **Go (Golang)**: Được chọn vì hiệu năng cao, khả năng xử lý đồng thời (concurrency) mạnh mẽ thông qua goroutines, và hệ sinh thái thư viện phong phú, rất phù hợp để xây dựng các hệ thống mạng.
* **gRPC & Protocol Buffers**: Cung cấp một phương thức hiệu quả và có cấu trúc rõ ràng để các node giao tiếp với nhau. Việc định nghĩa API thông qua file `.proto` giúp đảm bảo tính nhất quán và dễ dàng mở rộng.
* **LevelDB**: Một thư viện lưu trữ key-value đơn giản, nhẹ và hiệu quả do Google phát triển. Nó rất phù-hợp cho việc lưu trữ dữ liệu blockchain và state trong một dự án mô phỏng mà không cần đến các hệ quản trị cơ sở dữ liệu phức tạp.
* **Docker & Docker Compose**: Công cụ không thể thiếu để mô phỏng một mạng lưới đa node trên một máy tính duy nhất. Nó giúp việc thiết lập, khởi chạy và quản lý môi trường trở nên vô cùng đơn giản và có thể tái tạo.

---

## 3. Hướng dẫn cài đặt và sử dụng

* Go (phiên bản 1.18 trở lên)
* Docker và Docker Compose

### Bước 1: Thiết Lập ban đầu (Chỉ làm một lần)

1. **Cài đặt và & Build**
  Từ thư mục gốc của dự án, chạy lệnh sau để build các Docker image cho các node:

  ```bash
  docker-compose build --no-cache
  ```

2. **Khởi chạy hệ thống**
  Lệnh này sẽ khởi động một mạng lưới gồm 4 node (1 leader, 3 follower).

  ```bash
  docker-compose up
  ```

  Bạn sẽ thấy log của cả 4 node xuất hiện trên màn hình

3. **Tương tác với hệ thống**
  Mở một terminal mới để chạy các lệnh client

  a. **Tạo Ví mới (cần ít nhất 2 ví để có thể chuyển tiền)**
  Tạo file thông tin `.json` chứa thông tin bao gồm private_key, public_key, và address bằng cách chạy lệnh.

  ```bash
    go run cmd/create_user/create_user.go create-user --name <name>
  ```

  lệnh này tạo ra thư mục `wallets` chứa các thông tin về ví

  note: nếu bạn đặt tên khác thì phải sửa lại tên file bên trong 
  `./cmd/client/sendtx.go`

  sửa lại đoạn này

  ```
  aliceWallet, err := wallet.LoadWallet("wallets/<<name>>.json")
  ```

  b. **Gửi dao dịch**
  chạy lệnh bên dưới để gửi dao dịch.

  ```bash
    go run cmd/client/sendtx.go
  ```

  chú ý log từ `docker-compose` để xem các node xử lý dao dịch và tạo khối mới

  c. **Kiểm tra số dư**
  sử dụng câu lệnh bên dưới để chạy file `checkbalance.go` để kiểm tra số dư bên trong tài khoản

  ```bash
    go run cmd/client/checkbalance.go --address <wallet_address>
  ```

 `wallet address` được lấy bên trong thư mục wallets bằng mở file `.json`

 ## 4. Cấu trúc thư mục


  ```
      
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

  ```