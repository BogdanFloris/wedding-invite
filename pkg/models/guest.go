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
	InvitationID        string
	Name                string
	Attending           sql.NullBool
	MealPreference      sql.NullString
	DietaryRestrictions sql.NullString
	LastUpdated         time.Time
}

// MealOptions defines available meal choices
var MealOptions = []string{
	"Normal",
	"Vegetarian",
	"Staff",
	"Child",
}

// GetGuestsByInvitation retrieves all guests for a specific invitation
func GetGuestsByInvitation(invitationID string) ([]Guest, error) {
	rows, err := db.DB.Query(`
		SELECT id, invitation_id, name, attending, meal_preference, 
		       dietary_restrictions, last_updated
		FROM guests
		WHERE invitation_id = ?
		ORDER BY id
	`, invitationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guests []Guest

	for rows.Next() {
		var g Guest
		if err := rows.Scan(
			&g.ID,
			&g.InvitationID,
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
func CreateGuest(invitationID, name string) (int64, error) {
	result, err := db.DB.Exec(`
		INSERT INTO guests (invitation_id, name)
		VALUES (?, ?)
	`, invitationID, name)
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
func UpdateGuestName(id int64, invitationID, name string) error {
	// Only allow updates for guests that belong to the given invitation
	_, err := db.DB.Exec(`
		UPDATE guests
		SET name = ?
		WHERE id = ? AND invitation_id = ?
	`, name, id, invitationID)

	return err
}

// DeleteGuest removes a guest from the database
func DeleteGuest(id int64, invitationID string) error {
	// Only allow deletion if the guest belongs to the given invitation
	result, err := db.DB.Exec(`
		DELETE FROM guests
		WHERE id = ? AND invitation_id = ?
	`, id, invitationID)
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
func GetGuestCount(invitationID string) (int, error) {
	var count int
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM guests
		WHERE invitation_id = ?
	`, invitationID).Scan(&count)

	return count, err
}

// GetMaxGuestCount retrieves the maximum allowed guests for an invitation
func GetMaxGuestCount(invitationID string) (int, error) {
	var maxGuests int
	err := db.DB.QueryRow(`
		SELECT max_guests FROM invitations
		WHERE id = ?
	`, invitationID).Scan(&maxGuests)

	return maxGuests, err
}

// CheckCanAddGuest verifies if another guest can be added to the invitation
func CheckCanAddGuest(invitationID string) (bool, error) {
	maxGuests, err := GetMaxGuestCount(invitationID)
	if err != nil {
		return false, err
	}

	currentCount, err := GetGuestCount(invitationID)
	if err != nil {
		return false, err
	}

	return currentCount < maxGuests, nil
}

// GetGuest retrieves a specific guest by ID
func GetGuest(id int64) (*Guest, error) {
	var g Guest
	err := db.DB.QueryRow(`
		SELECT id, invitation_id, name, attending, meal_preference, 
		       dietary_restrictions, last_updated
		FROM guests
		WHERE id = ?
	`, id).Scan(
		&g.ID,
		&g.InvitationID,
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

// GetInvitationName returns the family name for an invitation
func GetInvitationName(invitationID string) (string, error) {
	var name string
	err := db.DB.QueryRow(`
		SELECT family_name FROM invitations
		WHERE id = ?
	`, invitationID).Scan(&name)

	return name, err
}
