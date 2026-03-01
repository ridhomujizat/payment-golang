# Prompt: WhatsApp Flows Endpoint — Decrypt/Encrypt (Golang Gin)

## Objective

Buatkan service endpoint menggunakan **Golang + Gin** yang menerima encrypted request dari WhatsApp Flows, mendekripsi payload, memproses action, lalu mengembalikan encrypted response. Gunakan **hanya standard library Go** (`crypto/*`, `encoding/*`) tanpa library pihak ketiga selain Gin.

---

## Tech Stack

- **Language:** Go (1.21+)
- **Framework:** Gin (`github.com/gin-gonic/gin`)
- **Crypto:** Standard library only — `crypto/rsa`, `crypto/aes`, `crypto/cipher`, `crypto/sha256`, `crypto/x509`, `encoding/pem`, `encoding/base64`, `encoding/json`

---

## WhatsApp Flows Encryption Specification

Referensi resmi: https://developers.facebook.com/docs/whatsapp/flows/guides/implementingyourflowendpoint

### Incoming Request Body (encrypted)

WhatsApp Flows mengirim POST request dengan JSON body berikut:

```json
{
  "encrypted_flow_data": "<BASE64_ENCODED_ENCRYPTED_PAYLOAD>",
  "encrypted_aes_key": "<BASE64_ENCODED_ENCRYPTED_AES_KEY>",
  "initial_vector": "<BASE64_ENCODED_IV>"
}
```

### Decryption Flow (Request)

1. **Decode semua field** dari Base64.
2. **Decrypt `encrypted_aes_key`** menggunakan RSA private key dengan:
   - Algorithm: **RSA-OAEP**
   - Hash: **SHA-256** (untuk both OAEP hash dan MGF1 hash)
   - Padding: `RSA_PKCS1_OAEP_PADDING` with SHA-256
3. **Decrypt `encrypted_flow_data`** menggunakan AES key yang sudah didekripsi:
   - Algorithm: **AES-128-GCM**
   - IV/Nonce: `initial_vector` (16 bytes)
   - **Auth tag** ada di **16 bytes terakhir** dari `encrypted_flow_data`
   - Ciphertext = `encrypted_flow_data` tanpa 16 bytes terakhir
   - Gunakan `cipher.NewGCMWithNonceSize(block, 16)` karena IV-nya 16 bytes (bukan default 12)

### Decrypted Payload Structure

```json
{
  "version": "3.0",
  "action": "ping | INIT | data_exchange | BACK",
  "screen": "<SCREEN_NAME>",
  "data": {
    "key1": "value1"
  },
  "flow_token": "<FLOW_TOKEN>"
}
```

### Encryption Flow (Response)

1. Buat response JSON sesuai action.
2. **Flip/invert semua byte** dari `initial_vector` (bitwise NOT: `flippedIV[i] = ^iv[i]`).
3. **Encrypt response** menggunakan:
   - Algorithm: **AES-128-GCM**
   - Key: AES key yang sama dari decryption
   - IV/Nonce: **flipped IV** (16 bytes, gunakan `NewGCMWithNonceSize` juga)
   - Output = ciphertext + auth tag (GCM Seal sudah otomatis append tag)
4. **Encode ke Base64** dan kirim sebagai **plain text** response (bukan JSON).

---

## Struktur File yang Diharapkan

```
├── main.go                    # Entry point, setup Gin router
├── config/
│   └── config.go              # Load private key, passphrase, env vars
├── handler/
│   └── flow_handler.go        # Gin handler untuk POST /flow-endpoint
├── crypto/
│   └── flow_crypto.go         # DecryptRequest() dan EncryptResponse()
├── model/
│   └── flow_model.go          # Struct untuk request/response
└── go.mod
```

---

## Model / Struct Definitions

```go
// EncryptedRequest — incoming encrypted body dari WhatsApp
type EncryptedRequest struct {
    EncryptedFlowData string `json:"encrypted_flow_data"`
    EncryptedAESKey   string `json:"encrypted_aes_key"`
    InitialVector     string `json:"initial_vector"`
}

// DecryptedRequest — payload setelah didekripsi
type DecryptedRequest struct {
    Version    string                 `json:"version"`
    Action     string                 `json:"action"`
    Screen     string                 `json:"screen"`
    Data       map[string]interface{} `json:"data"`
    FlowToken  string                 `json:"flow_token"`
}

// FlowResponse — response sebelum dienkripsi
type FlowResponse struct {
    Version string                 `json:"version"`
    Screen  string                 `json:"screen,omitempty"`
    Data    map[string]interface{} `json:"data,omitempty"`
}
```

---

## Crypto Module Requirements (`crypto/flow_crypto.go`)

### `DecryptRequest(privateKey *rsa.PrivateKey, body EncryptedRequest) (*DecryptedRequest, []byte, []byte, error)`

Langkah:
1. Base64 decode `encrypted_aes_key`, `initial_vector`, `encrypted_flow_data`.
2. RSA-OAEP decrypt `encrypted_aes_key` → hasilnya 16-byte AES key.
   ```go
   aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedAESKey, nil)
   ```
3. Pisahkan encrypted flow data:
   - `ciphertext` = data[0 : len(data)-16]
   - `authTag` = data[len(data)-16:]
   - `taggedCiphertext` = append(ciphertext, authTag...) — ini format yang dibutuhkan GCM `Open()`
4. Buat AES-GCM cipher **dengan nonce size 16**:
   ```go
   block, _ := aes.NewCipher(aesKey)
   gcm, _ := cipher.NewGCMWithNonceSize(block, 16) // PENTING: 16 bukan 12
   ```
5. Decrypt:
   ```go
   plaintext, err := gcm.Open(nil, iv, taggedCiphertext, nil)
   ```
6. JSON unmarshal plaintext → `DecryptedRequest`.
7. Return `DecryptedRequest`, `aesKey`, `iv`.

### `EncryptResponse(aesKey []byte, iv []byte, response FlowResponse) (string, error)`

Langkah:
1. JSON marshal response.
2. Flip IV:
   ```go
   flippedIV := make([]byte, len(iv))
   for i := range iv {
       flippedIV[i] = ^iv[i]
   }
   ```
3. Buat AES-GCM cipher **dengan nonce size 16**:
   ```go
   block, _ := aes.NewCipher(aesKey)
   gcm, _ := cipher.NewGCMWithNonceSize(block, 16)
   ```
4. Encrypt:
   ```go
   sealed := gcm.Seal(nil, flippedIV, plaintextBytes, nil)
   ```
   `Seal()` otomatis append auth tag di akhir.
5. Base64 encode `sealed` → return string.

---

## Handler Logic (`handler/flow_handler.go`)

```
POST /flow-endpoint
```

1. Bind JSON body → `EncryptedRequest`.
2. Call `DecryptRequest()` → dapat `DecryptedRequest`, `aesKey`, `iv`.
3. Switch berdasarkan `action`:

   - **`"ping"`** → Response: `{ "data": { "status": "active" } }`
   - **`"INIT"`** → Response: `{ "screen": "FIRST_SCREEN", "data": { ... } }`
   - **`"data_exchange"`** → Process business logic berdasarkan `screen` dan `data`, return next screen
   - **`"BACK"`** → Return data untuk previous screen

4. Call `EncryptResponse()` → dapat Base64 string.
5. Return response sebagai **plain text** (bukan JSON):
   ```go
   c.Data(http.StatusOK, "text/plain", []byte(encryptedResponse))
   ```

---

## Config: Loading Private Key

Private key dalam format PEM (PKCS#1 atau PKCS#8). Bisa encrypted atau unencrypted.

**Untuk unencrypted private key:**
```go
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
    keyData, _ := os.ReadFile(path)
    block, _ := pem.Decode(keyData)
    // Coba PKCS#8 dulu, fallback ke PKCS#1
    key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
    if err != nil {
        return x509.ParsePKCS1PrivateKey(block.Bytes)
    }
    return key.(*rsa.PrivateKey), nil
}
```

**Untuk encrypted private key (dengan passphrase):**
Gunakan `openssl pkcs8 -topk8 -nocrypt` untuk convert ke unencrypted sebelum deploy, atau decrypt manual menggunakan `x509.DecryptPEMBlock` (deprecated tapi masih jalan) / library tambahan.

---

## Critical Notes / Gotchas

1. **Nonce size harus 16 bytes** — WhatsApp Flows menggunakan 128-bit IV, bukan default GCM 96-bit (12 bytes). Wajib pakai `cipher.NewGCMWithNonceSize(block, 16)`. Kalau pakai `cipher.NewGCM()` biasa akan error.

2. **Auth tag di 16 bytes terakhir** — `encrypted_flow_data` setelah di-decode Base64, 16 bytes terakhirnya adalah GCM auth tag. Go's `gcm.Open()` expects ciphertext+tag digabung, jadi bisa langsung pass tanpa split (karena GCM Open otomatis memisahkan).

3. **Flipped IV untuk response** — Response harus diencrypt dengan IV yang di-bitwise-NOT, bukan IV original.

4. **Response content type** — Response harus dikirim sebagai plain text Base64 string, **BUKAN** JSON wrapped.

5. **RSA-OAEP SHA-256** — Hashing function untuk OAEP **dan** MGF1 harus sama-sama SHA-256. Di Go, `rsa.DecryptOAEP(sha256.New(), ...)` sudah handle kedua-duanya.

6. **Health check (ping)** — WhatsApp periodik mengirim action `"ping"`. Endpoint harus merespons dengan `{ "data": { "status": "active" } }` (tetap di-encrypt).

7. **Error handling** — Jangan pernah return error detail ke WhatsApp. Log internal, return generic encrypted error response.

---

## Environment Variables

```env
PRIVATE_KEY_PATH=./keys/private.pem
PORT=3000
# Optional: APP_SECRET untuk validasi signature
```

---

## Contoh Full Decrypt → Encrypt Lifecycle

```
[WhatsApp Client]
       |
       v
POST /flow-endpoint
{
  "encrypted_flow_data": "abc123...",
  "encrypted_aes_key": "xyz789...",
  "initial_vector": "iv000..."
}
       |
       v
[1] Base64 decode semua field
[2] RSA-OAEP(SHA256) decrypt encrypted_aes_key → AES key (16 bytes)
[3] AES-128-GCM decrypt encrypted_flow_data pakai AES key + IV
    (nonce size = 16, auth tag = 16 bytes terakhir)
       |
       v
Decrypted: { "action": "INIT", "version": "3.0", ... }
       |
       v
[4] Process action → build response JSON
[5] Flip IV (bitwise NOT)
[6] AES-128-GCM encrypt response pakai AES key + flipped IV
[7] Base64 encode → return sebagai plain text
       |
       v
Response: "SGVsbG8gV29ybGQ=..." (base64 string)
```

---

## Testing Tips

- Gunakan openssl untuk generate test key pair:
  ```bash
  openssl genrsa -out private.pem 2048
  openssl rsa -in private.pem -pubout -out public.pem
  ```
- Buat unit test di `crypto/flow_crypto_test.go` yang encrypt data pakai public key, lalu verify decrypt → encrypt → decrypt roundtrip.
- Test health check (ping) endpoint terlebih dahulu sebelum test flow action lainnya.