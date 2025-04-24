package models

import (
	"database/sql"
	"fmt"
	"time"
	"wedding-invite/pkg/db"
)

// Guest represents a single guest attending the wedding
type Guest struct {
	ID                  int64
	InvitationEmail     string
	Name                string
	Attending           sql.NullBool
	MealPreference      sql.NullString
	DietaryRestrictions sql.NullString
	LastUpdated         time.Time
}

// MealOptions defines available meal choices
var MealOptions = []string{
	"Standard",
	"Vegetarian",           // Meniu vegetarian nunta
	"Ovo-Lacto Vegetarian", // Meniu ovo-lacto vegetarian nunta
	"Ovo-Lacto with Fish",  // Meniu ovo-lacto cu peste nunta
	"Muslim",               // Meniu musulmani nunta
	"Gluten-Free",          // Meniu fara Gluten
	"Lactose-Free",         // Meniu fara lactoza
	"Child",
}

// GetGuestsByInvitation retrieves all guests for a specific invitation
func GetGuestsByInvitation(email string) ([]Guest, error) {
	rows, err := db.DB.Query(`
		SELECT id, invitation_email, name, attending, meal_preference, 
		       dietary_restrictions, last_updated
		FROM guests
		WHERE invitation_email = ?
		ORDER BY id
	`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []Guest

	for rows.Next() {
		var g Guest
		if err := rows.Scan(
			&g.ID,
			&g.InvitationEmail,
			&g.Name,
			&g.Attending,
			&g.MealPreference,
			&g.DietaryRestrictions,
			&g.LastUpdated,
		); err != nil {
			return nil, err
		}

		guests = append(guests, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return guests, nil
}

// GetAllGuests retrieves all guests from the database
func GetAllGuests() ([]Guest, error) {
	rows, err := db.DB.Query(`
		SELECT id, invitation_email, name, attending, meal_preference, 
		       dietary_restrictions, last_updated
		FROM guests
		ORDER BY invitation_email, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []Guest

	for rows.Next() {
		var g Guest
		if err := rows.Scan(
			&g.ID,
			&g.InvitationEmail,
			&g.Name,
			&g.Attending,
			&g.MealPreference,
			&g.DietaryRestrictions,
			&g.LastUpdated,
		); err != nil {
			return nil, err
		}

		guests = append(guests, g)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return guests, nil
}

// CreateGuest adds a new guest to the database
func CreateGuest(email, name string) (int64, error) {
	result, err := db.DB.Exec(`
		INSERT INTO guests (invitation_email, name)
		VALUES (?, ?)
	`, email, name)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

// UpdateGuestRSVP updates a guest's RSVP status
func UpdateGuestRSVP(id int64, attending bool, mealPreference, dietaryRestrictions string) error {
	_, err := db.DB.Exec(`
		UPDATE guests
		SET attending = ?,
		    meal_preference = ?,
		    dietary_restrictions = ?,
		    last_updated = CURRENT_TIMESTAMP
		WHERE id = ?
	`, attending, mealPreference, dietaryRestrictions, id)

	return err
}

// UpdateGuestName updates a guest's name
func UpdateGuestName(id int64, email, name string) error {
	// Only allow updates for guests that belong to the given invitation
	_, err := db.DB.Exec(`
		UPDATE guests
		SET name = ?
		WHERE id = ? AND invitation_email = ?
	`, name, id, email)

	return err
}

// DeleteGuest removes a guest from the database
func DeleteGuest(id int64, email string) error {
	// Only allow deletion if the guest belongs to the given invitation
	result, err := db.DB.Exec(`
		DELETE FROM guests
		WHERE id = ? AND invitation_email = ?
	`, id, email)
	if err != nil {
		return err
	}

	// Check if any row was actually deleted
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("guest not found or not authorized")
	}

	return nil
}

// GetGuestCount returns the number of guests for an invitation
func GetGuestCount(email string) (int, error) {
	var count int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM guests
		WHERE invitation_email = ?
	`, email).Scan(&count)

	return count, err
}

// GetMaxGuestCount retrieves the maximum allowed guests for an invitation
func GetMaxGuestCount(email string) (int, error) {
	var maxGuests int
	err := db.DB.QueryRow(`
		SELECT max_guests FROM invitations
		WHERE email = ?
	`, email).Scan(&maxGuests)

	return maxGuests, err
}

// CheckCanAddGuest verifies if another guest can be added to the invitation
func CheckCanAddGuest(email string) (bool, error) {
	maxGuests, err := GetMaxGuestCount(email)
	if err != nil {
		return false, err
	}

	currentCount, err := GetGuestCount(email)
	if err != nil {
		return false, err
	}

	return currentCount < maxGuests, nil
}

// GetGuest retrieves a specific guest by ID
func GetGuest(id int64) (*Guest, error) {
	var g Guest
	err := db.DB.QueryRow(`
		SELECT id, invitation_email, name, attending, meal_preference, 
		       dietary_restrictions, last_updated
		FROM guests
		WHERE id = ?
	`, id).Scan(
		&g.ID,
		&g.InvitationEmail,
		&g.Name,
		&g.Attending,
		&g.MealPreference,
		&g.DietaryRestrictions,
		&g.LastUpdated,
	)
	if err != nil {
		return nil, err
	}

	return &g, nil
}

// RecordAttendanceStatus records overall attendance status when no guests are present
func RecordAttendanceStatus(email string, attending bool) error {
	// Create a minimal guest entry to record attendance
	result, err := db.DB.Exec(`
		INSERT INTO guests (invitation_email, name, attending, last_updated)
		VALUES (?, 'Primary Contact', ?, CURRENT_TIMESTAMP)
	`, email, attending)
	if err != nil {
		return err
	}

	_, err = result.LastInsertId()
	return err
}

// RemovePrimaryContactGuest removes the auto-generated "Primary Contact" guest entry
// if it exists for the given invitation
func RemovePrimaryContactGuest(email string) error {
	_, err := db.DB.Exec(`
		DELETE FROM guests 
		WHERE invitation_email = ? AND name = 'Primary Contact'
	`, email)

	return err
}