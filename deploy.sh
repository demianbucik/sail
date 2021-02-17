#!/bin/bash

gcloud functions \
    deploy \
    send-mail \
    --source=. \
    --region=europe-west3 \
    --trigger-http \
    --allow-unauthenticated \
    --entry-point=SendEmailHandler \
    --memory=256MB \
    --runtime=go113 \
    --timeout=30 \
    --max-instances=2 \
    --env-vars-file=./env.yaml
