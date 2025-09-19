# Build Farm Metadata Adder

This project provides a Kubernetes Job that adds custom metadata (labels and annotations) to pods and jobs after completion.

## test-job.yaml

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: metadata-editor-sa
  namespace: test
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: job-pod-editor-role
  namespace: test
rules:
- apiGroups: [""] # "" indicates the core API group
  resources: ["pods"]
  verbs: ["get", "patch"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: job-editor-binding
  namespace: test
subjects:
- kind: ServiceAccount
  name: metadata-editor-sa
roleRef:
  kind: Role
  name: job-pod-editor-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: batch/v1
kind: Job
metadata:
  name: test-metadata-adder-job
  namespace: test
spec:
  template:
    spec:
      # Use the ServiceAccount that has the correct permissions
      serviceAccountName: metadata-editor-sa
      restartPolicy: Never
      containers:
      # This is a placeholder container to simulate some work
      - name: main-task
        image: busybox:latest
        command: ["/bin/sh", "-c", "echo 'Simulating main job task...'; sleep 15; echo 'Main task complete.'"]
      # This is your final metadata update step
      - name: step-update-metadata
        # ❗ IMPORTANT: Replace this with the image you just built and pushed.
        image: quay.io/mcornea/build-farm-metadata-adder
        imagePullPolicy: IfNotPresent
        env:
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: POD_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
          - name: JOB_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.labels['job-name']
          # --- Configure your desired metadata here ---
          # Add labels as a JSON string.
          # Use "<TIMESTAMP>" and "<RANDSTRING>" tags in both keys and values.
          - name: CUSTOM_LABELS
            value: |
              {
                "status": "completed",
                "stage": "testing",
                "build-<RANDSTRING>": "unique-build"
              }
          # Add annotations as a JSON string.
          # Use "<TIMESTAMP>" to have the script insert the completion time.
          # Use "<RANDSTRING>" to have the script insert a random 8-character string.
          # These tags work in both keys and values.
          - name: CUSTOM_ANNOTATIONS
            value: |
              {
                "company.com/completed-at": "<TIMESTAMP>",
                "company.com/build-id": "<RANDSTRING>",
                "<RANDSTRING>-suffix": "some-value",
                "tested-by": "automation-script"
              }
          # --- Configure iteration settings (optional) ---
          # Number of times to apply metadata updates (default: 1)
          - name: ITERATIONS
            value: "3"
          # Delay between iterations (e.g., 5s, 2m, 1h) (default: no delay)
          - name: DELAY
            value: "10s"
  backoffLimit: 1
```