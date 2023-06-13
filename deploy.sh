#!/usr/bin/env bash

NAME="$1"
ENV_FILE="${2:-env.yaml}"

gcloud alpha functions deploy \
    "$NAME" \
    --gen2 \
    --trigger-http \
    --allow-unauthenticated \
    --source=. \
    --region=europe-west3 \
    --entry-point=SendEmailHandler \
    --timeout=10 \
    --max-instances=2 \
    --concurrency=1 \
    --cpu=0.083 \
    --memory=128Mi \
    --runtime=go120 \
    --env-vars-file="$ENV_FILE"

