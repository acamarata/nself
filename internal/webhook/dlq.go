package webhook

import (
	"context"
	"fmt"
	"time"
)

// DLQEntry represents a row in the np_webhook_dlq table.
type DLQEntry struct {
	ID           string     `json:"id"`
	EndpointID   string     `json:"endpoint_id"`
	DeliveryID   string     `json:"delivery_id"`
	Payload      []byte     `json:"payload"`
	FinalError   string     `json:"final_error"`
	QuarantinedAt time.Time `json:"quarantined_at"`
	ReEnqueuedAt *time.Time `json:"re_enqueued_at,omitempty"`
}

// DLQStore defines the persistence contract for DLQ operations.
// Implementations provide the DB-specific logic (Postgres, mock, etc.).
type DLQStore interface {
	// Save persists a new DLQ entry (quarantine).
	Save(ctx context.Context, entry DLQEntry) error
	// List returns paginated DLQ entries, most recent first.
	List(ctx context.Context, limit, offset int) ([]DLQEntry, error)
	// ReEnqueue marks the entry as re-enqueued and resets its parent delivery for retry.
	// Returns ErrDLQEntryNotFound if id does not exist.
	ReEnqueue(ctx context.Context, id string) error
}

// ErrDLQEntryNotFound is returned when a DLQ entry ID does not exist.
var ErrDLQEntryNotFound = fmt.Errorf("webhook: DLQ entry not found")

// DLQManager wraps a DLQStore and exposes higher-level operations.
type DLQManager struct {
	store DLQStore
}

// NewDLQManager returns a DLQManager backed by store.
func NewDLQManager(store DLQStore) *DLQManager {
	return &DLQManager{store: store}
}

// Quarantine moves a delivery to the DLQ after exhausting all retries.
func (m *DLQManager) Quarantine(ctx context.Context, endpointID, deliveryID string, payload []byte, finalErr string) error {
	entry := DLQEntry{
		EndpointID:    endpointID,
		DeliveryID:    deliveryID,
		Payload:       payload,
		FinalError:    finalErr,
		QuarantinedAt: time.Now().UTC(),
	}
	if err := m.store.Save(ctx, entry); err != nil {
		return fmt.Errorf("webhook: quarantine delivery %s: %w", deliveryID, err)
	}
	return nil
}

// ReEnqueue re-queues a DLQ entry for immediate retry.
// The entry's parent delivery has its attempt_count reset to 0 and next_retry_at set to now.
// Re-enqueue completes within 5 seconds (enforced by the store implementation).
func (m *DLQManager) ReEnqueue(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := m.store.ReEnqueue(ctx, id); err != nil {
		return fmt.Errorf("webhook: re-enqueue DLQ entry %s: %w", id, err)
	}
	return nil
}

// List returns a paginated slice of DLQ entries.
func (m *DLQManager) List(ctx context.Context, limit, offset int) ([]DLQEntry, error) {
	return m.store.List(ctx, limit, offset)
}
