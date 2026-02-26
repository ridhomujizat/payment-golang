# 🏗️ Project Prompt: Payment Gateway Midtrans

## Konteks Project

Membangun sistem payment gateway terintegrasi dengan **Midtrans** untuk WhatsApp Bot yang menjual produk. User dari WhatsApp Bot akan di-redirect ke halaman website payment untuk menyelesaikan pembayaran.

**Library:** [midtrans-go](https://github.com/Midtrans/midtrans-go) (Official Go API Client)

---

## 📐 Arsitektur Sistem

```
┌──────────────┐     ┌──────────────────┐     ┌─────────────────┐     ┌──────────────┐
│  WhatsApp    │────▶│   Backend API    │────▶│   Midtrans API  │────▶│  Payment     │
│  Bot         │     │   (Golang)       │◀────│   (Snap)        │     │  Provider    │
└──────────────┘     └──────────────────┘     └─────────────────┘     └──────────────┘
                            │  ▲                       │
                            │  │ callback              │
                            ▼  │                       │
                     ┌──────────────────┐              │
                     │   Database       │              │
                     │   (PostgreSQL/   │              │
                     │    MySQL)        │              │
                     └──────────────────┘              │
                                                       │
                     ┌──────────────────┐              │
                     │   Frontend       │◀─────────────┘
                     │   (Payment Page) │
                     └──────────────────┘
```

---

## 🔄 Payment Flow

```
User (WhatsApp)
    │
    ▼
[1] Bot kirim data produk ke Backend API ──▶ POST /api/v1/payments/create
    │
    ▼
[2] Backend generate payment via Midtrans Snap API
    │
    ▼
[3] Backend simpan data transaksi ke Database (status: pending)
    │
    ▼
[4] Backend return URL payment ke Bot ──▶ Bot kirim URL ke User
    │
    ▼
[5] User buka URL ──▶ Frontend: Halaman Payment (embed snap.js)
    │
    ▼
[6] User pilih metode bayar & submit payment
    │
    ▼
[7] Midtrans kirim callback/notification ke Backend ──▶ POST /api/v1/payments/callback
    │
    ▼
[8] Backend update status transaksi di Database
    │
    ▼
[9] User redirect ke halaman status payment
```

---

## 📦 Tech Stack

| Layer     | Technology                             |
|-----------|----------------------------------------|
| Backend   | Go (Golang) + Gin/Echo/Fiber           |
| Library   | `github.com/midtrans/midtrans-go`      |
| Database  | PostgreSQL / MySQL                     |
| Frontend  | HTML + Tailwind CSS + snap.js          |
| Bot       | WhatsApp Bot (existing)                |

---

## ⚙️ Environment Variables

```env
# Midtrans
MIDTRANS_SERVER_KEY=SB-Mid-server-xxxxxxxxxxxxx
MIDTRANS_CLIENT_KEY=SB-Mid-client-xxxxxxxxxxxxx
MIDTRANS_ENVIRONMENT=sandbox  # sandbox | production

# App
APP_BASE_URL=https://yourdomain.com
APP_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=payment_db
DB_USER=postgres
DB_PASSWORD=secret
```

---

## 🗃️ Database Schema

### Tabel: `transactions`

```sql
CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id        VARCHAR(100) UNIQUE NOT NULL,
    
    -- Data dari WhatsApp Bot (dynamic)
    customer_name   VARCHAR(255),
    customer_phone  VARCHAR(50),
    customer_email  VARCHAR(255),
    
    -- Payment info
    gross_amount    BIGINT NOT NULL,
    payment_type    VARCHAR(50),
    
    -- Item detail (simpan sebagai JSON karena dynamic)
    items           JSONB NOT NULL,
    
    -- Metadata tambahan dari bot (dynamic, bisa apa saja)
    metadata        JSONB,
    
    -- Midtrans response
    snap_token      VARCHAR(255),
    snap_url        TEXT,
    transaction_id  VARCHAR(255),
    
    -- Status
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    -- pending, settlement, capture, deny, cancel, expire, refund
    
    fraud_status    VARCHAR(50),
    status_code     VARCHAR(10),
    signature_key   TEXT,
    
    -- Timestamps
    created_at      TIMESTAMP DEFAULT NOW(),
    updated_at      TIMESTAMP DEFAULT NOW(),
    paid_at         TIMESTAMP
);

CREATE INDEX idx_transactions_order_id ON transactions(order_id);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_customer_phone ON transactions(customer_phone);
```

---

## 🔧 Backend API

### Install Dependencies

```bash
go get -u github.com/midtrans/midtrans-go
go get -u github.com/midtrans/midtrans-go/snap
go get -u github.com/midtrans/midtrans-go/coreapi
```

### Inisialisasi Midtrans Client

```go
package payment

import (
    "github.com/midtrans/midtrans-go"
    "github.com/midtrans/midtrans-go/snap"
    "github.com/midtrans/midtrans-go/coreapi"
)

var snapClient snap.Client
var coreAPIClient coreapi.Client

func InitMidtrans(serverKey string, env midtrans.EnvironmentType) {
    snapClient.New(serverKey, env)
    coreAPIClient.New(serverKey, env)
}
```

---

### API 1: Create Payment

**Endpoint:** `POST /api/v1/payments/create`

**Deskripsi:** Menerima data dari WhatsApp Bot, generate Snap transaction, simpan ke DB, return URL payment.

**Request Body (dynamic dari bot):**

```json
{
    "order_id": "WA-ORDER-20240101-001",
    "customer": {
        "name": "John Doe",
        "phone": "081234567890",
        "email": "john@example.com"
    },
    "items": [
        {
            "id": "PROD-001",
            "name": "Paket Premium",
            "price": 150000,
            "qty": 1
        },
        {
            "id": "PROD-002",
            "name": "Add-on Feature",
            "price": 50000,
            "qty": 1
        }
    ],
    "metadata": {
        "wa_chat_id": "6281234567890@c.us",
        "bot_session": "sess_abc123",
        "promo_code": "DISKON10"
    }
}
```

**Logic:**

```go
func CreatePayment(req CreatePaymentRequest) (*CreatePaymentResponse, error) {
    // 1. Hitung gross amount dari items
    var grossAmount int64
    var midtransItems []midtrans.ItemDetails
    for _, item := range req.Items {
        grossAmount += item.Price * int64(item.Qty)
        midtransItems = append(midtransItems, midtrans.ItemDetails{
            ID:    item.ID,
            Name:  item.Name,
            Price: item.Price,
            Qty:   int32(item.Qty),
        })
    }

    // 2. Buat Snap Request
    snapReq := &snap.Request{
        TransactionDetails: midtrans.TransactionDetails{
            OrderID:  req.OrderID,
            GrossAmt: grossAmount,
        },
        CustomerDetail: &midtrans.CustomerDetails{
            FName: req.Customer.Name,
            Email: req.Customer.Email,
            Phone: req.Customer.Phone,
        },
        Items: &midtransItems,
    }

    // 3. Request ke Midtrans Snap API
    snapResp, err := snapClient.CreateTransaction(snapReq)
    if err != nil {
        return nil, err
    }

    // 4. Simpan ke database (semua data dynamic)
    transaction := Transaction{
        OrderID:       req.OrderID,
        CustomerName:  req.Customer.Name,
        CustomerPhone: req.Customer.Phone,
        CustomerEmail: req.Customer.Email,
        GrossAmount:   grossAmount,
        Items:         req.Items,     // simpan sebagai JSONB
        Metadata:      req.Metadata,  // simpan sebagai JSONB
        SnapToken:     snapResp.Token,
        SnapURL:       snapResp.RedirectURL,
        Status:        "pending",
    }
    db.Create(&transaction)

    // 5. Return response dengan URL
    return &CreatePaymentResponse{
        OrderID:    req.OrderID,
        PaymentURL: fmt.Sprintf("%s/pay/%s", appBaseURL, snapResp.Token),
        SnapToken:  snapResp.Token,
        SnapURL:    snapResp.RedirectURL,
        Amount:     grossAmount,
    }, nil
}
```

**Response:**

```json
{
    "success": true,
    "data": {
        "order_id": "WA-ORDER-20240101-001",
        "payment_url": "https://yourdomain.com/pay/SNAP-TOKEN-xxxx",
        "snap_token": "SNAP-TOKEN-xxxx",
        "snap_url": "https://app.sandbox.midtrans.com/snap/v3/redirection/xxx",
        "amount": 200000
    }
}
```

---

### API 2: Check Payment Status

**Endpoint:** `GET /api/v1/payments/status/:order_id`

**Deskripsi:** Check status payment dari Midtrans dan database.

**Logic:**

```go
func CheckPaymentStatus(orderID string) (*PaymentStatusResponse, error) {
    // 1. Check dari Midtrans langsung (real-time)
    transactionStatusResp, err := coreAPIClient.CheckTransaction(orderID)
    if err != nil {
        // fallback ke database jika midtrans error
        var trx Transaction
        db.Where("order_id = ?", orderID).First(&trx)
        return &PaymentStatusResponse{
            OrderID: trx.OrderID,
            Status:  trx.Status,
            Amount:  trx.GrossAmount,
        }, nil
    }

    // 2. Update database jika status berubah
    updateTransactionStatus(orderID, transactionStatusResp)

    // 3. Return response
    return &PaymentStatusResponse{
        OrderID:       orderID,
        Status:        transactionStatusResp.TransactionStatus,
        PaymentType:   transactionStatusResp.PaymentType,
        Amount:        transactionStatusResp.GrossAmount,
        TransactionID: transactionStatusResp.TransactionID,
    }, nil
}
```

**Response:**

```json
{
    "success": true,
    "data": {
        "order_id": "WA-ORDER-20240101-001",
        "status": "settlement",
        "payment_type": "bank_transfer",
        "amount": "200000",
        "transaction_id": "xxx-xxx-xxx"
    }
}
```

---

### API 3: Handle Payment (Frontend Submit)

**Endpoint:** `POST /api/v1/payments/process`

**Deskripsi:** Frontend mengirim hasil dari snap.js callback setelah user melakukan pembayaran. Digunakan untuk update UI secara langsung.

**Request Body (dari snap.js callback):**

```json
{
    "order_id": "WA-ORDER-20240101-001",
    "transaction_id": "xxx-xxx-xxx",
    "transaction_status": "settlement",
    "payment_type": "bank_transfer",
    "gross_amount": "200000.00",
    "status_code": "200",
    "signature_key": "xxxxxx"
}
```

**Logic:**

```go
func HandlePayment(req PaymentResultRequest) error {
    // ⚠️ PENTING: Data dari frontend TIDAK bisa dipercaya 100%
    // Selalu verifikasi ke Midtrans API
    
    transactionStatusResp, err := coreAPIClient.CheckTransaction(req.OrderID)
    if err != nil {
        return err
    }

    // Update berdasarkan data yang terverifikasi dari Midtrans
    updateTransactionStatus(req.OrderID, transactionStatusResp)
    return nil
}
```

> ⚠️ **Catatan Keamanan:** Data dari frontend (snap.js callback) harus selalu diverifikasi ulang ke Midtrans API karena bisa dimanipulasi oleh user.

---

### API 4: Callback / Notification dari Midtrans

**Endpoint:** `POST /api/v1/payments/callback`

**Deskripsi:** Endpoint yang menerima HTTP POST notification dari Midtrans setiap kali status transaksi berubah. URL ini harus didaftarkan di **Midtrans Dashboard > Settings > Configuration > Payment Notification URL**.

**Midtrans Notification Payload (contoh):**

```json
{
    "transaction_time": "2024-01-09 18:27:19",
    "transaction_status": "settlement",
    "transaction_id": "57d5293c-e65f-4a29-95e4-5959c3fa335b",
    "status_message": "midtrans payment notification",
    "status_code": "200",
    "signature_key": "16d6f84b2fb0468e...",
    "payment_type": "bank_transfer",
    "order_id": "WA-ORDER-20240101-001",
    "gross_amount": "200000.00",
    "fraud_status": "accept",
    "currency": "IDR"
}
```

**Logic:**

```go
func MidtransCallback(w http.ResponseWriter, r *http.Request) {
    // 1. Parse notification payload
    var notificationPayload map[string]interface{}
    err := json.NewDecoder(r.Body).Decode(&notificationPayload)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // 2. Ambil order_id
    orderID, exists := notificationPayload["order_id"].(string)
    if !exists {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // 3. Verifikasi ke Midtrans API (WAJIB!)
    transactionStatusResp, err := coreAPIClient.CheckTransaction(orderID)
    if err != nil {
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    // 4. Handle berdasarkan transaction status
    switch transactionStatusResp.TransactionStatus {
    case "capture":
        if transactionStatusResp.FraudStatus == "accept" {
            // Pembayaran sukses (credit card)
            updateStatus(orderID, "capture", transactionStatusResp)
            notifyWhatsAppBot(orderID, "success")
        }
    case "settlement":
        // Pembayaran sukses (bank transfer, e-wallet, dll)
        updateStatus(orderID, "settlement", transactionStatusResp)
        notifyWhatsAppBot(orderID, "success")
    case "pending":
        // Menunggu pembayaran
        updateStatus(orderID, "pending", transactionStatusResp)
    case "deny":
        // Pembayaran ditolak
        updateStatus(orderID, "deny", transactionStatusResp)
        notifyWhatsAppBot(orderID, "failed")
    case "expire":
        // Pembayaran expired
        updateStatus(orderID, "expire", transactionStatusResp)
        notifyWhatsAppBot(orderID, "expired")
    case "cancel":
        // Pembayaran dibatalkan
        updateStatus(orderID, "cancel", transactionStatusResp)
        notifyWhatsAppBot(orderID, "cancelled")
    case "refund":
        // Refund
        updateStatus(orderID, "refund", transactionStatusResp)
        notifyWhatsAppBot(orderID, "refunded")
    }

    // 5. Response 200 OK ke Midtrans (WAJIB!)
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

**Verifikasi Signature Key:**

```go
func verifySignatureKey(orderID, statusCode, grossAmount, serverKey, signatureKey string) bool {
    // SHA512(order_id + status_code + gross_amount + ServerKey)
    input := orderID + statusCode + grossAmount + serverKey
    hash := sha512.Sum512([]byte(input))
    calculatedSignature := hex.EncodeToString(hash[:])
    return calculatedSignature == signatureKey
}
```

> ⚠️ **Best Practices Callback:**
> - Selalu verifikasi notifikasi dengan memanggil `CheckTransaction` ke Midtrans API
> - Implementasi idempotent (gunakan `order_id` sebagai key, hindari double entry)
> - Gunakan HTTPS endpoint
> - Response time harus di bawah 5 detik
> - Midtrans HTTP timeout adalah 30 detik
> - Handle kemungkinan notifikasi out-of-order

---

## 🗂️ Route Summary

```go
func SetupRoutes(r *gin.Engine) {
    api := r.Group("/api/v1/payments")
    {
        api.POST("/create",   CreatePaymentHandler)    // Bot -> Backend
        api.GET("/status/:id", CheckStatusHandler)     // Frontend/Bot -> Backend
        api.POST("/process",   HandlePaymentHandler)   // Frontend -> Backend
        api.POST("/callback",  MidtransCallbackHandler) // Midtrans -> Backend
    }

    // Frontend pages
    r.GET("/pay/:token",    PaymentPageHandler)   // Halaman Payment
    r.GET("/status/:id",    StatusPageHandler)    // Halaman Status
}
```

---

## 🖥️ Frontend

### Halaman 1: Payment Page (`/pay/:token`)

Halaman ini menampilkan detail order dan embed **Midtrans Snap.js** untuk proses pembayaran.

```html
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Pembayaran</title>
    <script src="https://cdn.tailwindcss.com"></script>
    
    <!-- Snap.js - Sandbox -->
    <script 
        src="https://app.sandbox.midtrans.com/snap/snap.js" 
        data-client-key="SB-Mid-client-xxxxxxxxxxxxx">
    </script>
    <!-- Production: https://app.midtrans.com/snap/snap.js -->
</head>
<body class="bg-gray-50 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-lg max-w-md w-full p-6">
        <!-- Header -->
        <div class="text-center mb-6">
            <h1 class="text-2xl font-bold text-gray-800">Pembayaran</h1>
            <p class="text-gray-500 text-sm mt-1">Order: <span id="order-id"></span></p>
        </div>

        <!-- Order Summary -->
        <div class="border rounded-xl p-4 mb-6">
            <h2 class="font-semibold text-gray-700 mb-3">Ringkasan Order</h2>
            <div id="item-list" class="space-y-2"></div>
            <hr class="my-3">
            <div class="flex justify-between font-bold text-lg">
                <span>Total</span>
                <span id="total-amount" class="text-blue-600"></span>
            </div>
        </div>

        <!-- Customer Info -->
        <div class="border rounded-xl p-4 mb-6">
            <h2 class="font-semibold text-gray-700 mb-2">Info Pembeli</h2>
            <p id="customer-name" class="text-gray-600"></p>
            <p id="customer-phone" class="text-gray-500 text-sm"></p>
        </div>

        <!-- Pay Button -->
        <button 
            id="pay-button" 
            class="w-full bg-blue-600 hover:bg-blue-700 text-white font-semibold 
                   py-3 px-6 rounded-xl transition-colors duration-200">
            Bayar Sekarang
        </button>

        <!-- Result -->
        <div id="result" class="mt-4 hidden">
            <div id="result-success" class="hidden bg-green-50 border border-green-200 
                        rounded-xl p-4 text-center">
                <p class="text-green-700 font-semibold">Pembayaran Berhasil! ✅</p>
                <p class="text-green-600 text-sm mt-1">Terima kasih atas pembayaran Anda.</p>
            </div>
            <div id="result-pending" class="hidden bg-yellow-50 border border-yellow-200 
                        rounded-xl p-4 text-center">
                <p class="text-yellow-700 font-semibold">Menunggu Pembayaran ⏳</p>
                <p class="text-yellow-600 text-sm mt-1">Silakan selesaikan pembayaran Anda.</p>
            </div>
            <div id="result-error" class="hidden bg-red-50 border border-red-200 
                        rounded-xl p-4 text-center">
                <p class="text-red-700 font-semibold">Pembayaran Gagal ❌</p>
                <p class="text-red-600 text-sm mt-1">Silakan coba lagi.</p>
            </div>
        </div>
    </div>

    <script>
        // Data dari backend (di-inject saat serve halaman)
        const SNAP_TOKEN = "{{.SnapToken}}";
        const ORDER_DATA = JSON.parse('{{.OrderDataJSON}}');

        // Render order info
        document.getElementById('order-id').textContent = ORDER_DATA.order_id;
        document.getElementById('customer-name').textContent = ORDER_DATA.customer_name;
        document.getElementById('customer-phone').textContent = ORDER_DATA.customer_phone;
        document.getElementById('total-amount').textContent = 
            'Rp ' + ORDER_DATA.gross_amount.toLocaleString('id-ID');

        // Render items
        const itemList = document.getElementById('item-list');
        ORDER_DATA.items.forEach(item => {
            const div = document.createElement('div');
            div.className = 'flex justify-between text-sm';
            div.innerHTML = `
                <span class="text-gray-600">${item.name} x${item.qty}</span>
                <span class="text-gray-800">Rp ${(item.price * item.qty).toLocaleString('id-ID')}</span>
            `;
            itemList.appendChild(div);
        });

        // Pay button handler
        document.getElementById('pay-button').addEventListener('click', function() {
            snap.pay(SNAP_TOKEN, {
                onSuccess: function(result) {
                    showResult('success');
                    sendResultToBackend(result);
                    // Redirect ke status page setelah 2 detik
                    setTimeout(() => {
                        window.location.href = '/status/' + ORDER_DATA.order_id;
                    }, 2000);
                },
                onPending: function(result) {
                    showResult('pending');
                    sendResultToBackend(result);
                },
                onError: function(result) {
                    showResult('error');
                    sendResultToBackend(result);
                },
                onClose: function() {
                    console.log('Payment popup closed');
                }
            });
        });

        function showResult(type) {
            document.getElementById('result').classList.remove('hidden');
            document.getElementById('result-' + type).classList.remove('hidden');
            document.getElementById('pay-button').classList.add('hidden');
        }

        function sendResultToBackend(result) {
            fetch('/api/v1/payments/process', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(result)
            });
        }
    </script>
</body>
</html>
```

---

### Halaman 2: Status Payment (`/status/:order_id`)

Halaman untuk melihat status pembayaran secara real-time.

```html
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Status Pembayaran</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-50 min-h-screen flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-lg max-w-md w-full p-6">
        <!-- Status Icon -->
        <div id="status-icon" class="text-center mb-6">
            <!-- Dynamic icon based on status -->
        </div>

        <!-- Status Info -->
        <div class="text-center mb-6">
            <h1 id="status-title" class="text-2xl font-bold"></h1>
            <p id="status-message" class="text-gray-500 mt-2"></p>
        </div>

        <!-- Transaction Detail -->
        <div class="border rounded-xl p-4 space-y-3">
            <div class="flex justify-between">
                <span class="text-gray-500">Order ID</span>
                <span id="detail-order-id" class="font-medium"></span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-500">Total</span>
                <span id="detail-amount" class="font-medium"></span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-500">Metode Bayar</span>
                <span id="detail-payment-type" class="font-medium"></span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-500">Status</span>
                <span id="detail-status" class="font-semibold"></span>
            </div>
            <div class="flex justify-between">
                <span class="text-gray-500">Waktu</span>
                <span id="detail-time" class="font-medium text-sm"></span>
            </div>
        </div>

        <!-- Back Button -->
        <div class="mt-6 text-center">
            <p class="text-gray-500 text-sm">Anda bisa menutup halaman ini dan
               kembali ke WhatsApp.</p>
        </div>
    </div>

    <script>
        const ORDER_ID = "{{.OrderID}}";

        const STATUS_CONFIG = {
            settlement: {
                icon: '✅', title: 'Pembayaran Berhasil',
                message: 'Terima kasih! Pembayaran telah diterima.',
                color: 'text-green-600', bg: 'bg-green-100'
            },
            capture: {
                icon: '✅', title: 'Pembayaran Berhasil',
                message: 'Pembayaran telah diproses.',
                color: 'text-green-600', bg: 'bg-green-100'
            },
            pending: {
                icon: '⏳', title: 'Menunggu Pembayaran',
                message: 'Silakan selesaikan pembayaran sesuai instruksi.',
                color: 'text-yellow-600', bg: 'bg-yellow-100'
            },
            deny: {
                icon: '❌', title: 'Pembayaran Ditolak',
                message: 'Pembayaran tidak dapat diproses.',
                color: 'text-red-600', bg: 'bg-red-100'
            },
            expire: {
                icon: '⏰', title: 'Pembayaran Kadaluarsa',
                message: 'Waktu pembayaran telah habis.',
                color: 'text-gray-600', bg: 'bg-gray-100'
            },
            cancel: {
                icon: '🚫', title: 'Pembayaran Dibatalkan',
                message: 'Transaksi telah dibatalkan.',
                color: 'text-gray-600', bg: 'bg-gray-100'
            }
        };

        async function fetchStatus() {
            try {
                const res = await fetch(`/api/v1/payments/status/${ORDER_ID}`);
                const data = await res.json();

                if (data.success) {
                    renderStatus(data.data);
                }
            } catch (err) {
                console.error('Error fetching status:', err);
            }
        }

        function renderStatus(data) {
            const config = STATUS_CONFIG[data.status] || STATUS_CONFIG.pending;

            document.getElementById('status-icon').innerHTML = 
                `<div class="inline-flex items-center justify-center w-20 h-20 
                      rounded-full ${config.bg} text-4xl">${config.icon}</div>`;
            document.getElementById('status-title').textContent = config.title;
            document.getElementById('status-title').className = 
                `text-2xl font-bold ${config.color}`;
            document.getElementById('status-message').textContent = config.message;

            document.getElementById('detail-order-id').textContent = data.order_id;
            document.getElementById('detail-amount').textContent = 
                'Rp ' + parseInt(data.amount).toLocaleString('id-ID');
            document.getElementById('detail-payment-type').textContent = 
                data.payment_type || '-';
            document.getElementById('detail-status').textContent = data.status;
            document.getElementById('detail-status').className = 
                `font-semibold ${config.color}`;
            document.getElementById('detail-time').textContent = 
                data.transaction_time || '-';
        }

        // Fetch status on load
        fetchStatus();

        // Auto-refresh setiap 5 detik jika masih pending
        setInterval(() => {
            const statusEl = document.getElementById('detail-status');
            if (statusEl && statusEl.textContent === 'pending') {
                fetchStatus();
            }
        }, 5000);
    </script>
</body>
</html>
```

---

## 📁 Struktur Project

```
payment-service/
├── main.go
├── go.mod
├── go.sum
├── .env
├── config/
│   └── config.go              # Load env & config
├── internal/
│   ├── handler/
│   │   ├── payment_handler.go # HTTP handlers
│   │   └── page_handler.go    # Frontend page handlers
│   ├── service/
│   │   ├── payment_service.go # Business logic
│   │   └── midtrans_service.go# Midtrans client wrapper
│   ├── repository/
│   │   └── transaction_repo.go# Database operations
│   ├── model/
│   │   ├── transaction.go     # DB model
│   │   └── request.go         # Request/Response DTOs
│   └── middleware/
│       └── cors.go            # CORS middleware
├── templates/
│   ├── payment.html           # Halaman Payment
│   └── status.html            # Halaman Status
├── static/
│   └── (css, js, images)
├── migrations/
│   └── 001_create_transactions.sql
└── README.md
```

---

## 🔐 Security Checklist

- [ ] Jangan expose `MIDTRANS_SERVER_KEY` di frontend
- [ ] Selalu verifikasi notification dengan `CheckTransaction` API
- [ ] Validasi `signature_key` menggunakan SHA512
- [ ] Gunakan HTTPS untuk semua endpoint
- [ ] Implementasi idempotent callback handler
- [ ] Jangan percaya data dari frontend snap.js callback (selalu verifikasi ulang)
- [ ] Rate limiting pada endpoint public
- [ ] Validasi input pada semua endpoint

---

## 🧪 Testing

### Sandbox Credentials

Gunakan credential sandbox dari [Midtrans Dashboard](https://dashboard.sandbox.midtrans.com).

### Test Cards (Sandbox)

| Card Number        | Scenario        |
|--------------------|-----------------|
| 4811 1111 1111 1114 | Success (3DS)  |
| 4911 1111 1111 1113 | Deny           |
| 4411 1111 1111 1118 | Challenge      |

### Test VA Numbers (Sandbox)

Semua Virtual Account number bisa menggunakan simulator di Midtrans Dashboard.

### Ngrok untuk Callback Testing

```bash
ngrok http 8080
# Copy URL dan set di Midtrans Dashboard > Configuration > Payment Notification URL
# Contoh: https://abc123.ngrok.io/api/v1/payments/callback
```

---

## 🚀 Deployment Checklist

- [ ] Ganti environment ke `midtrans.Production`
- [ ] Ganti Server Key dan Client Key ke production
- [ ] Ganti snap.js URL ke `https://app.midtrans.com/snap/snap.js`
- [ ] Set Payment Notification URL di Midtrans Dashboard (production)
- [ ] Pastikan HTTPS aktif
- [ ] Test end-to-end di production dengan nominal kecil
- [ ] Monitor callback logs

---

## 📚 Referensi

- [Midtrans Go Library](https://github.com/Midtrans/midtrans-go)
- [Midtrans Snap Integration Guide](https://docs.midtrans.com/docs/snap-snap-integration-guide)
- [Midtrans Notification/Webhook Docs](https://docs.midtrans.com/docs/https-notification-webhooks)
- [Midtrans Sandbox Dashboard](https://dashboard.sandbox.midtrans.com)
- [Midtrans API Reference](https://docs.midtrans.com)
