#!/bin/bash

gcloud functions \
    deploy \
    send-email \
    --source=. \
    --region=europe-west3 \
    --trigger-http \
    --allow-unauthenticated \
    --entry-point=SendEmailHandler \
    --memory=256MB \
    --runtime=go113 \
    --timeout=10 \
    --max-instances=2 \
    --env-vars-file=./env.yaml
