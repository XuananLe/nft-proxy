package services

import (
	"github.com/gagliardetto/solana-go"
	"github.com/joho/godotenv"
	"log"
	"os"
	"testing"
)

// Add more tests in here

func init() {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Print("Error loading .env file")
	}
}

// Helper function to initialize the service with RPC_URL check
func initSolanaService() (*SolanaService, error) {
	svc := SolanaService{}
	err := svc.Start()
	if err != nil {
		return nil, err
	}
	return &svc, nil
}

func TestSolanaService_FetchMetadata(t *testing.T) {
	// Initialize SolanaService
	svc, err := initSolanaService()
	if err != nil {
		t.Fatalf("Failed to start SolanaService: %v", err)
	}

	// Test valid public key
	t.Run("Valid Public Key", func(t *testing.T) {
		pk := solana.MustPublicKeyFromBase58("CJ9AXYbSUPoR95oMvWzgCV3GbG3ZubQjFUpRHN7xqAVb")
		data, _, err := svc.TokenData(pk)
		if err != nil {
			t.Fatalf("Error fetching token data for valid public key: %v", err)
		}
		if data == nil {
			t.Fatal("Expected valid metadata, got nil")
		}
		t.Logf("Metadata: %+v\n", data)
	})

	// Test invalid public key
	t.Run("Invalid Public Key", func(t *testing.T) {
		invalidPk := solana.MustPublicKeyFromBase58("invalidPublicKey1234567890")
		_, _, err := svc.TokenData(invalidPk)
		if err == nil {
			t.Fatal("Expected error for invalid public key, got none")
		}
		t.Logf("Error for invalid public key: %v\n", err)
	})

	// Test empty public key
	t.Run("Empty Public Key", func(t *testing.T) {
		emptyPk := solana.PublicKey{}
		_, _, err := svc.TokenData(emptyPk)
		if err == nil {
			t.Fatal("Expected error for empty public key, got none")
		}
		t.Logf("Error for empty public key: %v\n", err)
	})
}

// Test for environment loading and service initialization
func TestSolanaService_Init(t *testing.T) {
	err := godotenv.Load("../.env")
	if err != nil {
		t.Fatal("Error loading .env file")
	}

	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		t.Fatal("RPC_URL not set in .env file")
	}

	svc, err := initSolanaService()
	if err != nil {
		t.Fatalf("Failed to initialize SolanaService: %v", err)
	}

	t.Logf("SolanaService initialized successfully with RPC_URL: %s", rpcURL)
}
