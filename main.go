// main.go
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PatchPayload defines the structure for our Kubernetes patch
type PatchPayload struct {
	Metadata map[string]interface{} `json:"metadata"`
}

// generateRandomString generates a random 8-character hexadecimal string
func generateRandomString() string {
	bytes := make([]byte, 4) // 4 bytes = 8 hex characters
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("⚠️ Warning: Failed to generate random string: %v", err)
		return "00000000" // fallback value
	}
	return hex.EncodeToString(bytes)
}

// processSpecialTags replaces special tags in both keys and values
func processSpecialTags(input map[string]string, timestamp, randomString string) map[string]string {
	result := make(map[string]string)
	for key, val := range input {
		// Process key - replace all occurrences of the tags
		processedKey := strings.ReplaceAll(key, "<TIMESTAMP>", timestamp)
		processedKey = strings.ReplaceAll(processedKey, "<RANDSTRING>", randomString)

		// Process value - replace all occurrences of the tags
		processedVal := strings.ReplaceAll(val, "<TIMESTAMP>", timestamp)
		processedVal = strings.ReplaceAll(processedVal, "<RANDSTRING>", randomString)

		result[processedKey] = processedVal
	}
	return result
}

func main() {
	// --- Kubernetes Client Setup ---
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("🚨 Failed to load in-cluster config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("🚨 Failed to create clientset: %v", err)
	}

	// --- Get Metadata from Environment ---
	podName := os.Getenv("POD_NAME")
	jobName := os.Getenv("JOB_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	customLabelsJSON := os.Getenv("CUSTOM_LABELS")
	customAnnotationsJSON := os.Getenv("CUSTOM_ANNOTATIONS")

	// --- Get Iteration Configuration from Environment ---
	iterations := 1
	if iterationsStr := os.Getenv("ITERATIONS"); iterationsStr != "" {
		if parsed, err := strconv.Atoi(iterationsStr); err != nil {
			log.Printf("⚠️ Warning: Invalid ITERATIONS value '%s', using default 1", iterationsStr)
		} else if parsed < 1 {
			log.Printf("⚠️ Warning: ITERATIONS must be at least 1, using default 1")
		} else {
			iterations = parsed
		}
	}

	var delay time.Duration
	if delayStr := os.Getenv("DELAY"); delayStr != "" {
		if parsed, err := time.ParseDuration(delayStr); err != nil {
			log.Printf("⚠️ Warning: Invalid DELAY value '%s', using no delay", delayStr)
		} else {
			delay = parsed
		}
	}

	if podName == "" || jobName == "" || namespace == "" {
		log.Fatal("🚨 Error: Missing required environment variables (POD_NAME, JOB_NAME, POD_NAMESPACE).")
	}

	log.Printf("Starting metadata update with %d iterations and %v delay between iterations", iterations, delay)

	// --- Run Multiple Iterations ---
	for i := 1; i <= iterations; i++ {
		log.Printf("--- Iteration %d/%d ---", i, iterations)

		// --- Build the Patch Payload Dynamically ---
		metadataPatch := make(map[string]interface{})

		// Process Labels
		if customLabelsJSON != "" {
			var labels map[string]string
			if err := json.Unmarshal([]byte(customLabelsJSON), &labels); err != nil {
				log.Printf("⚠️ Warning: Could not parse CUSTOM_LABELS JSON: %v. Skipping labels.", err)
			} else if len(labels) > 0 {
				// Process special tags in both keys and values
				timestamp := time.Now().UTC().Format(time.RFC3339)
				randomString := generateRandomString()
				processedLabels := processSpecialTags(labels, timestamp, randomString)
				metadataPatch["labels"] = processedLabels
				log.Printf("Found labels to apply: %v", processedLabels)
			}
		}

		// Process Annotations
		if customAnnotationsJSON != "" {
			var annotations map[string]string
			if err := json.Unmarshal([]byte(customAnnotationsJSON), &annotations); err != nil {
				log.Printf("⚠️ Warning: Could not parse CUSTOM_ANNOTATIONS JSON: %v. Skipping annotations.", err)
			} else if len(annotations) > 0 {
				// Process special tags in both keys and values (updated for each iteration)
				timestamp := time.Now().UTC().Format(time.RFC3339)
				randomString := generateRandomString()
				processedAnnotations := processSpecialTags(annotations, timestamp, randomString)
				metadataPatch["annotations"] = processedAnnotations
				log.Printf("Found annotations to apply: %v", processedAnnotations)
			}
		}

		// If there's nothing to patch, exit gracefully
		if len(metadataPatch) == 0 {
			log.Println("✅ No labels or annotations provided. Exiting.")
			return
		}

		// Construct the final payload
		payload := PatchPayload{Metadata: metadataPatch}
		patchBytes, err := json.Marshal(payload)
		if err != nil {
			log.Fatalf("🚨 Failed to marshal patch payload: %v", err)
		}

		log.Printf("Preparing to patch with payload: %s", string(patchBytes))

		// --- Apply the Patch ---
		ctx := context.TODO()

		// Patch the Pod
		_, err = clientset.CoreV1().Pods(namespace).Patch(ctx, podName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			log.Printf("❌ Failed to patch Pod %s: %v", podName, err)
		} else {
			log.Printf("✅ Successfully patched Pod: %s", podName)
		}

		// Patch the Job
		_, err = clientset.BatchV1().Jobs(namespace).Patch(ctx, jobName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
		if err != nil {
			log.Printf("❌ Failed to patch Job %s: %v", jobName, err)
		} else {
			log.Printf("✅ Successfully patched Job: %s", jobName)
		}

		// Wait for the specified delay before the next iteration (except for the last iteration)
		if i < iterations && delay > 0 {
			log.Printf("⏳ Waiting %v before next iteration...", delay)
			time.Sleep(delay)
		}
	}

	log.Printf("🎉 Completed all %d iterations", iterations)
}
