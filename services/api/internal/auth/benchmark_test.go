package auth

import (
	"sync"
	"testing"
	"time"
)

func BenchmarkArgon2idHash(b *testing.B) {
	password := "Correct-Horse-Battery-Staple-12345"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := HashPassword(password, DefaultArgon2Params)
		if err != nil {
			b.Fatalf("failed to hash: %v", err)
		}
	}
}

func BenchmarkArgon2idVerify(b *testing.B) {
	password := "Correct-Horse-Battery-Staple-12345"
	hash, err := HashPassword(password, DefaultArgon2Params)
	if err != nil {
		b.Fatalf("failed to hash: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valid, err := VerifyPassword(password, hash)
		if err != nil || !valid {
			b.Fatalf("failed to verify: %v", err)
		}
	}
}

func TestArgon2CapacityAndConcurrency(t *testing.T) {
	password := "Correct-Horse-Battery-Staple-12345"

	// Measure single hash latency
	start := time.Now()
	hash, err := HashPassword(password, DefaultArgon2Params)
	hashDuration := time.Since(start)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	t.Logf("Argon2id 64MB Single Hash Latency: %v", hashDuration)
	if hashDuration > 1*time.Second {
		t.Errorf("hash duration %v exceeds 1s threshold", hashDuration)
	}

	// Measure single verify latency
	start = time.Now()
	valid, err := VerifyPassword(password, hash)
	verifyDuration := time.Since(start)
	if err != nil || !valid {
		t.Fatalf("failed to verify password: %v", err)
	}
	t.Logf("Argon2id 64MB Single Verify Latency: %v", verifyDuration)
	if verifyDuration > 1*time.Second {
		t.Errorf("verify duration %v exceeds 1s threshold", verifyDuration)
	}

	// Test concurrent execution under bounded semaphore
	const concurrentOps = 16
	var wg sync.WaitGroup
	errCount := 0
	var mu sync.Mutex

	t.Logf("Running %d concurrent Argon2id operations to test bounded concurrency...", concurrentOps)
	benchStart := time.Now()
	for i := 0; i < concurrentOps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := VerifyPassword(password, hash)
			if err != nil {
				mu.Lock()
				errCount++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	totalDuration := time.Since(benchStart)
	t.Logf("Completed %d concurrent ops in %v (errors: %d)", concurrentOps, totalDuration, errCount)
}

