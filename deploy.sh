#!/bin/bash

gcloud functions \
    deploy \
    staging-send-email \
    --source=. \
    --region=europe-west3 \
    --trigger-http \
    --allow-unauthenticated \
    --entry-point=SendEmailHandler \
    --memory=128MB \
    --runtime=go116 \
    --timeout=5 \
    --max-instances=2 \
    --env-vars-file=./env.yaml
