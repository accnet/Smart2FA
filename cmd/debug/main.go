package main

import (
	"fmt"
	"log"

	"smart2fa/internal/db"
)

func main() {
	db.Init("data/smart2fa.db")

	rows, err := db.DB.Query("SELECT id, entries_count, length(encrypted_blob) FROM vaults v JOIN vault_blobs vb ON vb.vault_id = v.id")
	if err != nil {
		log.Fatalf("query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, count, blobLen int64
		rows.Scan(&id, &count, &blobLen)
		fmt.Printf("vault_id=%d entries_count=%d blob_len=%d\n", id, count, blobLen)
	}
}
