package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupRoundTripAndVerify(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "vault.meta.json")
	encPath := filepath.Join(dir, "vault.db.enc")
	attachmentsDir := filepath.Join(dir, "attachments")
	if err := os.MkdirAll(attachmentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metaPath, []byte(`{"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(encPath, []byte("encrypted-db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(attachmentsDir, "a.bin"), []byte("photo"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, recoveryRaw, err := GenerateRecoveryKey()
	if err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dir, "portfolio.mtbackup")
	manifest, err := CreateBackupFile(backupPath, recoveryRaw, BackupSource{
		MetaPath:       metaPath,
		EncDBPath:      encPath,
		AttachmentsDir: attachmentsDir,
		UnitCount:      3,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if manifest.UnitCount != 3 || manifest.AttachmentCount != 1 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}

	verify, err := VerifyBackupFile(backupPath, recoveryRaw)
	if err != nil || !verify.Valid {
		t.Fatalf("verify: %+v err=%v", verify, err)
	}

	restoreDir := filepath.Join(dir, "restored")
	restored, err := RestoreBackupFile(backupPath, recoveryRaw, restoreDir)
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if restored.UnitCount != 3 {
		t.Fatalf("restored unit count %d", restored.UnitCount)
	}
	metaOut, err := os.ReadFile(filepath.Join(restoreDir, "vault.meta.json"))
	if err != nil || string(metaOut) != `{"version":1}` {
		t.Fatalf("restored meta: %s err=%v", metaOut, err)
	}
	attachmentOut, err := os.ReadFile(filepath.Join(restoreDir, "attachments", "a.bin"))
	if err != nil || string(attachmentOut) != "photo" {
		t.Fatalf("restored attachment: %s err=%v", attachmentOut, err)
	}
}

func TestBackupRejectsWrongRecoveryKey(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "vault.meta.json")
	encPath := filepath.Join(dir, "vault.db.enc")
	_ = os.WriteFile(metaPath, []byte(`{}`), 0o600)
	_ = os.WriteFile(encPath, []byte("x"), 0o600)
	_, recoveryRaw, _ := GenerateRecoveryKey()
	_, otherRaw, _ := GenerateRecoveryKey()
	backupPath := filepath.Join(dir, "bad.mtbackup")
	if _, err := CreateBackupFile(backupPath, recoveryRaw, BackupSource{
		MetaPath: metaPath,
		EncDBPath: encPath,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyBackupFile(backupPath, otherRaw); err == nil {
		t.Fatal("expected wrong recovery key to fail")
	}
}
