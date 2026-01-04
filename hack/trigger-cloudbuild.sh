#!/bin/bash
set -e

echo "Submitting build to Cloud Build..."
gcloud builds submit --async \
--substitutions=SHORT_SHA=$(git rev-parse --short HEAD) \
--config image-build/cloudbuild.yaml .
