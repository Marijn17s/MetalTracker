package storage

import (
	"database/sql"
	"fmt"
	"time"

	"MetalTracker/internal/domain"

	"github.com/google/uuid"
)

func (database *DB) CreateAttachment(attachment domain.Attachment) (domain.Attachment, error) {
	if attachment.ID == "" {
		attachment.ID = uuid.NewString()
	}
	if attachment.CreatedAt == "" {
		attachment.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if attachment.RelativePath == "" {
		attachment.RelativePath = attachment.ID + ".bin"
	}
	_, err := database.conn.Exec(`
		INSERT INTO attachments(id, owner_type, owner_id, kind, filename, content_type, relative_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		attachment.ID,
		attachment.OwnerType,
		attachment.OwnerID,
		attachment.Kind,
		attachment.Filename,
		attachment.ContentType,
		attachment.RelativePath,
		attachment.CreatedAt,
	)
	if err != nil {
		return domain.Attachment{}, err
	}
	return attachment, nil
}

func (database *DB) ListAttachments(ownerType string, ownerID string) ([]domain.Attachment, error) {
	rows, err := database.conn.Query(`
		SELECT id, owner_type, owner_id, kind, filename, content_type, relative_path, created_at
		FROM attachments
		WHERE owner_type = ? AND owner_id = ?
		ORDER BY created_at DESC`, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAttachments(rows)
}

func (database *DB) ListAllAttachments() ([]domain.Attachment, error) {
	rows, err := database.conn.Query(`
		SELECT id, owner_type, owner_id, kind, filename, content_type, relative_path, created_at
		FROM attachments
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAttachments(rows)
}

func scanAttachments(rows *sql.Rows) ([]domain.Attachment, error) {
	items := make([]domain.Attachment, 0)
	for rows.Next() {
		var item domain.Attachment
		if err := rows.Scan(
			&item.ID,
			&item.OwnerType,
			&item.OwnerID,
			&item.Kind,
			&item.Filename,
			&item.ContentType,
			&item.RelativePath,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (database *DB) GetAttachment(attachmentID string) (domain.Attachment, error) {
	var item domain.Attachment
	err := database.conn.QueryRow(`
		SELECT id, owner_type, owner_id, kind, filename, content_type, relative_path, created_at
		FROM attachments WHERE id = ?`, attachmentID,
	).Scan(
		&item.ID,
		&item.OwnerType,
		&item.OwnerID,
		&item.Kind,
		&item.Filename,
		&item.ContentType,
		&item.RelativePath,
		&item.CreatedAt,
	)
	if err != nil {
		return domain.Attachment{}, err
	}
	return item, nil
}

func (database *DB) DeleteAttachment(attachmentID string) error {
	result, err := database.conn.Exec(`DELETE FROM attachments WHERE id = ?`, attachmentID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("attachment not found")
	}
	return nil
}

func (database *DB) DeleteAttachmentsForOwner(ownerType string, ownerID string) ([]domain.Attachment, error) {
	items, err := database.ListAttachments(ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	if _, err := database.conn.Exec(
		`DELETE FROM attachments WHERE owner_type = ? AND owner_id = ?`,
		ownerType, ownerID,
	); err != nil {
		return nil, err
	}
	return items, nil
}
