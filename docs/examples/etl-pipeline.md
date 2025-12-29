# ETL Pipeline Example

This example demonstrates how to coordinate a multi-stage ETL (Extract, Transform, Load) pipeline using Konductor barriers and semaphores.

## Scenario

You have a data pipeline with three stages:
1. **Extract**: 5 parallel jobs extract data from different sources
2. **Transform**: 3 jobs transform the extracted data (must wait for all extracts)
3. **Load**: 2 jobs load data into the target system (must wait for all transforms, limited concurrency)

## Architecture

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Extract 1  │  │  Extract 2  │  │  Extract 3  │
└─────────────┘  └─────────────┘  └─────────────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                ┌───────▼────────┐
                │ extract-barrier │
                │  Expected: 5   │
                └───────┬────────┘
                        │
       ┌────────────────┼────────────────┐
       │                │                │
┌─────▼─────┐  ┌─────▼─────┐  ┌─────▼─────┐
│Transform 1│  │Transform 2│  │Transform 3│
└─────┬─────┘  └─────┬─────┘  └─────┬─────┘
      └──────────────┼──────────────┘
                     │
             ┌───────▼────────┐
             │transform-barrier│
             │  Expected: 3   │
             └───────┬────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
   ┌────▼────┐  ┌────▼────┐  ┌────▼────┐
   │ Load 1  │  │ Load 2  │  │ Semaphore│
   └─────────┘  └─────────┘  │Permits: 2│
                              └─────────┘
```

## Implementation

### 1. Create Coordination Resources

```yaml
# barriers.yaml
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: extract-complete
  labels:
    pipeline: etl-demo
    stage: extract
spec:
  expected: 5
  timeout: 2h
---
apiVersion: konductor.io/v1
kind: Barrier
metadata:
  name: transform-complete
  labels:
    pipeline: etl-demo
    stage: transform
spec:
  expected: 3
  timeout: 1h
---
apiVersion: konductor.io/v1
kind: Semaphore
metadata:
  name: load-concurrency
  labels:
    pipeline: etl-demo
    stage: load
spec:
  permits: 2
  ttl: 30m
```

### 2. Extract Jobs

```yaml
# extract-jobs.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: extract-customers
  labels:
    pipeline: etl-demo
    stage: extract
    source: customers
spec:
  template:
    metadata:
      labels:
        pipeline: etl-demo
        stage: extract
    spec:
      containers:
      - name: extractor
        image: my-registry/data-extractor:v1.0
        env:
        - name: SOURCE_TYPE
          value: \"customers\"\n        - name: OUTPUT_PATH\n          value: \"/data/customers.json\"\n        command:\n        - /bin/sh\n        - -c\n        - |\n          echo \"Starting customer data extraction...\"\n          \n          # Simulate data extraction\n          extract-data --source=customers --output=/data/customers.json\n          \n          echo \"Customer extraction complete, signaling barrier\"\n          koncli barrier arrive extract-complete\n        volumeMounts:\n        - name: data-volume\n          mountPath: /data\n      volumes:\n      - name: data-volume\n        persistentVolumeClaim:\n          claimName: etl-data-pvc\n      restartPolicy: Never\n---\napiVersion: batch/v1\nkind: Job\nmetadata:\n  name: extract-orders\n  labels:\n    pipeline: etl-demo\n    stage: extract\n    source: orders\nspec:\n  template:\n    metadata:\n      labels:\n        pipeline: etl-demo\n        stage: extract\n    spec:\n      containers:\n      - name: extractor\n        image: my-registry/data-extractor:v1.0\n        env:\n        - name: SOURCE_TYPE\n          value: \"orders\"\n        - name: OUTPUT_PATH\n          value: \"/data/orders.json\"\n        command:\n        - /bin/sh\n        - -c\n        - |\n          echo \"Starting order data extraction...\"\n          extract-data --source=orders --output=/data/orders.json\n          echo \"Order extraction complete, signaling barrier\"\n          koncli barrier arrive extract-complete\n        volumeMounts:\n        - name: data-volume\n          mountPath: /data\n      volumes:\n      - name: data-volume\n        persistentVolumeClaim:\n          claimName: etl-data-pvc\n      restartPolicy: Never\n# Add 3 more extract jobs for products, inventory, and analytics...\n```\n\n### 3. Transform Jobs\n\n```yaml\n# transform-jobs.yaml\napiVersion: batch/v1\nkind: Job\nmetadata:\n  name: transform-customer-orders\n  labels:\n    pipeline: etl-demo\n    stage: transform\nspec:\n  template:\n    metadata:\n      labels:\n        pipeline: etl-demo\n        stage: transform\n    spec:\n      initContainers:\n      - name: wait-extract\n        image: logiciq/koncli:latest\n        command:\n        - koncli\n        - barrier\n        - wait\n        - extract-complete\n        - --timeout=2h\n      containers:\n      - name: transformer\n        image: my-registry/data-transformer:v1.0\n        env:\n        - name: TRANSFORM_TYPE\n          value: \"customer-orders\"\n        command:\n        - /bin/sh\n        - -c\n        - |\n          echo \"All extracts complete, starting customer-order transformation...\"\n          \n          # Join customer and order data\n          transform-data \\\n            --input-customers=/data/customers.json \\\n            --input-orders=/data/orders.json \\\n            --output=/data/customer-orders-transformed.json\n          \n          echo \"Customer-order transformation complete, signaling barrier\"\n          koncli barrier arrive transform-complete\n        volumeMounts:\n        - name: data-volume\n          mountPath: /data\n      volumes:\n      - name: data-volume\n        persistentVolumeClaim:\n          claimName: etl-data-pvc\n      restartPolicy: Never\n# Add 2 more transform jobs...\n```\n\n### 4. Load Jobs\n\n```yaml\n# load-jobs.yaml\napiVersion: batch/v1\nkind: Job\nmetadata:\n  name: load-warehouse\n  labels:\n    pipeline: etl-demo\n    stage: load\nspec:\n  template:\n    metadata:\n      labels:\n        pipeline: etl-demo\n        stage: load\n    spec:\n      initContainers:\n      - name: wait-transform\n        image: logiciq/koncli:latest\n        command:\n        - koncli\n        - barrier\n        - wait\n        - transform-complete\n        - --timeout=1h\n      - name: acquire-load-permit\n        image: logiciq/koncli:latest\n        command:\n        - koncli\n        - semaphore\n        - acquire\n        - load-concurrency\n        - --wait\n        - --ttl=30m\n      containers:\n      - name: loader\n        image: my-registry/data-loader:v1.0\n        env:\n        - name: TARGET_DB\n          value: \"warehouse\"\n        command:\n        - /bin/sh\n        - -c\n        - |\n          echo \"All transforms complete, loading to warehouse...\"\n          \n          # Load transformed data\n          load-data \\\n            --source=/data/customer-orders-transformed.json \\\n            --target=warehouse \\\n            --table=customer_orders\n          \n          echo \"Warehouse load complete, releasing permit\"\n          koncli semaphore release load-concurrency\n        volumeMounts:\n        - name: data-volume\n          mountPath: /data\n      volumes:\n      - name: data-volume\n        persistentVolumeClaim:\n          claimName: etl-data-pvc\n      restartPolicy: Never\n```\n\n## Deployment\n\n### 1. Create Persistent Volume\n\n```yaml\n# storage.yaml\napiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  name: etl-data-pvc\nspec:\n  accessModes:\n    - ReadWriteMany\n  resources:\n    requests:\n      storage: 10Gi\n  storageClassName: nfs-client\n```\n\n### 2. Deploy Pipeline\n\n```bash\n# Deploy coordination resources\nkubectl apply -f barriers.yaml\n\n# Deploy storage\nkubectl apply -f storage.yaml\n\n# Deploy extract jobs\nkubectl apply -f extract-jobs.yaml\n\n# Deploy transform jobs (will wait for extracts)\nkubectl apply -f transform-jobs.yaml\n\n# Deploy load jobs (will wait for transforms)\nkubectl apply -f load-jobs.yaml\n```\n\n## Monitoring\n\n### Check Pipeline Progress\n\n```bash\n# Monitor barriers\nkubectl get barriers -l pipeline=etl-demo\nkoncli barrier status extract-complete\nkoncli barrier status transform-complete\n\n# Monitor semaphore\nkoncli semaphore status load-concurrency\n\n# Watch jobs\nkubectl get jobs -l pipeline=etl-demo -w\n\n# Check job logs\nkubectl logs -l stage=extract\nkubectl logs -l stage=transform\nkubectl logs -l stage=load\n```\n\n### Pipeline Status Dashboard\n\n```bash\n#!/bin/bash\n# pipeline-status.sh\n\necho \"=== ETL Pipeline Status ===\"\necho\n\necho \"Extract Stage:\"\nkubectl get jobs -l stage=extract --no-headers | awk '{print $1 \": \" $2}'\necho \"Barrier: $(koncli barrier status extract-complete --output=json | jq -r '.phase')\"\necho \"Arrived: $(koncli barrier status extract-complete --output=json | jq -r '.arrived')/5\"\necho\n\necho \"Transform Stage:\"\nkubectl get jobs -l stage=transform --no-headers | awk '{print $1 \": \" $2}'\necho \"Barrier: $(koncli barrier status transform-complete --output=json | jq -r '.phase')\"\necho \"Arrived: $(koncli barrier status transform-complete --output=json | jq -r '.arrived')/3\"\necho\n\necho \"Load Stage:\"\nkubectl get jobs -l stage=load --no-headers | awk '{print $1 \": \" $2}'\necho \"Semaphore: $(koncli semaphore status load-concurrency --output=json | jq -r '.available')/2 permits available\"\n```\n\n## Error Handling\n\n### Timeout Handling\n\n```yaml\n# Add timeout handling to jobs\nspec:\n  activeDeadlineSeconds: 3600  # 1 hour timeout\n  backoffLimit: 3              # Retry up to 3 times\n```\n\n### Cleanup on Failure\n\n```bash\n#!/bin/bash\n# cleanup-pipeline.sh\n\necho \"Cleaning up failed pipeline...\"\n\n# Delete jobs\nkubectl delete jobs -l pipeline=etl-demo\n\n# Reset barriers\nkubectl delete barriers -l pipeline=etl-demo\nkubectl apply -f barriers.yaml\n\n# Reset semaphore\nkubectl delete semaphore load-concurrency\nkubectl apply -f - <<EOF\napiVersion: konductor.io/v1\nkind: Semaphore\nmetadata:\n  name: load-concurrency\nspec:\n  permits: 2\n  ttl: 30m\nEOF\n\necho \"Pipeline reset complete\"\n```\n\n## Advanced Patterns\n\n### Conditional Processing\n\n```yaml\n# Add conditions based on data quality\ninitContainers:\n- name: check-data-quality\n  image: my-registry/data-validator:v1.0\n  command:\n  - /bin/sh\n  - -c\n  - |\n    if validate-data /data/customers.json; then\n      echo \"Data quality check passed\"\n      koncli barrier arrive extract-complete\n    else\n      echo \"Data quality check failed\"\n      exit 1\n    fi\n```\n\n### Dynamic Scaling\n\n```yaml\n# Scale based on data volume\nspec:\n  parallelism: 1\n  completions: 1\n  template:\n    spec:\n      containers:\n      - name: dynamic-processor\n        command:\n        - /bin/sh\n        - -c\n        - |\n          DATA_SIZE=$(wc -l < /data/input.json)\n          if [ $DATA_SIZE -gt 10000 ]; then\n            # Large dataset - request more permits\n            koncli semaphore acquire load-concurrency --permits=2\n          else\n            # Small dataset - single permit\n            koncli semaphore acquire load-concurrency --permits=1\n          fi\n```\n\n## Best Practices\n\n1. **Use meaningful names** for barriers and semaphores\n2. **Set appropriate timeouts** based on expected processing time\n3. **Monitor resource usage** and adjust semaphore permits accordingly\n4. **Implement proper cleanup** for failed pipelines\n5. **Use labels** for easy resource management\n6. **Test failure scenarios** to ensure robust error handling\n\n## Related Examples\n\n- [Batch Processing](./batch-processing.md) - Semaphore usage patterns\n- [Database Migrations](./database-migrations.md) - Singleton execution\n- [MapReduce Workflows](./mapreduce.md) - Large-scale coordination