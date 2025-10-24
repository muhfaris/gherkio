# Catatan Perilaku API

File ini digunakan untuk mencatat perilaku spesifik API yang ditemukan selama pengujian, terutama yang berbeda dari asumsi atau yang tidak terdokumentasi dengan jelas.

---

## Tipe Dokumen (`/api/document-types`)

### `POST /api/document-types`

*   **Request:** Payload dapat berisi `code`, `name`, `description`, `reminder` (integer), dan `emailReceivers` (array of strings).
*   **Response:** Respons sukses (`201 Created`) hanya mengembalikan `data.id` dari objek yang baru dibuat. **Respons tidak berisi seluruh objek yang baru dibuat.**

### `GET /api/document-types/{id}`

*   **Response:** Respons sukses (`200 OK`) berisi objek Tipe Dokumen yang lengkap.
*   **Perilaku Penting:**
    *   Field `reminder` yang dikirim dalam request `POST` tidak dikembalikan.
    *   Sebagai gantinya, respons berisi field `reminders` dengan nilai berupa array kosong (`[]`).

---
