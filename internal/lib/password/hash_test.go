package password

import (
	"testing"
)

func TestGetHash(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "regular password",
			password: "password123",
			wantErr:  false,
		},
		{
			name:     "password with special chars",
			password: "p@ssw0rd!@#$%^&*()",
			wantErr:  false,
		},
		{
			name:     "long password",
			password: "verylongpasswordwithmorethanfiftycharacters",
			wantErr:  false,
		},
		{
			name:     "short password",
			password: "short",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHash, err := GetHash(tt.password)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && gotHash == "" {
				t.Error("GetHash() returned empty hash")
			}

			if !tt.wantErr {
				err = CompareHash(gotHash, tt.password)
				if err != nil {
					t.Errorf("Generated hash doesn't work with original password: %v", err)
				}
			}
		})
	}
}

func TestCompareHash(t *testing.T) {
	correctHash, err := GetHash("correct_password")
	if err != nil {
		t.Fatalf("Failed to create test hash: %v", err)
	}

	anotherHash, err := GetHash("another_password")
	if err != nil {
		t.Fatalf("Failed to create test hash: %v", err)
	}

	tests := []struct {
		name        string
		hash        string
		password    string
		shouldMatch bool
	}{
		{
			name:        "matching password",
			hash:        correctHash,
			password:    "correct_password",
			shouldMatch: true,
		},
		{
			name:        "wrong password",
			hash:        correctHash,
			password:    "wrong_password",
			shouldMatch: false,
		},
		{
			name:        "different hash same password",
			hash:        anotherHash,
			password:    "correct_password",
			shouldMatch: false,
		},
		{
			name:        "empty password",
			hash:        correctHash,
			password:    "",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CompareHash(tt.hash, tt.password)

			if tt.shouldMatch && err != nil {
				t.Errorf("CompareHash() should succeed, got error: %v", err)
			}

			if !tt.shouldMatch && err == nil {
				t.Error("CompareHash() should fail, but got no error")
			}
		})
	}
}

func TestGetHash_DifferentPasswordsProduceDifferentHashes(t *testing.T) {
	hash1, err := GetHash("password1")
	if err != nil {
		t.Fatalf("GetHash failed: %v", err)
	}

	hash2, err := GetHash("password2")
	if err != nil {
		t.Fatalf("GetHash failed: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Different passwords produced identical hashes")
	}
}
