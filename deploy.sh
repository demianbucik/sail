#!/bin/bash

NAME="$1"
ENV_FILE="${2:-env.yaml}"

gcloud alpha functions \
    deploy \
    "$NAME" \
    --gen2 \
    --trigger-http \
    --allow-unauthenticated \
    --source=. \
    --region=europe-west3 \
    --entry-point=SendEmailHandler \
    --memory=256MB \
    --cpu=1 \
    --runtime=go120 \
    --timeout=10 \
    --max-instances=2 \
    --concurrency=50 \
    --env-vars-file="$ENV_FILE"
