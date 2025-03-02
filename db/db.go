package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// Initialize sets up the database connection and creates tables if they don't exist
func Initialize() error {
	// Create data directory if it doesn't exist
	dataDir := "./data"
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		if err := os.Mkdir(dataDir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	// Open database connection
	var err error
	dbPath := filepath.Join(dataDir, "poe2bot.db")
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables if they don't exist
	if err := createTables(); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// createTables creates the necessary tables in the database
func createTables() error {
	// Items table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			estimated_value INTEGER NOT NULL,
			status TEXT NOT NULL,
			assigned_to TEXT,
			sale_amount INTEGER,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// Participants table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS participants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			item_id INTEGER NOT NULL,
			user_id TEXT NOT NULL,
			share_amount INTEGER,
			FOREIGN KEY (item_id) REFERENCES items(id),
			UNIQUE(item_id, user_id)
		)
	`)
	if err != nil {
		return err
	}

	// Profit history table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS profit_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			item_id INTEGER NOT NULL,
			amount INTEGER NOT NULL,
			transaction_date TIMESTAMP NOT NULL,
			FOREIGN KEY (item_id) REFERENCES items(id)
		)
	`)
	if err != nil {
		return err
	}

	return nil
}

// Item represents an item in the database
type Item struct {
	ID             int64
	Name           string
	EstimatedValue int64
	Status         string // "assigned", "sold", "distributed"
	AssignedTo     string
	SaleAmount     int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Participants   []Participant
}

// Participant represents a participant in an item drop
type Participant struct {
	ID          int64
	ItemID      int64
	UserID      string
	ShareAmount int64
}

// ProfitRecord represents a profit transaction in the history
type ProfitRecord struct {
	ID              int64
	UserID          string
	ItemID          int64
	ItemName        string
	Amount          int64
	TransactionDate time.Time
}

// AddItem adds a new item to the database
func AddItem(name string, estimatedValue int64, participants []string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Insert item
	now := time.Now()
	result, err := tx.Exec(
		"INSERT INTO items (name, estimated_value, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		name, estimatedValue, "assigned", now, now,
	)
	if err != nil {
		return 0, err
	}

	// Get the ID of the inserted item
	itemID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Insert participants
	for _, userID := range participants {
		// Clean up user ID (remove mentions if present)
		userID = strings.TrimPrefix(userID, "<@")
		userID = strings.TrimPrefix(userID, "!")
		userID = strings.TrimSuffix(userID, ">")

		_, err := tx.Exec(
			"INSERT INTO participants (item_id, user_id) VALUES (?, ?)",
			itemID, userID,
		)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return itemID, nil
}

// AssignItem assigns an item to a user for selling
func AssignItem(itemID int64, userID string) error {
	// Clean up user ID
	userID = strings.TrimPrefix(userID, "<@")
	userID = strings.TrimPrefix(userID, "!")
	userID = strings.TrimSuffix(userID, ">")

	// Update item status
	_, err := db.Exec(
		"UPDATE items SET status = ?, assigned_to = ?, updated_at = ? WHERE id = ?",
		"assigned", userID, time.Now(), itemID,
	)
	return err
}

// MarkItemAsSoldAndDistribute marks an item as sold, calculates shares, and records profit history
func MarkItemAsSoldAndDistribute(itemID int64, saleAmount int64, shares map[string]int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if item exists and is in assigned status
	var status string
	var itemName string
	err = tx.QueryRow("SELECT status, name FROM items WHERE id = ?", itemID).Scan(&status, &itemName)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("item with ID %d not found", itemID)
		}
		return err
	}

	if status != "assigned" {
		return fmt.Errorf("item must be assigned before it can be sold")
	}

	// Update item status
	now := time.Now()
	_, err = tx.Exec(
		"UPDATE items SET status = ?, sale_amount = ?, updated_at = ? WHERE id = ?",
		"distributed", saleAmount, now, itemID,
	)
	if err != nil {
		return err
	}

	// Update participant shares and record profit history
	for userID, shareAmount := range shares {
		// Clean up user ID
		cleanUserID := strings.TrimPrefix(userID, "<@")
		cleanUserID = strings.TrimPrefix(cleanUserID, "!")
		cleanUserID = strings.TrimSuffix(cleanUserID, ">")

		// Update participant share
		_, err = tx.Exec(
			"UPDATE participants SET share_amount = ? WHERE item_id = ? AND user_id = ?",
			shareAmount, itemID, cleanUserID,
		)
		if err != nil {
			return err
		}

		// Record profit history
		_, err = tx.Exec(
			"INSERT INTO profit_history (user_id, item_id, amount, transaction_date) VALUES (?, ?, ?, ?)",
			cleanUserID, itemID, shareAmount, now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// MarkItemAsSold marks an item as sold and calculates shares
func MarkItemAsSold(itemID int64, saleAmount int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Check if item exists and is in assigned status
	var status string
	err = tx.QueryRow("SELECT status FROM items WHERE id = ?", itemID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("item with ID %d not found", itemID)
		}
		return err
	}

	if status != "assigned" {
		return fmt.Errorf("item must be assigned before it can be sold")
	}

	// Update item status
	_, err = tx.Exec(
		"UPDATE items SET status = ?, sale_amount = ?, updated_at = ? WHERE id = ?",
		"sold", saleAmount, time.Now(), itemID,
	)
	if err != nil {
		return err
	}

	// Count participants
	var count int
	err = tx.QueryRow("SELECT COUNT(*) FROM participants WHERE item_id = ?", itemID).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("no participants found for item %d", itemID)
	}

	// Calculate share amount per participant
	shareAmount := saleAmount / int64(count)

	// Update participant shares
	_, err = tx.Exec(
		"UPDATE participants SET share_amount = ? WHERE item_id = ?",
		shareAmount, itemID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// MarkItemAsDistributed marks an item as distributed
func MarkItemAsDistributed(itemID int64) error {
	// Check if item exists and is in sold status
	var status string
	err := db.QueryRow("SELECT status FROM items WHERE id = ?", itemID).Scan(&status)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("item with ID %d not found", itemID)
		}
		return err
	}

	if status != "sold" {
		return fmt.Errorf("item must be sold before it can be distributed")
	}

	// Update item status
	_, err = db.Exec(
		"UPDATE items SET status = ?, updated_at = ? WHERE id = ?",
		"distributed", time.Now(), itemID,
	)
	return err
}

// GetItem retrieves an item by ID
func GetItem(itemID int64) (*Item, error) {
	// Get item details
	item := &Item{}
	var nullSaleAmount sql.NullInt64
	var nullAssignedTo sql.NullString
	
	err := db.QueryRow(`
		SELECT id, name, estimated_value, status, assigned_to, sale_amount, created_at, updated_at
		FROM items WHERE id = ?
	`, itemID).Scan(
		&item.ID, &item.Name, &item.EstimatedValue, &item.Status,
		&nullAssignedTo, &nullSaleAmount, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("item with ID %d not found", itemID)
		}
		return nil, err
	}
	
	// Convert NullInt64 to int64 (0 if NULL)
	if nullSaleAmount.Valid {
		item.SaleAmount = nullSaleAmount.Int64
	} else {
		item.SaleAmount = 0
	}
	
	// Convert NullString to string (empty if NULL)
	if nullAssignedTo.Valid {
		item.AssignedTo = nullAssignedTo.String
	} else {
		item.AssignedTo = ""
	}

	// Get participants
	rows, err := db.Query(`
		SELECT id, item_id, user_id, share_amount
		FROM participants WHERE item_id = ?
	`, itemID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Participant
		var nullShareAmount sql.NullInt64
		
		if err := rows.Scan(&p.ID, &p.ItemID, &p.UserID, &nullShareAmount); err != nil {
			return nil, err
		}
		
		// Convert NullInt64 to int64 (0 if NULL)
		if nullShareAmount.Valid {
			p.ShareAmount = nullShareAmount.Int64
		} else {
			p.ShareAmount = 0
		}
		
		item.Participants = append(item.Participants, p)
	}

	return item, nil
}

// ListItems retrieves all items
func ListItems() ([]Item, error) {
	rows, err := db.Query(`
		SELECT id, name, estimated_value, status, assigned_to, sale_amount, created_at, updated_at
		FROM items ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		var nullSaleAmount sql.NullInt64
		var nullAssignedTo sql.NullString
		
		if err := rows.Scan(
			&item.ID, &item.Name, &item.EstimatedValue, &item.Status,
			&nullAssignedTo, &nullSaleAmount, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		
		// Convert NullInt64 to int64 (0 if NULL)
		if nullSaleAmount.Valid {
			item.SaleAmount = nullSaleAmount.Int64
		} else {
			item.SaleAmount = 0
		}
		
		// Convert NullString to string (empty if NULL)
		if nullAssignedTo.Valid {
			item.AssignedTo = nullAssignedTo.String
		} else {
			item.AssignedTo = ""
		}
		
		items = append(items, item)
	}

	return items, nil
}

// GetUserProfitHistory retrieves profit history for a specific user
func GetUserProfitHistory(userID string) ([]ProfitRecord, error) {
	// Clean up user ID (remove mentions if present)
	userID = strings.TrimPrefix(userID, "<@")
	userID = strings.TrimPrefix(userID, "!")
	userID = strings.TrimSuffix(userID, ">")

	rows, err := db.Query(`
		SELECT p.id, p.user_id, p.item_id, i.name, p.amount, p.transaction_date
		FROM profit_history p
		JOIN items i ON p.item_id = i.id
		WHERE p.user_id = ?
		ORDER BY p.transaction_date DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []ProfitRecord
	for rows.Next() {
		var record ProfitRecord
		if err := rows.Scan(
			&record.ID, &record.UserID, &record.ItemID, &record.ItemName,
			&record.Amount, &record.TransactionDate,
		); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

// GetTotalUserProfit calculates the total profit for a user
func GetTotalUserProfit(userID string) (int64, error) {
	var total int64
	err := db.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
		FROM profit_history
		WHERE user_id = ?
	`, userID).Scan(&total)
	
	return total, err
}

// Close closes the database connection
func Close() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
} 