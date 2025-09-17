// main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
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

	if podName == "" || jobName == "" || namespace == "" {
		log.Fatal("🚨 Error: Missing required environment variables (POD_NAME, JOB_NAME, POD_NAMESPACE).")
	}

	// --- Build the Patch Payload Dynamically ---
	metadataPatch := make(map[string]interface{})

	// Process Labels
	if customLabelsJSON != "" {
		var labels map[string]string
		if err := json.Unmarshal([]byte(customLabelsJSON), &labels); err != nil {
			log.Printf("⚠️ Warning: Could not parse CUSTOM_LABELS JSON: %v. Skipping labels.", err)
		} else if len(labels) > 0 {
			metadataPatch["labels"] = labels
			log.Printf("Found labels to apply: %v", labels)
		}
	}

	// Process Annotations
	if customAnnotationsJSON != "" {
		var annotations map[string]string
		if err := json.Unmarshal([]byte(customAnnotationsJSON), &annotations); err != nil {
			log.Printf("⚠️ Warning: Could not parse CUSTOM_ANNOTATIONS JSON: %v. Skipping annotations.", err)
		} else if len(annotations) > 0 {
			// Check for the special <TIMESTAMP> value and replace it
			timestamp := time.Now().UTC().Format(time.RFC3339)
			for key, val := range annotations {
				if strings.ToUpper(val) == "<TIMESTAMP>" {
					annotations[key] = timestamp
				}
			}
			metadataPatch["annotations"] = annotations
			log.Printf("Found annotations to apply: %v", annotations)
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
}
