package db

func (db *DB) GetOrCreateKV(key, value string) (string, error) {
	_, err := db.Exec(
		`INSERT INTO kv_store (key, value, updated_at) VALUES ($1, $2, NOW()) ON CONFLICT (key) DO NOTHING`,
		key, value)
	if err != nil {
		return "", err
	}

	var v string
	if err := db.QueryRow(`SELECT value FROM kv_store WHERE key = $1`, key).Scan(&v); err != nil {
		return "", err
	}
	return v, nil
}
