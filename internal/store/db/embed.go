package db

import (
	_ "embed"
)

//go:embed sql/schema.sql
var Schema []byte
